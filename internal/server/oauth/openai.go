package oauth

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type responseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responseOutputItem struct {
	Type      string            `json:"type"`
	Role      string            `json:"role"`
	CallID    string            `json:"call_id"`
	Name      string            `json:"name"`
	Arguments string            `json:"arguments"`
	Content   []responseContent `json:"content"`
}

type codexRequest struct {
	Model        string           `json:"model"`
	Instructions string           `json:"instructions"`
	Input        []codexInputItem `json:"input"`
	Tools        []codexTool      `json:"tools,omitempty"`
	ToolChoice   string           `json:"tool_choice,omitempty"`
	Reasoning    *codexReasoning  `json:"reasoning,omitempty"`
	Store        bool             `json:"store"`
	Stream       bool             `json:"stream"`
}

type codexReasoning struct {
	Effort string `json:"effort"`
}

type codexTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      bool           `json:"strict,omitempty"`
}

type codexInputItem struct {
	Role      string  `json:"role,omitempty"`
	Content   any     `json:"content,omitempty"`
	Type      string  `json:"type,omitempty"`
	CallID    string  `json:"call_id,omitempty"`
	Name      string  `json:"name,omitempty"`
	Arguments string  `json:"arguments,omitempty"`
	Output    *string `json:"output,omitempty"`
}

type codexContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Usage struct {
	InputTokens        int64              `json:"input_tokens"`
	OutputTokens       int64              `json:"output_tokens"`
	TotalTokens        int64              `json:"total_tokens"`
	InputTokenDetails  InputTokenDetails  `json:"input_tokens_details"`
	OutputTokenDetails OutputTokenDetails `json:"output_tokens_details"`
}

type InputTokenDetails struct {
	CachedTokens int64 `json:"cached_tokens"`
}

type OutputTokenDetails struct {
	ReasoningTokens int64 `json:"reasoning_tokens"`
}

func MakeOauthRequest(provider string, model string, reasoningEffort string, messages []models.Message, system string, tools []responses.ToolUnionParam) (*openai.ChatCompletion, error) {
	authVal, err := config.GetAuthVal(provider)
	if err != nil {
		return nil, err
	}
	if authVal.Expires == 0 || authVal.Expires < time.Now().Unix() {
		// fmt.Printf("Tokens for provider %s have expired or are missing, refreshing...\n", provider)
		var fetchErr error
		authVal, fetchErr = FetchTokens(authVal)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to refresh tokens: %w", fetchErr)
		}
		if err := config.UpdateAuthVal(provider, authVal); err != nil {
			return nil, err
		}
	}

	inputItems := []codexInputItem{{
		Role:    "developer",
		Content: system,
	}}

	for _, m := range messages {
		data := models.DecodeMessageData(m.Data)
		switch data.Role {
		case "user":
			inputItems = append(inputItems, codexInputItem{
				Role: "user",
				Content: []codexContentPart{{
					Type: "input_text",
					Text: data.Content,
				}},
			})
		case "assistant":
			inputItems = append(inputItems, codexInputItem{
				Role:    "assistant",
				Content: data.Content,
			})
		case "tool_call":
			if len(data.ToolCalls) == 0 {
				continue
			}
			toolCall := data.ToolCalls[0]
			inputItems = append(inputItems, codexInputItem{
				Type:      "function_call",
				CallID:    toolCall.ID,
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
			})
			inputItems = append(inputItems, codexInputItem{
				Type:   "function_call_output",
				CallID: toolCall.ID,
				Output: &data.Content,
			})

		}
	}
	var reqTools []codexTool
	for _, tool := range tools {
		if tool.OfFunction == nil {
			continue
		}
		reqTools = append(reqTools, codexTool{
			Type:        "function",
			Name:        tool.OfFunction.Name,
			Description: tool.OfFunction.Description.Or(""),
			Parameters:  tool.OfFunction.Parameters,
		})
	}

	body := codexRequest{
		Model:        model,
		Instructions: system,
		Input:        inputItems,
		Tools:        reqTools,
		ToolChoice:   "auto",
		Store:        false,
		Stream:       true,
	}
	if reasoningEffort != "" {
		body.Reasoning = &codexReasoning{Effort: reasoningEffort}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	send := func(authVal config.AuthVal) (int, string, []byte, error) {
		req, err := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", bytes.NewReader(bodyBytes))
		if err != nil {
			return 0, "", nil, err
		}
		req.Header.Set("Authorization", "Bearer "+authVal.Access)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("ChatGPT-Account-Id", authVal.AccountId)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, "", nil, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, "", nil, err
		}
		return resp.StatusCode, resp.Status, respBody, nil
	}

	statusCode, status, respBody, err := send(authVal)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized {
		refreshedAuth, refreshErr := refreshAuthAfterUnauthorized(provider, authVal)
		if refreshErr != nil {
			return nil, refreshErr
		}
		statusCode, status, respBody, err = send(refreshedAuth)
		if err != nil {
			return nil, err
		}
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("oauth request failed: %s: %s", status, string(respBody))
	}
	if looksLikeStreamingResponse(respBody) {
		return parseStreamingResponse(respBody)
	}

	return parseJSONResponse(respBody)
}

