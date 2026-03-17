package components

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

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
	s.title = lipgloss.NewStyle().MarginLeft(2)
	s.item = lipgloss.NewStyle().PaddingLeft(4)
	s.selectedItem = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	s.pagination = list.DefaultStyles(darkBG).PaginationStyle.PaddingLeft(4)
	s.help = list.DefaultStyles(darkBG).HelpStyle.PaddingLeft(4).PaddingBottom(1)
	s.quitText = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	return s
}

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct {
	styles *styles
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.selectedItem.Render("> " + strings.Join(s, " "))
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
	items := []list.Item{
		item("sessions"),
		item("new_session"),
		item("mcp"),
		item("rename_session"),
		item("delete_session"),
		item("skills"),
		item("editor"),
		item("models"),
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
		if strings.Contains(strings.ToLower(string(i.(item))), strings.ToLower(term)) {
			filtered = append(filtered, i)
		}
	}
	m.list.SetItems(filtered)
}

func (m *ModelCmdList) updateStyles(isDark bool) {
	m.styles = newStyles(isDark)
	m.list.Styles.Title = m.styles.title
	m.list.Styles.PaginationStyle = m.styles.pagination
	m.list.Styles.HelpStyle = m.styles.help
	m.list.SetDelegate(itemDelegate{styles: &m.styles})
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
		// case "enter":
		// 	i, ok := m.list.SelectedItem().(item)
		// 	if ok {
		// 		m.choice = string(i)
		// 	}
		// 	return m, nil
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
			// case "default":
			// 	return m, nil
		}
	}
	return m, nil
}

func (m ModelCmdList) View() tea.View {
	if m.choice != "" {
		return tea.NewView(m.styles.quitText.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice)))
	}
	if m.quitting {
		return tea.NewView(m.styles.quitText.Render("Not hungry? That’s cool."))
	}
	return tea.NewView("\n" + m.list.View())
}

func (m ModelCmdList) StringView() string {
	return m.list.View()
}

func (m ModelCmdList) Current() string {
	return string(m.list.SelectedItem().(item))
}

func (m ModelCmdList) Height() int {
	return m.list.Height()
}

func LaunchCommandList() ModelCmdList {
	return initialModel()
}
