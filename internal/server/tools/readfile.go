package tools

import (
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(ctx ToolContext, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", nil
	}
	offset, ok := args["offset"].(int)
	if !ok {
		offset = 1
	}
	limit, ok := args["limit"].(int)
	if !ok {
		limit = 1000
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.WorkingDirectory, path)
	}
	gitignore, err := os.ReadFile(filepath.Join(ctx.WorkingDirectory, ".gitignore"))
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
	content := string(data)
	lines := strings.Split(content, "\n")
	lines = lines[offset-1:]
	if len(lines) < limit {
		limit = len(lines)
	}
	lines = lines[:limit]
	content = strings.Join(lines, "\n")
	return content, nil
}

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
				"offset": map[string]any{
					"type":        "number",
					"description": "The offset to read from the file, starting line number",
					"default":     1,
				},
				"limit": map[string]any{
					"type":        "number",
					"description": "The number of lines to read from the file",
					"default":     1000,
				},
			},
			"required": []string{"path"},
		},
	}, ReadFile)
}
