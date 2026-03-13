package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Kartik-2239/lightcode/internal/server/agent"
	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func Initialise() {
	http.HandleFunc("GET /list-sessions", listSessions)
	http.HandleFunc("GET /get-session-data", getSessionData)
	http.HandleFunc("GET /chatcompletion", chatcompletion)
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
		// json.NewEncoder(w).Encode(result)
		fmt.Fprintf(w, "data: %s\n\n", result)
		flusher.Flush()
	}

}
