package views

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Kartik-2239/lightcode/internal/server/api"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
	"github.com/Kartik-2239/lightcode/internal/tui/components"
)

func (m model) handleModelsListInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "enter":
		selectedModel := m.listModels.Current()
		m.isModelsListWin = false
		m.textarea.SetValue("")
		m.textarea.Placeholder = "Send a message..."
		if msg.String() == "enter" && selectedModel.Model != "" {
			m.modelsListIndex = findModelIndex(m.modelsList, selectedModel)
			if selectedModel.ApiKey == "" && !isAuthBackedModel(selectedModel) {
				m.enter_api_win = true
				m.textarea.Placeholder = "enter api key for " + selectedModel.Model
			}
			err := client.SetCurrentModel(selectedModel)
			if err != nil {

			}
		}

		m.textarea.Focus()
		(&m).syncLayout()
		return m, nil
	case "up", "down":
		updatedModel, cmd := m.listModels.Update(msg)
		m.listModels = updatedModel.(components.ModelModelsList)
		return m, cmd
	case "right":
		m.listModels.NextPage()
		return m, nil
	case "left":
		m.listModels.PrevPage()
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.listModels.Filter(m.textarea.Value())
		m.syncLayout()
		return m, cmd
	}
}

func isAuthBackedModel(model api.ModelInfo) bool {
	baseURL := strings.TrimSpace(model.BaseUrl)
	return baseURL != "" && !strings.HasPrefix(baseURL, "http")
}

func findModelIndex(modelsList []api.ModelInfo, selectedModel api.ModelInfo) int {
	for i, model := range modelsList {
		if model.Model == selectedModel.Model && model.BaseUrl == selectedModel.BaseUrl {
			return i
		}
	}
	return 0
}
