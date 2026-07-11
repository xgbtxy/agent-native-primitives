package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xgbtxy/agent-native-primitives/internal/discovery"
	"github.com/xgbtxy/agent-native-primitives/internal/managed"
	"github.com/xgbtxy/agent-native-primitives/internal/model"
	"github.com/xgbtxy/agent-native-primitives/internal/search"
	"github.com/xgbtxy/agent-native-primitives/internal/tooling"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var version = "0.1.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "scan":
		return runScan(args[1:])
	case "find":
		return runFind(args[1:])
	case "show":
		return runShow(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "repair":
		return runRepair(args[1:])
	case "exec":
		return runManaged(args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(args []string) error {
	jsonOutput := takeBool(&args, "--json")
	project, err := takeValue(&args, "--project", ".")
	if err != nil {
		return err
	}
	if len(args) != 0 {
		return fmt.Errorf("unexpected scan arguments: %s", strings.Join(args, " "))
	}
	index, err := discovery.Scan(discovery.Options{Project: project})
	if err != nil {
		return err
	}
	present, projectCount, missingRuntime := counts(index)
	result := struct {
		OK             bool   `json:"ok"`
		ScopeID        string `json:"scope_id"`
		Project        string `json:"project"`
		Present        int    `json:"present"`
		ProjectDefined int    `json:"project_defined"`
		MissingRuntime int    `json:"missing_runtime"`
		Persistence    string `json:"persistence"`
	}{true, index.Scope.ID, index.Scope.ProjectName, present, projectCount, missingRuntime, "none"}
	if jsonOutput {
		return printJSON(result)
	}
	fmt.Printf("Resolved %d capabilities (%d project-defined, %d missing runtime)\n", present, projectCount, missingRuntime)
	fmt.Printf("Scope: %s (%s)\n", index.Scope.ID, index.Scope.ProjectName)
	fmt.Println("Persistence: none")
	return nil
}

func runFind(args []string) error {
	jsonOutput := takeBool(&args, "--json")
	project, err := takeValue(&args, "--project", ".")
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("find requires an intent, for example: tooltruth find \"search logs\"")
	}
	query := strings.Join(args, " ")
	options := discovery.Options{Project: project}
	index, err := discovery.Scan(options)
	if err != nil {
		return err
	}
	result := search.Find(index, query)
	if result.Match == nil {
		if exact, ok, err := discovery.ResolveExact(query, options); err != nil {
			return err
		} else if ok {
			index.Tools = append(index.Tools, exact)
			result = search.Find(index, query)
		}
	}
	if jsonOutput {
		return printJSON(result)
	}
	if result.Match == nil {
		fmt.Println(result.Status)
		return nil
	}
	candidate := result.Match
	fmt.Println(candidate.Command)
	if candidate.Claim != "" {
		fmt.Printf("  Claim: %s\n", candidate.Claim)
	}
	fmt.Printf("  Signal: semantics=%s; availability=%s; behavior=%s; match=%s\n", candidate.Signal.Semantics, candidate.Signal.Availability, candidate.Signal.Behavior, candidate.Signal.Match)
	if candidate.DeclaredExample != "" {
		fmt.Printf("  Declared example: %s\n", candidate.DeclaredExample)
	}
	return nil
}

func runShow(args []string) error {
	jsonOutput := takeBool(&args, "--json")
	project, err := takeValue(&args, "--project", ".")
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return errors.New("show requires exactly one tool id or command")
	}
	index, err := discovery.Scan(discovery.Options{Project: project})
	if err != nil {
		return err
	}
	wanted := strings.ToLower(args[0])
	for _, tool := range index.Tools {
		if strings.ToLower(tool.ID) != wanted && strings.ToLower(tool.Command) != wanted {
			continue
		}
		if jsonOutput {
			return printJSON(tool)
		}
		fmt.Printf("%s (%s)\n", tool.Command, tool.Status)
		fmt.Println("  id:", tool.ID)
		fmt.Println("  semantic source:", tool.SemanticSource)
		fmt.Println("  resolver source:", tool.ResolverSource)
		fmt.Println("  resolved path:", tool.ResolvedPath)
		fmt.Println("  description:", tool.Description)
		fmt.Println("  capabilities:", strings.Join(tool.Capabilities, ", "))
		return nil
	}
	return fmt.Errorf("tool %q is not present in the active scope", args[0])
}

func runDoctor(args []string) error {
	jsonOutput := takeBool(&args, "--json")
	project, err := takeValue(&args, "--project", ".")
	if err != nil {
		return err
	}
	if len(args) == 1 {
		return runToolDoctor(args[0], project, jsonOutput)
	}
	if len(args) != 0 {
		return fmt.Errorf("unexpected doctor arguments: %s", strings.Join(args, " "))
	}
	started := time.Now()
	index, err := discovery.Scan(discovery.Options{Project: project})
	if err != nil {
		return err
	}
	present, projectCount, missingRuntime := counts(index)
	report := struct {
		OK             bool   `json:"ok"`
		ScopeID        string `json:"scope_id"`
		Project        string `json:"project"`
		Resolver       string `json:"resolver"`
		Present        int    `json:"present"`
		ProjectDefined int    `json:"project_defined"`
		MissingRuntime int    `json:"missing_runtime"`
		ElapsedMS      int64  `json:"elapsed_ms"`
		ExecProbes     int    `json:"executable_probes"`
		Persistence    string `json:"persistence"`
	}{true, index.Scope.ID, index.Scope.ProjectName, index.Scope.Resolver, present, projectCount, missingRuntime, time.Since(started).Milliseconds(), 0, "none"}
	if jsonOutput {
		return printJSON(report)
	}
	fmt.Println("OK: active capability scope resolved")
	fmt.Printf("Scope: %s (%s)\n", report.ScopeID, report.Project)
	fmt.Printf("Capabilities: %d present, %d project-defined, %d missing runtime\n", present, projectCount, missingRuntime)
	fmt.Printf("Resolver: %s; executable probes: 0; persistence: none; elapsed: %dms\n", report.Resolver, report.ElapsedMS)
	return nil
}

