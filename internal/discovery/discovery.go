package discovery

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/xgbtxy/agent-native-primitives/internal/catalog"
	"github.com/xgbtxy/agent-native-primitives/internal/managed"
	"github.com/xgbtxy/agent-native-primitives/internal/model"
	"github.com/xgbtxy/agent-native-primitives/internal/tooling"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	makeTargetPattern = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*):(?:\s|$)`)
	exactCommand      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.+\-]*$`)
)

type Options struct {
	Project     string
	PathEnv     string
	PathExt     string
	ManagedHome string
}

func Scan(options Options) (model.Index, error) {
	options, scope, err := normalize(options)
	if err != nil {
		return model.Index{}, err
	}

	resolver := newPathResolver(options)
	tools := scanCatalog(resolver, options.ManagedHome)
	descriptors, err := scanDescriptors(options, resolver)
	if err != nil {
		return model.Index{}, err
	}
	tools = append(tools, descriptors...)
	tools = append(tools, scanPackageScripts(options, resolver)...)
	tools = append(tools, scanMakefile(options, resolver)...)
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].ProjectDefined != tools[j].ProjectDefined {
			return tools[i].ProjectDefined
		}
		return tools[i].Command < tools[j].Command
	})

	return model.Index{
		SchemaVersion: model.SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Scope:         scope,
		Tools:         tools,
	}, nil
}

