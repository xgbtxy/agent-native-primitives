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
		{"search panic logs with context", "ripgrep"},
		{"find all TypeScript files", "fd"},
		{"query the port in a JSON config", "jq"},
		{"read server.port from YAML", "yq"},
		{"show current Git changes", "git"},
		{"send an HTTP API request", "curl"},
		{"run all Go tests", "go"},
		{"run a Python script", "python"},
		{"inspect video codec and duration metadata", "ffprobe"},
		{"extract a zip archive", "7zip"},
		{"view GitHub pull requests", "gh"},
		{"list running containers", "docker"},
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
