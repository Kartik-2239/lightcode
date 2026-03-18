package tools

import "os/exec"

func init() {
	Register("grep", ToolDef{
		Name:        "grep",
		Description: "Fast content search tool that searches file contents using regular expressions",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]string{
					"type":        "string",
					"description": "The pattern to search for in the files",
				},
				"include": map[string]string{
					"type":        "string",
					"description": "The type of file to search in (e.g. '*.md', '*.go', '*.txt')",
				},
				"path": map[string]string{
					"type":        "string",
					"description": "The path of Folder or file to search in",
				},
			},
			"required": []string{"pattern"},
		},
	}, func(args map[string]any) (string, error) {
		pattern, ok := args["pattern"].(string)
		if !ok {
			return "", nil
		}
		path, ok := args["path"].(string)
		if !ok {
			path = "."
		}
		include, ok := args["include"].(string)
		if !ok {
			include = "*"
		}
		cmd, err := exec.Command("grep", "-r", pattern, path, "--include="+include).Output()
		if err != nil {
			return "", err
		}
		return string(cmd), nil
	})
}
