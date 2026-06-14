package oauth

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CopilotModel struct {
	ID                  string                   `json:"id"`
	ModelPickerCategory string                   `json:"model_picker_category"`
	ModelPickerEnabled  bool                     `json:"model_picker_enabled"`
	Name                string                   `json:"name"`
	Object              string                   `json:"object"`
	Policy              CopilotModelPolicy       `json:"policy"`
	Preview             bool                     `json:"preview"`
	SupportedEndpoints  []string                 `json:"supported_endpoints"`
	Vendor              string                   `json:"vendor"`
	Version             string                   `json:"version"`
	Capabilities        CopilotModelCapabilities `json:"capabilities"`
}

type CopilotModelPolicy struct {
	State string `json:"state"`
	Terms string `json:"terms"`
}

type CopilotModelCapabilities struct {
	Family    string               `json:"family"`
	Limits    CopilotModelLimits   `json:"limits"`
	Object    string               `json:"object"`
	Supports  CopilotModelSupports `json:"supports"`
	Tokenizer string               `json:"tokenizer"`
	Type      string               `json:"type"`
}

type CopilotModelLimits struct {
	MaxContextWindowTokens      int                      `json:"max_context_window_tokens"`
	MaxNonStreamingOutputTokens int                      `json:"max_non_streaming_output_tokens"`
	MaxOutputTokens             int                      `json:"max_output_tokens"`
	MaxPromptTokens             int                      `json:"max_prompt_tokens"`
	Vision                      CopilotModelVisionLimits `json:"vision"`
}

type CopilotModelVisionLimits struct {
	MaxPromptImageSize  int      `json:"max_prompt_image_size"`
	MaxPromptImages     int      `json:"max_prompt_images"`
	SupportedMediaTypes []string `json:"supported_media_types"`
}

type CopilotModelSupports struct {
	AdaptiveThinking  bool     `json:"adaptive_thinking"`
	MaxThinkingBudget int      `json:"max_thinking_budget"`
	MinThinkingBudget int      `json:"min_thinking_budget"`
	ParallelToolCalls bool     `json:"parallel_tool_calls"`
	ReasoningEffort   []string `json:"reasoning_effort"`
	Streaming         bool     `json:"streaming"`
	StructuredOutputs bool     `json:"structured_outputs"`
	ToolCalls         bool     `json:"tool_calls"`
	Vision            bool     `json:"vision"`
}

func DefaultCopilotPickerModelNames() []string {
	defaults := defaultCopilotPickerModels()
	names := make([]string, 0, len(defaults))
	for _, model := range defaults {
		names = append(names, model.Name)
	}
	return names
}

func ResolveCopilotModel(name string, liveModels []CopilotModel) (CopilotModel, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return CopilotModel{}, false
	}

	for _, model := range defaultCopilotPickerModels() {
		if trimmed == model.Name || trimmed == model.ID {
			return model, true
		}
	}

	for _, model := range liveModels {
		if trimmed == model.Name || trimmed == model.ID {
			return model, true
		}
	}

	return CopilotModel{}, false
}

func defaultCopilotPickerModels() []CopilotModel {
	return []CopilotModel{
		{
			Name:                "Auto",
			ID:                  "gpt-5.4-mini",
			ModelPickerEnabled:  true,
			SupportedEndpoints:  []string{"/responses"},
			ModelPickerCategory: "lightweight",
		},
		{
			Name:                "GPT-5.4 mini (default)",
			ID:                  "gpt-5.4-mini",
			ModelPickerEnabled:  true,
			SupportedEndpoints:  []string{"/responses"},
			ModelPickerCategory: "lightweight",
		},
		{
			Name:                "GPT-5 mini",
			ID:                  "gpt-5-mini",
			ModelPickerEnabled:  true,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
			ModelPickerCategory: "lightweight",
		},
		{
			Name:                "Claude Haiku 4.5",
			ID:                  "claude-haiku-4.5",
			ModelPickerEnabled:  true,
			SupportedEndpoints:  []string{"/chat/completions", "/v1/messages"},
			ModelPickerCategory: "lightweight",
		},
		{
			Name:                "Gemini 3.1 Pro (Preview)",
			ID:                  "gemini-3.1-pro-preview",
			ModelPickerEnabled:  true,
			SupportedEndpoints:  []string{"/chat/completions"},
			ModelPickerCategory: "powerful",
			Preview:             true,
		},
	}
}

type copilotModelsResponse []CopilotModel

func (r *copilotModelsResponse) UnmarshalJSON(data []byte) error {
	var direct []CopilotModel
	if err := json.Unmarshal(data, &direct); err == nil {
		*r = direct
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for _, key := range []string{"data", "models", "items"} {
		models, ok, err := decodeCopilotModels(raw[key])
		if err != nil {
			return err
		}
		if ok {
			*r = models
			return nil
		}
	}

	for _, value := range raw {
		models, ok, err := decodeCopilotModels(value)
		if err != nil {
			return err
		}
		if ok {
			*r = models
			return nil
		}
	}

	return fmt.Errorf("copilot models response did not contain a model list")
}

func decodeCopilotModels(data json.RawMessage) ([]CopilotModel, bool, error) {
	if len(data) == 0 {
		return nil, false, nil
	}

	var models []CopilotModel
	if err := json.Unmarshal(data, &models); err == nil {
		return nonEmptyCopilotModels(models), len(models) > 0, nil
	}

	var keyed map[string]CopilotModel
	if err := json.Unmarshal(data, &keyed); err == nil && len(keyed) > 0 {
		models = make([]CopilotModel, 0, len(keyed))
		for key, model := range keyed {
			if model.Name == "" {
				model.Name = key
			}
			models = append(models, model)
		}
		return nonEmptyCopilotModels(models), len(models) > 0, nil
	}

	return nil, false, nil
}

func nonEmptyCopilotModels(models []CopilotModel) []CopilotModel {
	filtered := models[:0]
	for _, model := range models {
		if model.Name != "" || model.ID != "" {
			filtered = append(filtered, model)
		}
	}
	return filtered
}
