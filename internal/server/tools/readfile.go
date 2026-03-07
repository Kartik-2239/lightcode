package tools

import (
	"os"
)

func init() {
	Register("read_file", ToolDef{
		Name:        "read_file",
		Description: "Read the contents of a file from the filesystem",
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

		// strings.HasSuffix(path, "")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	})
}
