package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	go func() {
		defer close(ch)
		var prior []models.Message
		database.Where("session_id = ?", session_id).Find(&prior)
		userTurn := models.Message{
			SessionID: session_id,
			ID:        fmt.Sprintf("%s-%d", session_id, len(prior)),
			Data:      models.EncodeMessageData(models.StoredMessageData{Role: "user", Content: prompt}),
		}
		if err := database.Create(&userTurn).Error; err != nil {
			fmt.Println("Error saving user message:", err)
		}

		for i := 0; i < MaxIterations; i++ {

			select {
			case <-ctx.Done():
				return
			default:
			}
			fmt.Println("Iteration:", i)
			var messages []models.Message
			database.Where("session_id = ?", session_id).Find(&messages)
			chats := make([]llm.Chat, 0, len(messages))
			for _, message := range messages {
				d := models.DecodeMessageData(message.Data)
				switch d.Role {
				case "tool_call":
					name, id := "tool", ""
					if len(d.ToolCalls) > 0 {
						name = d.ToolCalls[0].Name
						id = d.ToolCalls[0].ID
					}
					chats = append(chats, llm.Chat{
						Role:    "user",
						Content: fmt.Sprintf("Tool %q (call_id=%s) output:\n%s", name, id, d.Content),
					})
				case "assistant":
					content := d.Content
					if content == "" && len(d.ToolCalls) > 0 {
						names := make([]string, len(d.ToolCalls))
						for j, tc := range d.ToolCalls {
							names[j] = tc.Name
						}
						content = "(Calling tools: " + strings.Join(names, ", ") + ")"
					}
					chats = append(chats, llm.Chat{Role: "assistant", Content: content})
				default:
					chats = append(chats, llm.Chat{Role: d.Role, Content: d.Content})
				}
			}
			resp := llm.ApiCall(ctx, "", chats)
			select {
			case <-ctx.Done():
				return
			default:
			}
			fmt.Println("================================================")
			fmt.Println("Tool calls:", resp.ToolCalls)
			fmt.Println("Number of tool calls:", len(resp.ToolCalls))
			fmt.Println("================================================")
			if len(resp.ToolCalls) == 0 {
				select {
				case <-ctx.Done():
					return
				default:
				}
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
				fmt.Println("Message created successfully!")
			}
			for _, tc := range resp.ToolCalls {
				fmt.Println("Executing tool call:", tc.Name)
				result, err := llm.ExecuteToolCall(tc)
				if err != nil {
					fmt.Println("Error executing tool call:", err)
					ch <- models.StoredMessageData{Role: "error", Content: fmt.Sprintf("Tool '%s' failed: %v", tc.Name, err)}
					continue
				}
				ch <- models.StoredMessageData{Role: "tool_call", Content: result, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}
				toolMsg := models.Message{
					SessionID: session_id,
					ID:        fmt.Sprintf("%s-%d", session_id, len(messages)+1+i), // Simplified ID generation
					Data:      models.EncodeMessageData(models.StoredMessageData{Role: "tool_call", Content: result, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}),
				}
				database.Create(&toolMsg)
				fmt.Println("Result of tool call:", result)
			}
		}
	}()
	return ch
}

func (a *Agent) TextSkill(skill_name string) (string, error) {
	result, err := llm.ExecuteToolCall(llm.ToolCall{Name: "skill", Arguments: fmt.Sprintf("{\"skill_name\": \"%s\"}", skill_name)})
	if err != nil {
		return "", err
	}
	return result, nil
}
