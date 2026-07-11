package discovery

import (
	"github.com/xgbtxy/agent-native-primitives/internal/managed"
	"github.com/xgbtxy/agent-native-primitives/internal/search"
	"github.com/xgbtxy/agent-native-primitives/internal/tooling"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestKnownNameIsResolvedButNeverExecuted(t *testing.T) {
	bin := t.TempDir()
	project := t.TempDir()
	marker := filepath.Join(t.TempDir(), "executed")
	pathExt := ""
	var fake string
	if runtime.GOOS == "windows" {
		fake = filepath.Join(bin, "git.cmd")
		pathExt = ".CMD"
		if err := os.WriteFile(fake, []byte("@echo off\r\ntype nul > \""+marker+"\"\r\n"), 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		fake = filepath.Join(bin, "git")
		if err := os.WriteFile(fake, []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	index, err := Scan(Options{Project: project, PathEnv: bin, PathExt: pathExt})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("scan executed a PATH command")
	}
	var gitFound bool
	for _, tool := range index.Tools {
		if tool.ID == "git" {
			gitFound = tool.Status == "present" && pathsEqual(tool.ResolvedPath, fake)
		}
	}
	if !gitFound {
		t.Fatalf("expected fake git to be presence-resolved without execution, got %#v", index.Tools)
	}
}

func TestScanDoesNotPersistUnknownPathInventory(t *testing.T) {
	bin := t.TempDir()
	project := t.TempDir()
	createCommand(t, bin, "mystery")
	index, err := Scan(Options{Project: project, PathEnv: bin, PathExt: testPathExt()})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range index.Tools {
		if tool.Command == "mystery" {
			t.Fatal("unknown PATH inventory must not be collected during scan")
		}
	}
	exact, ok, err := ResolveExact("mystery", Options{Project: project, PathEnv: bin, PathExt: testPathExt()})
	if err != nil || !ok || exact.Status != "present_unclassified" {
		t.Fatalf("expected query-time exact resolution, got %#v, %v, %v", exact, ok, err)
	}
}

func TestProjectDescriptorMakesOpaqueCommandDiscoverableByIntent(t *testing.T) {
	bin := t.TempDir()
	project := t.TempDir()
	createCommand(t, bin, "fwx")
	manifest := `{
  "capabilities": [{
    "id": "firmware-unpack",
    "command": "fwx",
    "description": "Extract a supported router firmware image into a filesystem tree.",
    "capabilities": ["firmware_extraction"],
    "intents": ["拆包路由器固件", "extract router firmware"],
    "examples": [{"intent":"拆包固件", "command":"fwx unpack image.bin"}],
    "risk": "medium"
  }]
}`
	if err := os.WriteFile(filepath.Join(project, ".tooltruth.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	index, err := Scan(Options{Project: project, PathEnv: bin, PathExt: testPathExt()})
	if err != nil {
		t.Fatal(err)
	}
	result := search.Find(index, "拆包路由器固件")
	if result.Match == nil || result.Match.ID != "firmware-unpack" || result.Match.Signal.Semantics != "project_declared" || result.Match.Signal.Behavior != "not_verified" {
		t.Fatalf("opaque described command was not found by intent: %#v", result.Match)
	}
}

func TestScopeChangesWithProjectAndEnvironment(t *testing.T) {
	first, err := Scan(Options{Project: t.TempDir(), PathEnv: t.TempDir(), PathExt: testPathExt()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scan(Options{Project: t.TempDir(), PathEnv: t.TempDir(), PathExt: testPathExt()})
	if err != nil {
		t.Fatal(err)
	}
	if first.Scope.ID == second.Scope.ID {
		t.Fatal("different project/environment scopes must not share an id")
	}
}

func TestPackageScriptRequiresDetectedRuntimeAndOmitsBody(t *testing.T) {
	project := t.TempDir()
	data := `{"packageManager":"pnpm@9.0.0","scripts":{"deploy":"curl https://internal.example/token=$SECRET"}}`
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	index, err := Scan(Options{Project: project, PathEnv: t.TempDir(), PathExt: testPathExt()})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range index.Tools {
		if tool.ID != "pnpm:deploy" {
			continue
		}
		if tool.Status != "missing_runtime" || tool.Command != "pnpm run deploy" {
			t.Fatalf("unexpected package task: %#v", tool)
		}
		encoded := tool.Description + strings.Join(tool.Intents, " ")
		if strings.Contains(encoded, "internal.example") || strings.Contains(encoded, "SECRET") {
			t.Fatal("script body leaked into semantic metadata")
		}
		return
	}
	t.Fatal("expected pnpm:deploy project task")
}

func TestResolvedAliasRewritesExample(t *testing.T) {
	bin := t.TempDir()
	project := t.TempDir()
	createCommand(t, bin, "fdfind")
	index, err := Scan(Options{Project: project, PathEnv: bin, PathExt: testPathExt()})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range index.Tools {
		if tool.ID == "fd" {
			if len(tool.Examples) == 0 || !strings.HasPrefix(tool.Examples[0].Command, "fdfind ") {
				t.Fatalf("example does not use resolved alias: %#v", tool.Examples)
			}
			return
		}
	}
	t.Fatal("expected fd capability resolved through fdfind")
}

func TestManagedHealthyBinwalkOutranksBrokenPathWithoutDuplication(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	project := t.TempDir()
	createCommand(t, bin, "binwalk")
	managedPath := filepath.Join(managed.ToolRoot(home, "binwalk"), tooling.BinwalkVersion, "content", "bin", executableNameForTest("binwalk"))
	if err := os.MkdirAll(filepath.Dir(managedPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedPath, []byte("managed binary"), 0700); err != nil {
		t.Fatal(err)
	}
	digest, err := managed.HashFile(managedPath)
	if err != nil {
		t.Fatal(err)
	}
	relative, _ := filepath.Rel(managed.ToolRoot(home, "binwalk"), managedPath)
	checked := time.Now().UTC()
	manifest := managed.Manifest{ID: "binwalk", Version: tooling.BinwalkVersion, Executable: filepath.ToSlash(relative), SHA256: digest, VerifiedAt: checked, HealthCheck: tooling.BinwalkProbeID, Source: "crates.io", SourceDigest: tooling.BinwalkCrateSHA256, OS: runtime.GOOS, Arch: runtime.GOARCH}
	if err := managed.WriteManifest(home, manifest); err != nil {
		t.Fatal(err)
	}
	if err := managed.WriteHealth(home, managed.Health{ID: "binwalk", Digest: digest, Probe: tooling.BinwalkProbeID, Status: "ready", CheckedAt: checked}); err != nil {
		t.Fatal(err)
	}
	index, err := Scan(Options{Project: project, PathEnv: bin, PathExt: testPathExt(), ManagedHome: home})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, tool := range index.Tools {
		if tool.ID == "binwalk" {
			count++
			if tool.Status != "ready" || !tool.Managed || tool.Command != "tooltruth exec binwalk --" {
				t.Fatalf("unexpected managed tool: %#v", tool)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one binwalk signal, got %d", count)
	}
}

func TestKnownBrokenPathDigestIsNotRecommended(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	project := t.TempDir()
	path := createCommand(t, bin, "binwalk")
	digest, err := managed.HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := managed.WriteHealth(home, managed.Health{ID: "binwalk", Digest: digest, Probe: tooling.BinwalkProbeID, Status: "broken", CheckedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	index, err := Scan(Options{Project: project, PathEnv: bin, PathExt: testPathExt(), ManagedHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if result := search.Find(index, "分析固件"); result.Match != nil {
		t.Fatalf("known broken digest must be suppressed: %#v", result.Match)
	}
}

func TestWindowsPathExtensionPrecedence(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATHEXT is Windows-specific")
	}
	bin := t.TempDir()
	for _, name := range []string{"same.bat", "same.exe"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("test"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	path, ok := resolveCommand("same", Options{PathEnv: bin, PathExt: ".EXE;.BAT"})
	if !ok || !strings.EqualFold(filepath.Ext(path), ".exe") {
		t.Fatalf("expected PATHEXT-preferred .exe, got %q", path)
	}
}

func pathsEqual(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func createCommand(t *testing.T, dir, command string) string {
	t.Helper()
	name := command
	if runtime.GOOS == "windows" {
		name += ".EXE"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("not executed"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func testPathExt() string {
	if runtime.GOOS == "windows" {
		return ".EXE"
	}
	return ""
}

func executableNameForTest(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