func runToolDoctor(id, project string, jsonOutput bool) error {
	if !tooling.Supports(id) {
		return fmt.Errorf("no curated health check for %q", id)
	}
	index, err := discovery.Scan(discovery.Options{Project: project})
	if err != nil {
		return err
	}
	path := ""
	source := "none"
	for _, tool := range index.Tools {
		if tool.ID == id {
			path = tool.ResolvedPath
			source = tool.ResolverSource
			break
		}
	}
	result := tooling.Doctor(context.Background(), id, path, source)
	if path != "" && (result.Status == "ready" || result.Status == "broken") {
		home, homeErr := managed.Home("")
		digest, digestErr := managed.HashFile(path)
		if homeErr == nil && digestErr == nil {
			if err := managed.WriteHealth(home, managed.Health{ID: id, Digest: digest, Probe: result.Check, Status: result.Status, CheckedAt: time.Now().UTC()}); err != nil {
				return err
			}
		}
	}
	if jsonOutput {
		if err := printJSON(result); err != nil {
			return err
		}
		if result.Status != "ready" {
			return fmt.Errorf("%s health status is %s", id, result.Status)
		}
		return nil
	}
	fmt.Printf("%s: %s\n", result.ID, result.Status)
	fmt.Printf("  source: %s\n", result.Source)
	if result.Path != "" {
		fmt.Printf("  path: %s\n", result.Path)
	}
	fmt.Printf("  check: %s\n", result.Check)
	if result.Observation != "" {
		fmt.Printf("  observation: %s\n", result.Observation)
	}
	if result.Status != "ready" {
		return fmt.Errorf("%s health status is %s", id, result.Status)
	}
	return nil
}

func runRepair(args []string) error {
	jsonOutput := takeBool(&args, "--json")
	if len(args) != 1 {
		return errors.New("repair requires exactly one supported tool id")
	}
	home, err := managed.Home("")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	progress := io.Writer(os.Stderr)
	if jsonOutput {
		progress = io.Discard
	}
	record, err := tooling.Repair(ctx, home, args[0], progress)
	if err != nil {
		return err
	}
	result := struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Version   string `json:"version"`
		Command   string `json:"command"`
		Behavior  string `json:"behavior"`
		Isolation string `json:"isolation"`
	}{record.Manifest.ID, "ready", record.Manifest.Version, "tooltruth exec " + record.Manifest.ID + " --", "help_signature_probe_passed", "tooltruth_managed"}
	if jsonOutput {
		return printJSON(result)
	}
	fmt.Printf("%s %s: ready\n", result.ID, result.Version)
	fmt.Printf("Command: %s\n", result.Command)
	fmt.Println("Isolation: Tooltruth-managed; global PATH unchanged")
	return nil
}

func runManaged(args []string) error {
	if len(args) == 0 {
		return errors.New("exec requires a managed tool id")
	}
	id := args[0]
	if !tooling.Supports(id) {
		return fmt.Errorf("managed tool id %q is not supported", id)
	}
	args = args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	home, err := managed.Home("")
	if err != nil {
		return err
	}
	record, ok, err := managed.Load(home, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("managed tool %q is absent or failed hash verification; run tooltruth repair %s", id, id)
	}
	return tooling.RunManaged(context.Background(), record, args, os.Stdin, os.Stdout, os.Stderr)
}

func counts(index model.Index) (present int, project int, missingRuntime int) {
	for _, tool := range index.Tools {
		if tool.Status == "present" || tool.Status == "present_unclassified" || tool.Status == "ready" {
			present++
		}
		if tool.ProjectDefined {
			project++
		}
		if tool.Status == "missing_runtime" {
			missingRuntime++
		}
	}
	return present, project, missingRuntime
}

func takeBool(args *[]string, name string) bool {
	out := (*args)[:0]
	found := false
	for _, arg := range *args {
		if arg == name {
			found = true
			continue
		}
		out = append(out, arg)
	}
	*args = out
	return found
}

func takeValue(args *[]string, name, fallback string) (string, error) {
	input := *args
	out := input[:0]
	value := fallback
	found := false
	for i := 0; i < len(input); i++ {
		if input[i] != name {
			out = append(out, input[i])
			continue
		}
		if found {
			return "", fmt.Errorf("%s provided more than once", name)
		}
		if i+1 >= len(input) {
			return "", fmt.Errorf("%s requires a value", name)
		}
		found = true
		value = input[i+1]
		i++
	}
	*args = out
	return value, nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func printUsage() {
	name := filepath.Base(os.Args[0])
	fmt.Printf(`Tooltruth resolves active local CLI capabilities for AI agents.

Usage:
  %s scan [--project DIR] [--json]
  %s find <intent> [--project DIR] [--json]
  %s show <tool> [--project DIR] [--json]
  %s doctor [<tool>] [--project DIR] [--json]
  %s repair <tool> [--json]
  %s exec <managed-tool> -- [ARGS...]
  %s version
`, name, name, name, name, name, name, name)
}
