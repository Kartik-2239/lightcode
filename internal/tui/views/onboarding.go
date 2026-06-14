package views

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
)

type onbTickMsg time.Time

func onbTick() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
		return onbTickMsg(t)
	})
}

type onboardingStage int

const (
	stageSelect onboardingStage = iota
	stageKeys
)

type onboardingProvider struct {
	name        string
	description string
	selected    bool
}

type onboardingModel struct {
	stage           onboardingStage
	providers       []onboardingProvider
	cursor          int
	selected        []string // provider names, in display order
	keyIndex        int
	keys            map[string]string
	baseUrls        map[string]string   // for the "other" provider, the typed base URL
	models          map[string][]string // for the "other" provider, the typed model list
	enteringBaseUrl bool                // collecting the base URL for "other"
	enteringModels  bool                // collecting the model list for "other"
	input           textinput.Model
	showEmpty       bool   // "select at least one" warning
	frame           int    // animation frame for pending dots
	validating      bool   // a key check is in flight for the current provider
	validateErr     string // error from the last key check, shown under the input
	done            bool
	err             error
}

// keyValidatedMsg carries the result of an async API-key check.
type keyValidatedMsg struct {
	index int
	err   error
}

func validateKeyCmd(index int, name, key string) tea.Cmd {
	return func() tea.Msg {
		return keyValidatedMsg{index: index, err: config.ValidateProviderKey(name, key)}
	}
}

func newOnboardingModel() onboardingModel {
	ti := textinput.New()
	ti.Placeholder = "paste API key (or leave blank to add later)"
	ti.Prompt = ""
	ti.SetWidth(60)

	return onboardingModel{
		stage: stageSelect,
		providers: []onboardingProvider{
			{name: "codex", description: "Codex       use ChatGPT login from codex login"},
			{name: "openrouter", description: "OpenRouter   300+ models via one API key"},
			{name: "openai", description: "OpenAI       e.g. GPT-5"},
			{name: "anthropic", description: "Anthropic    e.g. Opus 4.8"},
			{name: "other", description: "Other        custom OpenAI-compatible endpoint"},
		},
		keys:     map[string]string{},
		baseUrls: map[string]string{},
		models:   map[string][]string{},
		input:    ti,
	}
}

func (m onboardingModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, onbTick())
}

func (m onboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case onbTickMsg:
		m.frame++
		return m, onbTick()
	case keyValidatedMsg:
		if msg.index != m.keyIndex || !m.validating {
			return m, nil // stale result
		}
		m.validating = false
		if msg.err != nil {
			m.validateErr = friendlyKeyError(msg.err)
			m.input.Focus()
			return m, textinput.Blink
		}
		return m.advanceKey()
	case tea.KeyPressMsg:
		switch m.stage {
		case stageSelect:
			return m.updateSelect(msg)
		case stageKeys:
			return m.updateKeys(msg)
		}
	}
	// Forward any other message (bracketed paste, the clipboard result of
	// ctrl+v, cursor blink) to the key input.
	if m.stage == stageKeys {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m onboardingModel) updateSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.providers)-1 {
			m.cursor++
		}
	case " ", "space":
		m.providers[m.cursor].selected = !m.providers[m.cursor].selected
		m.showEmpty = false
	case "enter":
		m.selected = nil
		for _, p := range m.providers {
			if p.selected {
				m.selected = append(m.selected, p.name)
			}
		}
		if len(m.selected) == 0 {
			m.showEmpty = true
			return m, nil
		}
		// move to API-key entry for the first selected provider
		m.stage = stageKeys
		m.keyIndex = 0
		m.setupProvider()
		return m, textinput.Blink
	}
	return m, nil
}

// setupProvider configures the input for the current provider. For "other" it
// first collects a base URL; for known providers it goes straight to the key.
func (m *onboardingModel) setupProvider() {
	m.validateErr = ""
	m.input.SetValue("")
	m.input.Focus()
	m.enteringBaseUrl = false
	m.enteringModels = false
	if m.selected[m.keyIndex] == "other" {
		m.enteringBaseUrl = true
		m.input.Placeholder = "enter base URL (e.g. https://api.example.com/v1)"
		return
	}
	m.input.Placeholder = keyPlaceholder(m.selected[m.keyIndex])
}

