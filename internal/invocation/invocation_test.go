package invocation

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xgbtxy/agent-native-primitives/internal/model"
)

type fakeRunner struct {
	output RunOutput
	path   string
	args   []string
	calls  int
	after  func()
}

func (runner *fakeRunner) Run(_ context.Context, path string, args []string) RunOutput {
	runner.calls++
	runner.path = path
	runner.args = append([]string(nil), args...)
	if runner.after != nil {
		runner.after()
	}
	return runner.output
}

func TestValidateObservesOnlyRequestedFlagsInExactSubcommandHelp(t *testing.T) {
	recipe, ok := RecipeForCommand("gh")
	if !ok {
		t.Fatal("gh recipe missing")
	}
	tool := fakeTool(t, "gh")
	runner := &fakeRunner{output: RunOutput{Text: "Usage: gh pr create [flags]\n  -t, --title string\n      --draft\n"}}
	result := Validate(context.Background(), recipe, tool, "gh", []string{"pr", "create", "--title", "hello", "--fake=yes"}, runner)
	if result.Status != "requested_flags_not_observed_in_local_help" {
		t.Fatalf("unexpected status: %#v", result)
	}
	if !reflect.DeepEqual(runner.args, []string{"pr", "create", "--help"}) {
		t.Fatalf("unsafe or incorrect probe args: %#v", runner.args)
	}
	if len(result.Flags) != 2 || result.Flags[0].Canonical != "--fake" || result.Flags[0].Status != "not_observed_in_local_help" || result.Flags[1].Status != "observed_in_local_help" {
		t.Fatalf("unexpected flag evidence: %#v", result.Flags)
	}
	if result.Evidence == nil || result.Evidence.ExecutableSHA256 == "" || result.Evidence.ProbeOutputSHA256 == "" || result.Evidence.ShellAliasesEvaluated {
		t.Fatalf("missing or overstated evidence: %#v", result.Evidence)
	}
}

func TestValidateNeverPassesUserValuesOrFlagsToProbe(t *testing.T) {
	recipe, _ := RecipeForCommand("git")
	tool := fakeTool(t, "git")
	runner := &fakeRunner{output: RunOutput{Text: "usage: git status [--short]\n  -s, --short\n"}}
	result := Validate(context.Background(), recipe, tool, "git", []string{"status", "--short", "$(dangerous)"}, runner)
	if result.Status != "requested_flags_observed_in_local_help" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !reflect.DeepEqual(runner.args, []string{"status", "-h"}) {
		t.Fatalf("probe inherited intended invocation data: %#v", runner.args)
	}
}

func TestGoTestUsesDedicatedLocalFlagHelp(t *testing.T) {
	recipe, _ := RecipeForCommand("go")
	tool := fakeTool(t, "go")
	runner := &fakeRunner{output: RunOutput{Text: "usage: go test [build/test flags]\n  -run regexp\n"}}
	result := Validate(context.Background(), recipe, tool, "go", []string{"test", "-run", "TestOne", "./..."}, runner)
	if result.Status != "requested_flags_observed_in_local_help" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !reflect.DeepEqual(runner.args, []string{"help", "testflag"}) {
		t.Fatalf("go test did not use its dedicated local flag surface: %#v", runner.args)
	}
}

func TestValidateAbstainsWhenGlobalFlagMakesSubcommandAmbiguous(t *testing.T) {
	recipe, _ := RecipeForCommand("gh")
	tool := fakeTool(t, "gh")
	runner := &fakeRunner{}
	result := Validate(context.Background(), recipe, tool, "gh", []string{"--repo", "owner/repo", "pr", "list"}, runner)
	if result.Status != "unverified" || runner.calls != 0 {
		t.Fatalf("expected no-exec abstention, got %#v calls=%d", result, runner.calls)
	}
}

