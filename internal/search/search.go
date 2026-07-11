package search

import (
	"github.com/xgbtxy/agent-native-primitives/internal/model"
	"sort"
	"strings"
	"unicode"
)

func Find(index model.Index, query string) model.FindResult {
	query = strings.TrimSpace(query)
	var candidates []model.Candidate
	for _, tool := range index.Tools {
		if tool.Status != "present" && tool.Status != "present_unclassified" && tool.Status != "ready" {
			continue
		}
		score, match := semanticMatch(tool, query)
		if score == 0 {
			continue
		}
		candidates = append(candidates, model.Candidate{
			ID: tool.ID, Family: tool.Family, Command: tool.Command,
			Claim: tool.Description,
			Signal: model.Evidence{
				Semantics:    semanticsEvidence(tool),
				Availability: availabilityEvidence(tool),
				Behavior:     behaviorEvidence(tool),
				Match:        match,
			},
			DeclaredExample: bestExample(tool, strings.ToLower(query)), Score: score,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Command < candidates[j].Command
		}
		return candidates[i].Score > candidates[j].Score
	})

	// One capability family contributes at most one signal. This prevents an AI
	// from receiving several interchangeable tools as separate "facts".
	deduped := candidates[:0]
	seenFamilies := map[string]bool{}
	for _, candidate := range candidates {
		family := candidate.Family
		if family == "" {
			family = candidate.ID
		}
		if seenFamilies[family] {
			continue
		}
		seenFamilies[family] = true
		deduped = append(deduped, candidate)
	}
	candidates = deduped
	result := model.FindResult{
		Scope: model.ResultScope{ID: index.Scope.ID, Project: index.Scope.ProjectName},
	}
	if len(candidates) > 0 {
		result.Match = &candidates[0]
	} else {
		result.Status = "no_supported_match"
	}
	return result
}

func semanticMatch(tool model.Tool, query string) (int, string) {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return 0, ""
	}
	if normalized == strings.ToLower(tool.ID) || normalized == strings.ToLower(tool.Command) {
		return 100, "exact_command"
	}
	for _, capability := range tool.Capabilities {
		readable := strings.ReplaceAll(strings.ToLower(capability), "_", " ")
		if normalized == readable {
			return 75, "capability:" + capability
		}
	}
	bestScore := 0
	bestMatch := ""
	for _, intent := range tool.Intents {
		score := intentMatchScore(normalized, strings.ToLower(strings.TrimSpace(intent)))
		if score > bestScore {
			bestScore = score
			bestMatch = "intent:" + intent
		}
	}
	if bestScore > 0 && tool.Status == "present" {
		bestScore += 4
	}
	return bestScore, bestMatch
}

func intentMatchScore(query, intent string) int {
	if query == "" || intent == "" {
		return 0
	}
	queryWords, intentWords := asciiWords(query), asciiWords(intent)
	if len(intentWords) > 0 && !containsAll(queryWords, intentWords) {
		return 0
	}
	if query == intent {
		return 80
	}
	if strings.Contains(query, intent) || strings.Contains(intent, query) {
		return 65
	}
	queryHan, intentHan := hanOnly(query), hanOnly(intent)
	if len([]rune(intentHan)) >= 2 && strings.Contains(queryHan, intentHan) {
		return 60
	}
	if len(intentWords) > 0 {
		return 55
	}
	queryBigrams, intentBigrams := hanBigrams(queryHan), hanBigrams(intentHan)
	overlap := intersectionSize(queryBigrams, intentBigrams)
	if len(intentBigrams) >= 2 && overlap >= 2 && float64(overlap)/float64(len(intentBigrams)) >= 0.5 {
		return 50
	}
	return 0
}

func availabilityEvidence(tool model.Tool) string {
	switch tool.ResolverSource {
	case "managed_digest_matched":
		return "managed_digest_matched"
	case "project_manifest+path":
		return "project_declared_and_runtime_resolved"
	case "path":
		if tool.Status == "ready" {
			return "path_digest_health_record"
		}
		return "path_resolved"
	default:
		return "resolved"
	}
}

func behaviorEvidence(tool model.Tool) string {
	if tool.Status == "ready" {
		return "help_signature_probe_passed"
	}
	return "not_verified"
}

func semanticsEvidence(tool model.Tool) string {
	switch tool.SemanticSource {
	case "builtin_catalog":
		return "curated_name_mapping"
	case "project_descriptor":
		return "project_declared"
	case "package.json", "Makefile", "makefile", "GNUmakefile":
		return "manifest_declared"
	case "none", "":
		return "none"
	default:
		return "declared:" + tool.SemanticSource
	}
}

func bestExample(tool model.Tool, query string) string {
	best := ""
	bestScore := -1
	queryTerms := terms(query)
	for _, example := range tool.Examples {
		score := 0
		intent := strings.ToLower(example.Intent)
		if strings.Contains(query, intent) || strings.Contains(intent, query) {
			score += 10
		}
		exampleTerms := terms(intent)
		for term := range queryTerms {
			if exampleTerms[term] {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			best = example.Command
		}
	}
	return best
}

func hanOnly(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func hanBigrams(value string) map[string]bool {
	runes := []rune(value)
	result := map[string]bool{}
	for i := 0; i+1 < len(runes); i++ {
		result[string(runes[i:i+2])] = true
	}
	return result
}

func intersectionSize(left, right map[string]bool) int {
	count := 0
	for value := range left {
		if right[value] {
			count++
		}
	}
	return count
}

func asciiWords(value string) map[string]bool {
	result := map[string]bool{}
	var builder strings.Builder
	flush := func() {
		if builder.Len() >= 2 {
			result[builder.String()] = true
		}
		builder.Reset()
	}
	for _, r := range strings.ToLower(value) {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			builder.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return result
}

func containsAll(haystack, needles map[string]bool) bool {
	for needle := range needles {
		if !haystack[needle] {
			return false
		}
	}
	return true
}

func terms(value string) map[string]bool {
	value = strings.ToLower(value)
	result := map[string]bool{}
	var ascii strings.Builder
	var han []rune
	flushASCII := func() {
		if ascii.Len() >= 2 {
			result[ascii.String()] = true
		}
		ascii.Reset()
	}
	flushHan := func() {
		if len(han) == 1 {
			result[string(han)] = true
		}
		for i := 0; i+1 < len(han); i++ {
			result[string(han[i:i+2])] = true
		}
		han = han[:0]
	}
	for _, r := range value {
		switch {
		case unicode.Is(unicode.Han, r):
			flushASCII()
			han = append(han, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			flushHan()
			ascii.WriteRune(r)
		default:
			flushASCII()
			flushHan()
		}
	}
	flushASCII()
	flushHan()
	return result
}
