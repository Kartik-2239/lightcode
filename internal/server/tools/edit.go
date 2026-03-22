package tools

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	Register("edit", ToolDef{
		Name:        "edit",
		Description: "Perform exact string replacements in existing files",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "The path to the file to edit",
				},
				"oldString": map[string]any{
					"type":        "string",
					"description": "The string to find and replace",
				},
				"newString": map[string]any{
					"type":        "string",
					"description": "The replacement string",
				},
				"replaceAll": map[string]any{
					"type":        "integer",
					"description": "Replace all occurrences of the old string. 0 for first occurrence, 1 for all occurrences",
				},
			},
			"required": []string{"filePath", "oldString", "newString", "replaceAll"},
		},
	}, func(ctx ToolContext, args map[string]any) (string, error) {
		path, ok := args["filePath"].(string)
		if !ok {
			return "", nil
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(ctx.WorkingDirectory, path)
		}
		oldString, ok := args["oldString"].(string)
		if !ok {
			return "", nil
		}
		newString, ok := args["newString"].(string)
		if !ok {
			return "", nil
		}

		var n int
		// JSON numbers are decoded as float64 into map[string]any
		if val, ok := args["replaceAll"].(float64); ok {
			if val == 1 {
				n = -1 // All occurrences
			} else {
				n = 1 // First occurrence
			}
		} else if val, ok := args["replaceAll"].(int); ok {
			if val == 1 {
				n = -1
			} else {
				n = 1
			}
		} else {
			return "", nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		content := string(data)

		// Check if oldString exists
		if !strings.Contains(content, oldString) {
			return "Old string not found in file", nil
		}

		newContent := strings.Replace(content, oldString, newString, n)
		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return "", err
		}

		return strings.Join([]string{oldString, newString}, "\n"), nil
	})
}
