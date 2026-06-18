package models

import "encoding/json"

type StoredMessageData struct {
	Role        string           `json:"role"`
	Content     string           `json:"content,omitempty"`
	CodeChanges []string         `json:"code_changes,omitempty"`
	Usage       *StoredUsage     `json:"usage,omitempty"`
	ToolCalls   []StoredToolCall `json:"tool_calls,omitempty"`
	// Mentions holds workspace-relative paths the user explicitly attached via
	// @file mentions in the TUI picker. They are expanded into synthetic
	// read_file/list_dir tool-output messages when the chat history is assembled.
	Mentions []string `json:"mentions,omitempty"`
}

type StoredUsage struct {
	PromptTokens     int64 `json:"prompt_tokens,omitempty"`
	CompletionTokens int64 `json:"completion_tokens,omitempty"`
	TotalTokens      int64 `json:"total_tokens,omitempty"`
}

type StoredToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

func EncodeMessageData(d StoredMessageData) string {
	b, _ := json.Marshal(d)
	return string(b)
}

func DecodeMessageData(s string) StoredMessageData {
	var d StoredMessageData
	json.Unmarshal([]byte(s), &d)
	return d
}
