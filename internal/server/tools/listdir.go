package tools

import (
	"os"
	"path/filepath"
)

func init() {
	Register("list_dir", ToolDef{
		Name:        "list_dir",
		Description: "List files and directories in a given path",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]string{
					"type":        "string",
					"description": "The directory path to list",
				},
			},
			"required": []string{"path"},
		},
	}, func(ctx ToolContext, args map[string]any) (string, error) {
		path, ok := args["path"].(string)
		if !ok {
			return "", nil
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(ctx.WorkingDirectory, path)
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}

		var result string
		for _, e := range entries {
			if e.IsDir() {
				result += e.Name() + "/\n"
			} else {
				result += e.Name() + "\n"
			}
		}
		return result, nil
	})
}
