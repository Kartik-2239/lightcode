package components

import "testing"

func TestRankPathsSubstringFirst(t *testing.T) {
	paths := []string{
		"cmd/lightcode/main.go",
		"internal/server/agent/compaction.go",
		"internal/server/config/customization.go",
		"internal/server/prompt/assistant_prompt.go",
		"internal/tui/views/homepage.go",
	}

	// "main.go" is a fuzzy subsequence of compaction.go etc., but only the real
	// main.go should rank — substring matches win and pure-fuzzy filler is dropped.
	items := rankPaths("main.go", paths)
	if len(items) != 1 {
		t.Fatalf("expected only main.go matches, got %d: %v", len(items), items)
	}
	if got := items[0].(fileItem).path; got != "cmd/lightcode/main.go" {
		t.Fatalf("top result = %q", got)
	}

	// An abbreviation with no substring hit falls back to fuzzy matching.
	got := rankPaths("hmpage", paths)
	if len(got) == 0 || got[0].(fileItem).path != "internal/tui/views/homepage.go" {
		t.Fatalf("fuzzy fallback failed: %v", got)
	}
}
