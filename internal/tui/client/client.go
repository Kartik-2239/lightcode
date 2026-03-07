package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func ListSession() []models.Session {
	resp, err := http.Get("http://localhost:8080/list-sessions")
	if err != nil {
		fmt.Println(err.Error())
	}
	body, _ := io.ReadAll(resp.Body)
	var sessions []models.Session
	json.Unmarshal(body, &sessions)
	if len(sessions[0].Title) > 30 {
		fmt.Println(sessions[0].Title[:30])
	}
	return sessions
}

func GetSessionData(session_id string) []models.Message {
	resp, err := http.Get("http://localhost:8080/get-session-data")
	if err != nil {
		fmt.Println(err.Error())
	}
	body, _ := io.ReadAll(resp.Body)
	var messages []models.Message
	json.Unmarshal(body, &messages)
	return messages
}
