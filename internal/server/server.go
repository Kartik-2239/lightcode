package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/agent"
	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func Initialise() {
	http.HandleFunc("GET /list-sessions", listSessions)
	http.HandleFunc("GET /get-session-data", getSessionData)
	http.HandleFunc("GET /chat-completion", chatcompletion)
	http.HandleFunc("POST /send-message", sendMessage)
	http.HandleFunc("POST /create-session", createSession)
	http.HandleFunc("POST /delete-session", deleteSession)
	http.ListenAndServe(":8080", nil)
}

func listSessions(w http.ResponseWriter, r *http.Request) {
	database, _ := db.Connect()
	var sessions []models.Session
	database.Table("sessions").Select("*").Find(&sessions)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func getSessionData(w http.ResponseWriter, r *http.Request) {
	database, _ := db.Connect()
	var messages []models.Message
	session_id := r.URL.Query().Get("session_id")
	database.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	// fmt.Println(messages)
	json.NewEncoder(w).Encode(messages)
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	database, _ := db.Connect()
	session_id := r.URL.Query().Get("session_id")
	message := r.URL.Query().Get("message")
	var messages []models.Message
	database.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	newMessage := models.Message{SessionID: session_id, ID: fmt.Sprintf("%s-user-%d", session_id, len(messages)), Data: models.EncodeMessageData(models.StoredMessageData{Role: "user", Content: message})}
	database.Create(&newMessage)
	json.NewEncoder(w).Encode(newMessage)
}

func deleteSession(w http.ResponseWriter, r *http.Request) {
	session_id := r.URL.Query().Get("session_id")
	database, _ := db.Connect()
	database.Table("messages").Where("session_id = ?", session_id).Delete(&models.Message{})
	database.Table("sessions").Where("id = ?", session_id).Delete(&models.Session{})
	fmt.Fprint(w, "Session deleted successfully")
}

func createSession(w http.ResponseWriter, r *http.Request) {
	prompt := r.URL.Query().Get("prompt")
	database, _ := db.Connect()
	session_id := randomSessionID()
	session := models.Session{ID: session_id, Title: prompt, Directory: "."}
	database.Create(&session)
	fmt.Fprint(w, session_id)
}

func chatcompletion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	session_id := r.URL.Query().Get("session_id")
	prompt := r.URL.Query().Get("prompt")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	for result := range agent.New().Run(r.Context(), prompt, session_id) {
		if r.Context().Err() != nil {
			return
		}
		fmt.Fprintf(w, "%s\n", models.EncodeMessageData(result))
		flusher.Flush()
	}
	fmt.Fprintf(w, "[DONE]\n")
	flusher.Flush()
}

func randomSessionID() string {
	var chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890-_"
	length := 10
	var result strings.Builder
	for range length {
		result.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	return result.String()
}
