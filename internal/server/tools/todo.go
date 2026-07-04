package tools

import (
	"fmt"
	"math"

	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func CreateTodo(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	raw, ok := args["descriptions"].([]any)
	if !ok {
		return ToolResponse{Content: "Error: descriptions is required and must be an array of strings", Printable: "Error: descriptions is required and must be an array of strings"}, nil
	}

	descriptions := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return ToolResponse{Content: "Error: each description must be a string", Printable: "Error: each description must be a string"}, nil
		}
		descriptions = append(descriptions, s)
	}
	todos := make([]models.ToDo, len(descriptions))
	database, _ := db.Connect()
	for i, description := range descriptions {
		todos[i] = models.ToDo{Index: i, Description: description, Completed: false}
	}
	result := database.Model(&models.Session{}).Where("id = ?", ctx.SessionID).Update("to_do_list", models.EncodeToDoList(todos))
	if result.Error != nil {
		return ToolResponse{Content: "Error: failed to update todo list", Printable: "Error: failed to update todo list"}, nil
	}
	return ToolResponse{Content: "Todo list Created successfully", Printable: "Todo list Created successfully"}, nil
}

func UpdateTodo(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	// JSON numbers are decoded as float64 into map[string]any (see ParseArgs / json.Unmarshal).
	var index int
	switch v := args["index"].(type) {
	case float64:
		if v != math.Trunc(v) || v < 0 || v > float64(math.MaxInt) {
			return ToolResponse{Content: "Error: index is required and must be a non-negative integer", Printable: "Error: index is required and must be a non-negative integer"}, nil
		}
		index = int(v)
	case int:
		if v < 0 {
			return ToolResponse{Content: "Error: index is required and must be a non-negative integer", Printable: "Error: index is required and must be a non-negative integer"}, nil
		}
		index = v
	default:
		return ToolResponse{Content: "Error: index is required and must be an integer", Printable: "Error: index is required and must be an integer"}, nil
	}
	completed, ok := args["completed"].(bool)
	if !ok {
		return ToolResponse{Content: "Error: completed is required and must be a boolean", Printable: "Error: completed is required and must be a boolean"}, nil
	}
	database, _ := db.Connect()
	var session models.Session
	database.Model(&models.Session{}).Where("id = ?", ctx.SessionID).First(&session)
	todos := models.DecodeToDoList(session.ToDoList)
	if index < 0 || index >= len(todos) {
		msg := fmt.Sprintf("Error: index %d is out of range (list has %d items)", index, len(todos))
		return ToolResponse{Content: msg, Printable: msg}, nil
	}
	todos[index] = models.ToDo{Index: index, Description: todos[index].Description, Completed: completed}
	result := database.Model(&models.Session{}).Where("id = ?", ctx.SessionID).Update("to_do_list", models.EncodeToDoList(todos))
	if result.Error != nil {
		return ToolResponse{Content: "Error: failed to update todo list", Printable: "Error: failed to update todo list"}, nil
	}
	msg := "Todo updated successfully: " + models.EncodeToDoList(todos)
	return ToolResponse{Content: msg, Printable: msg}, nil
}

func init() {
	// Register("create_todo", ToolDef{
	// 	Name:        "create_todo",
	// 	Description: "Create a todo list by providing a list of string descriptions in order",
	// 	Params: map[string]any{
	// 		"type": "object",
	// 		"properties": map[string]any{
	// 			"descriptions": map[string]any{
	// 				"type": "array",
	// 				"items": map[string]any{
	// 					"type":        "string",
	// 					"description": "The description of the todo",
	// 				},
	// 			},
	// 		},
	// 		"required": []string{"descriptions"},
	// 	},
	// }, CreateTodo)
	// Register("update_todo", ToolDef{
	// 	Name:        "update_todo",
	// 	Description: "Update a todo by providing an index and a completed boolean, only provide the data of a specific item in the list not all the fucking items in the list, only ONE AT A TIME.",
	// 	Params: map[string]any{
	// 		"type": "object",
	// 		"properties": map[string]any{
	// 			"index": map[string]any{
	// 				"type":        "integer",
	// 				"description": "The index of the todo to update",
	// 			},
	// 			"completed": map[string]any{
	// 				"type":        "boolean",
	// 				"description": "The completed status of the todo",
	// 			},
	// 		},
	// 		"required": []string{"index", "completed"},
	// 	},
	// }, UpdateTodo)
}
