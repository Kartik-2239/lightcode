package tools

import (
	"fmt"
	"os/exec"
)

func Glob(ctx ToolContext, args map[string]any) (ToolResponse, error) {
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
	cmd := exec.Command("find", path, "-name", fmt.Sprintf("%s", pattern))
	cmd.Dir = ctx.WorkingDirectory
	output, err := cmd.Output()
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}
	return ToolResponse{Content: string(output), Printable: string(output)}, nil
}

func init() {
	Prompt := `Fast file pattern matching tool that works with any codebase size
# Rules to follow :-
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns
- When you are doing an open-ended search that may require multiple rounds of globbing and grepping, use the Task tool instead
`
	Register("glob", ToolDef{
		Name:        "glob",
		Description: Prompt,
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
	}, Glob)
}
