package views

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
)

func withoutEphemeralTodoStatus(msgs []models.Message) []models.Message {
	var out []models.Message
	for _, msg := range msgs {
		if models.DecodeMessageData(msg.Data).Role == "todo_status" {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func renderTodoStatusBlock(todos []models.ToDo, width int) string {
	title := styleTodoTitle.Render("Todo list")
	boxStyle := styleTodoBox
	if width > 8 {
		boxStyle = boxStyle.MaxWidth(width - 4)
	}
	var inner strings.Builder
	if len(todos) == 0 {
		inner.WriteString(styleTodoEmpty.Render("No tasks in this session."))
	} else {
		for _, t := range todos {
			mark := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[ ] ")
			line := styleTodoOpen.Render(t.Description)
			if t.Completed {
				mark = lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Render("[x] ")
				line = styleTodoDone.Render(t.Description)
			}
			_, _ = inner.WriteString(mark + line + "\n")
		}
	}
	boxed := boxStyle.Render(strings.TrimSuffix(inner.String(), "\n"))
	return title + "\n" + boxed
}

func formatToolCall(tc models.StoredToolCall, width int, border bool) string {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(tc.Arguments), &args)
	if err != nil {
		return styleToolName.Render(tc.Name + "()")
	}

	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		cur := fmt.Sprintf("%v", args[key])
		if filepath.IsAbs(cur) {
			home, _ := os.UserHomeDir()
			cur = strings.Replace(cur, home, "~", 1)
		}
		cur = strings.TrimSpace(cur)
		if key == "path" || key == "filePath" || key == "command" {
			values = append(values, cur)
		} else {
			values = append(values, fmt.Sprintf("%s=%s", key, cur))
		}
	}

	if border {
		return styleToolName.Border(lipgloss.Border{
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
		}).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
	}
	return styleToolName.Border(lipgloss.RoundedBorder()).Width(width).Render(tc.Name + "(" + strings.Join(values, ", ") + ")")
}

func formatToolResult(content string, codeChanges []string, width int, tc models.StoredToolCall) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return styleResultText.Render("(no output)")
	}
	if strings.Contains(tc.Name, "todo") {
		return renderTodoStatusBlock(models.DecodeToDoList(content), width)
	}
	if len(codeChanges) == 0 {
		lines := strings.Split(content, "\n")
		home, _ := os.UserHomeDir()
		var content string
		if len(lines) >= 8 {
			req := append(lines[:8], fmt.Sprintf("...%d more lines", len(lines)-8))
			content = strings.Replace(strings.Join(req, "\n"), home, "~", 0)
		} else {
			content = strings.Replace(strings.Join(lines, "\n"), home, "~", 0)
		}
		return styleToolResult.Border(lipgloss.RoundedBorder()).BorderTop(false).Width(width).PaddingLeft(1).Render(content)
	}

	var sb strings.Builder
	var oldlines []string
	var newlines []string
	oldlines = strings.Split(codeChanges[0], "\n")
	newlines = strings.Split(codeChanges[1], "\n")

	top := lipgloss.NewStyle().Foreground(lipgloss.Red).Render(fmt.Sprintf("-%d", len(oldlines))) + " " + lipgloss.NewStyle().Foreground(lipgloss.Green).Render(fmt.Sprintf("+%d", len(newlines))) + "\n"
	// sb.WriteString()

	if tc.Name == "write_file" {
		line_limit := 20
		if len(newlines) > line_limit {
			newlines = newlines[:line_limit]
			newlines = append(newlines, fmt.Sprintf("...\n%d more lines", len(newlines)))
		}
		if len(oldlines) > line_limit {
			oldlines = oldlines[:line_limit]
			oldlines = append(oldlines, fmt.Sprintf("...\n%d more lines", len(oldlines)))
		}
	}

	for i, line := range oldlines {
		if i == len(oldlines)-1 {
			sb.WriteString(styleRemoved.Foreground(lipgloss.BrightBlack).Width(width).Render("- " + line))
		} else {
			sb.WriteString(styleRemoved.Width(width).Render("- " + line))
		}
		sb.WriteString("\n")
	}
	for i, line := range newlines {
		if i == len(newlines) {
			sb.WriteString(styleAdded.Foreground(lipgloss.BrightBlack).Width(width).Render("+ " + line))
		} else {
			sb.WriteString(styleAdded.Width(width).Render("+ " + line))
		}
		sb.WriteString("\n")

	}
	return top + lipgloss.NewStyle().Margin(2).MarginTop(0).MarginBottom(0).Render(sb.String())
}

