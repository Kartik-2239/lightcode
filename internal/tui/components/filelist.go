package components

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

// fileItem is a single selectable path in the @mention picker. A trailing "/"
// marks a directory.
type fileItem struct {
	path string
}

func (i fileItem) FilterValue() string { return i.path }

type fileItemDelegate struct {
	styles *styles
}

func (d fileItemDelegate) Height() int                             { return 1 }
func (d fileItemDelegate) Spacing() int                            { return 0 }
func (d fileItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d fileItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(fileItem)
	if !ok {
		return
	}
	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			temp := lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Render("→ " + strings.Join(s, " "))
			return d.styles.selectedItem.Render(temp)
		}
	}
	fmt.Fprint(w, fn(i.path))
}

// ModelFileList is the @file mention picker. It mirrors ModelCmdList but ranks
// entries with fuzzy matching over the file index.
type ModelFileList struct {
	list     list.Model
	allPaths []string
	styles   styles
}

func NewFileList(paths []string) ModelFileList {
	l := list.New(nil, fileItemDelegate{}, 20, 8)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowTitle(false)
	l.SetShowFilter(false)

	m := ModelFileList{list: l, allPaths: paths}
	m.styles = newStyles(true)
	m.list.SetDelegate(fileItemDelegate{styles: &m.styles})
	m.Filter("")
	return m
}

// SetItems replaces the candidate file index (e.g. after lazily building it).
func (m *ModelFileList) SetItems(paths []string) {
	m.allPaths = paths
	m.Filter("")
}

// Filter ranks the index against term. An empty term shows everything in index
// order; otherwise results are fuzzy-ranked.
func (m *ModelFileList) Filter(term string) {
	var items []list.Item
	if term == "" {
		items = make([]list.Item, 0, len(m.allPaths))
		for _, p := range m.allPaths {
			items = append(items, fileItem{path: p})
		}
	} else {
		matches := fuzzy.Find(term, m.allPaths)
		items = make([]list.Item, 0, len(matches))
		for _, mt := range matches {
			items = append(items, fileItem{path: mt.Str})
		}
	}
	m.list.ResetSelected()
	m.list.SetItems(items)
}

func (m ModelFileList) Init() tea.Cmd { return nil }

func (m ModelFileList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "right":
			m.list.NextPage()
		case "left":
			m.list.PrevPage()
		}
	}
	return m, nil
}

func (m ModelFileList) View() tea.View { return tea.NewView("\n" + m.list.View()) }

func (m ModelFileList) StringView() string { return m.list.View() }

func (m ModelFileList) Height() int { return m.list.Height() }

// Current returns the highlighted path, or "" if the index is empty.
func (m ModelFileList) Current() string {
	selected := m.list.SelectedItem()
	if selected == nil {
		return ""
	}
	it, ok := selected.(fileItem)
	if !ok {
		return ""
	}
	return it.path
}
