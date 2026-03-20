package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func ListSession() []models.Session {
	resp, err := http.Get("http://localhost:8080/list-sessions")
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sessions []models.Session
	json.Unmarshal(body, &sessions)
	return sessions
}

func GetSessionData(session_id string) []models.Message {
	resp, err := http.Get("http://localhost:8080/get-session-data?session_id=" + url.QueryEscape(session_id))
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var messages []models.Message
	json.Unmarshal(body, &messages)
	return messages
}

func CreateSession(prompt string) string {
	resp, err := http.Post("http://localhost:8080/create-session?prompt="+url.QueryEscape(prompt), "application/json", nil)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
	return strings.TrimSpace(string(body))
}

func ChatCompletion(ctx context.Context, session_id string, prompt string) chan models.StoredMessageData {
	ch := make(chan models.StoredMessageData)
	go func() {
		defer close(ch)
		url := "http://localhost:8080/chat-completion?session_id=" + url.QueryEscape(session_id) + "&prompt=" + url.QueryEscape(prompt)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || line == "[DONE]" {
				break
			}
			var message models.StoredMessageData
			if err := json.Unmarshal([]byte(line), &message); err != nil {
				continue
			}
			if message.Role == "" {
				continue
			}
			ch <- message
		}
	}()
	return ch
}

func SendMessage(session_id string, message string) models.Message {
	resp, err := http.Post("http://localhost:8080/send-message?session_id="+url.QueryEscape(session_id)+"&message="+url.QueryEscape(message), "application/json", nil)
	if err != nil {
		fmt.Println(err.Error())
		return models.Message{}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var newMessage models.Message
	json.Unmarshal(body, &newMessage)
	return newMessage
}

func DeleteSession(session_id string) {
	resp, err := http.Post("http://localhost:8080/delete-session?session_id="+url.QueryEscape(session_id), "application/json", nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}
