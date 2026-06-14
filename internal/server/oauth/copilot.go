package oauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Kartik-2239/lightcode/internal/server/config"
)

const copilotBaseURL = "https://api.githubcopilot.com"
const githubDeviceCodeURL = "https://github.com/login/device/code"
const githubAccessTokenURL = "https://github.com/login/oauth/access_token"
const githubCopilotClientID = "178c6fc778ccc68e1d6a"
const githubCopilotScope = "read:org"

type AuthResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type CopilotResponsesRequest struct {
	Model      string                      `json:"model"`
	Input      []CopilotResponsesInputItem `json:"input"`
	Tools      []CopilotResponsesTool      `json:"tools,omitempty"`
	ToolChoice any                         `json:"tool_choice,omitempty"`
	Reasoning  *CopilotReasoning           `json:"reasoning,omitempty"`
	Include    []string                    `json:"include,omitempty"`
	Text       *CopilotTextOptions         `json:"text,omitempty"`
	Stream     bool                        `json:"stream,omitempty"`
}

type CopilotResponsesInputItem struct {
	Role      string `json:"role,omitempty"`
	Content   any    `json:"content,omitempty"`
	Type      string `json:"type,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Output    string `json:"output,omitempty"`
}

type CopilotResponsesContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type CopilotResponsesTool struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type CopilotReasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type CopilotTextOptions struct {
	Verbosity string `json:"verbosity,omitempty"`
}

type CopilotResponsesResponse struct {
	ID     string                       `json:"id,omitempty"`
	Output []CopilotResponsesOutputItem `json:"output"`
	Usage  CopilotResponsesUsage        `json:"usage"`
}

type CopilotResponsesOutputItem struct {
	Type      string                          `json:"type"`
	Role      string                          `json:"role,omitempty"`
	CallID    string                          `json:"call_id,omitempty"`
	Name      string                          `json:"name,omitempty"`
	Arguments string                          `json:"arguments,omitempty"`
	Content   []CopilotResponsesOutputContent `json:"content,omitempty"`
}

type CopilotResponsesOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type CopilotResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type CopilotChatCompletionRequest struct {
	Model       string               `json:"model"`
	Messages    []CopilotChatMessage `json:"messages"`
	Tools       []CopilotChatTool    `json:"tools,omitempty"`
	ToolChoice  any                  `json:"tool_choice,omitempty"`
	Temperature *float64             `json:"temperature,omitempty"`
	Stream      bool                 `json:"stream,omitempty"`
}

type CopilotChatMessage struct {
	Role       string                `json:"role"`
	Content    any                   `json:"content,omitempty"`
	ToolCalls  []CopilotChatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string                `json:"tool_call_id,omitempty"`
}

type CopilotChatTool struct {
	Type     string              `json:"type"`
	Function CopilotToolFunction `json:"function"`
}

type CopilotToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type CopilotChatToolCall struct {
	ID       string                  `json:"id"`
	Type     string                  `json:"type"`
	Function CopilotChatToolFunction `json:"function"`
}

type CopilotChatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

type CopilotChatCompletionResponse struct {
	ID      string              `json:"id"`
	Choices []CopilotChatChoice `json:"choices"`
	Usage   CopilotChatUsage    `json:"usage"`
}

type CopilotChatChoice struct {
	Index        int                        `json:"index"`
	Message      CopilotChatResponseMessage `json:"message"`
	FinishReason string                     `json:"finish_reason"`
}

type CopilotChatResponseMessage struct {
	Role            string                `json:"role"`
	Content         string                `json:"content"`
	ReasoningOpaque string                `json:"reasoning_opaque,omitempty"`
	ToolCalls       []CopilotChatToolCall `json:"tool_calls,omitempty"`
}

type CopilotChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func StartAuthFlow() (DeviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", githubCopilotClientID)
	form.Set("scope", githubCopilotScope)
	req, err := http.NewRequest("POST", githubDeviceCodeURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "lightcode")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DeviceCodeResponse{}, fmt.Errorf("github device code request failed: %s: %s", resp.Status, string(respBody))
	}

	var deviceCodeResp DeviceCodeResponse
	err = json.Unmarshal(respBody, &deviceCodeResp)
	if err != nil {
		return DeviceCodeResponse{}, err
	}

	return deviceCodeResp, nil
}

func PollForAccessToken(deviceCode string) (AuthResponse, error) {
	form := url.Values{}
	form.Set("client_id", githubCopilotClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	req, err := http.NewRequest("POST", githubAccessTokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return AuthResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "lightcode")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AuthResponse{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return AuthResponse{}, fmt.Errorf("github access token request failed: %s: %s", resp.Status, string(respBody))
	}

	var tokenResp AuthResponse
	err = json.Unmarshal(respBody, &tokenResp)
	if err != nil {
		return AuthResponse{}, err
	}

	return tokenResp, nil
}

func WaitForAccessToken(device DeviceCodeResponse) (AuthResponse, error) {
	interval := device.Interval
	if interval < 5 {
		interval = 5
	}
	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)
	time.Sleep(time.Duration(interval) * time.Second)

	for time.Now().Before(deadline) {
		resp, err := PollForAccessToken(device.DeviceCode)
		if err != nil {
			return AuthResponse{}, err
		}
		if resp.AccessToken != "" {
			return resp, nil
		}
		switch resp.Error {
		case "authorization_pending":
		case "slow_down":
			interval += 5
		case "access_denied":
			return AuthResponse{}, fmt.Errorf("github authorization was denied")
		case "expired_token":
			return AuthResponse{}, fmt.Errorf("github device code expired")
		default:
			if resp.ErrorDescription != "" {
				return AuthResponse{}, errors.New(resp.ErrorDescription)
			}
			if resp.Error != "" {
				return AuthResponse{}, errors.New(resp.Error)
			}
			return AuthResponse{}, fmt.Errorf("github oauth response did not include an access token")
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
	return AuthResponse{}, fmt.Errorf("github device code expired")
}

func SaveAccessToken(token string) error {
	val := config.AuthVal{
		Type:      "oauth",
		Access:    token,
		Refresh:   "",
		Expires:   0,
		AccountId: "",
		Models:    []string{},
	}
	if err := config.UpdateAuthVal(config.CopilotAuthProvider, val); err != nil {
		return err
	}
	return RefreshSavedCopilotModels()
}

func RefreshSavedCopilotModels() error {
	val, err := config.GetAuthVal(config.CopilotAuthProvider)
	if err != nil {
		return err
	}
	if val.Access == "" {
		return fmt.Errorf("no access token found for github copilot")
	}
	models, err := MakeModelsRequest()
	if err != nil {
		return err
	}
	val.Models = DefaultCopilotPickerModelNames()
	validModels := make([]string, 0, len(val.Models))
	for _, model := range val.Models {
		if _, ok := ResolveCopilotModel(model, models); ok {
			validModels = append(validModels, model)
		}
	}
	val.Models = validModels
	if err := config.UpdateAuthVal(config.CopilotAuthProvider, val); err != nil {
		return err
	}
	return config.ReconcileAuthProviderModels(config.CopilotAuthProvider, val.Models)
}

func MakeModelsRequest() ([]CopilotModel, error) {
	var response copilotModelsResponse
	val, err := getCopilotAuthVal()
	if err != nil {
		return nil, err
	}
	if val.Access == "" {
		return nil, fmt.Errorf("no access token found for github copilot")
	}
	err = makeCopilotRequest("/models", nil, &response)
	return []CopilotModel(response), err
}

func MakeCopilotResponsesRequest(request CopilotResponsesRequest) (CopilotResponsesResponse, error) {
	var response CopilotResponsesResponse
	err := makeCopilotRequest("/responses", request, &response)
	return response, err
}

func MakeCopilotChatCompletionRequest(request CopilotChatCompletionRequest) (CopilotChatCompletionResponse, error) {
	var response CopilotChatCompletionResponse
	err := makeCopilotRequest("/chat/completions", request, &response)
	return response, err
}

func makeCopilotRequest(endpoint string, request any, response any) error {
	val, err := getCopilotAuthVal()
	if err != nil {
		return err
	}
	accessToken := val.Access
	if accessToken == "" {
		return fmt.Errorf("no access token found for github copilot")
	}

	method := http.MethodPost
	var body io.Reader
	if request == nil {
		method = http.MethodGet
	} else {
		bodyBytes, err := json.Marshal(request)
		if err != nil {
			return err
		}
		body = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(method, copilotBaseURL+endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Openai-Intent", "conversation-edits")
	req.Header.Set("User-Agent", "opencode/0.1.0")
	req.Header.Set("x-initiator", "user")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("copilot request failed: %s: %s", resp.Status, string(respBody))
	}

	return json.Unmarshal(respBody, response)
}

func getCopilotAuthVal() (config.AuthVal, error) {
	return config.GetAuthVal(config.CopilotAuthProvider)
}
