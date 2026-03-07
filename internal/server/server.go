package server

import (
	"encoding/json"
	"net/http"

	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func Initialise() {
	http.HandleFunc("GET /list-sessions", listSessions)
	http.HandleFunc("GET /get-session-data", getSessionData)
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
	database.Table("chats").Select("*").Where("session_id ?", session_id).Find(&messages)
	json.NewEncoder(w).Encode(messages)
}
