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
	viewport         viewport.Model
	islistSessionWin bool
	listSession      components.Model
	sessions         []models.Session
	currentSession   models.Session
	messages         []models.Message
	textarea         textarea.Model
	senderStyle      lipgloss.Style
	err              error
	cache            map[int]string
	streamCh         chan models.StoredMessageData
	current_cache    int
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 32000

	ta.SetWidth(100)
	ta.SetHeight(3)

	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(30), viewport.WithHeight(10))
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	ta.KeyMap.InsertNewline.SetEnabled(false)
	sessions := client.ListSession()
	sessionItems := make([]list.Item, len(sessions))
	for i, s := range sessions {
		sessionItems[i] = components.NewItem(s.Title, s.Directory)
	}

	return model{
		textarea:         ta,
		messages:         []models.Message{},
		viewport:         vp,
		islistSessionWin: false,
		listSession:      components.LaunchSessionList(sessionItems),
		sessions:         sessions,
		senderStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:              nil,
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.textarea.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - m.textarea.Height())

		if len(m.messages) > 0 {
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(decodeMessages(m.messages)))
		}
		m.viewport.GotoBottom()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "esc":
			m.islistSessionWin = true
		case "ctrl+v":
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
			if m.currentSession.ID == "" {
				session_id := client.CreateSession((m.textarea.Value()))
				m.currentSession = models.Session{ID: session_id, Title: m.textarea.Value(), Directory: "."}
			}
			textareaValue := strings.Trim(m.textarea.Value(), "\n")
			newMessage := client.SendMessage(m.currentSession.ID, textareaValue)
			m.messages = append(m.messages, newMessage)

			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(decodeMessages(m.messages)))
			ch := client.ChatCompletion(m.currentSession.ID, textareaValue)
			m.streamCh = ch
			m.textarea.Reset()
			m.viewport.GotoBottom()
			return m, waitForMessages(ch)
		case "up", "down":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
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
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(decodeMessages(m.messages)))
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
	m.viewport.SetContent(m.currentSession.ID + "\n" + decodeMessages(m.messages))
	viewportView := m.viewport.View()
	v := tea.NewView(viewportView + "\n" + m.textarea.View())
	c := m.textarea.Cursor()
	if c != nil {
		c.Y += lipgloss.Height(viewportView)
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}

func decodeMessages(msgs []models.Message) string {
	var lines []string
	for _, msg := range msgs {
		d := models.DecodeMessageData(msg.Data)
		if d.Role == "" || d.Role == "error" {
			continue
		}
		if d.Content != "" {
			lines = append(lines, fmt.Sprintf("[%s] %s", strings.ToUpper(d.Role), d.Content))
		}
		for _, tc := range d.ToolCalls {
			lines = append(lines, fmt.Sprintf("[TOOL] %s(%s)", tc.Name, tc.Arguments))
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
