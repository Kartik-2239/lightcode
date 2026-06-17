package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm"
	"github.com/Kartik-2239/lightcode/internal/server/llm/llmModel"
	"github.com/Kartik-2239/lightcode/internal/server/tools"
	"gorm.io/gorm"
)

const MaxIterations = 25

// const DEBUG = false
const CONTEXT_WINDOW int64 = 128_000

type Agent struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Agent {
	return &Agent{db}
}

// isDir reports whether the workspace-relative path p resolves to a directory
// inside workingDir. Used to pick read_file vs list_dir for @file mentions.
func isDir(workingDir, p string) bool {
	resolved, err := tools.ValidatePath(tools.ToolContext{WorkingDirectory: workingDir}, p)
	if err != nil {
		return false
	}
	info, err := os.Stat(resolved)
	return err == nil && info.IsDir()
}

func (a *Agent) Run(ctx context.Context, model config.ResModel, prompt string, b64_imgs [][]byte, session_id string, mode string) <-chan models.StoredMessageData {
	ch := make(chan models.StoredMessageData)
	// currentPrompt := prompt
	database := a.db
	// if err != nil {
	// 	ch <- models.StoredMessageData{Role: "error", Content: "Ran into error: " + err.Error()}
	// 	close(ch)
	// 	return ch
	// }
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

		for i := range MaxIterations {

			select {
			case <-ctx.Done():
				return
			default:
			}
			if config.Debug {
				fmt.Println(strings.Repeat("=", 30))
				fmt.Println("Iteration:", i)
			}
			var messages []models.Message
			database.Where("session_id = ?", session_id).Order("time_created ASC").Find(&messages)
			chats := make([]llmModel.Chat, 0, len(messages)+2)

			// before starting the agent check if the token usage till the last memory compaction.
			/*
				token_count := 0
				for _, message := range messages {
					d := models.DecodeMessageData(message.Data)
					token_count += d.Usage.TotalTokens
				}
			*/
			var token_count int64 = 0
			slices.Reverse(messages)

			// Session (working directory) is needed below to expand @file mentions
			// while building the chat history; seen dedups repeated mentions across
			// turns, keeping the most recent (the loop runs newest-first).
			var session models.Session
			database.Where("id = ?", session_id).First(&session)
			seen := map[string]bool{}

			for _, message := range messages {
				d := models.DecodeMessageData(message.Data)
				if strings.HasPrefix(d.Content, "<memory>") && strings.HasSuffix(d.Content, "</memory>") {
					chats = append(chats, llmModel.Chat{
						Role:    "user",
						Content: d.Content,
					})
					break
				}
				switch d.Role {
				case "tool_call":
					name, id := "tool", ""
					if len(d.ToolCalls) > 0 {
						name = d.ToolCalls[0].Name
						id = d.ToolCalls[0].ID
					}
					chats = append(chats, llmModel.Chat{
						Role:    "user",
						Content: fmt.Sprintf("Tool %q (call_id=%s) output:\n%s", name, id, d.Content),
					})
				case "assistant":
					content := d.Content
					chats = append(chats, llmModel.Chat{Role: "assistant", Content: content})
				default:
					// Expand explicit @file mentions into read_file/list_dir tool-output
					// blocks appended to this user message, so the model sees the file as
					// if it had run the tool itself. The blocks are embedded in the same
					// Chat (rather than separate entries) to preserve the 1:1 alignment
					// between chats and messages that the rest of Run/ApiCall relies on.
					// Dedup keeps the most-recent mention of each path (loop is newest-first).
					content := d.Content
					for _, p := range d.Mentions {
						if p == "" || seen[p] {
							continue
						}
						seen[p] = true
						toolName := "read_file"
						if isDir(session.Directory, p) {
							toolName = "list_dir"
						}
						res, _ := tools.Execute(toolName, tools.ToolContext{WorkingDirectory: session.Directory, SessionID: session_id}, map[string]any{"path": p})
						content += fmt.Sprintf("\n\nTool %q (call_id=%s) output:\n%s", toolName, "mention-"+p, res.Content)
					}
					chats = append(chats, llmModel.Chat{Role: "user", Content: content})
				}
			}
			slices.Reverse(chats)
			slices.Reverse(messages)
			// memory compaction
			if config.Debug {
				for _, item := range messages[len(messages)-len(chats):] {
					data := models.DecodeMessageData(item.Data).Content
					if len(data) > 100 {
						fmt.Println(data[:100])
					} else {
						fmt.Println(data)
					}
				}
			}
			token_count = predictTokenCount(messages[len(messages)-len(chats):], len(chats)-1)

			if config.Debug {
				fmt.Println("token_count", token_count)
				// fmt.Println("CHATS: ", chats)
			}
			if token_count >= CONTEXT_WINDOW {
				// leave the last few chats before compacting memory
				compactedMemory, err := CompactMemory(chats)
				if err != nil {
					if config.Debug {
						fmt.Println("Failed compacting memory with error : ", err)
					}
				}
				compactedMsg := models.Message{ // wrap it
					SessionID: session_id,
					Data:      models.EncodeMessageData(compactedMemory),
				}

				if err := database.Create(&compactedMsg).Error; err != nil {
					if config.Debug {
						fmt.Println("Error creating message:", err, "?")
					}
				}
				ch <- models.StoredMessageData{Role: "assistant", Content: "Compacting context"}
				continue
			}

			// cur_list := session.ToDoList
			// chats = append(chats, llm.Chat{Role: "user", Content: cur_list})
			agents_md, err := ReadAgentsMd(session.Directory)
			if err != nil {
				if config.Debug {
					fmt.Println("Error reading AGENTS.md:", err)
				}
				agents_md = ""
			}
			var resp llmModel.Response
			if len(b64_imgs) > 0 {
				if len(chats) <= 1 {
					resp, err = llm.ApiCall(ctx, model, prompt, []llmModel.Chat{}, []models.Message{}, mode, b64_imgs, agents_md)
				} else {
					resp, err = llm.ApiCall(ctx, model, prompt, chats[:len(chats)-1], messages, mode, b64_imgs, agents_md)
				}
			} else {
				resp, err = llm.ApiCall(ctx, model, prompt, chats, messages, mode, [][]byte{}, agents_md)
			}
			if err != nil {
				errorMessage := models.StoredMessageData{Role: "error", Content: resp.Text, Usage: &models.StoredUsage{}}
				ch <- errorMessage
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			if config.Debug {
				fmt.Println("")
				// fmt.Println("Tool calls:", resp.ToolCalls)
				fmt.Println("Number of tool calls:", len(resp.ToolCalls))
				fmt.Println("")
			}
			if len(resp.ToolCalls) == 0 {
				select {
				case <-ctx.Done():
					return
				default:
				}
				assistantMessage := models.StoredMessageData{Role: "assistant", Content: resp.Text, Usage: &models.StoredUsage{PromptTokens: resp.CompleteResponse.Usage.PromptTokens, CompletionTokens: resp.CompleteResponse.Usage.CompletionTokens, TotalTokens: resp.CompleteResponse.Usage.TotalTokens}}
				newMessage := models.Message{
					SessionID: session_id,
					// ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
					Data: models.EncodeMessageData(assistantMessage),
				}
				if config.Debug {
					fmt.Println("Creating message:", newMessage)
				}
				if err := database.Create(&newMessage).Error; err != nil {
					if config.Debug {
						fmt.Println("Error creating message:", err)
					}
					return
				} else {
					if config.Debug {
						fmt.Println("Message created successfully! LAST!")
					}
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
				// ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
				Data: models.EncodeMessageData(assistantMessage),
			}
			ch <- assistantMessage
			if config.Debug {
				fmt.Println("Creating message:", resp.CompleteResponse.Usage)
			}
			if err := database.Create(&assistantMsg).Error; err != nil {
				if config.Debug {
					fmt.Println("Error creating message:", err)
				}
			} else {
				if config.Debug {
					fmt.Println("Message created successfully!")
				}
			}
			for _, tc := range resp.ToolCalls {
				if config.Debug {
					fmt.Println("Executing tool call:", tc.Name)
				}
				if tc.Name == "question" {
					ch <- models.StoredMessageData{Role: "question", Content: tc.Arguments}
					questionMsg := models.Message{
						SessionID: session_id,
						Data:      models.EncodeMessageData(models.StoredMessageData{Role: "question", Content: tc.Arguments, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}),
					}
					database.Create(&questionMsg)
					return
				}
				result, err := llm.ExecuteToolCall(tc, session.Directory, session_id)
				if err != nil {
					if config.Debug {
						fmt.Println("Error executing tool call:", err)
					}
					ch <- models.StoredMessageData{Role: "error", Content: fmt.Sprintf("Tool '%s' failed: %v", tc.Name, err)}
					continue
				}
				ch <- models.StoredMessageData{Role: "tool_call", Content: result.Content, CodeChanges: result.CodeChanges, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}
				toolMsg := models.Message{
					SessionID: session_id,
					Data:      models.EncodeMessageData(models.StoredMessageData{Role: "tool_call", Content: result.Content, CodeChanges: result.CodeChanges, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}),
				}
				database.Create(&toolMsg)
				if config.Debug {
					if len(result.Content) > 100 {
						fmt.Println("Result of tool call:", result.Content[:100], ".....")
					} else {
						fmt.Println("Result of tool call:", result.Content)
					}
				}
			}
		}
	}()
	return ch
}

// func (a *Agent) TextSkill(skill_name string) (string, error) {
// 	result, err := llm.ExecuteToolCall(llm.ToolCall{Name: "skill", Arguments: fmt.Sprintf("{\"skill_name\": \"%s\"}", skill_name)})
// 	if err != nil {
// 		return "", err
// 	}
// 	return result, nil
// }
