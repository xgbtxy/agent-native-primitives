package managed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const ManifestSchema = 1

var validID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type Manifest struct {
	Schema       int       `json:"schema"`
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	Executable   string    `json:"executable"`
	SHA256       string    `json:"sha256"`
	VerifiedAt   time.Time `json:"verified_at"`
	HealthCheck  string    `json:"health_check"`
	Source       string    `json:"source"`
	SourceDigest string    `json:"source_digest,omitempty"`
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
}

type Health struct {
	Schema    int       `json:"schema"`
	ID        string    `json:"id"`
	Digest    string    `json:"digest"`
	Probe     string    `json:"probe"`
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checked_at"`
}

type Record struct {
	Manifest Manifest
	Path     string
}

func Home(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return filepath.Abs(override)
	}
	if value := strings.TrimSpace(os.Getenv("TOOLRECALL_HOME")); value != "" {
		return filepath.Abs(value)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}
	return filepath.Join(cache, "tooltruth"), nil
}

func ToolRoot(home, id string) string {
	return filepath.Join(home, "tools", id)
}

func ManifestPath(home, id string) string {
	return filepath.Join(ToolRoot(home, id), "current.json")
}

// Load verifies the executable content against the manifest. It never runs it.
func Load(home, id string) (Record, bool, error) {
	if !ValidID(id) {
		return Record{}, false, fmt.Errorf("invalid managed tool id %q", id)
	}
	data, err := os.ReadFile(ManifestPath(home, id))
	if os.IsNotExist(err) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Record{}, false, fmt.Errorf("decode managed manifest for %s: %w", id, err)
	}
	if manifest.Schema != ManifestSchema || manifest.ID != id || manifest.Executable == "" || manifest.SHA256 == "" || manifest.OS != runtime.GOOS || manifest.Arch != runtime.GOARCH {
		return Record{}, false, nil
	}
	root, err := filepath.Abs(ToolRoot(home, id))
	if err != nil {
		return Record{}, false, err
	}
	path, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(manifest.Executable)))
	if err != nil || !within(root, path) {
		return Record{}, false, nil
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		if os.IsNotExist(err) {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}
	realRoot, rootErr := filepath.EvalSymlinks(root)
	realPath, pathErr := filepath.EvalSymlinks(path)
	if rootErr != nil || pathErr != nil || !within(realRoot, realPath) {
		return Record{}, false, nil
	}
	digest, err := HashFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}
	if !strings.EqualFold(digest, manifest.SHA256) {
		return Record{}, false, nil
	}
	return Record{Manifest: manifest, Path: path}, true, nil
}

func ManifestExists(home, id string) bool {
	if !ValidID(id) {
		return false
	}
	_, err := os.Stat(ManifestPath(home, id))
	return err == nil
}

func ValidID(id string) bool {
	return validID.MatchString(id)
}

func HealthPath(home, id, digest string) string {
	return filepath.Join(home, "health", id, strings.ToLower(digest)+".json")
}

func LoadHealth(home, id, digest, probe string) (Health, bool, error) {
	if !ValidID(id) || len(digest) != 64 {
		return Health{}, false, nil
	}
	data, err := os.ReadFile(HealthPath(home, id, digest))
	if os.IsNotExist(err) {
		return Health{}, false, nil
	}
	if err != nil {
		return Health{}, false, err
	}
	var health Health
	if err := json.Unmarshal(data, &health); err != nil {
		return Health{}, false, nil
	}
	if health.Schema != ManifestSchema || health.ID != id || !strings.EqualFold(health.Digest, digest) || health.Probe != probe || (health.Status != "ready" && health.Status != "broken") {
		return Health{}, false, nil
	}
	return health, true, nil
}

func WriteHealth(home string, health Health) error {
	if !ValidID(health.ID) || len(health.Digest) != 64 || health.Probe == "" || (health.Status != "ready" && health.Status != "broken") {
		return errors.New("invalid managed health record")
	}
	health.Schema = ManifestSchema
	return writeJSONAtomic(HealthPath(home, health.ID, health.Digest), health)
}

func WriteManifest(home string, manifest Manifest) error {
	if !ValidID(manifest.ID) {
		return fmt.Errorf("invalid managed tool id %q", manifest.ID)
	}
	manifest.Schema = ManifestSchema
	return writeJSONAtomic(ManifestPath(home, manifest.ID), manifest)
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.CreateTemp(filepath.Dir(path), ".manifest-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := replaceFile(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
