package llm

import (
	"errors"
	"testing"

	"github.com/Kartik-2239/lightcode/internal/server/oauth"
)

func TestPreferredCopilotEndpointPrefersChat(t *testing.T) {
	got := preferredCopilotEndpoint([]string{"/responses", "/chat/completions"})
	if got != "/chat/completions" {
		t.Fatalf("expected chat completions endpoint, got %q", got)
	}
}

func TestPreferredCopilotEndpointFallsBackToResponses(t *testing.T) {
	got := preferredCopilotEndpoint([]string{"/responses", "ws:/responses"})
	if got != "/responses" {
		t.Fatalf("expected responses endpoint, got %q", got)
	}
}

func TestCopilotModelIDUsesID(t *testing.T) {
	got := copilotModelID(oauth.CopilotModel{Name: "GPT-5.5", ID: "gpt-5.5"})
	if got != "gpt-5.5" {
		t.Fatalf("expected model ID, got %q", got)
	}
}

func TestCopilotIntegratorFallbackModelForGeminiPreview(t *testing.T) {
	got, ok := copilotIntegratorFallbackModel(
		oauth.CopilotModel{Name: "Gemini 3.1 Pro (Preview)", ID: "gemini-3.1-pro-preview"},
		errors.New(`copilot request failed: model_not_available_for_integrator`),
	)
	if !ok || got != "gemini-2.5-pro" {
		t.Fatalf("expected gemini fallback, got %q ok=%v", got, ok)
	}
}

func TestCopilotIntegratorFallbackModelDoesNotMaskOtherErrors(t *testing.T) {
	if got, ok := copilotIntegratorFallbackModel(oauth.CopilotModel{ID: "gpt-5-mini"}, errors.New("model_not_available_for_integrator")); ok {
		t.Fatalf("did not expect fallback for non-gemini model, got %q", got)
	}
	if got, ok := copilotIntegratorFallbackModel(oauth.CopilotModel{ID: "gemini-3.1-pro-preview"}, errors.New("network error")); ok {
		t.Fatalf("did not expect fallback for unrelated error, got %q", got)
	}
}

func TestCopilotResponseTextJoinsOutputText(t *testing.T) {
	got := copilotResponseText(oauth.CopilotResponsesResponse{
		Output: []oauth.CopilotResponsesOutputItem{
			{Content: []oauth.CopilotResponsesOutputContent{{Text: "one"}}},
			{Content: []oauth.CopilotResponsesOutputContent{{Text: "two"}}},
		},
	})
	if got != "one\ntwo" {
		t.Fatalf("expected joined text, got %q", got)
	}
}
