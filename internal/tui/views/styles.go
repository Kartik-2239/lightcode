package views

import "charm.land/lipgloss/v2"

// Brand theme palette (top -> bottom gradient)
const (
	themeTop     = "#2ECAF8" // RGB 46, 202, 248
	themeMiddle  = "#2CD5CF" // RGB 44, 213, 207
	themeBorder  = "#20938f" // RGB 44, 213, 207
	themeBottom  = "#22DA9C" // RGB 34, 218, 156
	themeSurface = "#0F2530" // dark surface tinted toward the theme
)

var (
	// styleToolName   = lipgloss.NewStyle().Foreground(lipgloss.Color(themeMiddle)).Bold(true)
	styleUser       = lipgloss.NewStyle().Foreground(lipgloss.Color(themeTop)).Bold(true).Background(lipgloss.Color(themeSurface)).Margin(1, 0).AlignVertical(lipgloss.Center).PaddingBottom(1).PaddingTop(1)
	styleThink      = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Bold(false)
	styleResultText = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleAdded      = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#aff5b4")).
			Background(lipgloss.Color("#1a3a2a")).
			PaddingLeft(1)
	styleRemoved = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffdcd7")).
			Background(lipgloss.Color("#3d1a1f")).
			PaddingLeft(1)
	styleTodoTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(themeTop)).Bold(true)
	styleTodoEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	styleTodoDone  = lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Strikethrough(true)
	styleTodoOpen  = lipgloss.NewStyle().Foreground(lipgloss.Color(themeBottom))
	styleTodoBox   = lipgloss.NewStyle().
			Padding(0, 1).
			MarginLeft(2)
	styleQuestionHeader = lipgloss.NewStyle().Foreground(lipgloss.Color(themeMiddle)).Bold(true)
	styleOptionNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleOptionSelected = lipgloss.NewStyle().Foreground(lipgloss.Color(themeMiddle)).Bold(true)

	styleStatusText = lipgloss.NewStyle().Foreground(lipgloss.Color(themeMiddle)).Bold(true)
	styleToolResult = lipgloss.NewStyle().Foreground(lipgloss.Color("#93f1f8")).Background(lipgloss.Color("#243d3f")).BorderBackground(lipgloss.Color("#243d3f")).BorderForeground(lipgloss.Color(themeBorder)).PaddingLeft(1).Bold(true)
	styleToolName   = lipgloss.NewStyle().Foreground(lipgloss.Color(themeMiddle)).Background(lipgloss.Color("#243d3f")).BorderBackground(lipgloss.Color("#243d3f")).BorderForeground(lipgloss.Color(themeBorder)).PaddingLeft(1).Bold(true)
)
