package main

import (
	"fmt"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	oauth "github.com/Kartik-2239/lightcode/internal/server/oauth"
)

const model = "gpt-5.5"

func main() {
	authVal, err := config.GetAuthVal(config.CopilotAuthProvider)
	if err != nil {
		panic(err)
	}
	token := authVal.Refresh
	if token == "" {
		token = authVal.Access
	}
	if token == "" {
		panic("missing github copilot token")
	}

	resp, err := oauth.MakeCopilotResponsesRequest(oauth.CopilotResponsesRequest{
		Model: model,
		Input: []oauth.CopilotResponsesInputItem{
			{Role: "system", Content: "You are a concise assistant."},
			{
				Role: "user",
				Content: []oauth.CopilotResponsesContentPart{{
					Type: "input_text",
					Text: "Say hello from GitHub Copilot in one sentence.",
				}},
			},
		},
		Text: &oauth.CopilotTextOptions{Verbosity: "low"},
	})
	if err != nil {
		panic(err)
	}
	for _, output := range resp.Output {
		for _, content := range output.Content {
			if content.Text != "" {
				fmt.Println(content.Text)
			}
		}
	}
}
