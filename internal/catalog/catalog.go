package catalog

import "github.com/xgbtxy/agent-native-primitives/internal/model"

type Entry struct {
	ID             string
	Family         string
	Commands       []string
	Description    string
	Capabilities   []string
	Intents        []string
	Examples       []model.Example
	Risk           string
	RewriteCommand bool
}

var entries = []Entry{
	{
		ID: "ripgrep", Family: "text_search", Commands: []string{"rg"},
		Description:  "Fast recursive text and source-code search with regex, file filters, line numbers, and context.",
		Capabilities: []string{"search_text", "search_source_code", "search_logs", "regex_search"},
		Intents:      []string{"find text in files", "search source code", "grep recursively", "search logs", "show matching context"},
		Examples:     []model.Example{{Intent: "search source code with line numbers", Command: `rg -n "validateToken" src`}, {Intent: "search logs with context", Command: `rg -n -C 10 "panic" error.log`}},
		Risk:         "safe",
	},
	{
		ID: "fd", Family: "file_discovery", Commands: []string{"fd", "fdfind"},
		Description:  "Fast filesystem search by file name, extension, type, or glob.",
		Capabilities: []string{"find_files", "find_directories", "filesystem_discovery"},
		Intents:      []string{"find files", "search by file name", "find directories", "find TypeScript files", "search by extension"},
		Examples:     []model.Example{{Intent: "find TypeScript files", Command: `fd -e ts`}},
		Risk:         "safe", RewriteCommand: true,
	},
	{
		ID: "git", Family: "version_control", Commands: []string{"git"},
		Description:  "Inspect and manage source history, diffs, branches, commits, and repositories.",
		Capabilities: []string{"version_control", "source_history", "diff", "repository_inspection"},
		Intents:      []string{"inspect git history", "show diff", "compare files", "version control", "find commit", "repository status", "show current Git changes"},
		Examples:     []model.Example{{Intent: "show current changes", Command: `git diff --stat`}, {Intent: "find the commit that changed a line", Command: `git blame -L 40,90 -- path/to/file`}},
		Risk:         "medium",
	},
	{
		ID: "jq", Family: "structured_data_query", Commands: []string{"jq"},
		Description:  "Query, filter, transform, and format JSON from files or stdin.",
		Capabilities: []string{"query_json", "transform_json", "format_json"},
		Intents:      []string{"parse JSON", "query JSON", "filter JSON", "format JSON", "transform JSON"},
		Examples:     []model.Example{{Intent: "read a JSON field", Command: `jq -r '.server.port' config.json`}},
		Risk:         "safe",
	},
	{
		ID: "yq", Family: "structured_data_query", Commands: []string{"yq"},
		Description:  "Query and transform YAML, JSON, XML, CSV, TOML, and properties files.",
		Capabilities: []string{"query_yaml", "transform_yaml", "query_structured_data"},
		Intents:      []string{"parse YAML", "query YAML", "read YAML", "edit YAML", "query yaml", "read yaml", "filter yaml", "transform yaml"},
		Examples:     []model.Example{{Intent: "read a YAML field", Command: `yq '.server.port' config.yaml`}},
		Risk:         "medium",
	},
	{
		ID: "curl", Family: "http_client", Commands: []string{"curl"},
		Description:  "Transfer data over HTTP and many other network protocols; inspect APIs and download files.",
		Capabilities: []string{"http_request", "api_call", "download", "network_probe"},
		Intents:      []string{"send an HTTP request", "call an API", "download a file", "test an endpoint", "http request", "call api", "download url"},
		Examples:     []model.Example{{Intent: "fetch a JSON API", Command: `curl -fsS https://example.com/api`}},
		Risk:         "medium",
	},
	{
		ID: "gh", Family: "github", Commands: []string{"gh"},
		Description:  "Work with GitHub repositories, issues, pull requests, releases, and API data.",
		Capabilities: []string{"github", "pull_requests", "issues", "github_api"},
		Intents:      []string{"view GitHub pull requests", "create an issue", "query GitHub", "github pull request", "github issues", "github api"},
		Examples:     []model.Example{{Intent: "list pull requests for the current repository", Command: `gh pr list`}},
		Risk:         "medium",
	},
	{
		ID: "go", Family: "go_toolchain", Commands: []string{"go"},
		Description:  "Build, test, format, inspect, and manage Go modules and programs.",
		Capabilities: []string{"build_go", "test_go", "format_go", "go_modules"},
		Intents:      []string{"build Go", "run Go tests", "format Go", "build go", "test go", "go modules"},
		Examples:     []model.Example{{Intent: "run all Go tests", Command: `go test ./...`}},
		Risk:         "medium",
	},
	{
		ID: "python", Family: "python_runtime", Commands: []string{"python", "python3", "py"},
		Description:  "Run Python programs and scripts.",
		Capabilities: []string{"run_python", "scripting", "data_processing"},
		Intents:      []string{"run Python", "execute a Python script", "process data", "run python", "python script"},
		Examples:     []model.Example{{Intent: "run a Python script", Command: `python script.py`}},
		Risk:         "medium", RewriteCommand: true,
	},
	{
		ID: "uv", Family: "python_environment", Commands: []string{"uv"},
		Description:  "Manage Python projects, dependencies, lockfiles, and environments.",
		Capabilities: []string{"python_packages", "python_environment", "python_lockfiles"},
		Intents:      []string{"install Python dependencies", "manage a Python environment", "manage a Python project", "python package manager", "python environment"},
		Examples:     []model.Example{{Intent: "sync Python project dependencies", Command: `uv sync`}},
		Risk:         "medium",
	},
	{
		ID: "uvx", Family: "ephemeral_python_tool", Commands: []string{"uvx"},
		Description:  "Run Python command-line tools in isolated, cached environments without global installation.",
		Capabilities: []string{"isolated_python_tools", "ephemeral_tool_execution"},
		Intents:      []string{"run an isolated Python tool", "run a temporary Python tool", "run python tool", "run tool without installing"},
		Examples:     []model.Example{{Intent: "run a temporary Python tool", Command: `uvx <tool> --help`}},
		Risk:         "medium",
	},
	{
		ID: "node", Family: "javascript_runtime", Commands: []string{"node"},
		Description:  "Run JavaScript and Node.js applications.",
		Capabilities: []string{"run_javascript", "node_runtime"},
		Intents:      []string{"run JavaScript", "execute a Node script", "run javascript", "node runtime"},
		Examples:     []model.Example{{Intent: "run a Node script", Command: `node script.js`}},
		Risk:         "medium",
	},
	{
		ID: "docker", Family: "container_runtime", Commands: []string{"docker"},
		Description:  "Build and run isolated containers and inspect container images, networks, and volumes.",
		Capabilities: []string{"containers", "container_build", "isolated_runtime"},
		Intents:      []string{"run containers", "build an image", "inspect containers", "list running containers", "run container", "build image", "inspect docker"},
		Examples:     []model.Example{{Intent: "list running containers", Command: `docker ps`}},
		Risk:         "dangerous",
	},
	{
		ID: "ffmpeg", Family: "media_transform", Commands: []string{"ffmpeg"},
		Description:  "Convert, extract, and transform audio and video media.",
		Capabilities: []string{"convert_media", "extract_audio", "video_processing"},
		Intents:      []string{"convert video", "extract audio", "transform media", "convert video", "extract audio", "convert media"},
		Examples:     []model.Example{{Intent: "convert video", Command: `ffmpeg -i input.mov output.mp4`}},
		Risk:         "medium",
	},
	{
		ID: "ffprobe", Family: "media_inspection", Commands: []string{"ffprobe"},
		Description:  "Inspect audio and video streams, codecs, duration, and metadata.",
		Capabilities: []string{"inspect_media", "media_metadata"},
		Intents:      []string{"inspect media", "read video metadata", "inspect media", "media metadata", "inspect video codec and duration metadata"},
		Examples:     []model.Example{{Intent: "inspect media metadata", Command: `ffprobe -v quiet -print_format json -show_format -show_streams input.mp4`}},
		Risk:         "safe",
	},
	{
		ID: "7zip", Family: "archive", Commands: []string{"7z", "7zz"},
		Description:  "List, test, compress, and extract archive files.",
		Capabilities: []string{"extract_archives", "create_archives", "inspect_archives"},
		Intents:      []string{"extract files", "create archives", "inspect archives", "extract archive", "create archive", "inspect zip"},
		Examples:     []model.Example{{Intent: "list archive contents", Command: `7z l archive.zip`}},
		Risk:         "medium", RewriteCommand: true,
	},
	{
		ID: "binwalk", Family: "firmware_analysis", Commands: []string{"binwalk"},
		Description:  "Identify and optionally extract files and data embedded in firmware and other binary images.",
		Capabilities: []string{"analyze_firmware", "identify_embedded_files", "extract_firmware"},
		Intents:      []string{"analyze firmware", "extract firmware", "identify embedded files", "scan firmware"},
		Examples:     []model.Example{{Intent: "analyze a firmware image", Command: `binwalk firmware.bin`}},
		Risk:         "medium",
	},
	{
		ID: "make", Family: "project_task", Commands: []string{"make", "gmake"},
		Description:  "Run declared build and automation targets from Makefiles.",
		Capabilities: []string{"build", "project_tasks", "automation"},
		Intents:      []string{"build a project", "run a Makefile", "execute a project task", "build project", "run make target"},
		Examples:     []model.Example{{Intent: "build a project", Command: `make`}},
		Risk:         "dangerous", RewriteCommand: true,
	},
}

func All() []Entry {
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}

func ByCommand(command string) (Entry, bool) {
	for _, entry := range entries {
		for _, candidate := range entry.Commands {
			if candidate == command {
				return entry, true
			}
		}
	}
	return Entry{}, false
}
