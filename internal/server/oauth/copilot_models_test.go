package oauth

import (
	"encoding/json"
	"testing"

	"github.com/Kartik-2239/lightcode/internal/server/config"
)

func TestCopilotModelsResponseAcceptsWrappedModels(t *testing.T) {
	var response copilotModelsResponse
	if err := json.Unmarshal([]byte(`{"models":[{"name":"gpt-5.5","supported_endpoints":["/chat/completions"]}]}`), &response); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(response) != 1 || response[0].Name != "gpt-5.5" {
		t.Fatalf("expected wrapped model to parse, got %#v", response)
	}
}

func TestCopilotModelsResponseAcceptsKeyedModels(t *testing.T) {
	var response copilotModelsResponse
	if err := json.Unmarshal([]byte(`{"data":{"gpt-5.5":{"supported_endpoints":["/chat/completions"]}}}`), &response); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(response) != 1 || response[0].Name != "gpt-5.5" {
		t.Fatalf("expected keyed model to parse, got %#v", response)
	}
}

func TestDefaultCopilotPickerModelNamesMatchCopilotPicker(t *testing.T) {
	got := DefaultCopilotPickerModelNames()
	want := []string{"Auto", "GPT-5.4 mini (default)", "GPT-5 mini", "Claude Haiku 4.5", "Gemini 3.1 Pro (Preview)"}
	if len(got) != len(want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %#v, got %#v", want, got)
		}
	}
}

func TestResolveCopilotModelMapsPickerLabelToID(t *testing.T) {
	model, ok := ResolveCopilotModel("Gemini 3.1 Pro (Preview)", nil)
	if !ok {
		t.Fatal("expected picker label to resolve")
	}
	if model.ID != "gemini-3.1-pro-preview" || model.SupportedEndpoints[0] != "/chat/completions" {
		t.Fatalf("unexpected model mapping: %#v", model)
	}
}

func TestResolveCopilotModelMapsAutoToDefaultID(t *testing.T) {
	model, ok := ResolveCopilotModel("Auto", nil)
	if !ok {
		t.Fatal("expected Auto to resolve")
	}
	if model.ID != "gpt-5.4-mini" || model.SupportedEndpoints[0] != "/responses" {
		t.Fatalf("unexpected Auto mapping: %#v", model)
	}
}

func TestGetCopilotAuthValDoesNotFallbackToLegacyGithub(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := config.UpdateAuthVal("github", config.AuthVal{
		Type:   "oauth",
		Access: "legacy-token",
	}); err != nil {
		t.Fatalf("UpdateAuthVal returned error: %v", err)
	}

	authVal, err := getCopilotAuthVal()
	if err == nil {
		t.Fatalf("expected missing copilot auth error, got auth %#v", authVal)
	}
}