func scanDescriptors(options Options, resolver *pathResolver) ([]model.Tool, error) {
	path := filepath.Join(options.Project, ".tooltruth.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read .tooltruth.json: %w", err)
	}
	if len(data) > 256*1024 {
		return nil, fmt.Errorf(".tooltruth.json exceeds 256 KiB")
	}
	var manifest struct {
		Capabilities []struct {
			ID           string          `json:"id"`
			Family       string          `json:"family"`
			Command      string          `json:"command"`
			Description  string          `json:"description"`
			Capabilities []string        `json:"capabilities"`
			Intents      []string        `json:"intents"`
			Examples     []model.Example `json:"examples"`
			Risk         string          `json:"risk"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode .tooltruth.json: %w", err)
	}
	seen := map[string]bool{}
	var tools []model.Tool
	for position, descriptor := range manifest.Capabilities {
		descriptor.Command = strings.TrimSpace(descriptor.Command)
		if !exactCommand.MatchString(descriptor.Command) {
			return nil, fmt.Errorf(".tooltruth.json capability %d has invalid command %q", position, descriptor.Command)
		}
		if descriptor.ID == "" {
			descriptor.ID = descriptor.Command
		}
		if descriptor.Family == "" {
			descriptor.Family = descriptor.ID
		}
		if seen[descriptor.ID] {
			return nil, fmt.Errorf(".tooltruth.json contains duplicate id %q", descriptor.ID)
		}
		seen[descriptor.ID] = true
		resolved, present := resolver.resolve(descriptor.Command)
		status := "missing_runtime"
		if present {
			status = "present"
		}
		risk := descriptor.Risk
		if risk == "" {
			risk = "unknown"
		}
		tools = append(tools, model.Tool{
			ID: descriptor.ID, Family: descriptor.Family, Command: descriptor.Command, ResolvedPath: resolved,
			Status: status, SemanticSource: "project_descriptor", ResolverSource: "path", Description: descriptor.Description,
			Capabilities: append([]string(nil), descriptor.Capabilities...),
			Intents:      append([]string(nil), descriptor.Intents...),
			Examples:     append([]model.Example(nil), descriptor.Examples...),
			Risk:         risk, ProjectDefined: true,
		})
	}
	return tools, nil
}

// ResolveExact is an explicit-name fallback, not semantic discovery. It allows
// an agent to ask whether an opaque command name resolves without persisting a
// whole-machine PATH inventory.
func ResolveExact(command string, options Options) (model.Tool, bool, error) {
	_, results, err := ResolveExactBatch([]string{command}, options)
	if err != nil {
		return model.Tool{}, false, err
	}
	if len(results) != 1 || results[0].Status != "present" {
		return model.Tool{}, false, nil
	}
	return results[0].Tool, true, nil
}

type ExactResolution struct {
	Query  string
	Status string
	Tool   model.Tool
}

// ResolveExactBatch resolves only caller-supplied command names. It checks only
// those names across PATH, preserves input order, and never executes them.
func ResolveExactBatch(commands []string, options Options) (model.Scope, []ExactResolution, error) {
	useNativePath := options.PathEnv == "" && options.PathExt == ""
	options, scope, err := normalize(options)
	if err != nil {
		return model.Scope{}, nil, err
	}
	keyResolver := &pathResolver{extensions: windowsExtensions(options.PathExt)}

	results := make([]ExactResolution, len(commands))
	for i, raw := range commands {
		results[i] = resolveExactOne(strings.TrimSpace(raw), options, keyResolver, useNativePath)
	}
	return scope, results, nil
}

func resolveExactOne(query string, options Options, keyResolver *pathResolver, useNativePath bool) ExactResolution {
	result := ExactResolution{Query: query, Status: "absent"}
	if !exactCommand.MatchString(query) {
		result.Status = "invalid_name"
		return result
	}
	key := keyResolver.key(query)
	entry, known := catalog.ByCommand(key)
	if known && tooling.Supports(entry.ID) {
		if tool, ok := resolveManagedExact(entry, options.ManagedHome); ok {
			result.Status = "present"
			result.Tool = tool
			return result
		}
	}
	path, present := "", false
	if useNativePath {
		var err error
		path, err = exec.LookPath(query)
		present = err == nil
	} else {
		path, present = resolveExactPath(query, options)
	}
	if !present {
		return result
	}
	result.Status = "present"
	if known {
		result.Tool = toolFromEntry(entry, query, path)
		return result
	}
	result.Tool = model.Tool{
		ID: query, Family: query, Command: query, ResolvedPath: path,
		Status: "present_unclassified", SemanticSource: "none", ResolverSource: "path",
		Description: "The exact command name resolves in the active PATH; its capability is unknown.",
		Risk:        "unknown",
	}
	return result
}

func resolveManagedExact(entry catalog.Entry, home string) (model.Tool, bool) {
	record, ok, err := managed.Load(home, entry.ID)
	if err != nil || !ok || !managedRecipeMatches(entry.ID, record.Manifest) {
		return model.Tool{}, false
	}
	health, ok, err := managed.LoadHealth(home, entry.ID, record.Manifest.SHA256, tooling.BinwalkProbeID)
	if err != nil || !ok || health.Status != "ready" {
		return model.Tool{}, false
	}
	tool := toolFromEntry(entry, "tooltruth exec "+entry.ID+" --", record.Path)
	tool.Status = "ready"
	tool.ResolverSource = "managed_digest_matched"
	tool.Managed = true
	tool.Version = record.Manifest.Version
	tool.VerifiedAt = health.CheckedAt
	tool.Examples = managedExamples(entry, tool.Command)
	return tool, true
}

func resolveExactPath(command string, options Options) (string, bool) {
	extensions := windowsExtensionOrder(options.PathExt)
	for _, dir := range filepath.SplitList(options.PathEnv) {
		dir = strings.TrimSpace(strings.Trim(dir, `"`))
		if dir == "" {
			continue
		}
		if runtime.GOOS != "windows" {
			path := filepath.Join(dir, command)
			info, err := os.Stat(path)
			if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
				return path, true
			}
			continue
		}

		extension := strings.ToLower(filepath.Ext(command))
		knownExtension := false
		for _, candidateExtension := range extensions {
			if extension == strings.ToLower(candidateExtension) {
				knownExtension = true
				break
			}
		}
		if knownExtension {
			path := filepath.Join(dir, command)
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				return path, true
			}
			continue
		}
		for _, candidateExtension := range extensions {
			path := filepath.Join(dir, command+candidateExtension)
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				return path, true
			}
		}
		path := filepath.Join(dir, command)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && hasExecutableHeader(path) {
			return path, true
		}
	}
	return "", false
}

