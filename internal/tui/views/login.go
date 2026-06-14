package views

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/oauth"
)

type loginProvider struct {
	name        string
	description string
	args        []string
}

type loginFinishedMsg struct {
	provider loginProvider
	err      error
}

type logoutFinishedMsg struct {
	provider loginProvider
	err      error
}

type copilotLoginStartedMsg struct {
	provider  loginProvider
	device    oauth.DeviceCodeResponse
	codexFlow *oauth.CodexLoginFlow
	openErr   error
	err       error
}

func defaultLoginProviders() []loginProvider {
	return []loginProvider{
		{
			name:        "codex",
			description: "ChatGPT device login",
			args:        []string{"login"},
		},
		{
			name:        "copilot",
			description: "GitHub Copilot device login",
		},
	}
}

func defaultLogoutProviders() []loginProvider {
	return []loginProvider{
		{
			name:        "codex",
			description: "ChatGPT OAuth",
			args:        []string{"logout"},
		},
		{
			name:        "copilot",
			description: "GitHub Copilot OAuth",
		},
	}
}

func openLoginProviderList(m *model) {
	m.textarea.Reset()
	m.loginProviders = defaultLoginProviders()
	m.isLoginProviderWin = true
	m.loginProviderIndex = 0
	m.loginAction = "login"
	m.textarea.Placeholder = "↑↓ select · Enter login · Esc close"
	m.textarea.Focus()
	m.syncLayout()
}

func openLogoutProviderList(m *model) {
	m.textarea.Reset()
	m.loginProviders = defaultLogoutProviders()
	m.isLoginProviderWin = true
	m.loginProviderIndex = 0
	m.loginAction = "logout"
	m.textarea.Placeholder = "↑↓ select · Enter logout · Esc close"
	m.textarea.Focus()
	m.syncLayout()
}

func (m model) handleLoginProviderInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.isLoginProviderWin = false
		m.textarea.Placeholder = "Send a message..."
		m.syncLayout()
		return m, nil
	case "up", "k":
		if m.loginProviderIndex > 0 {
			m.loginProviderIndex--
		}
		return m, nil
	case "down", "j":
		if m.loginProviderIndex < len(m.loginProviders)-1 {
			m.loginProviderIndex++
		}
		return m, nil
	case "enter":
		if len(m.loginProviders) == 0 {
			return m, nil
		}
		provider := m.loginProviders[m.loginProviderIndex]
		m.isLoginProviderWin = false
		m.textarea.Placeholder = "Send a message..."
		m.syncLayout()
		if m.loginAction == "logout" {
			return m, runLogoutProviderCmd(provider)
		}
		return m, runLoginProviderCmd(provider)
	}
	return m, nil
}

func runLoginProviderCmd(provider loginProvider) tea.Cmd {
	if provider.name == "copilot" {
		return startCopilotLoginCmd(provider)
	}
	if provider.name == "codex" || provider.name == "codex-device" {
		return startCodexLoginCmd(provider)
	}
	return func() tea.Msg {
		return loginFinishedMsg{provider: provider, err: fmt.Errorf("login provider %q is not supported", provider.name)}
	}
}

func runLogoutProviderCmd(provider loginProvider) tea.Cmd {
	if provider.name == "copilot" {
		err := clearCopilotAuthState()
		return func() tea.Msg {
			return logoutFinishedMsg{provider: provider, err: err}
		}
	}
	if provider.name != "codex" {
		return func() tea.Msg {
			return logoutFinishedMsg{provider: provider, err: fmt.Errorf("logout provider %q is not supported", provider.name)}
		}
	}
	if err := clearCodexAuthState(); err != nil {
		return func() tea.Msg {
			return logoutFinishedMsg{provider: provider, err: err}
		}
	}
	return func() tea.Msg {
		return logoutFinishedMsg{provider: provider, err: nil}
	}
}

func startCopilotLoginCmd(provider loginProvider) tea.Cmd {
	return func() tea.Msg {
		device, err := oauth.StartAuthFlow()
		if err != nil {
			return copilotLoginStartedMsg{provider: provider, err: err}
		}
		return copilotLoginStartedMsg{
			provider: provider,
			device:   device,
			openErr:  openURL(device.VerificationURI),
		}
	}
}

func startCodexLoginCmd(provider loginProvider) tea.Cmd {
	return func() tea.Msg {
		if err := config.DeleteAuthVal(config.CodexAuthProvider); err != nil {
			return copilotLoginStartedMsg{provider: provider, err: err}
		}
		flow, err := oauth.StartCodexLoginFlow()
		if err != nil {
			return copilotLoginStartedMsg{provider: provider, err: err}
		}
		msg := copilotLoginStartedMsg{
			provider:  provider,
			codexFlow: &flow,
			openErr:   openURL(flow.AuthURL),
		}
		if flow.Device != nil {
			msg.device = *flow.Device
		}
		return msg
	}
}

func pollCopilotLoginCmd(provider loginProvider, device oauth.DeviceCodeResponse) tea.Cmd {
	return func() tea.Msg {
		token, err := oauth.WaitForAccessToken(device)
		if err == nil {
			err = oauth.SaveAccessToken(token.AccessToken)
		}
		return loginFinishedMsg{provider: provider, err: err}
	}
}

func pollCodexLoginCmd(provider loginProvider, flow oauth.CodexLoginFlow) tea.Cmd {
	return func() tea.Msg {
		authVal, err := oauth.WaitForCodexLogin(flow)
		if err == nil {
			err = oauth.SaveCodexAuth(authVal)
		}
		return loginFinishedMsg{provider: provider, err: err}
	}
}

func openURL(uri string) error {
	if strings.TrimSpace(uri) == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", uri)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", uri)
	default:
		cmd = exec.Command("xdg-open", uri)
	}
	return cmd.Start()
}

