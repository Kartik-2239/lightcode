package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/tools"
)

// expandMentions appends synthetic tool output for each @mentioned path so file
// contents (or explicit errors) are baked into the stored user message.
func expandMentions(content string, mentions []string, sessionID, workingDir string) string {
	if len(mentions) == 0 {
		return content
	}
	dir := workingDir
	if dir == "" {
		dir = "."
	}
	toolCtx := tools.ToolContext{WorkingDirectory: dir, SessionID: sessionID}
	seen := map[string]bool{}
	for _, p := range mentions {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		toolName := "read_file"
		if mentionIsDir(toolCtx, p) {
			toolName = "list_dir"
		}
		res, err := tools.Execute(toolName, toolCtx, map[string]any{"path": p})
		content += formatMentionToolOutput(toolName, p, res, err)
	}
	return content
}

func formatMentionToolOutput(toolName, path string, res tools.ToolResponse, err error) string {
	return fmt.Sprintf("\n\nTool %q (call_id=mention-%s) output:\n%s", toolName, path, mentionToolOutput(res, err))
}

// mentionToolOutput returns tool result text, always surfacing failures to the model.
func mentionToolOutput(res tools.ToolResponse, err error) string {
	if strings.HasPrefix(res.Content, "Error:") {
		return res.Content
	}
	if err != nil {
		if res.Content != "" {
			return res.Content + "\nError: " + err.Error()
		}
		return "Error: " + err.Error()
	}
	return res.Content
}

// mentionIsDir reports whether p resolves to a directory inside the workspace.
func mentionIsDir(ctx tools.ToolContext, p string) bool {
	resolved, err := tools.ValidatePath(ctx, p)
	if err != nil {
		return false
	}
	info, err := os.Stat(resolved)
	return err == nil && info.IsDir()
}
