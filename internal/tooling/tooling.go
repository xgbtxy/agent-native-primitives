package tooling

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/xgbtxy/agent-native-primitives/internal/managed"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	BinwalkVersion     = "3.1.0"
	BinwalkProbeID     = "binwalk_help_signature_v1"
	BinwalkCrateSHA256 = "99d5dc70f5021c55624f098adae4d1de9c1c07d000475664704b675aad7e0811"
	binwalkSource      = "https://crates.io/crates/binwalk/3.1.0"
	rustToolchain      = "1.97.0-x86_64-pc-windows-msvc"
	rustupSHA256       = "86478e53f769379d7f0ebfa7c9aa97cb76ca92233f79aa2cc0dbee2efaac73c7"
)

type DoctorResult struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	Path        string `json:"path,omitempty"`
	Check       string `json:"check"`
	ExitCode    int    `json:"exit_code,omitempty"`
	Observation string `json:"observation,omitempty"`
}

func Supports(id string) bool {
	return id == "binwalk"
}

func Doctor(ctx context.Context, id, path, source string) DoctorResult {
	result := DoctorResult{ID: id, Status: "unsupported", Source: source, Path: path}
	if id != "binwalk" {
		result.Observation = "no curated health check"
		return result
	}
	result.Check = BinwalkProbeID
	if path == "" {
		result.Status = "missing"
		result.Observation = "no executable resolved"
		return result
	}
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, exitCode, err := runCapture(checkCtx, path, "--help")
	normalized := strings.ToLower(output)
	if err == nil && strings.Contains(normalized, "binwalk") && strings.Contains(normalized, "usage") {
		result.Status = "ready"
		result.Observation = "expected help signature observed"
		return result
	}
	result.Status = "broken"
	result.ExitCode = exitCode
	if errors.Is(checkCtx.Err(), context.DeadlineExceeded) {
		result.Observation = "health check timed out"
	} else if summary := summarize(output); summary != "" {
		result.Observation = summary
	} else if err != nil {
		result.Observation = err.Error()
	} else {
		result.Observation = "expected help signature was absent"
	}
	return result
}

func Repair(ctx context.Context, home, id string, progress io.Writer) (managed.Record, error) {
	if id != "binwalk" {
		return managed.Record{}, fmt.Errorf("no curated repair recipe for %q", id)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return managed.Record{}, fmt.Errorf("binwalk repair currently supports windows/amd64 only")
	}
	return repairBinwalkWindows(ctx, home, progress)
}

