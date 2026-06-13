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
)

func ApiCall(ctx context.Context, m config.ResModel, input string, chats []llmModel.Chat, originalMessages []models.Message, mode string, img_bytes [][]byte, agentsMd string) (llmModel.Response, error) {
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

	// Project-level instructions from AGENTS.md are injected into the system
	// prompt so they apply on every provider path (OpenAI and OAuth) without
	// disturbing the chats<->originalMessages alignment.
	agentsSuffix := ""
	if strings.TrimSpace(agentsMd) != "" {
		agentsSuffix = "\n\n<agents_md>\n" + agentsMd + "\n</agents_md>"
	}

	var messages []openai.ChatCompletionMessageParamUnion
	if mode == "plan" {
		messages = append(messages, openai.SystemMessage(prompt.Plan_prompt()+prompt.AvailableSkills()+agentsSuffix))
	}
	if mode == "chat" {
		messages = append(messages, openai.SystemMessage(prompt.SystemPrompt()+" Available skills: "+" "+prompt.AvailableSkills()+agentsSuffix))
	}
	if mode == "assistant" {
		messages = append(messages, openai.SystemMessage(prompt.Assistant_prompt()+prompt.ExplorePrompt()+agentsSuffix))
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
		var endpoint string
		for _, model := range models {
			if model.Name == m.Model {
				endpoint = model.SupportedEndpoints[0]
			}
		}
		if endpoint == "" {
			return llmModel.Response{Text: "Error: model not found in copilot"}, errors.New("model not found in copilot")
		}
		if endpoint == "/chat/completions" {
			copilotReq := oauth.CopilotChatCompletionRequest{Model: m.Model, Stream: false}
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
			msgs := make([]oauth.CopilotChatMessage, len(chats))
			for i, msg := range chats {
				msgs[i] = oauth.CopilotChatMessage{
					Role:    string(msg.Role),
					Content: msg.Content,
				}
			}
			copilotReq.Messages = msgs
			copilotReq.Tools = copilot_tools
			response, err := oauth.MakeCopilotChatCompletionRequest(copilotReq)
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
						FinishReason: resp.Choices[0].FinishReason,
						Index:        resp.Choices[0].Index,
						Logprobs:     resp.Choices[0].Logprobs,
						Message:      resp.Choices[0].Message,
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
		if endpoint == "/v1/responses" {
			// copilotMessages := oauth.CopilotResponsesRequest{}
		}
	} else {
		resp, err = oauth.MakeOauthRequest(m.BaseUrl, m.Model, trimmedMessages, "WRITE CODE DON'T KEEP SAYING HI AGAIN AND AGAIN AFTER USER ASKS YOU TO DO SOMETHING.\n"+" Available skills: "+" "+prompt.AvailableSkills()+agentsSuffix, tools.GetAllTools())
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

func ExecuteToolCall(tc llmModel.ToolCall, workingDirectory string, sessionID string) (tools.ToolResponse, error) {
	args, err := tools.ParseArgs(tc.Arguments)
	if err != nil {
		return tools.ToolResponse{Content: "Error: " + err.Error()}, err
	}
	return tools.Execute(tc.Name, tools.ToolContext{WorkingDirectory: workingDirectory, SessionID: sessionID}, args)
}
