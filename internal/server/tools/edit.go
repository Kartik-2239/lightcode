package tools

import (
	"os"
	"strings"
)

func Edit(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	path, ok := args["filePath"].(string)
	if !ok {
		return ToolResponse{Content: "Error: filePath is required and must be a string", Printable: "Error: filePath is required and must be a string"}, nil
	}
	resolved, err := ValidatePath(ctx, path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, nil
	}
	path = resolved
	oldString, ok := args["oldString"].(string)
	if !ok {
		return ToolResponse{Content: "Error: oldString is required and must be a string", Printable: "Error: oldString is required and must be a string"}, nil
	}
	newString, ok := args["newString"].(string)
	if !ok {
		return ToolResponse{Content: "Error: newString is required and must be a string", Printable: "Error: newString is required and must be a string"}, nil
	}
	copyNewString := newString

	var n int
	// JSON numbers are decoded as float64 into map[string]any
	if val, ok := args["replaceAll"].(float64); ok {
		if val == 1 {
			n = -1 // All occurrences
		} else {
			n = 1 // First occurrence
		}
	} else if val, ok := args["replaceAll"].(int); ok {
		if val == 1 {
			n = -1
		} else {
			n = 1
		}
	} else {
		return ToolResponse{Content: "Error: replaceAll is required and must be an integer", Printable: "Error: replaceAll is required and must be an integer"}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}
	content := string(data)

	// Check if oldString exists
	if !strings.Contains(content, oldString) {
		return ToolResponse{Content: "Old string not found in file", Printable: "Old string not found in file"}, nil
	}

	newContent := strings.Replace(content, oldString, newString, n)
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}
	// new_old, new_new := string_differentiator(oldString, copyNewString)
	msg := strings.Join([]string{"file_path: " + path, "old_string: ", oldString, "========", "new_string: ", copyNewString}, "\n")
	return ToolResponse{Content: msg, Printable: msg, CodeChanges: []string{"---" + oldString, "+++" + copyNewString}}, nil
}

func init() {
	Prompt := `Perform exact string replacements in existing files
- Read the file before using edit tool and provide the exact old string and filepath.
- DO NOT GUESS THE OLD STRING AND FILE PATH.
- Only use old strings that are changed don't generate TOO many tokens only input code that needs to be changed.`
	Register("edit", ToolDef{
		Name:        "edit",
		Description: Prompt,
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "The path to the file to edit in string",
				},
				"oldString": map[string]any{
					"type":        "string",
					"description": "The string to find and replace",
				},
				"newString": map[string]any{
					"type":        "string",
					"description": "The replacement string",
				},
				"replaceAll": map[string]any{
					"type":        "integer",
					"description": "Replace all occurrences of the old string. 0 for first occurrence, 1 for all occurrences",
				},
			},
			"required": []string{"filePath", "oldString", "newString", "replaceAll"},
		},
	}, Edit)
}

func string_differentiator(old_string string, new_string string) (string, string) { // old, new
	old_lines := strings.Split(old_string, "\n")
	new_lines := strings.Split(new_string, "\n")

	old_lines_reverse := reverseSlice(old_lines)
	new_lines_reverse := reverseSlice(new_lines)

	minLength := min(len(old_lines), len(new_lines))

	var starting_idx int
	var ending_idx int

	for i := range minLength {
		if old_lines[i] != new_lines[i] {
			starting_idx = i
		}
	}
	for i := range minLength {
		if old_lines_reverse[i] != new_lines_reverse[i] {
			ending_idx = i
		}
	}
	return strings.Join(old_lines[starting_idx:ending_idx], "\n"), strings.Join(new_lines[starting_idx:ending_idx], "\n")
}

func reverseSlice(a []string) []string {
	n := len(a)
	res := make([]string, n)

	for i := 0; i < n; i++ {
		res[i] = a[i]
	}
	return res
}