func windowsExtensionOrder(pathExt string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	extensions := filepath.SplitList(pathExt)
	if len(extensions) == 0 {
		extensions = []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	result := make([]string, 0, len(extensions))
	seen := map[string]bool{}
	for _, extension := range extensions {
		extension = strings.TrimSpace(extension)
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		key := strings.ToLower(extension)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, extension)
	}
	return result
}

func normalize(options Options) (Options, model.Scope, error) {
	if options.Project == "" {
		options.Project = "."
	}
	project, err := filepath.Abs(options.Project)
	if err != nil {
		return Options{}, model.Scope{}, fmt.Errorf("resolve project path: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(project); err == nil {
		project = resolved
	}
	options.Project = filepath.Clean(project)
	if options.PathEnv == "" {
		options.PathEnv = os.Getenv("PATH")
	}
	if options.PathExt == "" {
		options.PathExt = os.Getenv("PATHEXT")
	}
	options.ManagedHome, err = managed.Home(options.ManagedHome)
	if err != nil {
		return Options{}, model.Scope{}, err
	}
	fingerprint := sha256.Sum256([]byte(strings.Join([]string{
		runtime.GOOS, runtime.GOARCH, options.Project, options.PathEnv, options.PathExt, options.ManagedHome,
	}, "\x00")))
	scopeID := hex.EncodeToString(fingerprint[:8])
	return options, model.Scope{
		ID: scopeID, Project: options.Project, ProjectName: filepath.Base(options.Project),
		OS: runtime.GOOS, Arch: runtime.GOARCH, Resolver: "managed+path",
	}, nil
}

func scanCatalog(resolver *pathResolver, home string) []model.Tool {
	var tools []model.Tool
	for _, entry := range catalog.All() {
		record, managedOK, managedErr := managed.Load(home, entry.ID)
		if managedErr != nil || (!managedOK && managed.ManifestExists(home, entry.ID)) {
			tool := toolFromEntry(entry, entry.Commands[0], "")
			tool.Status = "managed_invalid"
			tool.ResolverSource = "managed_invalid"
			tools = append(tools, tool)
			continue
		}
		if managedOK {
			tool := toolFromEntry(entry, "tooltruth exec "+entry.ID+" --", record.Path)
			tool.Status = "managed_invalid"
			tool.ResolverSource = "managed_digest_matched"
			tool.Managed = true
			tool.Version = record.Manifest.Version
			tool.VerifiedAt = record.Manifest.VerifiedAt
			tool.Examples = managedExamples(entry, tool.Command)
			if managedRecipeMatches(entry.ID, record.Manifest) {
				health, healthOK, _ := managed.LoadHealth(home, entry.ID, record.Manifest.SHA256, tooling.BinwalkProbeID)
				if healthOK {
					tool.Status = health.Status
					tool.VerifiedAt = health.CheckedAt
				}
			}
			tools = append(tools, tool)
			continue
		}
		for _, command := range preferredCommands(entry) {
			path, ok := resolver.resolve(command)
			if !ok {
				continue
			}
			tool := toolFromEntry(entry, command, path)
			if entry.ID == "binwalk" {
				if digest, err := managed.HashFile(path); err == nil {
					if health, ok, _ := managed.LoadHealth(home, entry.ID, digest, tooling.BinwalkProbeID); ok {
						tool.Status = health.Status
						tool.VerifiedAt = health.CheckedAt
					}
				}
			}
			tools = append(tools, tool)
			break
		}
	}
	return tools
}

func managedRecipeMatches(id string, manifest managed.Manifest) bool {
	return id == "binwalk" &&
		manifest.Version == tooling.BinwalkVersion &&
		manifest.HealthCheck == tooling.BinwalkProbeID &&
		strings.EqualFold(manifest.SourceDigest, tooling.BinwalkCrateSHA256)
}

func managedExamples(entry catalog.Entry, prefix string) []model.Example {
	examples := append([]model.Example(nil), entry.Examples...)
	if len(entry.Commands) == 0 {
		return examples
	}
	for i := range examples {
		command := examples[i].Command
		if command == entry.Commands[0] {
			examples[i].Command = prefix
		} else if strings.HasPrefix(command, entry.Commands[0]+" ") {
			examples[i].Command = prefix + " " + strings.TrimPrefix(command, entry.Commands[0]+" ")
		}
	}
	return examples
}

func preferredCommands(entry catalog.Entry) []string {
	commands := append([]string(nil), entry.Commands...)
	if runtime.GOOS == "windows" && entry.ID == "python" {
		return []string{"py", "python", "python3"}
	}
	if runtime.GOOS != "windows" && entry.ID == "python" {
		return []string{"python3", "python"}
	}
	return commands
}

func toolFromEntry(entry catalog.Entry, command, path string) model.Tool {
	examples := append([]model.Example(nil), entry.Examples...)
	if entry.RewriteCommand && len(entry.Commands) > 0 && command != entry.Commands[0] {
		prefix := entry.Commands[0] + " "
		for i := range examples {
			if strings.HasPrefix(examples[i].Command, prefix) {
				examples[i].Command = command + " " + strings.TrimPrefix(examples[i].Command, prefix)
			}
		}
	}
	return model.Tool{
		ID: entry.ID, Family: entry.Family, Command: command, ResolvedPath: path,
		Status: "present", SemanticSource: "builtin_catalog", ResolverSource: "path", Description: entry.Description,
		Capabilities: append([]string(nil), entry.Capabilities...),
		Intents:      append([]string(nil), entry.Intents...), Examples: examples, Risk: entry.Risk,
	}
}

type pathResolver struct {
	commands   map[string]string
	extensions map[string]int
}

func newPathResolver(options Options) *pathResolver {
	resolver := &pathResolver{commands: map[string]string{}, extensions: windowsExtensions(options.PathExt)}
	for _, dir := range filepath.SplitList(options.PathEnv) {
		dir = strings.TrimSpace(strings.Trim(dir, `"`))
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		if runtime.GOOS == "windows" {
			sort.SliceStable(entries, func(i, j int) bool {
				leftBase, leftRank := resolver.windowsName(entries[i].Name())
				rightBase, rightRank := resolver.windowsName(entries[j].Name())
				if leftBase != rightBase {
					return leftBase < rightBase
				}
				return leftRank < rightRank
			})
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			command, ok := resolver.commandName(entry, path)
			if !ok {
				continue
			}
			key := resolver.key(command)
			if _, exists := resolver.commands[key]; !exists {
				resolver.commands[key] = path
			}
		}
	}
	return resolver
}

func windowsExtensions(pathExt string) map[string]int {
	result := map[string]int{}
	if runtime.GOOS != "windows" {
		return result
	}
	extensions := filepath.SplitList(pathExt)
	if len(extensions) == 0 {
		extensions = []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	for rank, extension := range extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		result[extension] = rank
	}
	return result
}

func (r *pathResolver) windowsName(filename string) (string, int) {
	ext := strings.ToLower(filepath.Ext(filename))
	rank, ok := r.extensions[ext]
	if !ok {
		return strings.ToLower(filename), 1 << 30
	}
	return strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename))), rank
}

