package tools

import (
	"fmt"
	"os/exec"
	"path/filepath"
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
	}, func(ctx ToolContext, args map[string]any) (string, error) {
		pattern, ok := args["pattern"].(string)
		if !ok {
			return "", nil
		}
		path, ok := args["path"].(string)
		if !ok {
			return "", nil
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(ctx.WorkingDirectory, path)
		}
		cmd := exec.Command("find", path, "-name", fmt.Sprintf("%s", pattern))
		cmd.Dir = ctx.WorkingDirectory
		output, err := cmd.Output()
		if err != nil {
			return "No matches found", err
		}
		return string(output), nil
	})
}
