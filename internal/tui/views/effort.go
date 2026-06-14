package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
)

type effortOption struct {
	label       string
	value       string
	description string
}

func defaultEffortOptions() []effortOption {
	return []effortOption{
		{label: "low", value: "low", description: "faster scoped tasks"},
		{label: "medium", value: "medium", description: "balanced default"},
		{label: "high", value: "high", description: "deeper debugging"},
		{label: "extra high", value: "xhigh", description: "long reasoning-heavy tasks"},
	}
}

func openEffortList(m *model) {
	current, err := client.GetCurrentModel()
	if err != nil || current.Model == "" {
		appendCommandStatusMessage(m, "No current model selected.")
		return
	}
	if !config.SupportsReasoningEffort(current) {
		appendCommandStatusMessage(m, fmt.Sprintf("%s does not support reasoning effort.", current.Model))
		return
	}

	m.textarea.Reset()
	m.effortOptions = defaultEffortOptions()
	m.effortIndex = effortIndexForValue(m.effortOptions, current.ReasoningEffort)
	m.isEffortWin = true
	m.textarea.Placeholder = "↑↓ select · Enter set effort · Esc close"
	m.textarea.Focus()
	m.syncLayout()
}

func (m model) handleEffortInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.isEffortWin = false
		m.textarea.Placeholder = "Send a message..."
		m.syncLayout()
		return m, nil
	case "up", "k":
		if m.effortIndex > 0 {
			m.effortIndex--
		}
		return m, nil
	case "down", "j":
		if m.effortIndex < len(m.effortOptions)-1 {
			m.effortIndex++
		}
		return m, nil
	case "enter":
		if len(m.effortOptions) == 0 {
			return m, nil
		}
		option := m.effortOptions[m.effortIndex]
		m.isEffortWin = false
		m.textarea.Placeholder = "Send a message..."
		if err := client.SetReasoningEffort(option.value); err != nil {
			appendCommandStatusMessage(&m, fmt.Sprintf("Effort update failed: %s", err.Error()))
			return m, nil
		}
		if m.modelsListIndex >= 0 && m.modelsListIndex < len(m.modelsList) {
			m.modelsList[m.modelsListIndex].ReasoningEffort = option.value
		}
		appendCommandStatusMessage(&m, fmt.Sprintf("Reasoning effort set to %s.", option.label))
		return m, nil
	}
	return m, nil
}

func (m model) renderEffortList() string {
	var sb strings.Builder
	sb.WriteString(effortTitleStyle.Render("Reasoning effort"))
	sb.WriteString("\n")
	for i, option := range m.effortOptions {
		cursor := "  "
		style := effortItemStyle
		if i == m.effortIndex {
			cursor = effortCursorStyle.Render("→ ")
			style = effortSelectedItemStyle
		}
		sb.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, style.Render(option.label), effortDescStyle.Render(option.description)))
	}
	sb.WriteString(effortHelpStyle.Render("↑↓ select · Enter set effort · Esc close"))
	return sb.String()
}

func effortIndexForValue(options []effortOption, value string) int {
	for i, option := range options {
		if option.value == value {
			return i
		}
	}
	return 1
}

func effortListHeight(items int) int {
	if items < 1 {
		items = 1
	}
	return items + 2
}

var (
	effortTitleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	effortCursorStyle       = lipgloss.NewStyle().Foreground(lipgloss.BrightCyan)
	effortItemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	effortSelectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Bold(true)
	effortDescStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	effortHelpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)