func (r *pathResolver) commandName(entry os.DirEntry, path string) (string, bool) {
	if runtime.GOOS == "windows" {
		base, rank := r.windowsName(entry.Name())
		if rank != 1<<30 {
			return base, true
		}
		if filepath.Ext(entry.Name()) == "" && hasExecutableHeader(path) {
			return entry.Name(), true
		}
		return "", false
	}
	info, err := entry.Info()
	if entry.Type()&os.ModeSymlink != 0 {
		info, err = os.Stat(path)
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return "", false
	}
	return entry.Name(), true
}

func hasExecutableHeader(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	var header [2]byte
	if _, err := file.Read(header[:]); err != nil {
		return false
	}
	return string(header[:]) == "MZ" || string(header[:]) == "#!"
}

func (r *pathResolver) key(command string) string {
	if runtime.GOOS != "windows" {
		return command
	}
	ext := strings.ToLower(filepath.Ext(command))
	if _, ok := r.extensions[ext]; ok {
		command = strings.TrimSuffix(command, filepath.Ext(command))
	}
	return strings.ToLower(command)
}

func (r *pathResolver) resolve(command string) (string, bool) {
	if command == "" || strings.ContainsAny(command, `/\\`) {
		return "", false
	}
	path, ok := r.commands[r.key(command)]
	return path, ok
}

