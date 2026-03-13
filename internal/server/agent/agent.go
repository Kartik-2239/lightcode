package agent

import (
	"errors"
	"fmt"

	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm"
	"gorm.io/gorm"
)

const MaxIterations = 10

type Agent struct{}

func New() *Agent {
	return &Agent{}
}

func (a *Agent) Run(prompt string, session_id string) string {
	// currentPrompt := prompt
	database, err := db.Connect()
	if err != nil {
		return "Ran into error: " + err.Error()
	}
	var session models.Session
	result := database.Where("id = ?", session_id).First(&session)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newSession := models.Session{
				ID:        session_id,
				Title:     prompt,
				Directory: ".",
			}
			database.Create(&newSession)
		}
	}

	currentPrompt := prompt

	userMessage := models.Message{
		SessionID: session_id,
		ID:        fmt.Sprintf("%s-user-%d", session_id, 0),
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "user", Content: currentPrompt}),
	}
	database.Create(&userMessage)

	for i := 0; i < MaxIterations; i++ {
		fmt.Println("Iteration:", i)
		var messages []models.Message
		database.Where("session_id = ?", session_id).Find(&messages)
		var chats []llm.Chat
		for _, message := range messages {
			d := models.DecodeMessageData(message.Data)
			chats = append(chats, llm.Chat{Role: d.Role, Content: d.Content})
		}
		fmt.Println("Calling API...")
		resp := llm.ApiCall(currentPrompt, chats)

		if len(resp.ToolCalls) == 0 {
			newMessage := models.Message{
				SessionID: session_id,
				ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
				Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: resp.Text}),
			}
			fmt.Println("Creating message:", newMessage)
			if err := database.Create(&newMessage).Error; err != nil {
				fmt.Println("Error creating message:", err)
			} else {
				fmt.Println("Message created successfully!")
			}
			return resp.Text
		}

		var storedToolCalls []models.StoredToolCall
		for _, tc := range resp.ToolCalls {
			storedToolCalls = append(storedToolCalls, models.StoredToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
		}
		assistantMsg := models.Message{
			SessionID: session_id,
			ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
			Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: resp.Text, ToolCalls: storedToolCalls, Usage: &models.StoredUsage{PromptTokens: resp.CompleteResponse.Usage.PromptTokens, CompletionTokens: resp.CompleteResponse.Usage.CompletionTokens, TotalTokens: resp.CompleteResponse.Usage.TotalTokens}}),
		}
		fmt.Println("Creating message:", assistantMsg)
		if err := database.Create(&assistantMsg).Error; err != nil {
			fmt.Println("Error creating message:", err)
		} else {
			fmt.Println("Message created successfully!")
		}

		for _, tc := range resp.ToolCalls {
			result, err := llm.ExecuteToolCall(tc)
			if err != nil {
				return fmt.Sprintf("Tool '%s' failed: %v", tc.Name, err)
			}
			fmt.Println(result)
			fmt.Println("========================")
			currentPrompt = "the result of the tool_call" + tc.Name + "is" + result
		}
	}

	return "Max iterations reached"
}
