package llm

import (
	"context"
	"fmt"

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

func ApiCall(prompt string, chats []Chat) Response {
	godotenv.Load()
	var toolCalls []ToolCall
	ctx := context.Background()
	client := openai.NewClient()

	var messages []openai.ChatCompletionMessageParamUnion

	for _, c := range chats {
		if c.Role == "user" {
			messages = append(messages, openai.UserMessage(c.Content))
		} else if c.Role == "assistant" {
			messages = append(messages, openai.AssistantMessage(c.Content))
		}
	}
	messages = append(messages, openai.UserMessage(prompt))

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Tools:    tools.GetToolsForChat(),
		Model:    "Minimax-M2.5",
	})
	if err != nil {
		fmt.Println("Error", err)
	}
	// fmt.Println("===================================")
	// fmt.Println("Response", resp)

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

func ExecuteToolCall(tc ToolCall) (string, error) {
	args, err := tools.ParseArgs(tc.Arguments)
	if err != nil {
		return "", err
	}
	return tools.Execute(tc.Name, args)
}