func resolveCommand(command string, options Options) (string, bool) {
	return newPathResolver(options).resolve(command)
}

func scanPackageScripts(options Options, resolver *pathResolver) []model.Tool {
	manifest := filepath.Join(options.Project, "package.json")
	data, err := os.ReadFile(manifest)
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts        map[string]string `json:"scripts"`
		PackageManager string            `json:"packageManager"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	manager := detectPackageManager(options.Project, pkg.PackageManager)
	_, runtimeReady := resolver.resolve(manager)
	names := make([]string, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	tools := make([]model.Tool, 0, len(names))
	for _, name := range names {
		status := "missing_runtime"
		if runtimeReady {
			status = "present"
		}
		command := packageScriptCommand(manager, name)
		tools = append(tools, model.Tool{
			ID: manager + ":" + name, Family: "project_task", Command: command, ResolvedPath: manifest,
			Status: status, SemanticSource: "package.json", ResolverSource: "project_manifest+path", Description: "Project-defined package script named " + name + ".",
			Capabilities: []string{"project_task", "package_script"},
			Intents:      []string{name, "运行项目脚本", "run project script"},
			Examples:     []model.Example{{Intent: "运行 " + name + " 项目脚本", Command: command}},
			Risk:         "dangerous", ProjectDefined: true,
		})
	}
	return tools
}

func detectPackageManager(project, declared string) string {
	if name := strings.TrimSpace(strings.SplitN(declared, "@", 2)[0]); name != "" {
		switch name {
		case "npm", "pnpm", "yarn", "bun":
			return name
		}
	}
	checks := []struct{ file, manager string }{
		{"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"bun.lock", "bun"}, {"bun.lockb", "bun"},
	}
	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(project, check.file)); err == nil {
			return check.manager
		}
	}
	return "npm"
}

func packageScriptCommand(manager, name string) string {
	if manager == "yarn" {
		return "yarn " + name
	}
	return manager + " run " + name
}

func scanMakefile(options Options, resolver *pathResolver) []model.Tool {
	_, makeReady := resolver.resolve("make")
	for _, filename := range []string{"Makefile", "makefile", "GNUmakefile"} {
		path := filepath.Join(options.Project, filename)
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		var tools []model.Tool
		seen := map[string]bool{}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			match := makeTargetPattern.FindStringSubmatch(scanner.Text())
			if len(match) != 2 || strings.HasPrefix(match[1], ".") || seen[match[1]] {
				continue
			}
			seen[match[1]] = true
			status := "missing_runtime"
			if makeReady {
				status = "present"
			}
			command := "make " + match[1]
			tools = append(tools, model.Tool{
				ID: "make:" + match[1], Family: "project_task", Command: command, ResolvedPath: path,
				Status: status, SemanticSource: filename, ResolverSource: "project_manifest+path", Description: "Project-defined Make target named " + match[1] + ".",
				Capabilities: []string{"project_task", "make_target"},
				Intents:      []string{match[1], "构建项目", "运行项目任务", "run make target"},
				Examples:     []model.Example{{Intent: "运行 " + match[1] + " target", Command: command}},
				Risk:         "dangerous", ProjectDefined: true,
			})
		}
		_ = file.Close()
		return tools
	}
	return nil
}
