package agent

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm"
	"github.com/openai/openai-go/v3"
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

	userMsgData, _ := json.Marshal(map[string]interface{}{
		"id":      fmt.Sprintf("%s-user-%d", session_id, 0),
		"object":  "chat.completion",
		"created": 0,
		"model":   "MiniMax-M2.5",
		"choices": []map[string]interface{}{
			{
				"finish_reason": "stop",
				"index":         0,
				"message": map[string]interface{}{
					"role":          "user",
					"name":          "",
					"audio_content": "",
					"content": map[string]interface{}{
						"thinking": "",
						"response": currentPrompt,
					},
				},
			},
		},
	})
	userMessage := models.Message{
		SessionID: session_id,
		ID:        fmt.Sprintf("%s-user-%d", session_id, 0),
		Data:      string(userMsgData),
	}
	database.Create(&userMessage)

	for i := 0; i < MaxIterations; i++ {
		fmt.Println("Iteration:", i)
		var messages []models.Message
		database.Where("session_id = ?", session_id).Find(&messages)
		var chats []llm.Chat
		for _, message := range messages {
			var m openai.ChatCompletion
			json.Unmarshal([]byte(message.Data), &m)
			chats = append(chats, llm.Chat{Role: string(m.Choices[0].Message.Role), Content: m.Choices[0].Message.Content})
		}
		fmt.Println("Calling API...")
		resp := llm.ApiCall(currentPrompt, chats)
		fmt.Println("API Response:", resp.Text)
		fmt.Println("ToolCalls:", len(resp.ToolCalls))

		if len(resp.ToolCalls) == 0 {
			data, _ := json.Marshal(resp.CompleteResponse)
			newMessage := models.Message{
				SessionID: session_id,
				ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
				Data:      string(data),
			}
			fmt.Println("Creating message:", newMessage)
			if err := database.Create(&newMessage).Error; err != nil {
				fmt.Println("Error creating message:", err)
			} else {
				fmt.Println("Message created successfully!")
			}
			return resp.Text
		}

		for _, tc := range resp.ToolCalls {
			result, err := llm.ExecuteToolCall(tc)
			if err != nil {
				return fmt.Sprintf("Tool '%s' failed: %v", tc.Name, err)
			}

			data, _ := json.Marshal(resp.CompleteResponse)
			newMessage := models.Message{
				SessionID: session_id,
				ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
				Data:      string(data),
			}
			fmt.Println("Creating message:", newMessage)
			if err := database.Create(&newMessage).Error; err != nil {
				fmt.Println("Error creating message:", err)
			} else {
				fmt.Println("Message created successfully!")
			}

			fmt.Println(result)
			fmt.Println("========================")
			currentPrompt = "the result of the tool_call" + tc.Name + "is" + result
		}
	}

	return "Max iterations reached"
}