func repairBinwalkWindows(ctx context.Context, home string, progress io.Writer) (managed.Record, error) {
	home, err := filepath.Abs(home)
	if err != nil {
		return managed.Record{}, err
	}
	release, err := acquireRepairLock(home, "binwalk")
	if err != nil {
		return managed.Record{}, err
	}
	defer release()
	cleanupStaleBuilds(home)
	build, err := os.MkdirTemp(filepath.Join(home, "build"), "binwalk-")
	if err != nil {
		if err := os.MkdirAll(filepath.Join(home, "build"), 0700); err != nil {
			return managed.Record{}, err
		}
		build, err = os.MkdirTemp(filepath.Join(home, "build"), "binwalk-")
		if err != nil {
			return managed.Record{}, err
		}
	}
	defer os.RemoveAll(build)

	stage := filepath.Join(build, "install")
	crateArchive := filepath.Join(build, "binwalk-"+BinwalkVersion+".crate")
	fmt.Fprintln(progress, "repair: downloading pinned binwalk crate")
	if err := download(ctx, "https://crates.io/api/v1/crates/binwalk/"+BinwalkVersion+"/download", crateArchive); err != nil {
		return managed.Record{}, fmt.Errorf("download binwalk crate: %w", err)
	}
	crateDigest, err := managed.HashFile(crateArchive)
	if err != nil {
		return managed.Record{}, err
	}
	if !strings.EqualFold(crateDigest, BinwalkCrateSHA256) {
		return managed.Record{}, fmt.Errorf("binwalk crate checksum mismatch: got %s", crateDigest)
	}
	sourceRoot := filepath.Join(build, "source")
	if err := extractCrate(crateArchive, sourceRoot); err != nil {
		return managed.Record{}, fmt.Errorf("extract binwalk crate: %w", err)
	}
	packageRoot := filepath.Join(sourceRoot, "binwalk-"+BinwalkVersion)
	rustup := filepath.Join(build, "rustup-init.exe")
	fmt.Fprintln(progress, "repair: downloading isolated Rust bootstrap from rustup.rs")
	if err := download(ctx, "https://win.rustup.rs/x86_64", rustup); err != nil {
		return managed.Record{}, fmt.Errorf("download rustup: %w", err)
	}
	bootstrapDigest, err := managed.HashFile(rustup)
	if err != nil {
		return managed.Record{}, err
	}
	if !strings.EqualFold(bootstrapDigest, rustupSHA256) {
		return managed.Record{}, fmt.Errorf("rustup bootstrap checksum mismatch: got %s", bootstrapDigest)
	}
	cargoHome := filepath.Join(build, "cargo")
	rustupHome := filepath.Join(build, "rustup")
	environment := append(minimalBuildEnvironment(),
		"CARGO_HOME="+cargoHome,
		"RUSTUP_HOME="+rustupHome,
		"CARGO_TARGET_DIR="+filepath.Join(build, "target"),
	)
	fmt.Fprintln(progress, "repair: installing a temporary minimal Rust toolchain (PATH unchanged)")
	if err := runStreaming(ctx, progress, environment, rustup, "-y", "--no-modify-path", "--profile", "minimal", "--default-toolchain", rustToolchain); err != nil {
		return managed.Record{}, fmt.Errorf("bootstrap rust: %w", err)
	}
	cargo := filepath.Join(cargoHome, "bin", "cargo.exe")
	fmt.Fprintln(progress, "repair: building official binwalk 3.1.0 from crates.io")
	if err := runStreaming(ctx, progress, environment, cargo, "install", "--path", packageRoot, "--locked", "--root", stage); err != nil {
		return managed.Record{}, fmt.Errorf("build binwalk: %w", err)
	}
	built := filepath.Join(stage, "bin", "binwalk.exe")
	result := Doctor(ctx, "binwalk", built, "staged")
	if result.Status != "ready" {
		return managed.Record{}, fmt.Errorf("staged binwalk failed health check: %s", result.Observation)
	}

	builtDigest, err := managed.HashFile(built)
	if err != nil {
		return managed.Record{}, err
	}
	versionRoot := filepath.Join(managed.ToolRoot(home, "binwalk"), BinwalkVersion, builtDigest)
	final := filepath.Join(versionRoot, "bin", "binwalk.exe")
	if err := os.MkdirAll(filepath.Dir(final), 0700); err != nil {
		return managed.Record{}, err
	}
	if existing, err := managed.HashFile(final); err == nil {
		if !strings.EqualFold(existing, builtDigest) {
			return managed.Record{}, errors.New("content-addressed destination has unexpected content")
		}
	} else if os.IsNotExist(err) {
		if err := copyFileExclusive(built, final); err != nil {
			return managed.Record{}, err
		}
	} else {
		return managed.Record{}, err
	}
	digest, err := managed.HashFile(final)
	if err != nil {
		return managed.Record{}, err
	}
	manifest := managed.Manifest{
		ID: "binwalk", Version: BinwalkVersion,
		Executable: filepath.ToSlash(filepath.Join(BinwalkVersion, builtDigest, "bin", "binwalk.exe")),
		SHA256:     digest, VerifiedAt: time.Now().UTC(),
		HealthCheck: result.Check, Source: binwalkSource, SourceDigest: BinwalkCrateSHA256,
		OS: runtime.GOOS, Arch: runtime.GOARCH,
	}
	if err := managed.WriteManifest(home, manifest); err != nil {
		return managed.Record{}, err
	}
	if err := managed.WriteHealth(home, managed.Health{ID: "binwalk", Digest: digest, Probe: BinwalkProbeID, Status: "ready", CheckedAt: manifest.VerifiedAt}); err != nil {
		return managed.Record{}, err
	}
	fmt.Fprintln(progress, "repair: health check passed; temporary build toolchain will be removed")
	record, ok, err := managed.Load(home, "binwalk")
	if err != nil {
		return managed.Record{}, err
	}
	if !ok {
		return managed.Record{}, errors.New("managed binwalk failed post-install hash verification")
	}
	return record, nil
}

