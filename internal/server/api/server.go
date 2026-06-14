package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/agent"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm/llmModel"
	"github.com/Kartik-2239/lightcode/internal/server/oauth"
	"gorm.io/gorm"
)

var DB *gorm.DB

type Request struct {
	Images [][]byte `json:"images"`
}

func Initialise(ready chan<- struct{}, port string, isDebug bool) {
	config.Debug = isDebug
	database, err := db.Connect()
	if err != nil {
		fmt.Println("Database error")
		return
	}
	DB = database
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "lightcode is running!")
	})
	http.HandleFunc("GET /list-sessions", listSessions)
	http.HandleFunc("GET /get-session-data", getSessionData)
	http.HandleFunc("GET /chat-completion", chatcompletion)
	http.HandleFunc("POST /send-message", sendMessage)
	http.HandleFunc("POST /create-session", createSession)
	http.HandleFunc("POST /delete-session", deleteSession)
	http.HandleFunc("GET /get-current-todo-list", getCurrentTodoList)
	http.HandleFunc("GET /get-context-size", getContextSize)
	http.HandleFunc("GET /get-available-skills", getAvailableSkills)
	http.HandleFunc("GET /get-current-model", getCurrentModel)
	http.HandleFunc("GET /get-models", getModels)
	http.HandleFunc("POST /set-api-key", setApiKey)
	http.HandleFunc("POST /set-current-model", setCurrentModel)
	http.HandleFunc("POST /set-reasoning-effort", setReasoningEffort)
	http.HandleFunc("GET /compact-memory", compactMemory)
	http.HandleFunc("GET /get-copilot-models", getCopilotModels)
	// http.ListenAndServe(":8080", nil)

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "address already in use") {
			fmt.Println("Running only tui")
			return
			// close(ready)
		}
		// log.Fatal(err)
	}
	close(ready)

	http.Serve(ln, nil)
}

