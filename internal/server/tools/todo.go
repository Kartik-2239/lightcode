package tools

import (
	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func CreateTodo(ctx ToolContext, args map[string]any) (string, error) {
	raw, ok := args["descriptions"].([]any)
	if !ok {
		return "Error: descriptions is required and must be an array of strings", nil
	}

	descriptions := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return "Error: each description must be a string", nil
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
		return "Error: failed to update todo list", nil
	}
	return "Todo list Created successfully", nil
}

func UpdateTodo(ctx ToolContext, args map[string]any) (string, error) {
	index, ok := args["index"].(int)
	if !ok {
		return "Error: index is required and must be an integer", nil
	}
	completed, ok := args["completed"].(bool)
	if !ok {
		return "Error: completed is required and must be a boolean", nil
	}
	database, _ := db.Connect()
	var session models.Session
	database.Model(&models.Session{}).Where("id = ?", ctx.SessionID).First(&session)
	todos := models.DecodeToDoList(session.ToDoList)
	todos[index] = models.ToDo{Index: index, Description: todos[index].Description, Completed: completed}
	result := database.Model(&models.Session{}).Where("id = ?", ctx.SessionID).Update("to_do_list", models.EncodeToDoList(todos))
	if result.Error != nil {
		return "Error: failed to update todo list", nil
	}
	return "Todo updated successfully: " + models.EncodeToDoList(todos), nil
}

func init() {
	Register("create_todo", ToolDef{
		Name:        "create_todo",
		Description: "Create a todo list by providing a list of string descriptions in order",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"descriptions": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":        "string",
						"description": "The description of the todo",
					},
				},
			},
			"required": []string{"descriptions"},
		},
	}, CreateTodo)
	Register("update_todo", ToolDef{
		Name:        "update_todo",
		Description: "Update a todo by providing an index and a description",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"index": map[string]any{
					"type":        "number",
					"description": "The index of the todo to update",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "The description of the todo",
				},
				"completed": map[string]any{
					"type":        "boolean",
					"description": "The completed status of the todo",
				},
			},
			"required": []string{"index", "description", "completed"},
		},
	}, UpdateTodo)
}
