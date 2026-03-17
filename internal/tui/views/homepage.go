package views

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"fmt"
	"html"
	"math"
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
				// var cmd tea.Cmd
				// updatedModel, cmd := m.listCommands.Update(msg)
				// m.listCommands = updatedModel.(components.ModelCmdList)
				if msg.String() == "up" {
					math.Min(float64(m.cacheIndex-1), 0)
					m.textarea.SetValue(m.cache[m.cacheIndex])
				} else {
					math.Min(float64(m.cacheIndex+1), float64(len(m.cache)))
					m.textarea.SetValue(m.cache[m.cacheIndex])
				}
				return m, nil
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
				m.pastedTexts[m.pasteCounter] = pasteValue
				m.pasteCounter++
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

		// case refreshSessionsMsg:
		// 	m.sessions = client.ListSession()
		// 	sessionItems := make([]list.Item, len(m.sessions))
		// 	for i, s := range m.sessions {
		// 		sessionItems[i] = components.NewItem(s.Title, s.Directory)
		// 	}
		// 	m.listSession.Refresh(m.sessions) // or m.listSession.Refresh(m.sessions)
		// 	return m, nil
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

func renderMessages(msgs []models.Message, width int) string {
	if width <= 0 {
		width = 80
	}
	r, _ := glamour.NewTermRenderer(glamour.WithWordWrap(width), glamour.WithStylePath("dark"))

	var lines []string
	lines = append(lines, mascott())
	for _, msg := range msgs {
		d := models.DecodeMessageData(msg.Data)
		if d.Role == "" || d.Role == "error" {
			continue
		}
		if d.Content != "" {
			content := d.Content
			if d.Role == "assistant" {
				content = html.UnescapeString(content)
				re := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
				matches := re.FindAllString(content, -1)
				matchedContent := strings.Join(matches, " ")
				content = content[len(strings.Join(matches, " ")):]

				if out, err := r.Render(content); err == nil {
					content = strings.TrimSpace(out)
				}
				if out, err := r.Render(matchedContent); err == nil {
					matchedContent = strings.TrimSpace(out)
				}

				if len(matches) == 0 {
					lines = append(lines, fmt.Sprintf("%s", "content"))
					continue
				} else {
					thinkStyle := lipgloss.NewStyle().
						Foreground(lipgloss.BrightBlack).
						Bold(false).
						Width(width)
					lines = append(lines, thinkStyle.Render(matchedContent))
					lines = append(lines, fmt.Sprintf("%s", content))
				}
			}
			if d.Role == "user" {
				userStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("20")).
					Bold(true).
					Width(width).
					Background(lipgloss.Color("2"))
				lines = append(lines, userStyle.Render(fmt.Sprintf("[%s] %s", strings.ToUpper(d.Role), content)))
			}
		}
		for _, tc := range d.ToolCalls {
			toolStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("20")).
				Bold(true).
				Width(width).
				Background(lipgloss.Color("2"))
			arguments := ""
			if len(tc.Arguments) > 20 {
				arguments = tc.Arguments[:17] + "..."
			} else {
				arguments = tc.Arguments
			}
			lines = append(lines, toolStyle.Render(fmt.Sprintf("[TOOL] %s(%s)", tc.Name, arguments)))
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