func (m model) handleCopilotLoginStarted(msg copilotLoginStartedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		appendCommandStatusMessage(&m, fmt.Sprintf("Login failed for %s: %s", msg.providerLabel(), msg.err.Error()))
		return m, nil
	}
	if msg.openErr != nil {
		appendCommandStatusMessage(&m, fmt.Sprintf("Could not open browser automatically: %s", msg.openErr.Error()))
	}
	switch msg.provider.name {
	case "codex", "codex-device":
		if msg.codexFlow == nil {
			appendCommandStatusMessage(&m, "Login failed for codex: codex login flow was not started")
			return m, nil
		}
		if msg.codexFlow.Device != nil {
			appendCommandStatusMessage(&m, fmt.Sprintf("Codex login: enter %s at %s. Waiting for authorization...", msg.device.UserCode, msg.device.VerificationURI))
		} else {
			appendCommandStatusMessage(&m, fmt.Sprintf("Codex login: complete sign-in in the browser at %s. Waiting for authorization...", msg.codexFlow.AuthURL))
		}
		return m, pollCodexLoginCmd(msg.provider, *msg.codexFlow)
	default:
		appendCommandStatusMessage(&m, fmt.Sprintf("GitHub Copilot login: enter %s at %s. Waiting for authorization...", msg.device.UserCode, msg.device.VerificationURI))
		return m, pollCopilotLoginCmd(msg.provider, msg.device)
	}
}

func (m model) handleLoginFinished(msg loginFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		appendCommandStatusMessage(&m, fmt.Sprintf("Login failed for %s: %s", msg.providerLabel(), msg.err.Error()))
		return m, nil
	}
	switch msg.provider.name {
	case "codex", "codex-device":
		modelsList, err := loadModelsList()
		if err == nil {
			m.modelsList = modelsList
			m.listModels.Refresh(modelsList)
			m.listModels.Filter("")
		}
		appendCommandStatusMessage(&m, "Codex login complete. Codex auth models are available in /models.")
		return m, nil
	case "copilot":
		modelsList, err := loadModelsList()
		if err == nil {
			m.modelsList = modelsList
			m.listModels.Refresh(modelsList)
			m.listModels.Filter("")
		}
		appendCommandStatusMessage(&m, "GitHub Copilot login complete. Copilot models are available in /models.")
		return m, nil
	}
	appendCommandStatusMessage(&m, fmt.Sprintf("Login complete for %s.", msg.providerLabel()))
	return m, nil
}

func (m model) handleLogoutFinished(msg logoutFinishedMsg) (tea.Model, tea.Cmd) {
	if err := clearAuthStateForProvider(msg.provider.name); err != nil && msg.err == nil {
		msg.err = err
	}
	modelsList, err := loadModelsList()
	if err == nil {
		m.modelsList = modelsList
		m.listModels.Refresh(modelsList)
		m.listModels.Filter("")
		m.modelsListIndex = 0
	}
	if msg.err != nil {
		appendCommandStatusMessage(&m, fmt.Sprintf("Logout failed for %s: %s", msg.providerLabel(), msg.err.Error()))
		return m, nil
	}
	appendCommandStatusMessage(&m, fmt.Sprintf("Logout complete for %s.", msg.providerLabel()))
	return m, nil
}

func clearAuthStateForProvider(provider string) error {
	switch provider {
	case "codex", "codex-device":
		return clearCodexAuthState()
	case "copilot":
		return clearCopilotAuthState()
	default:
		return nil
	}
}

func clearCodexAuthState() error {
	if err := config.DeleteAuthVal(config.CodexAuthProvider); err != nil {
		return err
	}
	return config.ClearModelProvider(config.CodexAuthProvider)
}

func clearCopilotAuthState() error {
	if err := config.DeleteAuthVal(config.CopilotAuthProvider); err != nil {
		return err
	}
	if err := config.DeleteAuthVal("github"); err != nil {
		return err
	}
	if err := config.ClearModelProvider(config.CopilotAuthProvider); err != nil {
		return err
	}
	return config.ClearModelProvider("github")
}

func (msg loginFinishedMsg) providerLabel() string {
	label := strings.TrimSpace(msg.provider.name)
	if label == "" {
		return "provider"
	}
	return label
}

func (msg copilotLoginStartedMsg) providerLabel() string {
	label := strings.TrimSpace(msg.provider.name)
	if label == "" {
		return "provider"
	}
	return label
}

func (msg logoutFinishedMsg) providerLabel() string {
	label := strings.TrimSpace(msg.provider.name)
	if label == "" {
		return "provider"
	}
	return label
}

func (m model) renderLoginProviderList() string {
	var sb strings.Builder
	title := "Login provider"
	help := "↑↓ select · Enter login · Esc close"
	if m.loginAction == "logout" {
		title = "Logout provider"
		help = "↑↓ select · Enter logout · Esc close"
	}
	sb.WriteString(loginTitleStyle.Render(title))
	sb.WriteString("\n")
	for i, provider := range m.loginProviders {
		cursor := "  "
		style := loginItemStyle
		if i == m.loginProviderIndex {
			cursor = loginCursorStyle.Render("→ ")
			style = loginSelectedItemStyle
		}
		sb.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, style.Render(provider.name), loginDescStyle.Render(provider.description)))
	}
	sb.WriteString(loginHelpStyle.Render(help))
	return sb.String()
}

func loginProviderListHeight(items int) int {
	if items < 1 {
		items = 1
	}
	return items + 2
}

var (
	loginTitleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	loginCursorStyle       = lipgloss.NewStyle().Foreground(lipgloss.BrightCyan)
	loginItemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	loginSelectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Bold(true)
	loginDescStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	loginHelpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)
