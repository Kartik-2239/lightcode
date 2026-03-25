package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Grep(ctx ToolContext, args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern is required and must be a string")
	}

	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required and must be a string")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.WorkingDirectory, path)
	}

	include, ok := args["include"].(string)
	if !ok {
		return "ERROR: include is required and must be a string", fmt.Errorf("include is required and must be a string")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "ERROR: path does not exist: " + path, fmt.Errorf("path does not exist: %s", path)
	}

	cmd := exec.Command("grep", "-r", "-l", "--include="+include, pattern, path)
	cmd.Dir = ctx.WorkingDirectory
	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return "No matches found", nil
			}
			return "ERROR: grep error: " + string(output), fmt.Errorf("grep error: %s", string(output))
		}
		return "ERROR: failed to execute grep: " + err.Error(), fmt.Errorf("failed to execute grep: %w", err)
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return "No matches found", nil
	}

	return result, nil
}

func init() {
	Register("grep", ToolDef{
		Name:        "grep",
		Description: "Search for a pattern in a file or directory",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]string{
					"type":        "string",
					"description": "The pattern to search for",
				},
				"path": map[string]string{
					"type":        "string",
					"description": "The path to search in",
				},
				"include": map[string]string{
					"type":        "string",
					"description": "The file extension to include",
					"default":     "*.go",
				},
			},
			"required": []string{"pattern", "path"},
		},
	}, Grep)
}