// parseModels splits a comma-separated model list, trimming blanks.
func parseModels(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// pendingDots returns a cycling "", ".", "..", "..." padded to a stable width.
func pendingDots(frame int) string {
	n := frame % 4
	return strings.Repeat(".", n) + strings.Repeat(" ", 3-n)
}

// keyPlaceholder spells out which provider's key to paste, with an example prefix.
func keyPlaceholder(provider string) string {
	switch provider {
	case "openrouter":
		return "paste your OpenRouter API key (sk-or-v1-...)"
	case "openai":
		return "paste your OpenAI API key (sk-...)"
	case "anthropic":
		return "paste your Anthropic API key (sk-ant-...)"
	case "codex":
		return "press Enter to import ~/.codex/auth.json"
	default:
		return "paste your " + provider + " API key"
	}
}

func (m onboardingModel) updateKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.validating {
		// ignore input while a key check is in flight (except ctrl+c)
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// go back to provider selection
		m.stage = stageSelect
		m.validateErr = ""
		m.input.Blur()
		return m, nil
	case "enter":
		name := m.selected[m.keyIndex]
		value := strings.TrimSpace(m.input.Value())
		if m.enteringBaseUrl {
			// "other" step 1: capture the base URL, then ask for models
			if value == "" {
				m.validateErr = "base URL is required"
				return m, nil
			}
			m.baseUrls[name] = value
			m.enteringBaseUrl = false
			m.enteringModels = true
			m.validateErr = ""
			m.input.SetValue("")
			m.input.Placeholder = "enter model(s), comma-separated (e.g. gpt-4o, llama-3.1-70b)"
			return m, nil
		}
		if m.enteringModels {
			// "other" step 2: capture the model list, then ask for the key
			parsed := parseModels(value)
			if len(parsed) == 0 {
				m.validateErr = "add at least one model"
				return m, nil
			}
			m.models[name] = parsed
			m.enteringModels = false
			m.validateErr = ""
			m.input.SetValue("")
			m.input.Placeholder = "paste API key (or leave blank if not needed)"
			return m, nil
		}
		m.keys[name] = value
		if name == "codex" {
			if err := config.ImportCodexAuth(); err != nil {
				m.validateErr = friendlyKeyError(fmt.Errorf("run codex login first: %w", err))
				return m, nil
			}
			return m.advanceKey()
		}
		if name == "other" {
			// custom endpoint: no known model to test against, so save as-is
			return m.advanceKey()
		}
		if value == "" {
			// blank = skip, no key to validate
			return m.advanceKey()
		}
		m.validating = true
		m.validateErr = ""
		return m, validateKeyCmd(m.keyIndex, name, value)
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// advanceKey moves to the next selected provider, or finishes onboarding by
// writing the config once every provider has been handled.
func (m onboardingModel) advanceKey() (tea.Model, tea.Cmd) {
	m.validateErr = ""
	m.keyIndex++
	if m.keyIndex >= len(m.selected) {
		m.err = config.CreateConfig(m.selected, m.keys, m.baseUrls, m.models)
		m.done = true
		return m, tea.Quit
	}
	m.setupProvider()
	return m, textinput.Blink
}

// friendlyKeyError shortens provider errors to a single readable line.
func friendlyKeyError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "context deadline exceeded") {
		return "request timed out — check your connection and try again"
	}
	if i := strings.Index(msg, "\n"); i >= 0 {
		msg = msg[:i]
	}
	if len(msg) > 80 {
		msg = msg[:80] + "…"
	}
	return msg
}

var (
	onbTitle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	onbSubtitle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	onbHint     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	onbChecked  = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
	onbCursor   = lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta).Bold(true)
	onbNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	onbErr      = lipgloss.NewStyle().Foreground(lipgloss.BrightRed)
	onbDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m onboardingModel) View() tea.View {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(onbTitle.Render("Welcome to Lightcode!"))
	sb.WriteString("\n\n")

	switch m.stage {
	case stageSelect:
		sb.WriteString(m.viewSelect())
	case stageKeys:
		sb.WriteString(m.viewKeys())
	}

	return tea.NewView(sb.String())
}

func (m onboardingModel) viewSelect() string {
	var sb strings.Builder
	sb.WriteString(onbSubtitle.Render("Step 1/2 · Select the providers you plan to use:"))
	sb.WriteString("\n\n")

	for i, p := range m.providers {
		cursor := "  "
		if i == m.cursor {
			cursor = onbCursor.Render("▸ ")
		}
		checkbox := "[ ]"
		style := onbNormal
		if p.name == "other" {
			style = onbDone // dimmer than the main providers
		}
		if p.selected {
			checkbox = "[✓]"
			style = onbChecked
		}
		sb.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, style.Render(p.description)))
	}

	sb.WriteString("\n")
	sb.WriteString(onbHint.Render("↑↓ navigate · Space toggle · Enter continue"))
	if m.showEmpty {
		sb.WriteString("\n")
		sb.WriteString(onbErr.Render("Select at least one provider"))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m onboardingModel) viewKeys() string {
	var sb strings.Builder
	sb.WriteString(onbSubtitle.Render(fmt.Sprintf("Step 2/2 · Configure providers (%d/%d):", m.keyIndex+1, len(m.selected))))
	sb.WriteString("\n\n")

	for i, name := range m.selected {
		switch {
		case i < m.keyIndex:
			status := "saved"
			if name == "other" {
				status = m.baseUrls[name] // show the custom endpoint
			} else if m.keys[name] == "" {
				status = "skipped"
			}
			marker := onbDone.Render("✓ ")
			label := onbDone.Render(fmt.Sprintf("%-12s", name))
			sb.WriteString(fmt.Sprintf("%s%s %s\n", marker, label, onbDone.Render(status)))
		case i == m.keyIndex:
			marker := onbCursor.Render("❯ ")
			label := onbChecked.Render(fmt.Sprintf("%-12s", name))
			trailing := m.input.View()
			if m.validating {
				trailing = onbDone.Render("validating" + pendingDots(m.frame))
			}
			sb.WriteString(fmt.Sprintf("%s%s %s\n", marker, label, trailing))
			if m.validateErr != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", onbErr.Render("✗ "+m.validateErr)))
			}
		default:
			marker := "  "
			label := onbNormal.Render(fmt.Sprintf("%-12s", name))
			sb.WriteString(fmt.Sprintf("%s%s %s\n", marker, label, onbDone.Render("pending"+pendingDots(m.frame))))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(onbHint.Render("Enter to verify & save · blank to skip keys · Esc back"))
	sb.WriteString("\n")
	return sb.String()
}

func RunOnboarding() error {
	p := tea.NewProgram(newOnboardingModel())
	result, err := p.Run()
	if err != nil {
		return err
	}
	m, ok := result.(onboardingModel)
	if !ok {
		return fmt.Errorf("unexpected onboarding model type")
	}
	if m.err != nil {
		return m.err
	}
	if !m.done {
		fmt.Fprintln(os.Stderr, "Onboarding cancelled.")
		os.Exit(0)
	}
	return nil
}
