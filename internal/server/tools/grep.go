package tools

import (
	"os"
	"os/exec"
	"strings"
)

func Grep(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return ToolResponse{Content: "Error: pattern is required and must be a string", Printable: "Error: pattern is required and must be a string"}, nil
	}

	path, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string", Printable: "Error: path is required and must be a string"}, nil
	}
	resolved, err := ValidatePath(ctx, path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, nil
	}
	path = resolved

	include, ok := args["include"].(string)
	if !ok {
		return ToolResponse{Content: "Error: include is required and must be a string", Printable: "Error: include is required and must be a string"}, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ToolResponse{Content: "Error: path does not exist: " + path, Printable: "Error: path does not exist: " + path}, nil
	}

	cmd := exec.Command("grep", "-r", "-l", "--include="+include, pattern, path)
	cmd.Dir = ctx.WorkingDirectory
	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return ToolResponse{Content: "Error: No matches found", Printable: "Error: No matches found"}, nil
			}
			return ToolResponse{Content: "Error: grep error: " + string(output), Printable: "Error: grep error: " + string(output)}, nil
		}
		return ToolResponse{Content: "Error: failed to execute grep: " + err.Error(), Printable: "Error: failed to execute grep: " + err.Error()}, err
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return ToolResponse{Content: "Error: No matches found", Printable: "Error: No matches found"}, nil
	}

	return ToolResponse{Content: result, Printable: result}, nil
}

func init() {
	Prompt := `Search for a pattern in a file or directory
# Rules to follow
- Fast content search tool that works with any codebase size
- Searches file contents using regular expressions
- Supports full regex syntax (eg. "log.*Error", "function\s+\w+", etc.)
- Filter files by pattern with the include parameter (eg. "*.js", "*.{ts,tsx}")
- Returns file paths and line numbers with at least one match sorted by modification time
- Use this tool when you need to find files containing specific patterns
- If you need to identify/count the number of matches within files, use the Bash tool with 'rg' (ripgrep) directly. Do NOT use 'grep'.
- When you are doing an open-ended search that may require multiple rounds of globbing and grepping, use the Task tool instead
`
	Register("grep", ToolDef{
		Name:        "grep",
		Description: Prompt,
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
