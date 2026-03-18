package tools

import (
	"os"
	"strings"
)

func init() {
	Register("edit", ToolDef{
		Name:        "edit",
		Description: "Perform exact string replacements in existing files",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]string{
					"type":        "string",
					"description": "The path to the file to edit",
				},
				"oldString": map[string]string{
					"type":        "string",
					"description": "The string to find and replace",
				},
				"newString": map[string]string{
					"type":        "string",
					"description": "The replacement string",
				},
				"replaceAll": map[string]any{
					"type":        "boolean",
					"description": "Replace all occurrences of the old string",
				},
				"required": []string{"filePath", "oldString", "newString", "replaceAll"},
			},
		},
	}, func(args map[string]any) (string, error) {
		filePath, ok := args["filePath"].(string)
		if !ok {
			return "", nil
		}
		oldString, ok := args["oldString"].(string)
		if !ok {
			return "", nil
		}
		newString, ok := args["newString"].(string)
		if !ok {
			return "", nil
		}
		replaceAll, ok := args["replaceAll"].(int)
		if !ok {
			return "", nil
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		content := string(data)
		content = strings.Replace(content, oldString, newString, replaceAll)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			return "", err
		}
		return "File edited successfully", nil
	})
}