func TestValidateDoesNotProbeWhenNoFlagsWereRequested(t *testing.T) {
	recipe, _ := RecipeForCommand("rg")
	tool := fakeTool(t, "ripgrep")
	runner := &fakeRunner{}
	result := Validate(context.Background(), recipe, tool, "rg", []string{"needle", "src"}, runner)
	if result.Status != "no_requested_flags" || runner.calls != 0 {
		t.Fatalf("expected no flag observation and no probe: %#v calls=%d", result, runner.calls)
	}
}

func TestValidateTreatsCombinedShortFlagsAsUnverified(t *testing.T) {
	recipe, _ := RecipeForCommand("rg")
	tool := fakeTool(t, "ripgrep")
	runner := &fakeRunner{output: RunOutput{Text: "USAGE: rg [OPTIONS]\n  -n, --line-number\n  -S, --smart-case\n"}}
	result := Validate(context.Background(), recipe, tool, "rg", []string{"-nS", "needle"}, runner)
	if result.Status != "partially_unverified" || result.Flags[0].Status != "combined_short_flag_unverified" {
		t.Fatalf("combined short flag was overstated: %#v", result)
	}
}

func TestValidateDoesNotClaimAbsenceFromTruncatedHelp(t *testing.T) {
	recipe, _ := RecipeForCommand("curl")
	tool := fakeTool(t, "curl")
	runner := &fakeRunner{output: RunOutput{Text: "Usage: curl [options]\n  --url URL\n", Truncated: true}}
	result := Validate(context.Background(), recipe, tool, "curl", []string{"--definitely-later"}, runner)
	if result.Status != "partially_unverified" || result.Flags[0].Status != "help_truncated_unverified" {
		t.Fatalf("truncated help must not prove absence: %#v", result)
	}
}

func TestValidateAbstainsOnTimeoutOrUnrecognizedOutput(t *testing.T) {
	recipe, _ := RecipeForCommand("jq")
	tool := fakeTool(t, "jq")
	for _, output := range []RunOutput{{TimedOut: true}, {Text: "network error"}} {
		result := Validate(context.Background(), recipe, tool, "jq", []string{"--raw-output"}, &fakeRunner{output: output})
		if result.Status != "unverified" || result.Evidence != nil {
			t.Fatalf("bad probe must not create evidence: %#v", result)
		}
	}
}

func TestValidateRejectsUnexpectedProbeExit(t *testing.T) {
	recipe, _ := RecipeForCommand("gh")
	tool := fakeTool(t, "gh")
	result := Validate(context.Background(), recipe, tool, "gh", []string{"unknown", "--helpful"}, &fakeRunner{output: RunOutput{Text: "Usage: gh <command> [flags]\n  --helpful\n", ExitCode: 1}})
	if result.Status != "unverified" || result.Evidence != nil {
		t.Fatalf("failed generic help must not validate a command: %#v", result)
	}
}

func TestValidateRejectsExecutableReplacementRace(t *testing.T) {
	recipe, _ := RecipeForCommand("rg")
	tool := fakeTool(t, "ripgrep")
	runner := &fakeRunner{
		output: RunOutput{Text: "Usage: rg [OPTIONS]\n  --glob GLOB\n"},
		after: func() {
			if err := os.WriteFile(tool.ResolvedPath, []byte("replaced executable identity"), 0700); err != nil {
				t.Fatal(err)
			}
		},
	}
	result := Validate(context.Background(), recipe, tool, "rg", []string{"--glob"}, runner)
	if result.Status != "unverified" || result.Evidence != nil {
		t.Fatalf("evidence was bound to replaced bytes: %#v", result)
	}
}

