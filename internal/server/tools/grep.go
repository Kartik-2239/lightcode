package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Grep recursively searches for a pattern in files matching the given include filter
func Grep(args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern is required and must be a string")
	}

	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required and must be a string")
	}

	include, ok := args["include"].(string)
	if !ok {
		include = "*.go"
	}

	// Validate that path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", path)
	}

	// Use -l flag to only output filenames, avoiding exit code issues when matches are found
	cmd := exec.Command("grep", "-r", "-l", "--include="+include, pattern, path)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	// grep returns exit code 1 when no matches are found, which is not an error
	// grep returns exit code 2 when there's an error (e.g., invalid pattern)
	if err != nil {
		// Check if it's just "no matches found" (exit code 1)
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				// No matches found - this is not an error
				return "No matches found", nil
			}
			// Exit code 2 - actual error
			return "", fmt.Errorf("grep error: %s", string(output))
		}
		// Some other error occurred
		return "", fmt.Errorf("failed to execute grep: %w", err)
	}

	// grep found matches (exit code 0)
	result := strings.TrimSpace(string(output))
	if result == "" {
		return "No matches found", nil
	}

	return result, nil
}
