package facts

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xgbtxy/agent-native-primitives/internal/model"
)

const (
	bundleTimeout = 4 * time.Second
	probeTimeout  = 3 * time.Second
	outputLimit   = 4096
)

type probeSpec struct {
	Args []string
}

var probes = map[string]probeSpec{
	"ripgrep": {Args: []string{"--version"}},
	"fd":      {Args: []string{"--version"}},
	"git":     {Args: []string{"--version"}},
	"jq":      {Args: []string{"--version"}},
	"yq":      {Args: []string{"--version"}},
	"curl":    {Args: []string{"--version"}},
	"gh":      {Args: []string{"--version"}},
	"go":      {Args: []string{"version"}},
	"python":  {Args: []string{"--version"}},
	"uv":      {Args: []string{"--version"}},
	"uvx":     {Args: []string{"--version"}},
	"node":    {Args: []string{"--version"}},
	"docker":  {Args: []string{"--version"}},
	"ffmpeg":  {Args: []string{"-version"}},
	"ffprobe": {Args: []string{"-version"}},
	"make":    {Args: []string{"--version"}},
}

type CommandFact struct {
	Command        string `json:"command"`
	Availability   string `json:"availability"`
	Version        string `json:"version,omitempty"`
	Implementation string `json:"implementation,omitempty"`
	Evidence       string `json:"evidence"`
}

type Bundle struct {
	Scope    model.ResultScope `json:"scope"`
	Commands []CommandFact     `json:"commands"`
	Limits   []string          `json:"limits"`
}

type RunOutput struct {
	Text     string
	ExitCode int
	TimedOut bool
}

type Runner interface {
	Run(context.Context, string, []string) RunOutput
}

type OSRunner struct{}

func (OSRunner) Run(parent context.Context, path string, args []string) RunOutput {
	ctx, cancel := context.WithTimeout(parent, probeTimeout)
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
	return RunOutput{Text: output.String(), ExitCode: exitCode, TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}
}

func Build(ctx context.Context, index model.Index, runner Runner) Bundle {
	var tools []model.Tool
	for _, tool := range index.Tools {
		if tool.ProjectDefined || (tool.Status != "present" && tool.Status != "ready") || tool.ResolvedPath == "" {
			continue
		}
		tools = append(tools, tool)
	}
	return Observe(ctx, model.ResultScope{ID: index.Scope.ID, Project: index.Scope.ProjectName}, tools, runner)
}

func Presence(scope model.ResultScope, tools []model.Tool) Bundle {
	result := make([]CommandFact, len(tools))
	for i, tool := range tools {
		result[i] = baseFact(tool)
	}
	return Bundle{
		Scope:    scope,
		Commands: result,
		Limits: []string{
			"presence only",
			"versions, flags, aliases, shell functions, and runtime behavior are not verified",
		},
	}
}

func Observe(ctx context.Context, scope model.ResultScope, tools []model.Tool, runner Runner) Bundle {
	ctx, cancel := context.WithTimeout(ctx, bundleTimeout)
	defer cancel()
	if runner == nil {
		runner = OSRunner{}
	}
	result := make([]CommandFact, len(tools))
	var wait sync.WaitGroup
	limit := make(chan struct{}, 4)
	for i, tool := range tools {
		fact := baseFact(tool)
		if tool.Version != "" {
			fact.Version = tool.Version
			result[i] = fact
			continue
		}
		spec, ok := probes[tool.ID]
		if !ok {
			result[i] = fact
			continue
		}
		wait.Add(1)
		go func(i int, tool model.Tool, spec probeSpec) {
			defer wait.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			fact := baseFact(tool)
			output := runner.Run(ctx, tool.ResolvedPath, spec.Args)
			if !output.TimedOut && output.ExitCode == 0 {
				fact.Version = extractVersion(tool.ID, output.Text)
				fact.Implementation = identifyImplementation(tool.ID, output.Text)
				if fact.Version != "" || fact.Implementation != "" {
					fact.Evidence = "path_resolved+fixed_version_probe"
				}
			}
			result[i] = fact
		}(i, tool, spec)
	}
	wait.Wait()

	return Bundle{
		Scope:    scope,
		Commands: result,
		Limits: []string{
			"presence and version identity only",
			"flags, aliases, shell functions, and runtime behavior are not verified",
		},
	}
}

func baseFact(tool model.Tool) CommandFact {
	if tool.Managed {
		return CommandFact{Command: tool.Command, Availability: "managed_digest_matched", Evidence: "managed_manifest+digest"}
	}
	return CommandFact{Command: tool.Command, Availability: "active_path_resolved", Evidence: "path_resolved"}
}

func Markdown(bundle Bundle) string {
	pathValues := make([]string, 0, len(bundle.Commands))
	managedValues := make([]string, 0, 1)
	for _, fact := range bundle.Commands {
		value := fact.Command
		if strings.ContainsAny(value, " \t") {
			value = "`" + value + "`"
		}
		if fact.Implementation != "" {
			value += "[" + fact.Implementation + "]"
		}
		if fact.Version != "" {
			value += "@" + fact.Version
		}
		if fact.Availability == "managed_digest_matched" {
			managedValues = append(managedValues, value)
		} else {
			pathValues = append(pathValues, value)
		}
	}
	lines := []string{
		"Verified local command facts (scope " + bundle.Scope.ID + "; trust presence/version without re-checking):",
		"- PATH-resolved: " + strings.Join(pathValues, ", "),
	}
	if len(managedValues) > 0 {
		lines = append(lines, "- Digest-bound managed: "+strings.Join(managedValues, ", "))
	}
	lines = append(lines, "- Limits: presence/version only; flags, aliases, shell functions, and runtime behavior remain unknown.")
	return strings.Join(lines, "\n")
}

var (
	versionPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])v?(\d+\.\d+(?:\.\d+)?(?:[-+][a-z0-9.-]+)?)\b`)
	goVersion      = regexp.MustCompile(`\bgo(\d+\.\d+(?:\.\d+)?)\b`)
)

func extractVersion(id, output string) string {
	if id == "go" {
		match := goVersion.FindStringSubmatch(output)
		if len(match) == 2 {
			return match[1]
		}
	}
	match := versionPattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func identifyImplementation(id, output string) string {
	if id != "yq" {
		return ""
	}
	normalized := strings.ToLower(output)
	if strings.Contains(normalized, "mikefarah") {
		return "mikefarah"
	}
	if strings.Contains(normalized, "kislyuk") {
		return "kislyuk"
	}
	return "unidentified-variant"
}

type limitedBuffer struct {
	buffer bytes.Buffer
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := outputLimit - b.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = b.buffer.Write(value)
	}
	return original, nil
}

func (b *limitedBuffer) String() string { return b.buffer.String() }