func formatTool(tc models.StoredToolCall, width int, content string, codeChanges []string) string {

	if should_print_tool_result(tc.Name) {
		result := formatToolCall(tc, width, true)
		resultSummary := formatToolResult(content, codeChanges, width, tc)
		if resultSummary != "" {
			result += "\n" + resultSummary
		}
		result += "\n"
		return result
	}
	return formatToolCall(tc, width, false) + "\n"

}

var lightcodeGlamourStyle = []byte(`{
  "document": {
    "block_prefix": "",
    "block_suffix": "",
    "color": "252",
    "margin": 0
  },
  "block_quote": {
    "indent": 1,
    "indent_token": "│ ",
    "color": "243",
    "italic": true
  },
  "paragraph": {},
  "list": {
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "bold": true
  },
  "h1": {
    "prefix": " ",
    "suffix": " ",
    "color": "232",
    "background_color": "43",
    "bold": true
  },
  "h2": {
    "prefix": "▌ ",
    "color": "86",
    "bold": true
  },
  "h3": {
    "prefix": "◆ ",
    "color": "43",
    "bold": true
  },
  "h4": {
    "prefix": "◇ ",
    "color": "6",
    "bold": false
  },
  "h5": {
    "prefix": "· ",
    "color": "244"
  },
  "h6": {
    "prefix": "· ",
    "color": "241"
  },
  "text": {},
  "strikethrough": { "crossed_out": true },
  "emph": { "italic": true, "color": "8" },
  "strong": { "bold": true, "color": "#41f7fa" },
  "hr": {
    "color": "8",
    "format": "\n──────────────────────────────────────\n"
  },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "link": { "color": "#41f7fa", "underline": true },
  "link_text": { "color": "43", "bold": true },
  "image": { "color": "86", "underline": true },
  "image_text": { "color": "8", "format": "Image: {{.text}} →" },
  "code": {
    "prefix": " ",
    "suffix": " ",
    "color": "#96cfd8",
    "background_color": "#212121"
  },
  "code_block": {
    "color": "252",
    "margin": 2,
    "chroma": {
      "text":                { "color": "#C4C4C4" },
      "error":               { "color": "#F1F1F1", "background_color": "#F05B5B" },
      "comment":             { "color": "#606060" },
      "comment_preproc":     { "color": "#FF875F" },
      "keyword":             { "color": "#41f7fa" },
      "keyword_reserved":    { "color": "#FF5FD2" },
      "keyword_namespace":   { "color": "#FF5F87" },
      "keyword_type":        { "color": "#86D0D0" },
      "operator":            { "color": "#8BE28B" },
      "punctuation":         { "color": "#C8C8A0" },
      "name":                { "color": "#C4C4C4" },
      "name_builtin":        { "color": "#FF8EC7" },
      "name_tag":            { "color": "#86D0D0" },
      "name_attribute":      { "color": "#7A7AE6" },
      "name_class":          { "color": "#F1F1F1", "underline": true, "bold": true },
      "name_constant":       {},
      "name_decorator":      { "color": "#FFFF87" },
      "name_exception":      {},
      "name_function":       { "color": "#41f7fa" },
      "name_other":          {},
      "literal_number":      { "color": "#6EEFC0" },
      "literal_string":      { "color": "#C69669" },
      "literal_string_escape": { "color": "#AFFFD7" },
      "generic_deleted":     { "color": "#FD5B5B" },
      "generic_emph":        { "italic": true },
      "generic_inserted":    { "color": "#41f7fa" },
      "generic_strong":      { "bold": true },
      "generic_subheading":  { "color": "#888888" },
      "background":          { "background_color": "#1e1e1e" }
    }
  },
  "table": {},
  "definition_list": {},
  "definition_term": {},
  "definition_description": { "block_prefix": "\n  → " },
  "html_block": {},
  "html_span": {}
}`)

