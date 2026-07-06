package views

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

var toolSpecialBorder = lipgloss.Border{
	Top:          "─",
	Bottom:       "─",
	Left:         "│",
	Right:        "│",
	TopLeft:      "╭",
	TopRight:     "╮",
	BottomLeft:   "├",
	BottomRight:  "┤",
	MiddleLeft:   "├",
	MiddleRight:  "┤",
	Middle:       "┼",
	MiddleTop:    "┬",
	MiddleBottom: "┴",
}

func parseToolArgs(tc models.StoredToolCall) (map[string]any, error) {
	var args map[string]any
	err := json.Unmarshal([]byte(tc.Arguments), &args)
	return args, err
}

func sortedArgKeys(args map[string]any, keep func(string) bool) []string {
	keys := make([]string, 0, len(args))
	for k := range args {
		if keep != nil && !keep(k) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatToolArg(key string, val any) string {
	cur := fmt.Sprintf("%v", val)
	if filepath.IsAbs(cur) {
		home, _ := os.UserHomeDir()
		cur = strings.Replace(cur, home, "~", 1)
	}
	cur = strings.TrimSpace(cur)
	if key == "path" || key == "filePath" || key == "command" {
		return cur
	}
	return key + "=" + cur
}

func summarizeContent(content string, width int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return styleResultText.Render("(no output)")
	}
	lines := strings.Split(content, "\n")
	home, _ := os.UserHomeDir()
	var out string
	if len(lines) >= 8 {
		req := append(lines[:8], fmt.Sprintf("...%d more lines", len(lines)-8))
		out = strings.Replace(strings.Join(req, "\n"), home, "~", 0)
	} else {
		out = strings.Replace(strings.Join(lines, "\n"), home, "~", 0)
	}
	return styleToolResult.Border(lipgloss.RoundedBorder()).BorderTop(false).Width(width).PaddingLeft(1).Render(out)
}

func renderDiff(oldStr, newStr string, width, lineLimit int, tc models.StoredToolCall) string {
	toolArgs, err := parseToolArgs(tc)
	filepath := ""
	if err == nil {
		filepath = fmt.Sprintf("%v", toolArgs["path"])
		if filepath == "<nil>" {
			filepath = fmt.Sprintf("%v", toolArgs["filePath"])
		}
	}
	if filepath == "<nil>" {
		filepath = ""
	} else {
		filepath = shortenDir(filepath)
	}
	oldlines := strings.Split(oldStr, "\n")
	if len(oldlines) == 1 && oldlines[0] == "" {
		oldlines = []string{}
	}
	newlines := strings.Split(newStr, "\n")
	if len(newlines) == 1 && newlines[0] == "" {
		newlines = []string{}
	}
	if lineLimit > 0 {
		if len(newlines) > lineLimit {
			newlines = newlines[:lineLimit]
			newlines = append(newlines, fmt.Sprintf("\n...%d more lines", len(newlines)))
		}
		if len(oldlines) > lineLimit {
			oldlines = oldlines[:lineLimit]
			oldlines = append(oldlines, fmt.Sprintf("\n...%d more lines", len(oldlines)))
		}
	}
	var removed strings.Builder
	var added strings.Builder
	if len(oldlines) != 1 {
		for _, line := range oldlines {
			removed.WriteString("- ")
			removed.WriteString(line)
			removed.WriteString("\n")
		}
	}
	if len(newlines) != 0 {
		for _, line := range newlines {
			added.WriteString("+ ")
			added.WriteString(line)
			added.WriteString("\n")
		}
	}
	final := ""
	if removed.Len() > 0 {
		if added.Len() > 0 {
			top := fmt.Sprintf("# edited file `%s`\n\n", filepath)
			final += styleRemoved.Width(width).BorderBottom(false).Render(top + strings.TrimSpace(removed.String()))
		} else {
			final += styleRemoved.Width(width).Render(strings.TrimSpace(removed.String()))
		}
	}
	if added.Len() > 0 {
		if removed.Len() > 0 {
			final += styleAdded.Width(width).BorderTop(false).Render(strings.TrimSpace(added.String()))
		} else {
			top := fmt.Sprintf("# edited file `%s`\n\n", filepath)
			final += styleAdded.Width(width).Render(top + strings.TrimSpace(added.String()))
		}
	}
	return final
}

func renderToolCall(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	switch tc.Name {
	case "bash":
		return renderBashTool(tc, width, content, codeChanges)
	case "edit":
		return renderEditTool(tc, width, content, codeChanges)
	case "write_file":
		return renderWriteFileTool(tc, width, content, codeChanges)
	case "read_file":
		return renderReadFileTool(tc, width, content, codeChanges)
	case "list_dir":
		return renderListDirTool(tc, width, content, codeChanges)
	case "glob":
		return renderGlobTool(tc, width, content, codeChanges)
	case "grep":
		return renderGrepTool(tc, width, content, codeChanges)
	case "skill":
		return renderSkillTool(tc, width, content, codeChanges)
	case "webfetch":
		return renderWebFetchTool(tc, width, content, codeChanges)
	case "question":
		return ""
	default:
		return renderGenericTool(tc, width, content, codeChanges)
	}
}

func renderBashTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	args, err := parseToolArgs(tc)
	var callOut string
	if err != nil {
		callOut = styleToolName.Render(tc.Name + "()")
	} else {
		keys := sortedArgKeys(args, nil)
		values := make([]string, 0, len(keys))
		for _, k := range keys {
			values = append(values, formatToolArg(k, args[k]))
		}
		callOut = styleToolName.Border(toolSpecialBorder).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
	}
	result := callOut
	if summary := summarizeContent(content, width); summary != "" {
		result += "\n" + summary
	}
	result += "\n"
	return result
}

func renderEditTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	return renderDiff(codeChanges[0], codeChanges[1], width, 4, tc)
}

func renderWriteFileTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	if len(codeChanges) < 2 {
		return summarizeContent(content, width)
	}
	return renderDiff(codeChanges[0], codeChanges[1], width, 20, tc)
}

func renderReadFileTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	keys := sortedArgKeys(args, func(k string) bool {
		return k == "filePath" || k == "path"
	})
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	return styleToolName.Border(lipgloss.RoundedBorder()).Width(width).Render(tc.Name+"("+strings.Join(values, ", ")+")") + "\n"
}

func renderListDirTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	result := ""
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	keys := sortedArgKeys(args, nil)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	result += styleToolName.Border(toolSpecialBorder).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
	if summary := summarizeContent(content, width); summary != "" {
		result += "\n" + summary
	}
	result += "\n"
	return result
}

func renderGlobTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	result := ""
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	keys := sortedArgKeys(args, nil)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	result += styleToolName.Border(toolSpecialBorder).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
	if summary := summarizeContent(content, width); summary != "" {
		result += "\n" + summary
	}
	result += "\n"
	return result
}

func renderGrepTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	result := ""
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	keys := sortedArgKeys(args, nil)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	result += styleToolName.Border(toolSpecialBorder).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
	if summary := summarizeContent(content, width); summary != "" {
		result += "\n" + summary
	}
	result += "\n"
	return result
}

func renderSkillTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	keys := sortedArgKeys(args, nil)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	return styleToolName.Border(lipgloss.RoundedBorder()).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
}

func renderWebFetchTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	keys := sortedArgKeys(args, nil)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	return styleToolName.Border(lipgloss.RoundedBorder()).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
}

func renderGenericTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	args, err := parseToolArgs(tc)
	var callOut string
	if err != nil {
		callOut = styleToolName.Render(tc.Name + "()")
	} else {
		keys := sortedArgKeys(args, nil)
		values := make([]string, 0, len(keys))
		for _, k := range keys {
			values = append(values, formatToolArg(k, args[k]))
		}
		callOut = styleToolName.Border(toolSpecialBorder).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
	}
	out := callOut
	if len(codeChanges) >= 2 {
		if diff := renderDiff(codeChanges[0], codeChanges[1], width, 0, tc); diff != "" {
			out += "\n" + diff
		}
	} else {
		if summary := summarizeContent(content, width); summary != "" {
			out += "\n" + summary
		}
	}
	out += "\n"
	return out
}

func formatTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {
	if tc.Name == "question" {
		return ""
	}
	return renderToolCall(tc, width, content, codeChanges)
}

func formatToolCall(tc models.StoredToolCall, width int, border bool) string {
	args, err := parseToolArgs(tc)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}
	var keep func(string) bool
	if tc.Name == "edit" || tc.Name == "write_file" || tc.Name == "read_file" {
		keep = func(k string) bool { return k == "filePath" || k == "path" }
	}
	keys := sortedArgKeys(args, keep)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, formatToolArg(k, args[k]))
	}
	borderStyle := lipgloss.RoundedBorder()
	if border && tc.Name != "edit" && tc.Name != "write_file" && tc.Name != "read_file" {
		borderStyle = toolSpecialBorder
	}
	return styleToolName.Border(borderStyle).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
}
