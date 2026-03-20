package views

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
	"github.com/Kartik-2239/lightcode/internal/tui/components"
	"github.com/charmbracelet/x/term"
	"golang.design/x/clipboard"
)

func LauchHomePage() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}

type streamMessageMsg models.StoredMessageData
type streamDoneMsg struct{}

type model struct {
	viewport          viewport.Model
	islistSessionWin  bool
	islistCommandsWin bool
	listSession       components.Model
	listCommands      components.ModelCmdList
	sessions          []models.Session
	currentSession    models.Session
	messages          []models.Message
	textarea          textarea.Model
	pasteCounter      int
	pastedTexts       map[int]string
	senderStyle       lipgloss.Style
	err               error
	cache             map[int]string
	cacheIndex        int
	streamCh          chan models.StoredMessageData
	width             int
	height            int
	bashMode          bool
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "
	// ta.SetPromptFunc(2, func(info textarea.PromptInfo) string {
	// 	if info.LineNumber == 0 {
	// 		return "❯ "
	// 	}
	// 	return " "
	// })

	ta.CharLimit = 32000

	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()

	s.Focused.Base = lipgloss.NewStyle()

	width, height := 80, 24
	if w, h, err := term.GetSize(os.Stdout.Fd()); err == nil {
		width, height = w, h
	}

	ta.SetWidth(width)
	ta.SetHeight(2)

	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height-2))
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	ta.KeyMap.InsertNewline.SetEnabled(false)
	sessions := client.ListSession()
	sessionItems := make([]list.Item, len(sessions))
	for i, s := range sessions {
		sessionItems[i] = components.NewItem(s.Title, s.Directory)
	}

	return model{
		textarea:          ta,
		pasteCounter:      0,
		pastedTexts:       make(map[int]string),
		messages:          []models.Message{},
		cacheIndex:        0,
		cache:             make(map[int]string),
		viewport:          vp,
		islistSessionWin:  false,
		islistCommandsWin: false,
		bashMode:          false,
		listSession:       components.LaunchSessionList(sessionItems),
		listCommands:      components.LaunchCommandList(),
		sessions:          sessions,
		senderStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:               nil,
		width:             width,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.islistSessionWin {
		var cmd tea.Cmd
		updatedModel, cmd := m.listSession.Update(msg)
		m.listSession = updatedModel.(components.Model)
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "enter":
				cur_idx := m.listSession.Current()
				m.currentSession = m.sessions[cur_idx]
				m.messages = client.GetSessionData(m.currentSession.ID)
				m.islistSessionWin = false
				m.textarea.SetWidth(m.width)
				m.viewport.SetWidth(m.width)
				m.viewport.SetHeight(m.height - m.textarea.Height())
				m.viewport.SetContent(renderMessages(m.messages, m.width))
				m.viewport.GotoBottom()
			}
		}
		return m, cmd
	}
	if m.islistCommandsWin {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "esc":
				m.islistCommandsWin = false
				m.viewport.SetHeight(m.viewport.Height() + m.listCommands.Height())
				return m, nil
			case "up", "down":
				var cmd tea.Cmd
				updatedModel, cmd := m.listCommands.Update(msg)
				m.listCommands = updatedModel.(components.ModelCmdList)
				return m, cmd
			case "enter":
				m.cacheIndex++
				cur_command := m.listCommands.Current()
				m.islistCommandsWin = false
				m.viewport.SetHeight(m.viewport.Height() + m.listCommands.Height())
				cmd := CmdHandler("/"+cur_command, &m)
				return m, cmd
			default:
				m.cache[m.cacheIndex] = m.textarea.Value()
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				val := m.textarea.Value()
				if !strings.HasPrefix(val, "/") {
					m.islistCommandsWin = false
					m.viewport.SetHeight(m.viewport.Height() + m.listCommands.Height())
				} else {
					m.listCommands.Filter(strings.TrimPrefix(val, "/"))
				}
				return m, cmd
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width)
		m.textarea.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - m.textarea.Height())

		if len(m.messages) > 0 {
			m.viewport.SetContent(renderMessages(m.messages, m.width))
		}
		m.viewport.GotoBottom()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "esc":
			m.sessions = client.ListSession()
			m.islistSessionWin = true

		case "ctrl+v", "super+v":
			curVal := m.textarea.Value()
			err := clipboard.Init()
			if err != nil {
				panic(err)
			}
			textBytes := clipboard.Read(clipboard.FmtText)
			pasteValue := string(textBytes)
			if strings.Count(pasteValue, "\n") > 1 {
				m.pasteCounter++
				m.pastedTexts[m.pasteCounter] = pasteValue
				placeholder := fmt.Sprintf("[pasted text #%d]", m.pasteCounter)
				m.textarea.SetValue(curVal + " " + placeholder)
			} else {
				m.textarea.SetValue(curVal + pasteValue)
			}
			return m, nil
		case "shift+enter":
			curVal := m.textarea.Value()
			m.textarea.SetValue(curVal + "\n")
			return m, nil
		case "enter":
			if strings.HasPrefix(m.textarea.Value(), "/") {
				cmd := CmdHandler(m.textarea.Value(), &m)
				return m, cmd
			}
			if m.currentSession.ID == "" {
				session_id := client.CreateSession((m.textarea.Value()))
				m.currentSession = models.Session{ID: session_id, Title: m.textarea.Value(), Directory: "."}
				m.sessions = append(m.sessions, m.currentSession)
				m.listSession.Refresh(m.sessions)
			}
			textareaValue := createPrompt(strings.Trim(m.textarea.Value(), "\n"), &m)

			newMessage := client.SendMessage(m.currentSession.ID, textareaValue)
			m.messages = append(m.messages, newMessage)

			m.viewport.SetContent(renderMessages(m.messages, m.width))
			ch := client.ChatCompletion(m.currentSession.ID, textareaValue)
			m.streamCh = ch
			m.textarea.SetValue("")
			m.textarea.Reset()
			m.viewport.GotoBottom()
			return m, waitForMessages(ch)
		case "up", "down":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case "/":
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			if len(m.textarea.Value()) == 1 {
				m.islistCommandsWin = true
			}
			m.viewport.SetHeight(m.viewport.Height() - m.listCommands.Height())
			return m, cmd
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)

			if strings.HasPrefix(m.textarea.Value(), "!") {
				m.bashMode = true
				BashModeHandler(m.textarea.Value())
				return m, nil
			} else {
				m.bashMode = false
				BashModeHandler(m.textarea.Value())
				return m, cmd
			}
		}

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case streamMessageMsg:
		m.messages = append(m.messages, models.Message{
			SessionID: m.currentSession.ID,
			ID:        fmt.Sprintf("%s-assistant-%d", m.currentSession.ID, len(m.messages)),
			Data:      models.EncodeMessageData(models.StoredMessageData(msg)),
		})
		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.viewport.GotoBottom()
		return m, waitForMessages(m.streamCh)

	case streamDoneMsg:
		m.streamCh = nil
		return m, nil

	}

	return m, nil
}