func TestValidateAbstainsForPassthroughSubcommands(t *testing.T) {
	for _, test := range []struct {
		command string
		args    []string
	}{
		{"go", []string{"run", "main.go", "--app-flag"}},
		{"uv", []string{"run", "script.py", "--app-flag"}},
		{"gh", []string{"extension", "exec", "name", "--app-flag"}},
	} {
		recipe, _ := RecipeForCommand(test.command)
		tool := fakeTool(t, recipe.ID)
		runner := &fakeRunner{}
		result := Validate(context.Background(), recipe, tool, test.command, test.args, runner)
		if result.Status != "unverified" || runner.calls != 0 {
			t.Fatalf("%s passthrough was probed: %#v calls=%d", test.command, result, runner.calls)
		}
	}
}

func TestRecipeCoverageIsExplicit(t *testing.T) {
	if _, ok := RecipeForCommand("python"); ok {
		t.Fatal("python script flags must not be confused with interpreter help")
	}
	if _, ok := RecipeForCommand("unknown"); ok {
		t.Fatal("unknown commands must not receive arbitrary help probes")
	}
	if _, ok := RecipeForCommand("yq"); ok {
		t.Fatal("incompatible yq implementations must not share a generic probe")
	}
	if _, ok := RecipeForCommand("docker"); ok {
		t.Fatal("unverified plugin-capable Docker surfaces must remain outside the pilot")
	}
}

func TestValidateRejectsExternalOrUnknownSubcommandsWithoutExecution(t *testing.T) {
	for _, test := range []struct {
		command string
		args    []string
	}{
		{"git", []string{"evil", "--flag"}},
		{"git", []string{"commit", "--amend"}},
		{"gh", []string{"third-party-extension", "--flag"}},
		{"gh", []string{"pr", "merge", "--squash"}},
		{"uv", []string{"third-party", "--flag"}},
	} {
		recipe, _ := RecipeForCommand(test.command)
		tool := fakeTool(t, recipe.ID)
		runner := &fakeRunner{}
		result := Validate(context.Background(), recipe, tool, test.command, test.args, runner)
		if result.Status != "unverified" || runner.calls != 0 {
			t.Fatalf("%s unknown/external subcommand was executed: %#v calls=%d", test.command, result, runner.calls)
		}
	}
}

func TestExtractFlagsUsesTokenBoundaries(t *testing.T) {
	flags := extractFlags("usage: demo [-q]\n  -q, --quiet\n  --quiet-mode\n  --[no-]short\n  text--not-a-flag\n", false)
	for _, expected := range []string{"-q", "--quiet", "--quiet-mode", "--short", "--no-short"} {
		if !flags[expected] {
			t.Fatalf("expected %s in %#v", expected, flags)
		}
	}
	if flags["--not-a-flag"] {
		t.Fatal("embedded text must not be treated as a flag token")
	}
	if flags["--qui"] {
		t.Fatal("prefix must not be treated as a flag")
	}
}

func TestLooksLikeHelpAcceptsHeadingWithoutColon(t *testing.T) {
	recipe, _ := RecipeForCommand("gh")
	if !looksLikeHelp(recipe, []string{"pr", "create"}, "description\n\nUSAGE\n  gh pr create [flags]\n") {
		t.Fatal("common heading-style help was not recognized")
	}
	if looksLikeHelp(recipe, []string{"pr", "create"}, "the usage rate increased") {
		t.Fatal("prose must not be accepted as a help surface")
	}
}

func TestCurlAllAcceptsAnOptionTableWithoutUsageHeading(t *testing.T) {
	recipe, _ := RecipeForCommand("curl")
	output := " --abstract-unix-socket <path>\n --alt-svc <file>\n --anyauth\n --append\n --basic\n --cacert <file>\n --cert <file>\n --compressed\n --connect-timeout <seconds>\n --fail\n"
	if !looksLikeHelp(recipe, nil, output) {
		t.Fatal("curl's fixed all-options surface was not recognized")
	}
}

func fakeTool(t *testing.T, id string) model.Tool {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool.bin")
	if err := os.WriteFile(path, []byte("test executable identity"), 0700); err != nil {
		t.Fatal(err)
	}
	return model.Tool{ID: id, Command: id, ResolvedPath: path, Status: "present"}
}
