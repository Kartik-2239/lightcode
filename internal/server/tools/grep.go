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
					"type": "string",
				},
				"include": map[string]string{
					"type": "string",
				},
				"path": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"pattern", "include", "path"},
		},
	}, func(args map[string]any) (string, error) {
		pattern, ok := args["pattern"].(string)
		if !ok {
			return "", nil
		}
		include, ok := args["include"].(string)
		if !ok {
			return "", nil
		}
		path, ok := args["path"].(string)
		if !ok {
			return "", nil
		}
		cmd, err := exec.Command("grep", "-r", pattern, include, path).Output()
		if err != nil {
			return "", err
		}
		return string(cmd), nil
	})
}
