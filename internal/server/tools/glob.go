package tools

import (
	"fmt"
	"os/exec"
)

func init() {
	Register("glob", ToolDef{
		Name:        "glob",
		Description: "Fast file pattern matching tool that works with any codebase size",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]string{
					"type":        "string",
					"description": "The pattern to match",
				},
				"path": map[string]string{
					"type":        "string",
					"description": "The path to search in",
				},
			},
			"required": []string{"pattern", "path"},
		},
	}, func(args map[string]any) (string, error) {
		pattern, ok := args["pattern"].(string)
		if !ok {
			return "", nil
		}
		path, ok := args["path"].(string)
		if !ok {
			return "", nil
		}
		cmd, err := exec.Command("find", path, "-name", fmt.Sprintf("%s", pattern)).Output()
		if err != nil {
			return "No matches found", err
		}
		return string(cmd), nil
	})
}
