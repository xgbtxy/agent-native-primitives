package invocation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xgbtxy/agent-native-primitives/internal/managed"
	"github.com/xgbtxy/agent-native-primitives/internal/model"
)

const (
	ProbeTimeout = 5 * time.Second
	outputLimit  = 256 << 10
)

type probeStyle string

const (
	suffixLongHelp  probeStyle = "suffix_long_help"
	suffixShortHelp probeStyle = "suffix_short_help"
	goHelp          probeStyle = "go_help"
	curlHelpAll     probeStyle = "curl_help_all"
)

type Recipe struct {
	ID             string
	Commands       []string
	Style          probeStyle
	DefaultDepth   int
	DepthByFirst   map[string]int
	allowedPaths   [][]string
	blockedPaths   [][]string
	allowedExits   map[int]bool
	singleDashLong bool
}

var recipes = []Recipe{
	{ID: "gh", Commands: []string{"gh"}, Style: suffixLongHelp, DefaultDepth: 1, DepthByFirst: map[string]int{
		"extension": 2, "issue": 2, "pr": 2, "release": 2, "repo": 2,
	}, allowedPaths: [][]string{
		{"api"}, {"browse"}, {"status"},
		{"issue", "create"}, {"issue", "list"}, {"issue", "view"},
		{"pr", "checks"}, {"pr", "checkout"}, {"pr", "create"}, {"pr", "diff"}, {"pr", "list"}, {"pr", "view"},
		{"release", "download"}, {"release", "list"}, {"release", "view"},
		{"repo", "clone"}, {"repo", "list"}, {"repo", "view"},
	}, blockedPaths: [][]string{{"extension", "exec"}}},
	{ID: "git", Commands: []string{"git"}, Style: suffixShortHelp, DefaultDepth: 1, allowedPaths: [][]string{
		{"blame"}, {"branch"}, {"diff"}, {"grep"}, {"log"}, {"show"}, {"stash"}, {"status"}, {"worktree"},
	}, allowedExits: map[int]bool{0: true, 129: true}},
	{ID: "go", Commands: []string{"go"}, Style: goHelp, DefaultDepth: 1, allowedPaths: [][]string{
		{"build"}, {"env"}, {"list"}, {"test"},
	}, blockedPaths: [][]string{{"run"}, {"tool"}}, singleDashLong: true},
	{ID: "ripgrep", Commands: []string{"rg"}, Style: suffixLongHelp},
	{ID: "fd", Commands: []string{"fd", "fdfind"}, Style: suffixLongHelp},
	{ID: "jq", Commands: []string{"jq"}, Style: suffixLongHelp},
	{ID: "curl", Commands: []string{"curl"}, Style: curlHelpAll},
	{ID: "uv", Commands: []string{"uv"}, Style: suffixLongHelp, DefaultDepth: 1, DepthByFirst: map[string]int{"pip": 2, "tool": 2}, allowedPaths: [][]string{
		{"add"}, {"lock"}, {"remove"}, {"sync"}, {"tree"}, {"venv"},
		{"pip", "install"}, {"pip", "list"}, {"pip", "show"},
	}, blockedPaths: [][]string{{"run"}, {"tool", "run"}}},
	{ID: "binwalk", Commands: []string{"binwalk"}, Style: suffixLongHelp},
}

type FlagObservation struct {
	Token     string `json:"token"`
	Canonical string `json:"canonical"`
	Status    string `json:"status"`
}

type Evidence struct {
	Scope                 string   `json:"scope"`
	ExecutableSHA256      string   `json:"executable_sha256"`
	ProbeArgv             []string `json:"probe_argv"`
	ProbeExitCode         int      `json:"probe_exit_code"`
	ProbeOutputSHA256     string   `json:"probe_output_sha256"`
	ProbeOutputBytes      int      `json:"probe_output_bytes"`
	ProbeOutputTruncated  bool     `json:"probe_output_truncated,omitempty"`
	HelpCompleteness      string   `json:"help_completeness"`
	ShellAliasesEvaluated bool     `json:"shell_aliases_evaluated"`
}

