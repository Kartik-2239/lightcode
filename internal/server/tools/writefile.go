package tools

import (
	"os"
	"path/filepath"
)

func init() {
	Register("write_file", ToolDef{
		Name:        "write_file",
		Description: "Write content to a file, creating it if it doesn't exist",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]string{
					"type":        "string",
					"description": "The path to the file to write",
				},
				"content": map[string]string{
					"type":        "string",
					"description": "The content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}, func(ctx ToolContext, args map[string]any) (string, error) {
		path, ok := args["path"].(string)
		if !ok {
			return "", nil
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(ctx.WorkingDirectory, path)
		}
		content, ok := args["content"].(string)
		if !ok {
			return "", nil
		}

		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}

		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return "", err
		}
		return "File written successfully", nil
	})
}
