package tools

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func ListDir(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	path_, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string", Printable: "Error: path is required and must be a string"}, nil
	}
	resolved, err := ValidatePath(ctx, path_)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, nil
	}
	path_ = resolved
	entries, err := os.ReadDir(path_)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}

	var result string
	for _, e := range entries {
		v, _ := e.Info()
		if e.IsDir() {
			result += fmt.Sprintf("dir '%s/' %v bytes\n", e.Name(), v.Size())
		} else {
			data, err := os.ReadFile(path.Join(path_, e.Name()))
			if err == nil {
				lines := len(strings.Split(string(data), "\n"))
				result += fmt.Sprintf("file: '%s' | size: %v kb | lines: %d\n", e.Name(), v.Size()/1000, lines)
			}
		}
	}
	return ToolResponse{Content: result, Printable: result}, nil
}

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
	}, ListDir)
}