type Result struct {
	Scope      model.ResultScope `json:"scope"`
	ToolID     string            `json:"tool_id"`
	Command    string            `json:"command"`
	Subcommand []string          `json:"subcommand,omitempty"`
	Flags      []FlagObservation `json:"flags"`
	Status     string            `json:"status"`
	Evidence   *Evidence         `json:"evidence,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

type RunOutput struct {
	Text      string
	ExitCode  int
	Truncated bool
	TimedOut  bool
}

type Runner interface {
	Run(context.Context, string, []string) RunOutput
}

type OSRunner struct{}

func (OSRunner) Run(parent context.Context, path string, args []string) RunOutput {
	ctx, cancel := context.WithTimeout(parent, ProbeTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	var output limitedBuffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}
	}
	return RunOutput{
		Text: output.String(), ExitCode: exitCode, Truncated: output.truncated,
		TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded),
	}
}

func RecipeForCommand(command string) (Recipe, bool) {
	command = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(command), ".exe"))
	for _, recipe := range recipes {
		for _, candidate := range recipe.Commands {
			if command == candidate {
				return recipe, true
			}
		}
	}
	return Recipe{}, false
}

func UnsupportedResult(requestedCommand string, args []string) Result {
	return Result{
		Command: requestedCommand,
		Flags:   requestedFlags(args),
		Status:  "unsupported_command",
		Reason:  "no compiled-in safe help probe exists for this command; Tooltruth did not execute it",
	}
}

func Validate(ctx context.Context, recipe Recipe, tool model.Tool, requestedCommand string, args []string, runner Runner) Result {
	result := Result{ToolID: recipe.ID, Command: requestedCommand, Flags: requestedFlags(args)}
	if tool.ResolvedPath == "" || (tool.Status != "present" && tool.Status != "ready") {
		result.Status = "command_not_resolved"
		result.Reason = "the curated command is not available in the active Tooltruth scope"
		return result
	}
	if runner == nil {
		runner = OSRunner{}
	}
	subcommands, ambiguous := deriveSubcommands(recipe, args)
	result.Subcommand = subcommands
	if ambiguous {
		result.Status = "unverified"
		result.Reason = "a flag appears before the subcommand path; Tooltruth will not guess the active help surface"
		return result
	}
	if len(result.Flags) == 0 {
		result.Status = "no_requested_flags"
		result.Reason = "validate currently observes flags only; positional arguments are not claimed"
		return result
	}
	if blockedProbePath(recipe, subcommands) {
		result.Status = "unverified"
		result.Reason = "this subcommand can pass arguments to another program; Tooltruth will not guess which parser owns the requested flags"
		return result
	}
	if recipe.allowedPaths != nil && !probePathAllowed(recipe, subcommands) {
		result.Status = "unverified"
		result.Reason = "the requested command path is not in the small compiled-in pilot set; Tooltruth did not execute it"
		return result
	}
	probeArgs := buildProbeArgs(recipe, subcommands)
	executableDigest, err := managed.HashFile(tool.ResolvedPath)
	if err != nil {
		result.Status = "unverified"
		result.Reason = "resolved executable could not be digest-bound before probing"
		return result
	}
	probe := runner.Run(ctx, tool.ResolvedPath, probeArgs)
	if probe.TimedOut {
		result.Status = "unverified"
		result.Reason = "curated help probe timed out"
		return result
	}
	if !exitCodeAllowed(recipe, probe.ExitCode) {
		result.Status = "unverified"
		result.Reason = "curated probe returned an unexpected exit code"
		return result
	}
	if !looksLikeHelp(recipe, subcommands, probe.Text) {
		result.Status = "unverified"
		result.Reason = "curated probe did not return a recognizable help surface"
		return result
	}
	postProbeDigest, err := managed.HashFile(tool.ResolvedPath)
	if err != nil {
		result.Status = "unverified"
		result.Reason = "resolved executable could not be digest-bound after probing"
		return result
	}
	if postProbeDigest != executableDigest {
		result.Status = "unverified"
		result.Reason = "resolved executable changed while the help probe was running"
		return result
	}
	helpDigest := sha256.Sum256([]byte(probe.Text))
	result.Evidence = &Evidence{
		Scope: "local_executable_help", ExecutableSHA256: executableDigest,
		ProbeArgv: append([]string{requestedCommand}, probeArgs...), ProbeExitCode: probe.ExitCode,
		ProbeOutputSHA256: hex.EncodeToString(helpDigest[:]), ProbeOutputBytes: len(probe.Text),
		ProbeOutputTruncated: probe.Truncated, HelpCompleteness: "unknown", ShellAliasesEvaluated: false,
	}
	observed := extractFlags(probe.Text, recipe.singleDashLong)
	missing := false
	unverified := false
	for i := range result.Flags {
		flag := &result.Flags[i]
		if !recipe.singleDashLong && strings.HasPrefix(flag.Canonical, "-") && !strings.HasPrefix(flag.Canonical, "--") && len([]rune(flag.Canonical)) > 2 {
			flag.Status = "combined_short_flag_unverified"
			unverified = true
			continue
		}
		if observed[flag.Canonical] {
			flag.Status = "observed_in_local_help"
		} else if probe.Truncated {
			flag.Status = "help_truncated_unverified"
			unverified = true
		} else {
			flag.Status = "not_observed_in_local_help"
			missing = true
		}
	}
	if missing {
		result.Status = "requested_flags_not_observed_in_local_help"
		result.Reason = "one or more requested flags were not observed in the local help surface; help completeness is unknown and this is not proof that the executable rejects them"
	} else if unverified {
		result.Status = "partially_unverified"
		result.Reason = "at least one requested flag could not be matched without guessing"
	} else {
		result.Status = "requested_flags_observed_in_local_help"
		result.Reason = "all requested flags were observed in the local help surface; positional values and runtime behavior remain unverified"
	}
	return result
}

func probePathAllowed(recipe Recipe, subcommands []string) bool {
	for _, allowed := range recipe.allowedPaths {
		if len(allowed) != len(subcommands) {
			continue
		}
		matched := true
		for i := range allowed {
			if !strings.EqualFold(allowed[i], subcommands[i]) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func blockedProbePath(recipe Recipe, subcommands []string) bool {
	for _, blocked := range recipe.blockedPaths {
		if len(subcommands) < len(blocked) {
			continue
		}
		matched := true
		for i := range blocked {
			if !strings.EqualFold(subcommands[i], blocked[i]) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func exitCodeAllowed(recipe Recipe, exitCode int) bool {
	if recipe.allowedExits != nil {
		return recipe.allowedExits[exitCode]
	}
	return exitCode == 0
}

func deriveSubcommands(recipe Recipe, args []string) ([]string, bool) {
	depth := recipe.DefaultDepth
	if len(args) > 0 {
		if selected, ok := recipe.DepthByFirst[strings.ToLower(args[0])]; ok {
			depth = selected
		}
	}
	if depth == 0 {
		return nil, false
	}
	var result []string
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			if len(result) < depth {
				return result, true
			}
			break
		}
		if len(result) < depth {
			if !safeSubcommand.MatchString(arg) {
				return result, true
			}
			result = append(result, arg)
		}
		if len(result) == depth {
			break
		}
	}
	return result, false
}

func buildProbeArgs(recipe Recipe, subcommands []string) []string {
	switch recipe.Style {
	case suffixShortHelp:
		return append(append([]string(nil), subcommands...), "-h")
	case goHelp:
		if len(subcommands) == 1 && strings.EqualFold(subcommands[0], "test") {
			return []string{"help", "testflag"}
		}
		return append([]string{"help"}, subcommands...)
	case curlHelpAll:
		return []string{"--help", "all"}
	default:
		return append(append([]string(nil), subcommands...), "--help")
	}
}

func requestedFlags(args []string) []FlagObservation {
	seen := map[string]bool{}
	var flags []FlagObservation
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			continue
		}
		canonical := arg
		if index := strings.IndexByte(canonical, '='); index >= 0 {
			canonical = canonical[:index]
		}
		if !requestedLongFlag.MatchString(canonical) && !requestedShortFlag.MatchString(canonical) {
			continue
		}
		if seen[canonical] {
			continue
		}
		seen[canonical] = true
		flags = append(flags, FlagObservation{Token: arg, Canonical: canonical, Status: "unverified"})
	}
	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Canonical < flags[j].Canonical })
	return flags
}

var (
	safeSubcommand     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	requestedLongFlag  = regexp.MustCompile(`^--[A-Za-z0-9][A-Za-z0-9_-]*$`)
	requestedShortFlag = regexp.MustCompile(`^-[A-Za-z]+$`)
	longFlag           = regexp.MustCompile(`(^|[[:space:],|(\[])(--[A-Za-z0-9][A-Za-z0-9_-]*)`)
	optionalNoLongFlag = regexp.MustCompile(`(^|[[:space:],|(\[])(--\[no-\]([A-Za-z0-9][A-Za-z0-9_-]*))`)
	shortFlag          = regexp.MustCompile(`(?:^|[\s,|[(])(-[A-Za-z0-9])(?:$|[\s,=|)\]])`)
	singleDashWordFlag = regexp.MustCompile(`(^|[[:space:],|(\[])(-[A-Za-z][A-Za-z0-9_-]+)(?:$|[[:space:],=|)\]])`)
	helpHeading        = regexp.MustCompile(`(?im)^usage(?::|[ \t]|$)`)
)

func extractFlags(help string, allowSingleDashWords bool) map[string]bool {
	result := map[string]bool{}
	for _, match := range longFlag.FindAllStringSubmatch(help, -1) {
		if len(match) == 3 {
			result[match[2]] = true
		}
	}
	for _, match := range optionalNoLongFlag.FindAllStringSubmatch(help, -1) {
		if len(match) == 4 {
			result["--"+match[3]] = true
			result["--no-"+match[3]] = true
		}
	}
	for _, match := range shortFlag.FindAllStringSubmatch(help, -1) {
		if len(match) == 2 {
			result[match[1]] = true
		}
	}
	if allowSingleDashWords {
		for _, match := range singleDashWordFlag.FindAllStringSubmatch(help, -1) {
			if len(match) == 3 {
				result[match[2]] = true
			}
		}
	}
	return result
}

func looksLikeHelp(recipe Recipe, subcommands []string, output string) bool {
	if helpHeading.MatchString(output) {
		return true
	}
	if recipe.Style == curlHelpAll {
		return len(extractFlags(output, false)) >= 10
	}
	return recipe.Style == goHelp && len(subcommands) == 1 && strings.EqualFold(subcommands[0], "test") &&
		strings.Contains(output, "flags are recognized by the 'go test' command")
}

type limitedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := outputLimit - b.buffer.Len()
	if remaining <= 0 {
		b.truncated = true
		return original, nil
	}
	if len(value) > remaining {
		value = value[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(value)
	return original, nil
}

func (b *limitedBuffer) String() string { return b.buffer.String() }
