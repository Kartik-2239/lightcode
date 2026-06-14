package views

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kartik-2239/lightcode/internal/server/api"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func TestDefaultLogoutProviders(t *testing.T) {
	providers := defaultLogoutProviders()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %#v", providers)
	}
	if providers[0].name != "codex" || providers[0].args[0] != "logout" {
		t.Fatalf("expected codex logout provider, got %#v", providers[0])
	}
	if providers[1].name != "copilot" {
		t.Fatalf("expected copilot logout provider, got %#v", providers[1])
	}
}

func TestDefaultLoginProvidersIncludesCopilot(t *testing.T) {
	providers := defaultLoginProviders()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %#v", providers)
	}
	if providers[0].name != "codex" {
		t.Fatalf("expected codex login provider first, got %#v", providers[0])
	}
	if providers[1].name != "copilot" {
		t.Fatalf("expected copilot login provider, got %#v", providers[1])
	}
}

func TestClearCodexAuthStateClearsCurrentModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := config.CreateConfig([]string{"codex"}, map[string]string{}, map[string]string{}, map[string][]string{}); err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}
	if err := config.SetCurrentModel(config.ResModel{Model: "gpt-5.5", BaseUrl: config.CodexAuthProvider}); err != nil {
		t.Fatalf("SetCurrentModel returned error: %v", err)
	}
	if err := clearCodexAuthState(); err != nil {
		t.Fatalf("clearCodexAuthState returned error: %v", err)
	}
	if current := config.GetCustomization().CurrentModel; current.Model != "" {
		t.Fatalf("expected current model to be cleared, got %#v", current)
	}
}

func TestClearCopilotAuthStateClearsCurrentModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := config.CreateConfig([]string{"copilot"}, map[string]string{}, map[string]string{}, map[string][]string{}); err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}
	if err := config.SetCurrentModel(config.ResModel{Model: "gpt-5.5", BaseUrl: config.CopilotAuthProvider}); err != nil {
		t.Fatalf("SetCurrentModel returned error: %v", err)
	}
	if err := clearCopilotAuthState(); err != nil {
		t.Fatalf("clearCopilotAuthState returned error: %v", err)
	}
	if current := config.GetCustomization().CurrentModel; current.Model != "" {
		t.Fatalf("expected current model to be cleared, got %#v", current)
	}
}

func TestDedupeModelsKeepsFirstEntry(t *testing.T) {
	models := []api.ModelInfo{
		{Model: "gpt-5.5", BaseUrl: "codex", Provider: "codex auth", LastUsed: 2},
		{Model: "gpt-5.5", BaseUrl: "codex", Provider: "codex auth", LastUsed: 0},
		{Model: "gpt-5.4-mini", BaseUrl: "codex", Provider: "codex auth", LastUsed: 0},
	}

	got := dedupeModels(models)
	if len(got) != 2 {
		t.Fatalf("expected 2 models, got %#v", got)
	}
	if got[0].LastUsed != 2 {
		t.Fatalf("expected first matching model to be kept, got %#v", got[0])
	}
}

func TestDefaultEffortOptionsIncludesExtraHigh(t *testing.T) {
	options := defaultEffortOptions()
	if len(options) != 4 {
		t.Fatalf("expected 4 options, got %#v", options)
	}
	last := options[len(options)-1]
	if last.label != "extra high" || last.value != "xhigh" {
		t.Fatalf("expected extra high to map to xhigh, got %#v", last)
	}
}

func TestEffortIndexForValueDefaultsToMedium(t *testing.T) {
	if got := effortIndexForValue(defaultEffortOptions(), ""); got != 1 {
		t.Fatalf("expected empty effort to default to medium index, got %d", got)
	}
}

func TestModelStatusNameIncludesEffort(t *testing.T) {
	model := api.ModelInfo{Model: "gpt-5.5", BaseUrl: "codex", ReasoningEffort: "high"}
	if got := modelStatusName(model); got != "gpt-5.5 high" {
		t.Fatalf("expected effort in model status, got %q", got)
	}
}

func TestContextWindowForCodexModel(t *testing.T) {
	model := api.ModelInfo{Model: "gpt-5.5", BaseUrl: "codex"}
	if got := contextWindowForModel(model); got != 258000 {
		t.Fatalf("expected 258000 context window, got %d", got)
	}
}

func TestRenderStatusLineUsesCachedGitInfo(t *testing.T) {
	model := api.ModelInfo{Model: "gpt-5.5", BaseUrl: "codex", ReasoningEffort: "high"}
	line := renderStatusLine(model, 620000, 120, statusLineGitInfo{Branch: "main", Changes: "No changes"})
	if !strings.Contains(line, "gpt-5.5 high") {
		t.Fatalf("expected model and effort in status line, got %q", line)
	}
	if !strings.Contains(line, "main") || !strings.Contains(line, "No changes") {
		t.Fatalf("expected cached git info in status line, got %q", line)
	}
}

func TestExpandWorkingDirExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := expandWorkingDir("~/repo")
	want := filepath.Join(home, "repo")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestShouldRefreshGitAfterFilesystemToolCall(t *testing.T) {
	if !shouldRefreshGitAfterToolCall(models.StoredMessageData{Role: "tool_call", ToolCalls: []models.StoredToolCall{{Name: "bash"}}}) {
		t.Fatal("expected bash tool call to refresh git status")
	}
	if shouldRefreshGitAfterToolCall(models.StoredMessageData{Role: "tool_call", ToolCalls: []models.StoredToolCall{{Name: "read_file"}}}) {
		t.Fatal("did not expect read-only tool call to refresh git status")
	}
}