func renderMessages(msgs []models.Message, width int) string {
	if width <= 0 {
		width = 80
	}
	r, _ := glamour.NewTermRenderer(glamour.WithWordWrap(width), glamour.WithStylesFromJSONBytes(lightcodeGlamourStyle))

	// tree := styleTree.Render("└─")

	type callKey struct {
		msgID string
		idx   int
	}
	hasResult := make(map[callKey]bool)
	var lastAssistantMsgID string
	var callIdx int
	for _, msg := range msgs {
		d := models.DecodeMessageData(msg.Data)
		if d.Role == "assistant" {
			lastAssistantMsgID = fmt.Sprintf("%d", msg.ID)
			callIdx = 0
		} else if d.Role == "tool_call" && lastAssistantMsgID != "" {
			hasResult[callKey{lastAssistantMsgID, callIdx}] = true
			callIdx++
		}
	}

	var lines []string
	lines = append(lines, mascot())
	for _, msg := range msgs {
		d := models.DecodeMessageData(msg.Data)
		if d.Role == "" || d.Role == "error" {
			continue
		}

		if d.Content != "" {
			content := d.Content
			switch d.Role {
			case "assistant":
				content = html.UnescapeString(content)
				re := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
				matches := re.FindAllString(content, -1)
				matchedContent := strings.Join(matches, " ")
				content = content[len(strings.Join(matches, " ")):]

				if out, err := r.Render(content); err == nil {
					content = strings.TrimSpace(out)
				}
				if matchedContent != "" {
					matchedContent = strings.TrimSpace(matchedContent)
					matchedContent = strings.ReplaceAll(matchedContent, "\n", "")
					matchedContent = strings.Replace(matchedContent, "<think>", "", 1)
					matchedContent = strings.Replace(matchedContent, "</think>", "", 1)
					if len(matchedContent) > width {
						matchedContent = "Thinking: " + matchedContent
						formattedData := styleThink.PaddingLeft(1).Width(width-1).Render(matchedContent[:width-1]) + styleThink.Width(width-2).PaddingLeft(2).Width(width).Render(matchedContent[width-1:])
						lines = append(lines, formattedData)
					} else {
						lines = append(lines, styleThink.Width(width).Render(matchedContent))
					}
				}
				if content != "" {
					if !strings.HasPrefix(content, "<memory>") && !strings.HasSuffix(content, "</memory>") {
						lines = append(lines, styleThink.Width(width).Render(content))
					}

				}

			case "tool_call":
				for _, toolcall := range d.ToolCalls {
					lines = append(lines, formatTool(toolcall, width, content, d.CodeChanges))
				}

			case "user":
				if !strings.HasPrefix(content, "<memory>") && !strings.HasSuffix(content, "</memory>") {

					lines = append(lines, styleUser.Width(width).Render(" "+content))
				}
			case "question":
				qs := parseQuestions(content)
				for _, q := range qs {
					line := styleQuestionHeader.Render("? " + q.question)
					if len(q.options) > 0 {
						line += " [" + strings.Join(q.options, ", ") + "]"
					}
					lines = append(lines, line)
				}
			case "todo_status":
				todos := models.DecodeToDoList(content)
				lines = append(lines, renderTodoStatusBlock(todos, width))
			}
		} else if d.Role == "assistant" && len(d.ToolCalls) > 0 {
			for i, tc := range d.ToolCalls {
				if !hasResult[callKey{fmt.Sprintf("%d", msg.ID), i}] {
					lines = append(lines, formatToolCall(tc, width, false))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func mascot() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#41f7fa")).Render(`
  ▐█████▌
  █  █  █
 ▘▜█████▛▘▘
   ▘▘ ▝▝`)
}

// func mascot2() string {
// 	return strings.TrimSpace(`
// [38;2;66;195;255m    ███[0m
// [38;2;60;208;242m      ██[0m
// [38;2;54;220;228m      ██[0m
// [38;2;48;230;214m      ██[0m
// [38;2;42;239;201m      ██[0m
// [38;2;36;245;189m      ██[0m
// [38;2;30;249;176m      ██     ██[0m
// [38;2;26;252;164m       ██   ██[0m
// [38;2;24;252;154m        █████[0m
// `)
// }
