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
		Intents:      []string{"搜索代码", "查找字符串", "搜索日志", "显示匹配上下文", "find text in files", "search source code", "grep recursively", "search logs"},
		Examples:     []model.Example{{Intent: "搜索代码并显示行号", Command: `rg -n "validateToken" src`}, {Intent: "搜索日志并显示上下文", Command: `rg -n -C 10 "panic" error.log`}},
		Risk:         "safe",
	},
	{
		ID: "fd", Family: "file_discovery", Commands: []string{"fd", "fdfind"},
		Description:  "Fast filesystem search by file name, extension, type, or glob.",
		Capabilities: []string{"find_files", "find_directories", "filesystem_discovery"},
		Intents:      []string{"查找文件", "按文件名搜索", "查找目录", "查找 TypeScript 文件", "find files", "find directories", "find TypeScript files", "search by extension"},
		Examples:     []model.Example{{Intent: "查找 TypeScript 文件", Command: `fd -e ts`}},
		Risk:         "safe", RewriteCommand: true,
	},
	{
		ID: "git", Family: "version_control", Commands: []string{"git"},
		Description:  "Inspect and manage source history, diffs, branches, commits, and repositories.",
		Capabilities: []string{"version_control", "source_history", "diff", "repository_inspection"},
		Intents:      []string{"查看代码历史", "查看改动", "比较文件", "版本控制", "inspect git history", "show diff", "find commit", "repository status"},
		Examples:     []model.Example{{Intent: "查看当前改动", Command: `git diff --stat`}, {Intent: "查找修改某行的提交", Command: `git blame -L 40,90 -- path/to/file`}},
		Risk:         "medium",
	},
	{
		ID: "jq", Family: "structured_data_query", Commands: []string{"jq"},
		Description:  "Query, filter, transform, and format JSON from files or stdin.",
		Capabilities: []string{"query_json", "transform_json", "format_json"},
		Intents:      []string{"解析 JSON", "查询 JSON", "筛选 JSON", "格式化 JSON", "query json", "filter json", "transform json"},
		Examples:     []model.Example{{Intent: "读取 JSON 字段", Command: `jq -r '.server.port' config.json`}},
		Risk:         "safe",
	},
	{
		ID: "yq", Family: "structured_data_query", Commands: []string{"yq"},
		Description:  "Query and transform YAML, JSON, XML, CSV, TOML, and properties files.",
		Capabilities: []string{"query_yaml", "transform_yaml", "query_structured_data"},
		Intents:      []string{"解析 YAML", "查询 YAML", "读取 YAML", "修改 YAML", "query yaml", "read yaml", "filter yaml", "transform yaml"},
		Examples:     []model.Example{{Intent: "读取 YAML 字段", Command: `yq '.server.port' config.yaml`}},
		Risk:         "medium",
	},
	{
		ID: "curl", Family: "http_client", Commands: []string{"curl"},
		Description:  "Transfer data over HTTP and many other network protocols; inspect APIs and download files.",
		Capabilities: []string{"http_request", "api_call", "download", "network_probe"},
		Intents:      []string{"发送 HTTP 请求", "调用 API", "下载文件", "测试接口", "http request", "call api", "download url"},
		Examples:     []model.Example{{Intent: "获取 JSON API", Command: `curl -fsS https://example.com/api`}},
		Risk:         "medium",
	},
	{
		ID: "gh", Family: "github", Commands: []string{"gh"},
		Description:  "Work with GitHub repositories, issues, pull requests, releases, and API data.",
		Capabilities: []string{"github", "pull_requests", "issues", "github_api"},
		Intents:      []string{"查看 GitHub PR", "创建 issue", "查询 GitHub", "github pull request", "github issues", "github api"},
		Examples:     []model.Example{{Intent: "查看当前仓库 PR", Command: `gh pr list`}},
		Risk:         "medium",
	},
	{
		ID: "go", Family: "go_toolchain", Commands: []string{"go"},
		Description:  "Build, test, format, inspect, and manage Go modules and programs.",
		Capabilities: []string{"build_go", "test_go", "format_go", "go_modules"},
		Intents:      []string{"编译 Go", "运行 Go 测试", "格式化 Go", "build go", "test go", "go modules"},
		Examples:     []model.Example{{Intent: "运行所有 Go 测试", Command: `go test ./...`}},
		Risk:         "medium",
	},
	{
		ID: "python", Family: "python_runtime", Commands: []string{"python", "python3", "py"},
		Description:  "Run Python programs and scripts.",
		Capabilities: []string{"run_python", "scripting", "data_processing"},
		Intents:      []string{"运行 Python", "执行 Python 脚本", "数据处理", "run python", "python script"},
		Examples:     []model.Example{{Intent: "运行 Python 脚本", Command: `python script.py`}},
		Risk:         "medium", RewriteCommand: true,
	},
	{
		ID: "uv", Family: "python_environment", Commands: []string{"uv"},
		Description:  "Manage Python projects, dependencies, lockfiles, and environments.",
		Capabilities: []string{"python_packages", "python_environment", "python_lockfiles"},
		Intents:      []string{"安装 Python 依赖", "管理 Python 环境", "管理 Python 项目", "python package manager", "python environment"},
		Examples:     []model.Example{{Intent: "同步 Python 项目依赖", Command: `uv sync`}},
		Risk:         "medium",
	},
	{
		ID: "uvx", Family: "ephemeral_python_tool", Commands: []string{"uvx"},
		Description:  "Run Python command-line tools in isolated, cached environments without global installation.",
		Capabilities: []string{"isolated_python_tools", "ephemeral_tool_execution"},
		Intents:      []string{"隔离运行 Python 工具", "临时运行 Python 工具", "run python tool", "run tool without installing"},
		Examples:     []model.Example{{Intent: "临时运行 Python 工具", Command: `uvx <tool> --help`}},
		Risk:         "medium",
	},
	{
		ID: "node", Family: "javascript_runtime", Commands: []string{"node"},
		Description:  "Run JavaScript and Node.js applications.",
		Capabilities: []string{"run_javascript", "node_runtime"},
		Intents:      []string{"运行 JavaScript", "执行 Node 脚本", "run javascript", "node runtime"},
		Examples:     []model.Example{{Intent: "运行 Node 脚本", Command: `node script.js`}},
		Risk:         "medium",
	},
	{
		ID: "docker", Family: "container_runtime", Commands: []string{"docker"},
		Description:  "Build and run isolated containers and inspect container images, networks, and volumes.",
		Capabilities: []string{"containers", "container_build", "isolated_runtime"},
		Intents:      []string{"运行容器", "构建镜像", "查看容器", "run container", "build image", "inspect docker"},
		Examples:     []model.Example{{Intent: "查看运行中的容器", Command: `docker ps`}},
		Risk:         "dangerous",
	},
	{
		ID: "ffmpeg", Family: "media_transform", Commands: []string{"ffmpeg"},
		Description:  "Convert, extract, and transform audio and video media.",
		Capabilities: []string{"convert_media", "extract_audio", "video_processing"},
		Intents:      []string{"转换视频", "提取音频", "转换媒体", "convert video", "extract audio", "convert media"},
		Examples:     []model.Example{{Intent: "转换视频", Command: `ffmpeg -i input.mov output.mp4`}},
		Risk:         "medium",
	},
	{
		ID: "ffprobe", Family: "media_inspection", Commands: []string{"ffprobe"},
		Description:  "Inspect audio and video streams, codecs, duration, and metadata.",
		Capabilities: []string{"inspect_media", "media_metadata"},
		Intents:      []string{"查看媒体信息", "读取视频元数据", "inspect media", "media metadata"},
		Examples:     []model.Example{{Intent: "查看媒体元数据", Command: `ffprobe -v quiet -print_format json -show_format -show_streams input.mp4`}},
		Risk:         "safe",
	},
	{
		ID: "7zip", Family: "archive", Commands: []string{"7z", "7zz"},
		Description:  "List, test, compress, and extract archive files.",
		Capabilities: []string{"extract_archives", "create_archives", "inspect_archives"},
		Intents:      []string{"解压文件", "压缩文件", "查看压缩包", "extract archive", "create archive", "inspect zip"},
		Examples:     []model.Example{{Intent: "查看压缩包内容", Command: `7z l archive.zip`}},
		Risk:         "medium", RewriteCommand: true,
	},
	{
		ID: "binwalk", Family: "firmware_analysis", Commands: []string{"binwalk"},
		Description:  "Identify and optionally extract files and data embedded in firmware and other binary images.",
		Capabilities: []string{"analyze_firmware", "identify_embedded_files", "extract_firmware"},
		Intents:      []string{"分析固件", "拆包固件", "识别固件内嵌文件", "提取路由器固件", "analyze firmware", "scan firmware", "extract firmware", "identify embedded files"},
		Examples:     []model.Example{{Intent: "分析固件镜像", Command: `binwalk firmware.bin`}},
		Risk:         "medium",
	},
	{
		ID: "make", Family: "project_task", Commands: []string{"make", "gmake"},
		Description:  "Run declared build and automation targets from Makefiles.",
		Capabilities: []string{"build", "project_tasks", "automation"},
		Intents:      []string{"构建项目", "运行 Makefile", "执行项目任务", "build project", "run make target"},
		Examples:     []model.Example{{Intent: "构建项目", Command: `make`}},
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
