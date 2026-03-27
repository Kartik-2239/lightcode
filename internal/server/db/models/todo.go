package models

import "encoding/json"

type ToDo struct {
	Index       int
	Description string
	Completed   bool
}

func EncodeToDoList(todos []ToDo) string {
	b, _ := json.Marshal(todos)
	return string(b)
}

func DecodeToDoList(s string) []ToDo {
	var todos []ToDo
	json.Unmarshal([]byte(s), &todos)
	return todos
}