func refreshAuthAfterUnauthorized(provider string, authVal config.AuthVal) (config.AuthVal, error) {
	if provider == config.CodexAuthProvider {
		if err := config.ImportCodexAuth(); err == nil {
			importedAuth, err := config.GetAuthVal(provider)
			if err == nil && importedAuth.Access != "" && importedAuth.Access != authVal.Access {
				return importedAuth, nil
			}
		}
	}

	refreshedAuth, err := FetchTokens(authVal)
	if err != nil {
		return config.AuthVal{}, fmt.Errorf("oauth token was rejected and refresh failed; run codex login again: %w", err)
	}
	if err := config.UpdateAuthVal(provider, refreshedAuth); err != nil {
		return config.AuthVal{}, err
	}
	return refreshedAuth, nil
}

func FetchTokens(authVal config.AuthVal) (config.AuthVal, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", authVal.Refresh)
	data.Set("client_id", "app_EMoamEEZ73f0CkXaXp7hrann")
	req, err := http.NewRequest(
		"POST",
		"https://auth.openai.com/oauth/token",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return config.AuthVal{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return config.AuthVal{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return config.AuthVal{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return config.AuthVal{}, fmt.Errorf("token refresh failed: %s: %s", resp.Status, string(respBody))
	}
	var tokenResp struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return config.AuthVal{}, err
	}
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = authVal.Refresh
	}
	expires := authVal.Expires
	if tokenResp.ExpiresIn > 0 {
		expires = time.Now().Unix() + tokenResp.ExpiresIn
	}

	return config.AuthVal{
		Type:      authVal.Type,
		Access:    tokenResp.AccessToken,
		Refresh:   tokenResp.RefreshToken,
		Expires:   expires,
		AccountId: authVal.AccountId,
		Models:    authVal.Models,
	}, nil
}

func looksLikeStreamingResponse(respBody []byte) bool {
	trimmed := bytes.TrimSpace(respBody)
	return bytes.HasPrefix(trimmed, []byte("event:")) || bytes.HasPrefix(trimmed, []byte("data:"))
}

func parseStreamingResponse(respBody []byte) (*openai.ChatCompletion, error) {
	result := &openai.ChatCompletion{}
	ensureAssistantChoice(result)
	scanner := bufio.NewScanner(bytes.NewReader(respBody))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			eventType = ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		if err := applyStreamEvent(result, eventType, payload); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func applyStreamEvent(result *openai.ChatCompletion, eventType string, payload string) error {
	ensureAssistantChoice(result)

	switch eventType {
	case "response.output_text.delta":
		var event struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return err
		}
		result.Choices[0].Message.Content += event.Delta
	case "response.output_item.done":
		var event struct {
			Item responseOutputItem `json:"item"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return err
		}
		appendOutputItem(result, event.Item)
	case "response.completed":
		var data struct {
			Response struct {
				Usage Usage `json:"usage"`
			} `json:"response"`
		}
		if err := json.Unmarshal([]byte(payload), &data); err != nil {
			return err
		}
		// fmt.Println("=========================USAGE=========================")
		// fmt.Println(data.Response.Usage)
		// fmt.Println("=========================USAGE END=========================")
		result.Usage = openai.CompletionUsage{
			CompletionTokens: data.Response.Usage.OutputTokens,
			PromptTokens:     data.Response.Usage.InputTokens,
			TotalTokens:      data.Response.Usage.TotalTokens,
			CompletionTokensDetails: openai.CompletionUsageCompletionTokensDetails{
				ReasoningTokens: data.Response.Usage.OutputTokenDetails.ReasoningTokens,
			},
			PromptTokensDetails: openai.CompletionUsagePromptTokensDetails{
				CachedTokens: data.Response.Usage.InputTokenDetails.CachedTokens,
			},
		}
	}

	return nil
}

func parseJSONResponse(respBody []byte) (*openai.ChatCompletion, error) {
	if looksLikeStreamingResponse(respBody) {
		return parseStreamingResponse(respBody)
	}

	var apiResp struct {
		Output []responseOutputItem `json:"output"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	result := &openai.ChatCompletion{}
	ensureAssistantChoice(result)
	for _, item := range apiResp.Output {
		appendOutputItem(result, item)
	}

	return result, nil
}

func appendOutputItem(result *openai.ChatCompletion, item responseOutputItem) {
	ensureAssistantChoice(result)

	switch item.Type {
	case "message":
		if result.Choices[0].Message.Content == "" {
			for _, content := range item.Content {
				if content.Type == "output_text" {
					result.Choices[0].Message.Content += content.Text
				}
			}
		}
	case "function_call":
		result.Choices[0].Message.ToolCalls = append(result.Choices[0].Message.ToolCalls, openai.ChatCompletionMessageToolCallUnion{
			ID:   item.CallID,
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
		})
	}
}

func ensureAssistantChoice(result *openai.ChatCompletion) {
	if len(result.Choices) > 0 {
		return
	}

	result.Choices = []openai.ChatCompletionChoice{{
		Index: 0,
		Message: openai.ChatCompletionMessage{
			Content:   "",
			ToolCalls: []openai.ChatCompletionMessageToolCallUnion{},
		},
	}}
}
