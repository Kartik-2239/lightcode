package db

import (
	"encoding/json"

	"github.com/Kartik-2239/lightcode/internal/server/db/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("/Users/kartikkannan/Desktop/lightcode/lightcode.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&models.Session{},
		&models.Message{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func SeedData(db *gorm.DB) error {
	sessions := []models.Session{
		{
			ID:        "session-1",
			Directory: "/Users/kartikkannan/Desktop/lightcode",
			Title:     "Initial Project Setup",
		},
		{
			ID:        "session-2",
			Directory: "/Users/kartikkannan/Desktop/lightcode",
			Title:     "Database Configuration",
		},
	}

	msgData1, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-1",
		"object":  "chat.completion",
		"created": 1772724309,
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
						"response": "What files are in my project?",
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 0,
			"total_tokens":      10,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 0,
			},
		},
	})
	msgData2, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-2",
		"object":  "chat.completion",
		"created": 1772724310,
		"model":   "MiniMax-M2.5",
		"choices": []map[string]interface{}{
			{
				"finish_reason": "stop",
				"index":         0,
				"message": map[string]interface{}{
					"role":          "assistant",
					"name":          "MiniMax AI",
					"audio_content": "",
					"content": map[string]interface{}{
						"thinking": "The user wants to know what files are in their project. Let me list them out.",
						"response": "I found several files in your project: go.mod, go.sum, main, and several directories (cmd, internal).",
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     373,
			"completion_tokens": 75,
			"total_tokens":      448,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 53,
			},
		},
	})
	msgData3, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-3",
		"object":  "chat.completion",
		"created": 1772724400,
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
						"response": "List the directories inside internal/",
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     20,
			"completion_tokens": 0,
			"total_tokens":      20,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 0,
			},
		},
	})
	msgData4, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-4",
		"object":  "chat.completion",
		"created": 1772724401,
		"model":   "MiniMax-M2.5",
		"choices": []map[string]interface{}{
			{
				"finish_reason": "stop",
				"index":         0,
				"message": map[string]interface{}{
					"role":          "assistant",
					"name":          "MiniMax AI",
					"audio_content": "",
					"content": map[string]interface{}{
						"thinking": "The user wants to see the directories inside internal/. Let me list them.",
						"response": "The internal/ directory contains: tools/, llm/, memory/, config/, agent/, cli/",
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     400,
			"completion_tokens": 50,
			"total_tokens":      450,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 30,
			},
		},
	})

	msgData5, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-5",
		"object":  "chat.completion",
		"created": 1772724500,
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
						"response": "Help me set up a new database",
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     25,
			"completion_tokens": 0,
			"total_tokens":      25,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 0,
			},
		},
	})
	msgData6, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-6",
		"object":  "chat.completion",
		"created": 1772724501,
		"model":   "MiniMax-M2.5",
		"choices": []map[string]interface{}{
			{
				"finish_reason": "tool_calls",
				"index":         0,
				"message": map[string]interface{}{
					"role":          "assistant",
					"name":          "MiniMax AI",
					"audio_content": "",
					"content": map[string]interface{}{
						"thinking": "The user wants help setting up a database. I need to check their current configuration first by reading the .env file.",
						"response": "I'll help you set up a database. Let me first check your current configuration.",
					},
					"tool_calls": []map[string]interface{}{
						{
							"id":       "call_1",
							"type":     "function",
							"function": map[string]interface{}{"name": "read_file", "arguments": `{"path": "/Users/kartikkannan/Desktop/lightcode/.env"}`},
						},
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     420,
			"completion_tokens": 80,
			"total_tokens":      500,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 45,
			},
		},
	})
	msgData7, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-7",
		"object":  "chat.completion",
		"created": 1772724502,
		"model":   "MiniMax-M2.5",
		"choices": []map[string]interface{}{
			{
				"finish_reason": "stop",
				"index":         0,
				"message": map[string]interface{}{
					"role":          "tool",
					"name":          "",
					"audio_content": "",
					"content": map[string]interface{}{
						"thinking": "",
						"response": `DATABASE_URL=sqlite:/Users/kartikkannan/Desktop/lightcode/lightcode.db`,
					},
					"tool_call_id": "call_1",
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     100,
			"completion_tokens": 10,
			"total_tokens":      110,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 0,
			},
		},
	})
	msgData8, _ := json.Marshal(map[string]interface{}{
		"id":      "resp-8",
		"object":  "chat.completion",
		"created": 1772724600,
		"model":   "MiniMax-M2.5",
		"choices": []map[string]interface{}{
			{
				"finish_reason": "stop",
				"index":         0,
				"message": map[string]interface{}{
					"role":          "assistant",
					"name":          "MiniMax AI",
					"audio_content": "",
					"content": map[string]interface{}{
						"thinking": "I can see your .env file contains the database URL. Let me help you configure the database connection.",
						"response": "I can see your .env file. Let me help you configure the database connection.",
					},
				},
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     600,
			"completion_tokens": 60,
			"total_tokens":      660,
			"total_characters":  0,
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 35,
			},
		},
	})

	messages := []models.Message{
		{SessionID: "session-1", ID: "msg-1", Data: string(msgData1)},
		{SessionID: "session-1", ID: "msg-2", Data: string(msgData2)},
		{SessionID: "session-1", ID: "msg-3", Data: string(msgData3)},
		{SessionID: "session-1", ID: "msg-4", Data: string(msgData4)},
		{SessionID: "session-2", ID: "msg-5", Data: string(msgData5)},
		{SessionID: "session-2", ID: "msg-6", Data: string(msgData6)},
		{SessionID: "session-2", ID: "msg-7", Data: string(msgData7)},
		{SessionID: "session-2", ID: "msg-8", Data: string(msgData8)},
	}

	for _, session := range sessions {
		db.Create(&session)
	}
	for _, msg := range messages {
		db.Create(&msg)
	}

	return nil
}
