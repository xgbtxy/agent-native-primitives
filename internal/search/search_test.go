package search

import (
	"encoding/json"
	"github.com/xgbtxy/agent-native-primitives/internal/model"
	"strings"
	"testing"
	"time"
)

func TestFindChineseIntentRanksRipgrepFirst(t *testing.T) {
	index := model.Index{
		GeneratedAt: time.Now(),
		Tools: []model.Tool{
			{ID: "git", Command: "git", Status: "present", Description: "source history", Intents: []string{"查看代码历史"}},
			{ID: "ripgrep", Command: "rg", Status: "present", Description: "search files", Capabilities: []string{"search_logs"}, Intents: []string{"搜索日志", "显示匹配上下文"}, Examples: []model.Example{{Intent: "搜索日志并显示上下文", Command: `rg -n -C 10 "panic" error.log`}}},
		},
	}

	result := Find(index, "搜索 panic 日志并显示上下文")
	if result.Match == nil {
		t.Fatal("expected at least one candidate")
	}
	if got := result.Match.ID; got != "ripgrep" {
		t.Fatalf("expected ripgrep first, got %q", got)
	}
}

func TestFindProjectScript(t *testing.T) {
	index := model.Index{
		GeneratedAt: time.Now(),
		Tools: []model.Tool{{
			ID: "npm:test", Command: "npm run test", Status: "present", ProjectDefined: true,
			Description: "Project-defined npm script: vitest run", Intents: []string{"test", "vitest run", "运行项目脚本"},
		}},
	}

	result := Find(index, "运行项目测试")
	if result.Match == nil || result.Match.ID != "npm:test" {
		t.Fatalf("expected npm:test, got %#v", result.Match)
	}
}

func TestFindReturnsNoCandidateForUnrelatedIntent(t *testing.T) {
	index := model.Index{GeneratedAt: time.Now(), Tools: []model.Tool{{ID: "rg", Command: "rg", Status: "present"}}}
	result := Find(index, "编辑视频字幕")
	if result.Match != nil {
		t.Fatalf("expected no candidates, got %#v", result.Match)
	}
}

func TestFindUnknownInstalledCommandByExactName(t *testing.T) {
	index := model.Index{GeneratedAt: time.Now(), Tools: []model.Tool{{ID: "acme-tool", Command: "acme-tool", Status: "present_unclassified"}}}
	result := Find(index, "acme-tool")
	if result.Match == nil || result.Match.Command != "acme-tool" {
		t.Fatalf("expected exact unknown command match, got %#v", result.Match)
	}
}

func TestFindDoesNotReturnGenericSingleKeywordOverlap(t *testing.T) {
	index := model.Index{GeneratedAt: time.Now(), Tools: []model.Tool{
		{ID: "jq", Command: "jq", Status: "present", Intents: []string{"查询 JSON", "筛选 JSON"}},
		{ID: "gh", Command: "gh", Status: "present", Intents: []string{"查询 GitHub", "查看 GitHub PR"}},
	}}
	result := Find(index, "查询 JSON 配置中的端口")
	if result.Match == nil || result.Match.ID != "jq" {
		t.Fatalf("expected only jq, got %#v", result.Match)
	}
}

func TestJSONQueryDoesNotReturnHTTPToolFromExampleOnly(t *testing.T) {
	index := model.Index{Scope: model.Scope{ID: "scope", ProjectName: "demo"}, Tools: []model.Tool{
		{ID: "jq", Command: "jq", Status: "present", Description: "Query JSON", Intents: []string{"查询 JSON"}},
		{ID: "curl", Command: "curl", Status: "present", Description: "HTTP client", Intents: []string{"发送 HTTP 请求"}, Examples: []model.Example{{Intent: "获取 JSON API", Command: "curl example"}}},
	}}
	result := Find(index, "查询 JSON 配置中的端口")
	if result.Match == nil || result.Match.ID != "jq" {
		t.Fatalf("expected only jq, got %#v", result.Match)
	}
}

func TestAgentCandidateOmitsSensitiveAndUncalibratedFields(t *testing.T) {
	index := model.Index{Scope: model.Scope{ID: "scope", ProjectName: "demo"}, Tools: []model.Tool{{
		ID: "jq", Family: "structured_data_query", Command: "jq", ResolvedPath: `C:\Users\private\jq.exe`, Status: "present", SemanticSource: "builtin_catalog", ResolverSource: "path",
		Description: "query json", Intents: []string{"query json"}, Risk: "safe",
	}}}
	result := Find(index, "query json")
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, forbidden := range []string{"resolved_path", "C:\\\\Users", "confidence", "risk", "score"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("agent result contains forbidden field/value %q: %s", forbidden, text)
		}
	}
}

func TestMissingRuntimeIsNeverRecommended(t *testing.T) {
	index := model.Index{Scope: model.Scope{ID: "scope", ProjectName: "demo"}, Tools: []model.Tool{{
		ID: "npm:deploy", Command: "npm run deploy", Status: "missing_runtime", SemanticSource: "package.json", ResolverSource: "project_manifest+path",
		Description: "Project deploy task", Intents: []string{"deploy project"},
	}}}
	result := Find(index, "deploy project")
	if result.Match != nil {
		t.Fatalf("missing runtime must not be recommended: %#v", result.Match)
	}
}

func TestReadySignalUsesNarrowProbeEvidence(t *testing.T) {
	index := model.Index{Tools: []model.Tool{{ID: "binwalk", Command: "tooltruth exec binwalk --", Status: "ready", Managed: true, ResolverSource: "managed_digest_matched", SemanticSource: "builtin_catalog", Intents: []string{"分析固件"}}}}
	result := Find(index, "分析固件")
	if result.Match == nil || result.Match.Signal.Availability != "managed_digest_matched" || result.Match.Signal.Behavior != "help_signature_probe_passed" {
		t.Fatalf("unexpected ready evidence: %#v", result.Match)
	}
}
