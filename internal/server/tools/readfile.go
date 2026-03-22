package tools

import (
	"os"
	"strings"
)

func init() {
	Register("read_file", ToolDef{
		Name:        "read_file",
		Description: "RETURNS THE CONTENTS OF A FILE FROM THE FILESYSTEM, don't keep calling it again and again like a fucking idiot",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]string{
					"type":        "string",
					"description": "The path to the file to read",
				},
			},
			"required": []string{"path"},
		},
	}, func(args map[string]any) (string, error) {
		path, ok := args["path"].(string)
		if !ok {
			return "", nil
		}
		gitignore, err := os.ReadFile(".gitignore")
		if err == nil {
			files_to_ignore := strings.Split(string(gitignore), "\n")
			for _, file := range files_to_ignore {
				if strings.HasSuffix(path, file) {
					return "Error: File is in .gitignore", nil
				}
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	})
}
