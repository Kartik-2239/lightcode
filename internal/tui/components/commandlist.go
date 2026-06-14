package components

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type item struct {
	name        string
	description string
}

var COMMANDS = []item{
	{name: "sessions", description: "List and switch sessions"},
	{name: "new_session", description: "Start a fresh session"},
	{name: "delete_session", description: "Delete the current session"},
	{name: "export", description: "Export the current session to markdown"},
	{name: "skills", description: "Show available skills"},
	{name: "login", description: "Sign in to OAuth providers"},
	{name: "logout", description: "Sign out of OAuth providers"},
	{name: "effort", description: "Set model reasoning effort"},
	{name: "models", description: "Browse and select models"},
	{name: "usage", description: "Show token usage for the session"},
	{name: "dir", description: "Show the current session directory"},
	{name: "compact", description: "Compact conversation"},
}

func longest_word() string {
	longest := ""
	for _, stuff := range COMMANDS {
		if len(stuff.name) > len(longest) {
			longest = stuff.name
		}
	}
	return longest
}

type styles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	selectedItem lipgloss.Style
	pagination   lipgloss.Style
	help         lipgloss.Style
	quitText     lipgloss.Style
}

func newStyles(darkBG bool) styles {
	var s styles
	s.title = lipgloss.NewStyle().MarginLeft(0)
	s.item = lipgloss.NewStyle().PaddingLeft(2)
	s.selectedItem = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.White)
	s.pagination = list.DefaultStyles(darkBG).PaginationStyle.PaddingLeft(1)
	s.help = list.DefaultStyles(darkBG).HelpStyle.PaddingLeft(1).PaddingBottom(1)
	s.quitText = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	return s
}

func (i item) FilterValue() string { return i.name + " " + i.description }
func (i item) Name() string        { return i.name }

type itemDelegate struct {
	styles       *styles
	longest_word string
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	pad := len(d.longest_word) - len(i.name) + 1
	if pad < 1 {
		pad = 1
	}
	str := i.name + strings.Repeat(" ", pad) + " " + i.description

	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			temp := lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Render("→ " + strings.Join(s, " "))
			return d.styles.selectedItem.Render(temp)
		}
	}

	fmt.Fprint(w, fn(str))
}

type ModelCmdList struct {
	list     list.Model
	allItems []list.Item
	choice   string
	styles   styles
	quitting bool
	current  int
}

func initialModel() ModelCmdList {
	items := []list.Item{}
	// 	item{name: "sessions", description: "List and switch sessions"},
	// 	item{name: "new_session", description: "Start a fresh session"},
	// 	item{name: "delete_session", description: "Delete the current session"},
	// 	item{name: "skills", description: "Show available skills"},
	// 	item{name: "models", description: "Browse and select models"},
	// 	item{name: "usage", description: "Show token usage for the session"},
	// 	item{name: "dir", description: "Show the current session directory"},
	// }
	for _, it := range COMMANDS {
		items = append(items, item{name: it.name, description: it.description})
	}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, 5)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowTitle(false)
	l.SetShowFilter(false)

	m := ModelCmdList{list: l, allItems: items}
	m.updateStyles(true) // default to dark styles.
	return m
}

func (m *ModelCmdList) Filter(term string) {
	if term == "" {
		m.list.SetItems(m.allItems)
		return
	}
	var filtered []list.Item
	for _, i := range m.allItems {
		if strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(term)) {
			filtered = append(filtered, i)
		}
	}
	m.list.ResetSelected()
	m.list.SetItems(filtered)
}

func (m *ModelCmdList) updateStyles(isDark bool) {
	m.styles = newStyles(isDark)
	m.list.Styles.Title = m.styles.title
	m.list.Styles.PaginationStyle = m.styles.pagination
	m.list.Styles.HelpStyle = m.styles.help
	m.list.SetDelegate(itemDelegate{styles: &m.styles, longest_word: longest_word()})
}

func (m ModelCmdList) Init() tea.Cmd {
	return nil
}

func (m ModelCmdList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyPressMsg:
		switch keypress := msg.String(); keypress {
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case "right":
			m.list.NextPage()
		case "left":
			m.list.PrevPage()
		}
	}
	return m, nil
}

func (m ModelCmdList) View() tea.View {
	return tea.NewView("\n" + m.list.View())
}

func (m ModelCmdList) StringView() string {
	return m.list.View()
}

func (m ModelCmdList) Current() string {
	selected := m.list.SelectedItem()
	if selected == nil {
		return ""
	}
	it, ok := selected.(item)
	if !ok {
		return ""
	}
	return it.name
}

func (m ModelCmdList) Reset() {
	m.list.ResetSelected()
}

func (m ModelCmdList) Height() int {
	return m.list.Height()
}

func LaunchCommandList() ModelCmdList {
	return initialModel()
}
