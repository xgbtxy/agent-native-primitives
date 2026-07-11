package managed

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestManifestAndHealthAreDigestBound(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(ToolRoot(home, "binwalk"), "3.1.0", "digest", "bin", executableName("binwalk"))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("first"), 0700); err != nil {
		t.Fatal(err)
	}
	digest, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "binwalk", Version: "3.1.0", Executable: relativeExecutable(t, home, path), SHA256: digest, VerifiedAt: time.Now().UTC(), HealthCheck: "probe", Source: "source", SourceDigest: "source-digest", OS: runtime.GOOS, Arch: runtime.GOARCH}
	if err := WriteManifest(home, manifest); err != nil {
		t.Fatal(err)
	}
	if err := WriteHealth(home, Health{ID: "binwalk", Digest: digest, Probe: "probe", Status: "ready", CheckedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := Load(home, "binwalk"); err != nil || !ok {
		t.Fatalf("expected valid managed record, ok=%v err=%v", ok, err)
	}
	if _, ok, err := LoadHealth(home, "binwalk", digest, "probe"); err != nil || !ok {
		t.Fatalf("expected valid health record, ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := Load(home, "binwalk"); err != nil || ok {
		t.Fatalf("tampered executable must fail digest binding, ok=%v err=%v", ok, err)
	}
}

func TestManifestReplacementIsAtomicAndIDsAreRestricted(t *testing.T) {
	home := t.TempDir()
	base := Manifest{ID: "binwalk", Version: "1", Executable: "x", SHA256: "digest", OS: runtime.GOOS, Arch: runtime.GOARCH}
	if err := WriteManifest(home, base); err != nil {
		t.Fatal(err)
	}
	base.Version = "2"
	if err := WriteManifest(home, base); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(home, "../outside"); err == nil {
		t.Fatal("traversal id must be rejected")
	}
}

func relativeExecutable(t *testing.T, home, path string) string {
	t.Helper()
	relative, err := filepath.Rel(ToolRoot(home, "binwalk"), path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(relative)
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
