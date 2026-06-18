package components

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
)

const maxProvidersListHeight = 5

type providerItem config.Provider

func (i providerItem) FilterValue() string {
	provider := config.Provider(i)
	return provider.Name() + " " + provider.BaseUrl
}

type providerItemDelegate struct {
	styles *styles
}

func (d providerItemDelegate) Height() int                             { return 1 }
func (d providerItemDelegate) Spacing() int                            { return 0 }
func (d providerItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d providerItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(providerItem)
	if !ok {
		return
	}

	provider := config.Provider(i)
	str := fmt.Sprintf("%s (%s)", provider.Name(), provider.BaseUrl)

	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.selectedItem.Render(lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Render("→ " + strings.Join(s, " ")))
		}
	}

	fmt.Fprint(w, fn(str))
}

type ModelProvidersList struct {
	list     list.Model
	allItems []list.Item
	styles   styles
	height   int
}

func initialProvidersList() ModelProvidersList {
	const defaultWidth = 20
	const defaultHeight = 5
	l := list.New([]list.Item{}, providerItemDelegate{}, defaultWidth, defaultHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetDelegate(list.DefaultDelegate{})
	m := ModelProvidersList{list: l, allItems: []list.Item{}, height: defaultHeight}
	m.updateStyles(true)
	return m
}

func (m *ModelProvidersList) Filter(term string) {
	if term == "" {
		m.list.SetItems(m.allItems)
		m.list.SetHeight(providersListHeight(len(m.allItems)))
		return
	}
	var filtered []list.Item
	for _, i := range m.allItems {
		if strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(term)) {
			filtered = append(filtered, i)
		}
	}
	m.list.SetItems(filtered)
	m.list.SetHeight(providersListHeight(len(filtered)))
}

func (m *ModelProvidersList) Refresh(items []config.Provider) {
	listItems := make([]list.Item, len(items))
	for i, provider := range items {
		listItems[i] = providerItem(provider)
	}
	m.allItems = listItems
	m.list.SetItems(listItems)
	m.list.SetHeight(providersListHeight(len(listItems)))
}

func (m *ModelProvidersList) updateStyles(isDark bool) {
	m.styles = newStyles(isDark)
	m.list.Styles.Title = m.styles.title
	m.list.Styles.PaginationStyle = m.styles.pagination
	m.list.Styles.HelpStyle = m.styles.help
	m.list.SetDelegate(providerItemDelegate{styles: &m.styles})
}

func (m ModelProvidersList) Init() tea.Cmd {
	return nil
}

func (m ModelProvidersList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m ModelProvidersList) View() tea.View {
	return tea.NewView("\n" + m.list.View())
}

func (m ModelProvidersList) StringView() string {
	return m.list.View()
}

func (m ModelProvidersList) NextPage() {
	m.list.NextPage()
}

func (m ModelProvidersList) PrevPage() {
	m.list.PrevPage()
}

func (m ModelProvidersList) Current() config.Provider {
	selected := m.list.SelectedItem()
	if selected == nil {
		return config.Provider{}
	}
	it, ok := selected.(providerItem)
	if !ok {
		return config.Provider{}
	}
	return config.Provider(it)
}

func (m ModelProvidersList) Height() int {
	return m.list.Height()
}

func providersListHeight(items int) int {
	if items <= 0 {
		return 1
	}
	if items > maxProvidersListHeight {
		return maxProvidersListHeight
	}
	return items
}

func LaunchProviderList() ModelProvidersList {
	return initialProvidersList()
}
