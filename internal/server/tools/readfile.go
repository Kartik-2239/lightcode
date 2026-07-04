package tools

import (
	"os"
	"strings"
)

func hideCreds(data string) string {
	safe_data := ""
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) < 2 {
			safe_data += line + "\n"
		}
		key, val := s[0], s[1]

		if len(val) <= 4 {
			safe_data += key + "=" + val + "\n"
			continue
		}

		masked := strings.Repeat("*", len(val)-4) + val[len(val)-4:]
		safe_data += key + "=" + masked + "\n"
	}
	return safe_data
}

func ReadFile(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	path, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string", Printable: "Error: path is required and must be a string"}, nil
	}
	offset, ok := args["offset"].(int)
	if !ok {
		offset = 1
	}
	limit, ok := args["limit"].(int)
	if !ok {
		limit = 1000
	}
	resolved, err := ValidatePath(ctx, path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, nil
	}
	path = resolved
	// gitignore, err := os.ReadFile(filepath.Join(ctx.WorkingDirectory, ".gitignore"))
	// if err == nil {
	// 	files_to_ignore := strings.Split(string(gitignore), "\n")
	// 	for _, file := range files_to_ignore {
	// 		if strings.HasSuffix(path, file) {
	// 			return ToolResponse{Content: "Error: File is in .gitignore"}, nil
	// 		}
	// 	}
	// }
	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}
	if strings.HasSuffix(path, ".env") {
		safeData := hideCreds(string(data))
		return ToolResponse{Content: safeData, Printable: safeData}, nil
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	lines = lines[offset-1:]
	if len(lines) < limit {
		limit = len(lines)
	}
	lines = lines[:limit]
	content = strings.Join(lines, "\n")
	return ToolResponse{Content: content, Printable: ""}, nil
}

func init() {
	Prompt := `Read tool for getting the content from a particular file`
	Register("read_file", ToolDef{
		Name:        "read_file",
		Description: Prompt,
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
