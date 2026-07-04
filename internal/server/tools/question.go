package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

type QuestionItem struct {
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
}

func init() {
	Prompt := `Ask one or more questions to the user
Rules to follow :-
- Use this tool for asking questions to the user about implementation of ideas when the prompt given by the user isn't enough
- Use this tool to get more information about how the project should be like, or what decisions to take.`
	Register("question", ToolDef{
		Name:        "question",
		Description: Prompt,
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"question": map[string]any{
								"type":        "string",
								"description": "The question to ask the user",
							},
							"options": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type":        "string",
									"description": "The options to ask the user",
								},
							},
						},
						"required": []string{"question"},
					},
				},
			},
			"required": []string{"question"},
		},
	}, Question)
}

func Question(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	rawQuestions, ok := args["question"].([]any)
	if !ok {
		return ToolResponse{Content: "Error: question must be an array of question objects", Printable: "Error: question must be an array of question objects"}, nil
	}

	questions := make([]QuestionItem, 0, len(rawQuestions))
	for _, item := range rawQuestions {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		q := QuestionItem{}
		q.Question, _ = obj["question"].(string)
		if rawOpts, ok := obj["options"].([]any); ok {
			for _, o := range rawOpts {
				if s, ok := o.(string); ok {
					q.Options = append(q.Options, s)
				}
			}
		}
		if q.Question == "" {
			continue
		}
		questions = append(questions, q)
	}

	if len(questions) == 0 {
		return ToolResponse{Content: "Error: no valid questions provided", Printable: "Error: no valid questions provided"}, nil
	}

	out, err := json.Marshal(questions)
	if err != nil {
		msg := fmt.Sprintf("Error: failed to encode questions: %v", err)
		return ToolResponse{Content: msg, Printable: msg}, nil
	}

	var sb strings.Builder
	for i, q := range questions {
		sb.WriteString(fmt.Sprintf("Q%d: %s", i+1, q.Question))
		if len(q.Options) > 0 {
			sb.WriteString("\nOptions: " + strings.Join(q.Options, ", "))
		}
		sb.WriteString("\n")
	}
	_ = out
	content := sb.String()
	return ToolResponse{Content: content, Printable: content}, nil
}
