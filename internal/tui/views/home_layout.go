package views

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/api"
)

func (m *model) syncLayout() {
	m.textarea.SetWidth(max(m.width-lipgloss.Width(textareaPrompt), 1))
	m.resizeTextareaToContent()
	m.viewport.SetWidth(m.width)

	reservedHeight := m.textarea.Height()
	if m.isGenerating || m.isCompacting {
		reservedHeight++
	}
	if len(m.mode) > 0 {
		reservedHeight++
	}
	if len(m.queue) > 0 {
		reservedHeight += len(m.queue) + 1
	}
	if previews, ok := m.currentKittyPreview(); ok {
		reservedHeight += previews[0].rows
	}
	if m.bashMode {
		reservedHeight++
	}
	if m.isError {
		reservedHeight++
	}
	if m.questionMode {
		reservedHeight += m.questionUIHeight()
	}

	if m.islistCommandsWin {
		reservedHeight += m.listCommands.Height()
	}
	if m.isModelsListWin {
		reservedHeight += m.listModels.Height()
	}
	if m.isLoginProviderWin {
		reservedHeight += loginProviderListHeight(len(m.loginProviders))
	}
	if m.isEffortWin {
		reservedHeight += effortListHeight(len(m.effortOptions))
	}
	// for textarea border
	reservedHeight += 2
	// for dir above textarea
	if strings.TrimSpace(m.currentSession.Directory) != "." || strings.TrimSpace(m.currentSession.Directory) != "" {
		reservedHeight += 1
	}

	viewportHeight := m.height - reservedHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport.SetHeight(viewportHeight)
}

func (m *model) resizeTextareaToContent() {
	height := max(countWrappedLines(m.textarea.Value(), m.textarea.Width(), m), 1)
	if height != m.textarea.Height() {
		m.textarea.SetHeight(height)
	}
}

func (m model) textareaView() string {
	value := m.textarea.Value()
	if value == "" {
		value = m.textarea.Placeholder
	}
	lines := wrapTextLines(value, m.textarea.Width())
	for len(lines) < m.textarea.Height() {
		lines = append(lines, "")
	}
	if len(lines) > m.textarea.Height() {
		lines = lines[:m.textarea.Height()]
	}
	return strings.Join(lines, "\n")
}

func (m model) View() tea.View {
	if m.islistSessionWin {
		return m.listSession.View()
	}
	m.viewport.SetContent(
		// m.currentSession.ID +
		// "\n" +
		renderMessages(m.messages, m.width))

	sections := make([]string, 0, 5)
	sections = append(sections, m.viewport.View())

	if m.questionMode {
		sections = append(sections, m.renderQuestionUI())
	}
	if len(m.queue) > 0 {
		sections = append(sections, m.renderQueueList())
	}

	if previews, ok := m.currentKittyPreview(); ok {
		images := []string{}
		slices.Reverse(previews)
		for _, preview := range previews {
			if placeholders := renderKittyPlaceholders(preview.id, preview.cols, preview.rows); placeholders != "" {
				images = append(images, placeholders)
			}
		}
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, images...))

	}

	// sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Render(shortenDir(m.currentSession.Directory)))
	if m.isError {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.BrightRed).Render(m.errorMessage))
		m.isError = false
	}

	if m.enter_api_win {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.BrightRed).Render("enter api key for "+m.listModels.Current().Model))
	}

	sections = append(sections, lipgloss.NewStyle().Render(strings.Repeat("—", m.width)))
	textareaSectionIndex := len(sections)

	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, textareaPrompt, m.textareaView()))
	sections = append(sections, lipgloss.NewStyle().Render(strings.Repeat("—", m.width)))

	if m.islistCommandsWin {
		sections = append(sections, m.listCommands.StringView())
	}
	if m.isModelsListWin {
		sections = append(sections, m.listModels.StringView())
	}
	if m.isLoginProviderWin {
		sections = append(sections, m.renderLoginProviderList())
	}
	if m.isEffortWin {
		sections = append(sections, m.renderEffortList())
	}

	currentModel := api.ModelInfo{}
	if m.modelsListIndex >= 0 && m.modelsListIndex < len(m.modelsList) {
		currentModel = m.modelsList[m.modelsListIndex]
	}
	sections = append(sections, renderStatusLine(currentModel, m.currentContextSize, m.width, m.gitStatus))

	if m.isGenerating || m.isCompacting {
		if m.showEscMsg {
			sections = append(sections, m.spinner.View()+lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(" Press Esc again to cancel..."))
		} else if m.isCompacting {
			sections = append(sections, m.spinner.View()+" Compacting...")
		} else {
			sections = append(sections, m.spinner.View()+" Generating...")
		}
	}

	v := tea.NewView(strings.Join(sections, "\n"))
	c := tea.NewCursor(wrappedCursorPosition(m.textarea.Value(), m.textarea.Line(), m.textarea.Column(), m.textarea.Width()))
	if m.isModelsListWin {
		c = nil
	} else if c != nil {
		c.X += lipgloss.Width(textareaPrompt)
		if textareaSectionIndex > 0 {
			c.Y += lipgloss.Height(strings.Join(sections[:textareaSectionIndex], "\n"))
		}
	}
	v.Cursor = c
	v.AltScreen = true
	// v.MouseMode = tea.MouseModeCellMotion
	return v
}
