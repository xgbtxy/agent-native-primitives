package search

import (
	"github.com/xgbtxy/agent-native-primitives/internal/catalog"
	"github.com/xgbtxy/agent-native-primitives/internal/model"
	"testing"
	"time"
)

// TestCatalogRankingSmoke catches accidental catalog/ranker drift. It is not an
// effectiveness benchmark: queries and expected IDs are authored with the same
// catalog and must never be reported as evidence that agents are improved.
func TestCatalogRankingSmoke(t *testing.T) {
	index := model.Index{GeneratedAt: time.Now()}
	for _, entry := range catalog.All() {
		index.Tools = append(index.Tools, model.Tool{
			ID: entry.ID, Family: entry.Family, Command: entry.Commands[0], Status: "present", SemanticSource: "test_catalog", ResolverSource: "path",
			Description: entry.Description, Capabilities: entry.Capabilities,
			Intents: entry.Intents, Examples: entry.Examples, Risk: entry.Risk,
		})
	}

	cases := []struct {
		query string
		want  string
	}{
		{"搜索 panic 日志并显示上下文", "ripgrep"},
		{"查找所有 TypeScript 文件", "fd"},
		{"查询 JSON 配置中的端口", "jq"},
		{"读取 YAML 里的 server.port", "yq"},
		{"查看当前 Git 代码改动", "git"},
		{"发送 HTTP API 请求", "curl"},
		{"运行所有 Go 测试", "go"},
		{"运行 Python 脚本", "python"},
		{"查看视频编码和时长元数据", "ffprobe"},
		{"解压 zip 文件", "7zip"},
		{"查看 GitHub pull requests", "gh"},
		{"查看正在运行的容器", "docker"},
	}

	hits := 0
	for _, test := range cases {
		result := Find(index, test.query)
		if result.Match != nil && result.Match.ID == test.want {
			hits++
			continue
		}
		got := "<none>"
		if result.Match != nil {
			got = result.Match.ID
		}
		t.Logf("query %q: want %s at rank 1, got %s", test.query, test.want, got)
	}
	if hits != len(cases) {
		t.Fatalf("catalog ranking smoke: %d/%d cases ranked as authored", hits, len(cases))
	}
}