func RunManaged(ctx context.Context, record managed.Record, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, record.Path, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

func runCapture(ctx context.Context, path string, args ...string) (string, int, error) {
	name, actualArgs := executableCommand(path, args)
	command := exec.CommandContext(ctx, name, actualArgs...)
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
	return output.String(), exitCode, err
}

func executableCommand(path string, args []string) (string, []string) {
	if runtime.GOOS != "windows" || filepath.Ext(path) != "" {
		return path, args
	}
	file, err := os.Open(path)
	if err != nil {
		return path, args
	}
	defer file.Close()
	line, _ := bufio.NewReader(io.LimitReader(file, 4096)).ReadString('\n')
	line = strings.TrimSpace(strings.TrimPrefix(line, "#!"))
	if line == "" {
		return path, args
	}
	return line, append([]string{path}, args...)
}

func runStreaming(ctx context.Context, output io.Writer, environment []string, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = environment
	command.Stdout = output
	command.Stderr = output
	return command.Run()
}

func download(ctx context.Context, url, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	const maximum = int64(100 << 20)
	if response.ContentLength > maximum {
		return fmt.Errorf("download is too large: %d bytes", response.ContentLength)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maximum+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if written > maximum {
		_ = os.Remove(destination)
		return errors.New("download exceeded 100 MiB limit")
	}
	return closeErr
}

func extractCrate(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	reader := tar.NewReader(gzipReader)
	var total int64
	for entries := 0; ; entries++ {
		if entries > 10000 {
			return errors.New("crate contains too many entries")
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(header.Name)))
		if err != nil || !pathWithin(root, target) {
			return fmt.Errorf("crate entry escapes destination: %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			total += header.Size
			if header.Size < 0 || total > 50<<20 {
				return errors.New("expanded crate exceeds 50 MiB limit")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(output, reader, header.Size)
			closeErr := output.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported crate entry type for %q", header.Name)
		}
	}
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func copyFileExclusive(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary := destination
	output, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(temporary)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}
	return nil
}

func minimalBuildEnvironment() []string {
	allowed := map[string]bool{
		"SYSTEMROOT": true, "WINDIR": true, "TEMP": true, "TMP": true,
		"USERPROFILE": true, "LOCALAPPDATA": true, "PROGRAMDATA": true,
		"PROGRAMFILES": true, "PROGRAMFILES(X86)": true, "COMSPEC": true,
		"NUMBER_OF_PROCESSORS": true, "PROCESSOR_ARCHITECTURE": true, "PATH": true,
	}
	var environment []string
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if ok && allowed[strings.ToUpper(name)] {
			environment = append(environment, value)
		}
	}
	return environment
}

func acquireRepairLock(home, id string) (func(), error) {
	root := managed.ToolRoot(home, id)
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(root, "repair.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return nil, fmt.Errorf("repair for %s is already running; remove %s only if no repair process exists", id, path)
	}
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(file, "%d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	_ = file.Close()
	return func() { _ = os.Remove(path) }, nil
}

func cleanupStaleBuilds(home string) {
	root := filepath.Join(home, "build")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "binwalk-") {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
	}
}

func summarize(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 240 {
		value = value[:240] + "..."
	}
	return value
}

type limitedBuffer struct {
	buffer bytes.Buffer
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := (64 << 10) - b.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = b.buffer.Write(value)
	}
	return original, nil
}

func (b *limitedBuffer) String() string { return b.buffer.String() }