func (m model) View() tea.View {
	if m.islistSessionWin {
		return m.listSession.View()
	}
	varListCommandsView := ""
	if m.islistCommandsWin {
		varListCommandsView = "\n" + m.listCommands.StringView()
	}
	var bashModeView = ""
	if m.bashMode {
		bashModeView = "Bash Mode"
	}
	m.viewport.SetContent(
		// m.currentSession.ID +
		// "\n" +
		renderMessages(m.messages, m.width))
	viewportView := m.viewport.View()

	v := tea.NewView(bashModeView + viewportView + "\n" + m.textarea.View() + varListCommandsView)
	c := m.textarea.Cursor()
	if c != nil {
		c.Y += lipgloss.Height(viewportView)
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}

// --- styles ---

var (
	styleDot        = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleToolName   = lipgloss.NewStyle().Bold(true)
	styleTree       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleUser       = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true).Background(lipgloss.Color("236")).Padding(0, 1)
	styleThink      = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Bold(false)
	styleResultText = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
)

// formatToolCall renders a tool call as ToolName(key_arg)
func formatToolCall(tc models.StoredToolCall) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err == nil && len(args) > 0 {
		priority := []string{"path", "file_path", "command", "pattern", "query", "url"}
		for _, key := range priority {
			if v, ok := args[key]; ok {
				val := fmt.Sprintf("%v", v)
				if len(val) > 50 {
					val = "..." + val[len(val)-47:]
				}
				return fmt.Sprintf("%s(%s)", styleToolName.Render(tc.Name), val)
			}
		}
		for _, v := range args {
			val := fmt.Sprintf("%v", v)
			if len(val) > 50 {
				val = "..." + val[len(val)-47:]
			}
			return fmt.Sprintf("%s(%s)", styleToolName.Render(tc.Name), val)
		}
	}
	return styleToolName.Render(tc.Name) + "()"
}

