package views

import "charm.land/lipgloss/v2"

const (
	brandGradientTop    = "#31b5dd" // RGB 46, 202, 248
	brandGradientMiddle = "#42cfd9" // RGB 44, 213, 207
	brandGradientBottom = "#22DA9C" // RGB 34, 218, 156

	appSurfaceBackground = "#0F2530" // dark surface tinted toward the theme
	mutedTealBorder      = "#255857"

	userMessageText       = brandGradientTop
	userMessageBackground = appSurfaceBackground
	thinkingText          = "8"
	resultText            = "7"

	addedDiffBackground = "#09210a"
	addedDiffText       = "#97c8b0"
	addedDiffBorder     = "#325134"

	removedDiffBackground = "#331212"
	removedDiffText       = "#d89da6"
	removedDiffBorder     = "#433532"

	todoTitleText = brandGradientTop
	todoEmptyText = "240"
	todoDoneText  = "247"
	todoOpenText  = brandGradientBottom

	questionHeaderText = brandGradientMiddle
	optionText         = "252"
	selectedOptionText = brandGradientMiddle
	statusText         = brandGradientMiddle

	toolResultText       = "#3aa9b1"
	toolPanelBackground  = "#132e30"
	toolPanelBorder      = mutedTealBorder
	toolNameText         = brandGradientMiddle
	toolNameBackground   = toolPanelBackground
	toolNameBorder       = toolPanelBorder
	toolResultBackground = toolPanelBackground
	toolResultBorder     = toolPanelBorder
)

var (
	styleUser       = lipgloss.NewStyle().Foreground(lipgloss.Color(userMessageText)).Bold(true).Background(lipgloss.Color(userMessageBackground)).Margin(1, 0).AlignVertical(lipgloss.Center).PaddingBottom(1).PaddingTop(1)
	styleThink      = lipgloss.NewStyle().Foreground(lipgloss.Color(thinkingText)).Bold(false)
	styleResultText = lipgloss.NewStyle().Foreground(lipgloss.Color(resultText))

	styleAdded = lipgloss.NewStyle().
			Foreground(lipgloss.Color(addedDiffText)).
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBackground(lipgloss.Color(addedDiffBackground)).
			BorderForeground(lipgloss.Color(addedDiffBorder)).
			Background(lipgloss.Color(addedDiffBackground)).
			PaddingLeft(1)
	styleRemoved = lipgloss.NewStyle().
			Foreground(lipgloss.Color(removedDiffText)).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(lipgloss.Color(removedDiffBackground)).
			BorderForeground(lipgloss.Color(removedDiffBorder)).
			BorderStyle(lipgloss.RoundedBorder()).
			Background(lipgloss.Color(removedDiffBackground)).
			PaddingLeft(1)

	styleTodoTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(todoTitleText)).Bold(true)
	styleTodoEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color(todoEmptyText)).Italic(true)
	styleTodoDone  = lipgloss.NewStyle().Foreground(lipgloss.Color(todoDoneText)).Strikethrough(true)
	styleTodoOpen  = lipgloss.NewStyle().Foreground(lipgloss.Color(todoOpenText))
	styleTodoBox   = lipgloss.NewStyle().
			Padding(0, 1).
			MarginLeft(2)

	styleQuestionHeader = lipgloss.NewStyle().Foreground(lipgloss.Color(questionHeaderText)).Bold(true)
	styleOptionNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color(optionText))
	styleOptionSelected = lipgloss.NewStyle().Foreground(lipgloss.Color(selectedOptionText)).Bold(true)

	styleStatusText = lipgloss.NewStyle().Foreground(lipgloss.Color(statusText)).Bold(true)
	styleToolResult = lipgloss.NewStyle().Foreground(lipgloss.Color(toolResultText)).Background(lipgloss.Color(toolResultBackground)).BorderBackground(lipgloss.Color(toolResultBackground)).BorderForeground(lipgloss.Color(toolResultBorder)).PaddingLeft(1)
	styleToolName   = lipgloss.NewStyle().Foreground(lipgloss.Color(toolNameText)).Background(lipgloss.Color(toolNameBackground)).BorderBackground(lipgloss.Color(toolNameBackground)).BorderForeground(lipgloss.Color(toolNameBorder)).PaddingLeft(1).Bold(true)
)
