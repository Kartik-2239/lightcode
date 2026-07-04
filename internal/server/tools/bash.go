package tools

import (
	"os/exec"
)

func Bash(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	command, ok := args["command"].(string)
	if !ok {
		return ToolResponse{Content: "Error: command is required and must be a string"}, nil
	}
	// parts := strings.Split(command, " ")
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = ctx.WorkingDirectory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error() + "\n" + string(output)}, nil
	}
	return ToolResponse{Content: string(output), Printable: bash_response_formatter(string(output))}, nil
}

func init() {
	Prompt := `Execute bash commands for various tasks such as locating/moving/copying files and many more. 
# Rules to follow
- Use ls on directories before creating/copying/moving files
- NEVER GUESS locations of files, always get the exact path and always check the contents of a folder before any file manipulation commands.
- Always use the proper bash syntax
- Follow the file structure of the codebase
	`

	Register("bash", ToolDef{
		Name:        "bash",
		Description: Prompt,
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
	}, Bash)

}

func bash_response_formatter(output string) string {
	return "printable" + output
}
