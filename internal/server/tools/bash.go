package tools

import (
	"os/exec"
	"strings"
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
	}, func(ctx ToolContext, args map[string]any) (string, error) {
		command, ok := args["command"].(string)
		if !ok {
			return "Error: command is required and must be a string", nil
		}
		parts := strings.Split(command, " ")
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = ctx.WorkingDirectory
		output, err := cmd.Output()
		if err != nil {
			return "Error: " + err.Error(), err
		}
		return string(output), nil
	})
}
