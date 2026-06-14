package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm/llmModel"
	"github.com/Kartik-2239/lightcode/internal/server/oauth"
	"github.com/Kartik-2239/lightcode/internal/server/prompt"
	"github.com/Kartik-2239/lightcode/internal/server/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared/constant"
)

func ApiCall(ctx context.Context, m config.ResModel, input string, chats []llmModel.Chat, originalMessages []models.Message, mode string, img_bytes [][]byte) (llmModel.Response, error) {
	trimmedMessages := originalMessages[len(originalMessages)-len(chats):]
	var toolCalls []llmModel.ToolCall
	// cur_model, err := config.GetCurrentModel()
	// if err != nil {
	// 	return Response{
	// 		Text:             "Ran into an error while getting the model",
	// 		ToolCalls:        []ToolCall{},
	// 		CompleteResponse: nil,
	// 	}, err
	// }

	var messages []openai.ChatCompletionMessageParamUnion
	if mode == "plan" {
		messages = append(messages, openai.SystemMessage(prompt.Plan_prompt()+prompt.AvailableSkills()))
	}
	if mode == "chat" {
		messages = append(messages, openai.SystemMessage(prompt.SystemPrompt()+" Available skills: "+" "+prompt.AvailableSkills()))
	}
	if mode == "assistant" {
		messages = append(messages, openai.SystemMessage(prompt.Assistant_prompt()+prompt.ExplorePrompt()))
	}

	for _, c := range chats {
		if c.Content != "" {
			switch c.Role {
			case "user":
				messages = append(messages, openai.UserMessage(c.Content))
			case "assistant":
				messages = append(messages, openai.AssistantMessage(c.Content))
			case "tool":
				messages = append(messages, openai.ToolMessage(c.Content, c.ToolCallID))
			}
		}

	}
	parts := []openai.ChatCompletionContentPartUnionParam{}

	if input != "" {
		messages = append(messages, openai.UserMessage(input))
	}
	for _, img := range img_bytes {
		b64 := base64.StdEncoding.EncodeToString(img)
		// fmt.Println("======b64======")
		// fmt.Println(b64)
		// fmt.Println("======b64======")
		parts = append(parts, openai.ImageContentPart(
			openai.ChatCompletionContentPartImageImageURLParam{
				URL:    "data:image/png;base64," + b64,
				Detail: "auto",
			},
		))
	}

	if len(parts) > 0 {
		messages = append(messages, openai.UserMessage(parts))
	}
	// m := cur_model.Model
	var resp *openai.ChatCompletion
	var err error
	if strings.HasPrefix(m.BaseUrl, "http") {
		client := openai.NewClient(option.WithAPIKey(m.ApiKey), option.WithBaseURL(m.BaseUrl))

		resp, err = client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: messages,
			Tools:    tools.GetToolsForChat(),
			Model:    m.Model,
		})
	} else if m.BaseUrl == "copilot" {
		models, err := oauth.MakeModelsRequest()
		if err != nil {
			return llmModel.Response{Text: "Error fetching models from copilot: " + err.Error()}, err
		}
		selectedModel, ok := oauth.ResolveCopilotModel(m.Model, models)
		if !ok {
			return llmModel.Response{Text: "Error: model not found in copilot"}, errors.New("model not found in copilot")
		}
		endpoint := preferredCopilotEndpoint(selectedModel.SupportedEndpoints)
		if endpoint == "/chat/completions" {
			copilotReq := oauth.CopilotChatCompletionRequest{Model: copilotModelID(selectedModel), Stream: false}
			tools := tools.GetToolsForChat()
			copilot_tools := make([]oauth.CopilotChatTool, len(tools))
			for i, tool := range tools {
				copilot_tools[i] = oauth.CopilotChatTool{
					Type: "function",
					Function: oauth.CopilotToolFunction{
						Name:        tool.GetFunction().Name,
						Description: tool.GetFunction().Description.String(),
						Parameters:  tool.GetFunction().Parameters,
					},
				}
			}
			msgs := make([]oauth.CopilotChatMessage, 0, len(chats)+1)
			for _, msg := range chats {
				msgs = append(msgs, oauth.CopilotChatMessage{
					Role:    string(msg.Role),
					Content: msg.Content,
				})
			}
			if input != "" {
				msgs = append(msgs, oauth.CopilotChatMessage{
					Role:    "user",
					Content: input,
				})
			}
			copilotReq.Messages = msgs
			copilotReq.Tools = copilot_tools
			response, err := oauth.MakeCopilotChatCompletionRequest(copilotReq)
			if err != nil {
				if fallbackModel, ok := copilotIntegratorFallbackModel(selectedModel, err); ok {
					copilotReq.Model = fallbackModel
					response, err = oauth.MakeCopilotChatCompletionRequest(copilotReq)
				}
				if err != nil {
					return llmModel.Response{Text: "Internal Error: " + err.Error()}, err
				}
			}
			if len(response.Choices) == 0 {
				return llmModel.Response{Text: "Ran into an error while calling Copilot"}, errors.New("copilot response did not include choices")
			}
			toolcalls := make([]llmModel.ToolCall, len(response.Choices[0].Message.ToolCalls))
			for i, tc := range response.Choices[0].Message.ToolCalls {
				toolcalls[i] = llmModel.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
			var completeResponse *openai.ChatCompletion
			completeResponse = &openai.ChatCompletion{
				ID: response.ID,
				Choices: []openai.ChatCompletionChoice{
					{
						FinishReason: response.Choices[0].FinishReason,
						Index:        int64(response.Choices[0].Index),
						Message: openai.ChatCompletionMessage{
							Content: response.Choices[0].Message.Content,
							Role:    constant.ValueOf[constant.Assistant](),
						},
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     int64(response.Usage.PromptTokens),
					CompletionTokens: int64(response.Usage.CompletionTokens),
					TotalTokens:      int64(response.Usage.TotalTokens),
				},
			}
			return llmModel.Response{
				Text:             response.Choices[0].Message.Content,
				ToolCalls:        toolcalls,
				CompleteResponse: completeResponse,
			}, err

		}
		if endpoint == "/responses" || endpoint == "/v1/responses" {
			copilotReq := oauth.CopilotResponsesRequest{
				Model:  copilotModelID(selectedModel),
				Input:  copilotResponsesInput(chats, input),
				Stream: false,
			}
			response, err := oauth.MakeCopilotResponsesRequest(copilotReq)
			if err != nil {
				return llmModel.Response{Text: "Internal Error: " + err.Error()}, err
			}
			text := copilotResponseText(response)
			completeResponse := &openai.ChatCompletion{
				ID: response.ID,
				Choices: []openai.ChatCompletionChoice{
					{
						FinishReason: "stop",
						Message: openai.ChatCompletionMessage{
							Content: text,
							Role:    constant.ValueOf[constant.Assistant](),
						},
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     int64(response.Usage.InputTokens),
					CompletionTokens: int64(response.Usage.OutputTokens),
					TotalTokens:      int64(response.Usage.TotalTokens),
				},
			}
			return llmModel.Response{
				Text:             text,
				ToolCalls:        []llmModel.ToolCall{},
				CompleteResponse: completeResponse,
			}, nil
		}
		return llmModel.Response{Text: "Error: selected Copilot model does not expose a supported endpoint"}, fmt.Errorf("copilot model %q has unsupported endpoints %v", m.Model, selectedModel.SupportedEndpoints)
	} else {
		resp, err = oauth.MakeOauthRequest(m.BaseUrl, m.Model, m.ReasoningEffort, trimmedMessages, "WRITE CODE DON'T KEEP SAYING HI AGAIN AND AGAIN AFTER USER ASKS YOU TO DO SOMETHING.\n"+" Available skills: "+" "+prompt.AvailableSkills(), tools.GetAllTools())
	}

	if err != nil {
		fmt.Println("Error", err.Error())
		var apierr *openai.Error
		if errors.As(err, &apierr) {
			return llmModel.Response{
				Text:             apierr.Message,
				ToolCalls:        []llmModel.ToolCall{},
				CompleteResponse: nil,
			}, err
		}
		return llmModel.Response{Text: "Internal Error: " + err.Error()}, err

	}
	if len(resp.Choices) == 0 {
		return llmModel.Response{
			Text:             "Ran into an error while calling the LLM",
			ToolCalls:        []llmModel.ToolCall{},
			CompleteResponse: nil,
		}, err
	}

	for _, item := range resp.Choices[0].Message.ToolCalls {
		toolCalls = append(toolCalls, llmModel.ToolCall{
			ID:        item.ID,
			Name:      item.Function.Name,
			Arguments: item.Function.Arguments,
		})
	}
	if config.Debug {
		fmt.Println("===========CACHED==============")
		fmt.Println("cached token", resp.Usage.PromptTokensDetails.CachedTokens)
		fmt.Println("usage: ", resp.Usage)
		fmt.Println("===========CACHED==============")
	}

	return llmModel.Response{
		Text:             resp.Choices[0].Message.Content,
		ToolCalls:        toolCalls,
		CompleteResponse: resp,
	}, nil
}

func preferredCopilotEndpoint(endpoints []string) string {
	for _, endpoint := range endpoints {
		if endpoint == "/chat/completions" {
			return endpoint
		}
	}
	for _, endpoint := range endpoints {
		if endpoint == "/responses" || endpoint == "/v1/responses" {
			return endpoint
		}
	}
	return ""
}

func copilotModelID(model oauth.CopilotModel) string {
	if model.ID != "" {
		return model.ID
	}
	return model.Name
}

func copilotIntegratorFallbackModel(model oauth.CopilotModel, err error) (string, bool) {
	if err == nil || model.ID != "gemini-3.1-pro-preview" {
		return "", false
	}
	message := err.Error()
	if strings.Contains(message, "model_not_available_for_integrator") ||
		strings.Contains(message, "not available for integrator") {
		return "gemini-2.5-pro", true
	}
	return "", false
}

func copilotResponsesInput(chats []llmModel.Chat, input string) []oauth.CopilotResponsesInputItem {
	items := make([]oauth.CopilotResponsesInputItem, 0, len(chats)+1)
	for _, msg := range chats {
		if msg.Content == "" {
			continue
		}
		items = append(items, oauth.CopilotResponsesInputItem{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}
	if input != "" {
		items = append(items, oauth.CopilotResponsesInputItem{
			Role:    "user",
			Content: input,
		})
	}
	return items
}

func copilotResponseText(response oauth.CopilotResponsesResponse) string {
	var sb strings.Builder
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Text == "" {
				continue
			}
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(content.Text)
		}
	}
	return sb.String()
}

func ExecuteToolCall(tc llmModel.ToolCall, workingDirectory string, sessionID string) (tools.ToolResponse, error) {
	args, err := tools.ParseArgs(tc.Arguments)
	if err != nil {
		return tools.ToolResponse{Content: "Error: " + err.Error()}, err
	}
	return tools.Execute(tc.Name, tools.ToolContext{WorkingDirectory: workingDirectory, SessionID: sessionID}, args)
}
