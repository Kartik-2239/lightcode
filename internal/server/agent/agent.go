package agent

import (
	"context"
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

func (a *Agent) Run(ctx context.Context, prompt string, session_id string) <-chan models.StoredMessageData {
	ch := make(chan models.StoredMessageData)
	// currentPrompt := prompt
	database, err := db.Connect()
	if err != nil {
		ch <- models.StoredMessageData{Role: "error", Content: "Ran into error: " + err.Error()}
		close(ch)
		return ch
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

	go func() {
		defer close(ch)
		for i := 0; i < MaxIterations; i++ {
			fmt.Println("Iteration:", i)
			var messages []models.Message
			database.Where("session_id = ?", session_id).Find(&messages)
			var chats []llm.Chat
			for _, message := range messages {
				d := models.DecodeMessageData(message.Data)
				chats = append(chats, llm.Chat{Role: d.Role, Content: d.Content})
			}
			// fmt.Println("Calling API...")
			resp := llm.ApiCall(currentPrompt, chats)

			if len(resp.ToolCalls) == 0 {
				assistantMessage := models.StoredMessageData{Role: "assistant", Content: resp.Text, Usage: &models.StoredUsage{PromptTokens: resp.CompleteResponse.Usage.PromptTokens, CompletionTokens: resp.CompleteResponse.Usage.CompletionTokens, TotalTokens: resp.CompleteResponse.Usage.TotalTokens}}
				newMessage := models.Message{
					SessionID: session_id,
					ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
					Data:      models.EncodeMessageData(assistantMessage),
				}
				// fmt.Println("Creating message:", newMessage)
				if err := database.Create(&newMessage).Error; err != nil {
					fmt.Println("Error creating message:", err)
				} else {
					fmt.Println("Message created successfully!")
				}
				ch <- assistantMessage
				return
			}

			var storedToolCalls []models.StoredToolCall
			for _, tc := range resp.ToolCalls {
				storedToolCalls = append(storedToolCalls, models.StoredToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
			}
			assistantMessage := models.StoredMessageData{Role: "assistant", Content: resp.Text, ToolCalls: storedToolCalls, Usage: &models.StoredUsage{PromptTokens: resp.CompleteResponse.Usage.PromptTokens, CompletionTokens: resp.CompleteResponse.Usage.CompletionTokens, TotalTokens: resp.CompleteResponse.Usage.TotalTokens}}
			assistantMsg := models.Message{
				SessionID: session_id,
				ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
				Data:      models.EncodeMessageData(assistantMessage),
			}
			ch <- assistantMessage
			fmt.Println("Creating message:", assistantMsg)
			if err := database.Create(&assistantMsg).Error; err != nil {
				fmt.Println("Error creating message:", err)
			} else {
				// Return the whole message object in the channel which can then be used to send the message to the client
				fmt.Println("Message created successfully!")
			}

			for _, tc := range resp.ToolCalls {
				result, err := llm.ExecuteToolCall(tc)
				if err != nil {
					ch <- models.StoredMessageData{Role: "error", Content: fmt.Sprintf("Tool '%s' failed: %v", tc.Name, err)}
					return
				}
				ch <- models.StoredMessageData{Role: "assistant", Content: result}
				currentPrompt = "the result of the tool_call" + tc.Name + "is" + result
			}
		}
	}()

	// ch <- "Max iterations reached"
	// close(ch)
	return ch
}