func formatToolResult(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return styleResultText.Render("(no output)")
	}
	lines := strings.Split(content, "\n")
	maxLines := 4
	if len(lines) <= maxLines {
		var out []string
		for _, l := range lines {
			out = append(out, "  "+styleResultText.Render(l))
		}
		return strings.Join(out, "\n")
	}
	var out []string
	for _, l := range lines[:maxLines] {
		out = append(out, "  "+styleResultText.Render(l))
	}
	out = append(out, styleTree.Render(fmt.Sprintf("  ... (%d more lines)", len(lines)-maxLines)))
	return strings.Join(out, "\n")
}

// lightcodeGlamourStyle is a custom glamour style tuned to match the app palette.
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
    "color": "37",
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
  "emph": { "italic": true, "color": "245" },
  "strong": { "bold": true, "color": "255" },
  "hr": {
    "color": "237",
    "format": "\n──────────────────────────────────────\n"
  },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "link": { "color": "51", "underline": true },
  "link_text": { "color": "43", "bold": true },
  "image": { "color": "212", "underline": true },
  "image_text": { "color": "243", "format": "Image: {{.text}} →" },
  "code": {
    "prefix": " ",
    "suffix": " ",
    "color": "215",
    "background_color": "235"
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

	dot := styleDot.Render("●")
	tree := styleTree.Render("└─")

	// Pre-pass: Find which tool calls in which messages have corresponding result messages
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
			lastAssistantMsgID = msg.ID
			callIdx = 0
		} else if d.Role == "tool_call" && lastAssistantMsgID != "" {
			hasResult[callKey{lastAssistantMsgID, callIdx}] = true
			callIdx++
		}
	}

	var lines []string
	lines = append(lines, mascott())
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
					lines = append(lines, dot+" Thinking: "+styleThink.Width(width).Render(matchedContent))
				}
				if content != "" {
					lines = append(lines, dot+" "+content)
				}

				// Only render tool calls that don't have results yet
				for i, tc := range d.ToolCalls {
					if !hasResult[callKey{msg.ID, i}] {
						lines = append(lines, dot+" "+formatToolCall(tc))
					}
				}

			case "tool_call":
				// For tool results, render BOTH the call and the result
				if len(d.ToolCalls) > 0 {
					lines = append(lines, dot+" "+formatToolCall(d.ToolCalls[0]))
				}
				resultSummary := formatToolResult(content)
				lines = append(lines, tree+" "+resultSummary)

			case "user":
				lines = append(lines, "")
				lines = append(lines, styleUser.Width(width).Render("> "+content))
				lines = append(lines, "")
			}
		} else if d.Role == "assistant" && len(d.ToolCalls) > 0 {
			// Assistant message with ONLY tool calls (no text content)
			for i, tc := range d.ToolCalls {
				if !hasResult[callKey{msg.ID, i}] {
					lines = append(lines, dot+" "+formatToolCall(tc))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func waitForMessages(ch chan models.StoredMessageData) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return streamMessageMsg(msg)
	}
}

func createPrompt(value string, m *model) string {
	re := regexp.MustCompile(`\[pasted text #(\d+)\]`)
	textareaValue := re.ReplaceAllStringFunc(value, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		var idx int
		fmt.Sscanf(sub[1], "%d", &idx)
		if real, ok := m.pastedTexts[idx]; ok {
			return real
		}
		return match
	})
	return textareaValue
}

type refreshSessionsMsg struct {
}

func CmdHandler(cmd string, m *model) tea.Cmd {
	switch cmd {
	case "/sessions":
		m.sessions = client.ListSession()
		m.islistSessionWin = true
		m.textarea.Reset()
	case "/new_session":
		m.currentSession = models.Session{ID: "", Title: "", Directory: "."}
		m.messages = []models.Message{}
		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.textarea.Reset()
		m.viewport.GotoBottom()
		return func() tea.Msg { return refreshSessionsMsg{} }

	case "/delete_session":
		session_id := m.currentSession.ID
		client.DeleteSession(session_id)
		var newSessions []models.Session
		for _, session := range m.sessions {
			if session.ID != session_id {
				newSessions = append(newSessions, session)
			}
		}
		m.sessions = newSessions
		m.currentSession = models.Session{ID: "", Title: "", Directory: "."}
		m.messages = []models.Message{}
		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.textarea.Reset()
		m.viewport.GotoBottom()
		return func() tea.Msg { return refreshSessionsMsg{} }
	}
	return nil
}

func BashModeHandler(cmd string) {

}

func mascott() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#41f7fa")).Render(`
  ▐█████▌
  █  █  █
 ▘▜█████▛▘▘
   ▘▘ ▝▝ 
`)
}
