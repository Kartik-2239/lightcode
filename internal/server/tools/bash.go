package tools

import (
	"os/exec"
)

func init() {
	Register("bash", ToolDef{
		Name:        "bash",
		Description: "Execute bash commands in a persistent shell session",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]string{
					"type":        "string",
					"description": "The command to execute",
				},
			},
			"required": []string{"command"},
		},
	}, func(args map[string]any) (string, error) {
		command, ok := args["command"].(string)
		if !ok {
			return "", nil
		}
		cmd, err := exec.Command(command).Output()
		if err != nil {
			return "", err
		}
		return string(string(cmd)), nil
	})
}
