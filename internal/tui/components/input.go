package components

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type doneLoadingMsg struct{}

func fakeSleep(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return doneLoadingMsg{}
	}
}

// func main() {
// 	p := tea.NewProgram(InitialModel())
// 	if _, err := p.Run(); err != nil {
// 		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
// 	}
// }

type model struct {
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
	spinner     spinner.Model
	loading     bool
}

func InitialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(30), viewport.WithHeight(5))
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		spinner:     spin,
		err:         nil,
		loading:     false,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.textarea.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - m.textarea.Height())

		if len(m.messages) > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n")))
		}
		m.viewport.GotoBottom()

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case doneLoadingMsg:
		m.loading = false
		m.messages = append(m.messages, "Bot: Hello!")
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n")))
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyPressMsg:
		if m.loading {
			// Ignore keypresses while loading
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "enter":
			m.messages = append(m.messages, m.senderStyle.Render("You: ")+m.textarea.Value())
			m.loading = true
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n")))
			m.textarea.Reset()
			m.viewport.GotoBottom()
			// Start spinner ticking + fake 2s sleep in parallel
			return m, tea.Batch(m.spinner.Tick, fakeSleep(2*time.Second))
		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() tea.View {
	viewportView := m.viewport.View()

	textareaView := m.textarea.View()
	if m.loading {
		// Place spinner inline to the left of the textarea
		textareaView = lipgloss.JoinHorizontal(lipgloss.Top,
			m.spinner.View()+" ",
			textareaView,
		)
	}

	v := tea.NewView(viewportView + "\n" + textareaView)
	c := m.textarea.Cursor()
	if c != nil {
		c.Y += lipgloss.Height(viewportView) // +1 for the newline separator
		if m.loading {
			c.X += lipgloss.Width(m.spinner.View() + " ")
		}
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}