func listSessions(w http.ResponseWriter, r *http.Request) {
	var sessions []models.Session
	DB.Table("sessions").Select("*").Find(&sessions)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func getSessionData(w http.ResponseWriter, r *http.Request) {
	var messages []models.Message
	session_id := r.URL.Query().Get("session_id")
	DB.Table("messages").Select("*").Where("session_id = ?", session_id).Limit(1000).Find(&messages)
	fmt.Println("Number of messages: ", len(messages))
	json.NewEncoder(w).Encode(messages)
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	session_id := r.URL.Query().Get("session_id")
	message := r.URL.Query().Get("message")

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {

	}
	var messages []models.Message
	DB.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	newMessage := models.Message{SessionID: session_id, Data: models.EncodeMessageData(models.StoredMessageData{Role: "user", Content: message})}
	DB.Create(&newMessage)
	json.NewEncoder(w).Encode(newMessage)
}

func deleteSession(w http.ResponseWriter, r *http.Request) {
	session_id := r.URL.Query().Get("session_id")
	DB.Table("messages").Where("session_id = ?", session_id).Delete(&models.Message{})
	DB.Table("sessions").Where("id = ?", session_id).Delete(&models.Session{})
	fmt.Fprint(w, "Session deleted successfully")
}

func createSession(w http.ResponseWriter, r *http.Request) {
	prompt := r.URL.Query().Get("prompt")
	workingDirectory := r.URL.Query().Get("working_directory")
	session_id := randomSessionID()
	session := models.Session{ID: session_id, Title: prompt, Directory: workingDirectory}
	DB.Create(&session)
	fmt.Fprint(w, session_id)
}

func chatcompletion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	session_id := r.URL.Query().Get("session_id")
	prompt := r.URL.Query().Get("prompt")
	mode := r.URL.Query().Get("mode")
	model := config.GetCustomization().CurrentModel

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {

	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	for result := range agent.New(DB).Run(r.Context(), model, prompt, req.Images, session_id, mode) {
		if r.Context().Err() != nil {
			return
		}
		fmt.Fprintf(w, "%s\n", models.EncodeMessageData(result))
		flusher.Flush()
	}
	fmt.Fprintf(w, "[DONE]\n")
	flusher.Flush()
}

func getCurrentTodoList(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, `{"error":"session_id required"}`, http.StatusBadRequest)
		return
	}
	// database, err := db.Connect()
	// if err != nil {
	// 	http.Error(w, `{"error":"database unavailable"}`, http.StatusInternalServerError)
	// 	return
	// }
	var session models.Session
	if err := DB.Where("id = ?", sessionID).First(&session).Error; err != nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}
	todos := models.DecodeToDoList(session.ToDoList)
	if todos == nil {
		todos = []models.ToDo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func getContextSize(w http.ResponseWriter, r *http.Request) {
	// database, _ := db.Connect()
	var messages []models.Message
	session_id := r.URL.Query().Get("session_id")
	DB.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	slices.Reverse(messages)
	var context_size int64
	var afterLastAssistant []models.Message
	for _, m := range messages {
		afterLastAssistant = append(afterLastAssistant, m)
		if models.DecodeMessageData(m.Data).Role == "assistant" {
			context_size = models.DecodeMessageData(m.Data).Usage.PromptTokens + int64(len(models.DecodeMessageData(m.Data).Content)/4)

			break
		}
	}
	for _, m := range afterLastAssistant {
		context_size += int64(len(models.DecodeMessageData(m.Data).Content) / 4)
	}
	fmt.Fprint(w, context_size)
}

func getAvailableSkills(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(config.SkillsPath())
	if err != nil {
		json.NewEncoder(w).Encode([]string{})
		fmt.Println("gone")
		return
	}
	skills := []string{}
	for _, f := range files {
		subFiles, _ := os.ReadDir(config.SkillsPath() + "/" + f.Name())
		for _, subf := range subFiles {
			if strings.Contains(subf.Name(), "SKILL.md") {
				skills = append(skills, f.Name())
				break
			}
		}
	}
	json.NewEncoder(w).Encode(skills)
}

func getCurrentModel(w http.ResponseWriter, r *http.Request) {
	cur_model := config.GetCustomization().CurrentModel
	if cur_model.Model != "" {
		json.NewEncoder(w).Encode(cur_model)
	}
}

func getModels(w http.ResponseWriter, r *http.Request) {
	list_models, recent_models, err := config.GetModels()
	if err != nil {

	}
	if copilotAuth, err := config.GetAuthVal(config.CopilotAuthProvider); err == nil && copilotAuth.Access != "" {
		_ = oauth.RefreshSavedCopilotModels()
	}
	authModels, err := config.GetAllAuthModels()
	if err != nil {
		// log.Println("fuck no models in openai codex shit bitch", err)
	}
	// models := make([]ModelInfo, 0, len(list_models)+len(authModels))
	models := []ModelInfo{}
	recent := make([]ModelInfo, len(recent_models))

	for _, m := range list_models {
		models = append(models, ModelInfo{
			Model:           m.Model,
			ApiKey:          m.ApiKey,
			BaseUrl:         m.BaseUrl,
			ReasoningEffort: m.ReasoningEffort,
			Provider:        providerFromBaseUrl(m.BaseUrl),
		})
	}
	for _, m := range authModels {
		models = append(models, ModelInfo{
			Model:           m.Model,
			ApiKey:          m.ApiKey,
			BaseUrl:         m.BaseUrl,
			ReasoningEffort: m.ReasoningEffort,
			Provider:        m.BaseUrl + " auth",
		})
	}
	for i, m := range recent_models {
		recent[i] = ModelInfo{
			Model:           m.Model,
			ApiKey:          m.ApiKey,
			BaseUrl:         m.BaseUrl,
			ReasoningEffort: m.ReasoningEffort,
			Provider:        providerLabelFromBaseUrl(m.BaseUrl),
			LastUsed:        m.LastUsed,
		}
	}
	var payload ModelTypes
	payload = ModelTypes{
		Models: models,
		Recent: recent,
	}
	json.NewEncoder(w).Encode(payload)
}

func setApiKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ApiKey string          `json:"api_key"`
		Model  config.ResModel `json:"model"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = config.SetApiKey(req.Model, req.ApiKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}
func setCurrentModel(w http.ResponseWriter, r *http.Request) {
	var req config.ResModel
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = config.SetCurrentModel(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func setReasoningEffort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Effort string `json:"effort"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = config.SetReasoningEffort(req.Effort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func compactMemory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, `{"error":"session_id required"}`, http.StatusBadRequest)
		return
	}

	var messages []models.Message
	DB.Where("session_id = ?", sessionID).Find(&messages)
	if len(messages) == 0 {
		fmt.Fprint(w, 0)
		return
	}

	chats := make([]llmModel.Chat, 0, len(messages))
	slices.Reverse(messages)
	for _, message := range messages {
		d := models.DecodeMessageData(message.Data)
		if strings.HasPrefix(d.Content, "<memory>") && strings.HasSuffix(d.Content, "</memory>") {
			chats = append(chats, llmModel.Chat{Role: "user", Content: d.Content})
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
			chats = append(chats, llmModel.Chat{Role: "assistant", Content: d.Content})
		default:
			chats = append(chats, llmModel.Chat{Role: "user", Content: d.Content})
		}
	}
	slices.Reverse(chats)

	compactedMemory, err := agent.CompactMemory(chats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	compactedMsg := models.Message{
		SessionID: sessionID,
		Data:      models.EncodeMessageData(compactedMemory),
	}
	if err := DB.Create(&compactedMsg).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, len(compactedMemory.Content)/4)
}

func getCopilotModels(w http.ResponseWriter, r *http.Request) {
	if _, err := config.GetAuthVal(config.CopilotAuthProvider); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	models := oauth.DefaultCopilotPickerModelNames()
	json.NewEncoder(w).Encode(models)
}

func getCopilotModelsRaw(w http.ResponseWriter, r *http.Request) {
	models, err := oauth.MakeModelsRequest()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	model_list := make([]string, len(models))
	for i, m := range models {
		model_list[i] = m.Name
	}
	json.NewEncoder(w).Encode(model_list)
}
