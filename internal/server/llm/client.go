package llm

import (
	"context"
	"fmt"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/prompt"
	"github.com/Kartik-2239/lightcode/internal/server/tools"
	"github.com/openai/openai-go/v3"

	"github.com/joho/godotenv"
)

type Response struct {
	Text             string
	ToolCalls        []ToolCall
	CompleteResponse *openai.ChatCompletion
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type Chat struct {
	Role    string
	Content string
}

func ApiCall(ctx context.Context, input string, chats []Chat) Response {
	godotenv.Load(config.EnvPath())
	var toolCalls []ToolCall
	// ctx := context.Background()
	client := openai.NewClient()

	var messages []openai.ChatCompletionMessageParamUnion
	messages = append(messages, openai.SystemMessage("You are a helpful assistant that can use the following skills to help the user: "+prompt.AvailableSkills()))

	for _, c := range chats {
		if c.Role == "user" {
			messages = append(messages, openai.UserMessage(c.Content))
		} else if c.Role == "assistant" {
			messages = append(messages, openai.AssistantMessage(c.Content))
		}
	}
	messages = append(messages, openai.UserMessage(input))

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Tools:    tools.GetToolsForChat(),
		Model:    "Minimax-M2.5",
	})
	if err != nil {
		fmt.Println("Error", err)
		return Response{}
	}

	for _, item := range resp.Choices[0].Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        item.ID,
			Name:      item.Function.Name,
			Arguments: item.Function.Arguments,
		})
	}

	return Response{
		Text:             resp.Choices[0].Message.Content,
		ToolCalls:        toolCalls,
		CompleteResponse: resp,
	}
}

func ExecuteToolCall(tc ToolCall, workingDirectory string, sessionID string) (string, error) {
	args, err := tools.ParseArgs(tc.Arguments)
	if err != nil {
		return "", err
	}
	return tools.Execute(tc.Name, tools.ToolContext{WorkingDirectory: workingDirectory, SessionID: sessionID}, args)
}
