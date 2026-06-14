package views

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/api"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
)

type refreshSessionsMsg struct{}
type compactMemoryDoneMsg struct {
	sessionID   string
	contextSize int64
	err         error
}

func compactMemoryCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		contextSize, err := client.CompactMemory(sessionID)
		return compactMemoryDoneMsg{sessionID: sessionID, contextSize: contextSize, err: err}
	}
}

func CmdHandler(cmd string, m *model) tea.Cmd {
	switch cmd {
	case "/sessions":
		sessions, err := client.ListSession()
		if err != nil {
			m.sessions = []models.Session{}
		} else {
			m.sessions = sessions
		}
		m.listSession.Refresh(m.sessions)
		m.islistSessionWin = true
		m.textarea.Reset()
	case "/new_session":
		resetCurrentSession(m)
		return tea.Batch(func() tea.Msg { return refreshSessionsMsg{} }, refreshGitStatusCmd(m.gitStatusDirectory()))

	case "/delete_session":
		deleteCurrentSession(m)
		return tea.Batch(func() tea.Msg { return refreshSessionsMsg{} }, refreshGitStatusCmd(m.gitStatusDirectory()))

	case "/export":
		path, err := exportCurrentSessionMarkdown(m, m.currentSession)
		if err != nil {
			appendCommandStatusMessage(m, fmt.Sprintf("Export failed: %s", err.Error()))
			return nil
		}
		home, _ := os.UserHomeDir()
		appendCommandStatusMessage(m, fmt.Sprintf("Exported session to %s", strings.Replace(path, home, "~", 1)))
		return nil

	case "/models":
		openModelsList(m)
		return nil

	case "/login":
		openLoginProviderList(m)
		return nil

	case "/logout":
		openLogoutProviderList(m)
		return nil

	case "/effort":
		openEffortList(m)
		return nil

	case "/usage":
		appendUsageMessage(m)
		return nil
	case "/skills":
		appendSkillsMessage(m)
	case "/dir":
		appendDirMessage(m)
	case "/compact":
		if m.currentSession.ID == "" {
			appendCommandStatusMessage(m, "No active session selected.")
			return nil
		}
		appendCommandStatusMessage(m, "\nCompacting...\n")
		m.isCompacting = true
		m.isGenerating = true
		m.syncLayout()
		m.viewport.GotoBottom()
		return tea.Batch(compactMemoryCmd(m.currentSession.ID), m.spinner.Tick)

	default:
		return nil
	}
	if m.isGenerating {
		return waitForMessages(m.streamCh)
	}
	return nil
}

func resetCurrentSession(m *model) {
	m.currentSession = models.Session{ID: "", Title: "", Directory: "."}
	m.messages = []models.Message{}
	m.completeMessages = []models.Message{}
	m.currentContextSize = 0
	m.gitStatus = defaultStatusLineGitInfo(m.gitStatusDirectory())
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.textarea.Reset()
	m.viewport.GotoBottom()
}

func deleteCurrentSession(m *model) {
	sessionID := m.currentSession.ID
	client.DeleteSession(sessionID)
	var newSessions []models.Session
	for _, session := range m.sessions {
		if session.ID != sessionID {
			newSessions = append(newSessions, session)
		}
	}
	m.sessions = newSessions
	resetCurrentSession(m)
}

type item config.ResModel

func (i item) FilterValue() string { return i.Model }

func loadModelsList() ([]api.ModelInfo, error) {
	modelsList, recentModels, err := client.GetModels()
	if err != nil {
		return []api.ModelInfo{}, err
	}

	sort.Slice(recentModels, func(i, j int) bool {
		return recentModels[i].LastUsed < recentModels[j].LastUsed
	})
	for _, recentModel := range recentModels {
		modelsList = append(modelsList, recentModel)
	}
	slices.Reverse(modelsList)
	return dedupeModels(modelsList), nil
}

func dedupeModels(modelsList []api.ModelInfo) []api.ModelInfo {
	seen := map[string]bool{}
	result := make([]api.ModelInfo, 0, len(modelsList))
	for _, model := range modelsList {
		key := model.BaseUrl + "\x00" + model.Model
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, model)
	}
	return result
}

func openModelsList(m *model) {
	modelsList, err := loadModelsList()
	if err != nil {
		modelsList = []api.ModelInfo{}
	}
	m.textarea.Reset()
	m.modelsList = modelsList
	m.listModels.Refresh(modelsList)
	m.listModels.Filter("")
	m.isModelsListWin = true
	m.textarea.Placeholder = "type to filter · ↑↓ select · Enter/Esc close"
	m.textarea.Focus()
	m.syncLayout()
}

func appendUsageMessage(m *model) {
	usageContent := buildUsageContent(m, m.currentSession)
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: usageContent}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}
func appendSkillsMessage(m *model) {
	available_skills := client.GetAvailableSkills(m.currentSession.ID)
	formatted_skills_list := lipgloss.NewStyle().Render("Available Skills: \n" + strings.Join(available_skills, ", "))
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: formatted_skills_list}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}
func appendDirMessage(m *model) {
	var dir string
	if m.currentSession.Directory == "" {
		dir = "start a session"
	} else {
		dir = shortenDir(m.currentSession.Directory)
	}
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: dir}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}

func buildUsageContent(m *model, session models.Session) string {
	if session.ID == "" {
		return "## Session Usage\n\nNo active session selected."
	}

	sessionMessages, err := client.GetSessionData(session.ID)
	if err != nil {
		m.isError = true
		m.errorMessage = "Unable to get usage data"
	}
	var promptTokens int64
	var completionTokens int64
	var totalTokens int64
	messageCount := 0
	for _, msg := range sessionMessages {
		data := models.DecodeMessageData(msg.Data)
		if data.Usage == nil {
			continue
		}
		promptTokens += data.Usage.PromptTokens
		completionTokens += data.Usage.CompletionTokens
		totalTokens += data.Usage.TotalTokens
		messageCount++
	}

	if messageCount == 0 {
		return "## Session Usage\n\nNo token usage has been recorded for this session yet."
	}

	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = "Untitled Session"
	}

	return fmt.Sprintf(
		"## Session Usage\n\n- Session: %s\n- Prompt tokens: %d\n- Completion tokens: %d\n- Total tokens: %d\n",
		title,
		promptTokens,
		completionTokens,
		totalTokens,
	)
}
