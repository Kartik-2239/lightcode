package components

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Item struct {
	title       string
	description string
}

func NewItem(title, description string) Item {
	return Item{title: title, description: description}
}

func (i Item) Title() string       { return i.title }
func (i Item) Description() string { return i.description }
func (i Item) FilterValue() string { return i.title }

type Model struct {
	list    list.Model
	current int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			return m, nil
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.current = m.list.Index()
	return m, cmd
}

func (m Model) View() tea.View {
	v := tea.NewView(docStyle.Render(m.list.View()))
	v.AltScreen = true
	return v
}

func (m Model) Current() int {
	return m.current
}

func LaunchSessionList(items []list.Item) Model {
	m := Model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "Sessions"
	return m
}
