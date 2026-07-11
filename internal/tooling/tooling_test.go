package tooling

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCrateRejectsTraversal(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bad.crate")
	writeTestCrate(t, archive, "../outside", "bad")
	if err := extractCrate(archive, filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected traversal archive to be rejected")
	}
}

func TestExtractCrateWritesRegularFile(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "ok.crate")
	writeTestCrate(t, archive, "binwalk-3.1.0/Cargo.toml", "[package]")
	destination := filepath.Join(t.TempDir(), "out")
	if err := extractCrate(archive, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "binwalk-3.1.0", "Cargo.toml"))
	if err != nil || string(data) != "[package]" {
		t.Fatalf("unexpected extracted data %q, err=%v", data, err)
	}
}

func TestMinimalBuildEnvironmentStripsRustOverrides(t *testing.T) {
	t.Setenv("RUSTC_WRAPPER", "malicious")
	t.Setenv("CARGO_REGISTRIES_CRATES_IO_INDEX", "malicious")
	for _, value := range minimalBuildEnvironment() {
		if strings.HasPrefix(value, "RUSTC_WRAPPER=") || strings.HasPrefix(value, "CARGO_REGISTRIES_CRATES_IO_INDEX=") {
			t.Fatalf("unsafe override leaked into build environment: %s", value)
		}
	}
}

func writeTestCrate(t *testing.T, path, name, content string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
