package views

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/tui/components"
)

func (m model) handleAddProviderListInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.isAddProviderWin = false
		m.textarea.Placeholder = "Type your message here..."
		m.textarea.Reset()
		(&m).syncLayout()
		return m, nil
	case "enter":
		m.isAddProviderWin = false
		if m.isAddingApiKey {
			m.isAddingApiKey = false
			m.errorMessage = "Added api key :)"
			m.textarea.Placeholder = "Type your message here..."
			(&m).syncLayout()
			return m, nil
		}
		m.isAddingApiKey = true
		m.textarea.SetValue("")
		for _, provider := range config.GetCustomization().Providers {
			if provider.BaseUrl == m.listProviders.Current().BaseUrl {
				m.textarea.SetValue(provider.ApiKey)
				break
			}
		}
		m.textarea.Placeholder = "Add API Key for " + m.listProviders.Current().BaseUrl + "..."
		m.textarea.Focus()
		(&m).syncLayout()
		return m, nil
	case "up", "down":
		updatedModel, cmd := m.listProviders.Update(msg)
		m.listProviders = updatedModel.(components.ModelProvidersList)
		return m, cmd
	case "right":
		m.listProviders.NextPage()
		return m, nil
	case "left":
		m.listProviders.PrevPage()
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.listProviders.Filter(m.textarea.Value())
		m.syncLayout()
		return m, cmd
	}
}

func (m model) handleProviderApiKeyWin(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.isAddingApiKey = false
		m.textarea.Placeholder = "Type your message here..."
		baseurl := m.listProviders.Current().BaseUrl
		apiKey := m.textarea.Value()
		for _, provider := range config.AllProviders() {
			if provider.BaseUrl == baseurl {
				config.UpdateModelsForProvider(baseurl, provider.Models, apiKey)
				break
			}
		}
		m.textarea.Reset()
		m.textarea.Focus()
		(&m).syncLayout()
		return m, nil

	case "esc":
		m.isAddingApiKey = false
		m.textarea.Placeholder = "Type your message here..."
		m.textarea.Reset()
		m.textarea.Focus()
		(&m).syncLayout()
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}
