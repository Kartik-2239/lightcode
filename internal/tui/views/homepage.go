package views

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"fmt"
	"os"
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
	senderStyle       lipgloss.Style
	err               error
	cache             map[int]string
	streamCh          chan models.StoredMessageData
	current_cache     int
	width             int
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

	ta.SetWidth(100)
	ta.SetHeight(2)

	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(100), viewport.WithHeight(10))
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
		messages:          []models.Message{},
		viewport:          vp,
		islistSessionWin:  false,
		islistCommandsWin: false,
		bashMode:          false,
		listSession:       components.LaunchSessionList(sessionItems),
		listCommands:      components.LaunchCommandList(),
		sessions:          sessions,
		senderStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:               nil,
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
			}
		}
		return m, cmd
	}
	if m.islistCommandsWin {
		var cmd tea.Cmd
		updatedModel, cmd := m.listCommands.Update(msg)
		m.listCommands = updatedModel.(components.ModelCmdList)
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			// case "left", "right":
			// 	var cmd tea.Cmd
			// 	m.textarea, cmd = m.textarea.Update(msg)
			// 	return m, cmd
			// m.islistCommandsWin = false
			case "esc":
				m.islistCommandsWin = false
				m.viewport.SetHeight(m.viewport.Height() + m.listCommands.Height())
				return m, nil
			case "up", "down":
				return m, nil
			case "enter":
				cur_command := m.listCommands.Current()
				m.textarea.SetValue("/" + cur_command)
				m.islistCommandsWin = false
				m.viewport.SetHeight(m.viewport.Height() + m.listCommands.Height())
				return m, nil
			default:
				m.islistCommandsWin = false
				m.viewport.SetHeight(m.viewport.Height() + m.listCommands.Height())
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
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
			m.islistSessionWin = true
		case "ctrl+v", "super+v":
			curVal := m.textarea.Value()
			err := clipboard.Init()
			if err != nil {
				panic(err)
			}
			textBytes := clipboard.Read(clipboard.FmtText)
			curVal += string(textBytes)
			m.textarea.SetValue(curVal)
			return m, nil
		case "shift+enter":
			curVal := m.textarea.Value()
			m.textarea.SetValue(curVal + "\n")
			return m, nil
		case "enter":
			if strings.HasPrefix(m.textarea.Value(), "/") {
				CmdHandler(m.textarea.Value(), &m)
				return m, nil
			}
			if m.currentSession.ID == "" {
				session_id := client.CreateSession((m.textarea.Value()))
				m.currentSession = models.Session{ID: session_id, Title: m.textarea.Value(), Directory: "."}
			}
			textareaValue := strings.Trim(m.textarea.Value(), "\n")
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
				if out, err := r.Render(content); err == nil {
					content = strings.TrimSpace(out)
				}
				lines = append(lines, fmt.Sprintf("%s", content))
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
			lines = append(lines, toolStyle.Render(fmt.Sprintf("[TOOL] %s(%s)", tc.Name, tc.Arguments)))
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

func CmdHandler(cmd string, m *model) {
	switch cmd {
	case "/sessions":
		m.islistSessionWin = true
		m.textarea.Reset()
	}
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
