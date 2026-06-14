package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/api"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

var baseUrl string

func init() {
	port := config.GetCustomization().Port
	portToUse := getPortRunningServer()

	if port == "" {
		port = "8080"
	}

	baseUrl = "http://localhost:" + portToUse
}

func ListSession() ([]models.Session, error) {
	resp, err := http.Get(baseUrl + "/list-sessions")
	if err != nil {
		// fmt.Println(err.Error())
		return []models.Session{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sessions []models.Session
	json.Unmarshal(body, &sessions)
	Reverse(sessions)
	return sessions, nil
}

func Reverse(arr []models.Session) []models.Session {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func GetSessionData(session_id string) ([]models.Message, error) {
	resp, err := http.Get(baseUrl + "/get-session-data?session_id=" + url.QueryEscape(session_id))
	if err != nil {
		// fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var messages []models.Message
	json.Unmarshal(body, &messages)
	return messages, nil
}

func CreateSession(prompt string) (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		// fmt.Println(err.Error())
		return "", err
	}
	resp, err := http.Post(baseUrl+"/create-session?prompt="+url.QueryEscape(prompt)+"&working_directory="+url.QueryEscape(workingDirectory), "application/json", nil)
	if err != nil {
		// fmt.Println(err.Error())
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(body))
	return strings.TrimSpace(string(body)), nil
}

func ChatCompletion(ctx context.Context, session_id string, prompt string, mode string, img_bytes [][]byte) chan models.StoredMessageData {
	ch := make(chan models.StoredMessageData)
	go func() {
		defer close(ch)
		payload := map[string]interface{}{
			"images": img_bytes,
		}
		bodybytes, err := json.Marshal(payload)
		if err != nil {

		}
		url := baseUrl + "/chat-completion?session_id=" + url.QueryEscape(session_id) + "&prompt=" + url.QueryEscape(prompt) + "&mode=" + url.QueryEscape(mode)
		req, err := http.NewRequestWithContext(ctx, "GET", url, bytes.NewReader(bodybytes))
		if err != nil {
			// fmt.Println(err.Error())
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// fmt.Println(err.Error())
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

func SendMessage(session_id string, message string, img_bytes [][]byte) (models.Message, error) {
	payload := map[string]interface{}{
		"images": img_bytes,
	}
	bodybytes, err := json.Marshal(payload)
	if err != nil {

	}
	resp, err := http.Post(baseUrl+"/send-message?session_id="+url.QueryEscape(session_id)+"&message="+url.QueryEscape(message), "application/json", bytes.NewReader(bodybytes))
	if err != nil {
		// fmt.Println(err.Error())
		return models.Message{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var newMessage models.Message
	json.Unmarshal(body, &newMessage)
	return newMessage, nil
}

func DeleteSession(session_id string) error {
	resp, err := http.Post(baseUrl+"/delete-session?session_id="+url.QueryEscape(session_id), "application/json", nil)
	if err != nil {
		// fmt.Println(err.Error())
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	return nil
	//body, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(body))
}

func GetCurrentTodoList(session_id string) ([]models.ToDo, error) {
	resp, err := http.Get(baseUrl + "/get-current-todo-list?session_id=" + url.QueryEscape(session_id))
	if err != nil {
		// fmt.Println(err.Error())
		return []models.ToDo{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var todoList []models.ToDo
	json.Unmarshal(body, &todoList)
	return todoList, nil
}

func GetAvailableSkills(session_id string) []string {
	resp, err := http.Get(baseUrl + "/get-available-skills?session_id=" + url.QueryEscape(session_id))
	if err != nil {
		// fmt.Println(err.Error())
		return []string{}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var skillsList []string
	json.Unmarshal(body, &skillsList)
	return skillsList
}

func GetContextSize(session_id string) (int64, error) {
	resp, err := http.Get(baseUrl + "/get-context-size?session_id=" + url.QueryEscape(session_id))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	contextSize, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return 0, err
	}
	return contextSize, nil
}

func CompactMemory(session_id string) (int64, error) {
	resp, err := http.Get(baseUrl + "/compact-memory?session_id=" + url.QueryEscape(session_id))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New(strings.TrimSpace(string(body)))
	}

	contextSize, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	if err != nil {
		return 0, err
	}

	return contextSize, nil
}

func GetModels() ([]api.ModelInfo, []api.ModelInfo, error) {
	resp, err := http.Get(baseUrl + "/get-models")
	if err != nil {
		// log.Fatal(err)
		return []api.ModelInfo{}, []api.ModelInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// log.Fatal(err)
		return []api.ModelInfo{}, []api.ModelInfo{}, err
	}
	var modelsList api.ModelTypes
	err = json.Unmarshal(body, &modelsList)
	if err != nil {
		// log.Fatal(err)
		return []api.ModelInfo{}, []api.ModelInfo{}, err
	}
	return modelsList.Models, modelsList.Recent, nil
}
func GetCurrentModel() (config.ResModel, error) {
	resp, err := http.Get(baseUrl + "/get-current-model")
	if err != nil {
		// log.Fatal(err)
		return config.ResModel{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// log.Fatal(err)
		return config.ResModel{}, err
	}
	var cur_model config.ResModel
	err = json.Unmarshal(body, &cur_model)
	if err != nil {
		// log.Fatal(err)
		return config.ResModel{}, err
	}
	return cur_model, nil
}

func SetApiKey(m api.ModelInfo, apikey string) error {
	body := struct {
		ApiKey string        `json:"api_key"`
		Model  api.ModelInfo `json:"model"`
	}{
		ApiKey: apikey,
		Model:  m,
	}
	bytess, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = http.Post(baseUrl+"/set-api-key", "application/json", bytes.NewReader(bytess))
	if err != nil {
		return err
	}
	return nil
}

func SetCurrentModel(m api.ModelInfo) error {
	bodybytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = http.Post(baseUrl+"/set-current-model", "application/json", bytes.NewReader(bodybytes))
	if err != nil {
		return err
	}
	return nil
}

func SetReasoningEffort(effort string) error {
	bodybytes, err := json.Marshal(struct {
		Effort string `json:"effort"`
	}{Effort: effort})
	if err != nil {
		return err
	}
	resp, err := http.Post(baseUrl+"/set-reasoning-effort", "application/json", bytes.NewReader(bodybytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(strings.TrimSpace(string(body)))
	}
	return nil
}

func GetCopilotModels() ([]string, error) {
	resp, err := http.Get(baseUrl + "/get-copilot-models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var models []string
	err = json.Unmarshal(body, &models)
	if err != nil {
		return nil, err
	}
	return models, nil
}
