package views

import (
	"context"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
)

type streamMessageMsg models.StoredMessageData
type streamDoneMsg struct{}
type clearEscMsgMsg struct{}

func waitForMessages(ch chan models.StoredMessageData) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		// if msg.Role == "error" {
		// 	return streamDoneMsg{}
		// }
		if !ok {
			return streamDoneMsg{}
		}
		return streamMessageMsg(msg)
	}
}

func clearEscMsg() tea.Cmd {
	return func() tea.Msg {
		return clearEscMsgMsg{}
	}
}

func (m *model) refreshMessagesView() {
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
}

func (m *model) ensureCurrentSession(prompt string) {
	if m.currentSession.ID != "" {
		return
	}
	dir, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	sessionID, err := client.CreateSession(prompt)
	if err != nil {
		return
	}
	m.currentSession = models.Session{ID: sessionID, Title: prompt, Directory: strings.Replace(dir, home, "~", 1)}
	client.Reverse(m.sessions)
	m.sessions = append(m.sessions, m.currentSession)
	client.Reverse(m.sessions)
	m.listSession.Refresh(m.sessions)
}

func streamError() tea.Cmd {
	return func() tea.Msg {
		return streamDoneMsg{}
	}
}
func (m *model) beginGeneration(prompt string) tea.Cmd {
	if len(m.modelsList) == 0 || m.modelsListIndex < 0 || m.modelsListIndex >= len(m.modelsList) {
		appendCommandStatusMessage(m, "No provider is logged in. Run `/login` to select and authenticate a provider.")
		return nil
	}
	m.isGenerating = true
	m.syncLayout()

	textareaValue, img_bytes := createPrompt(strings.Trim(prompt, "\n"), m)
	m.clearPastedInput()
	m.syncLayout()
	newMessage, err := client.SendMessage(m.currentSession.ID, textareaValue, img_bytes)
	if err != nil {
		m.isError = true
		m.errorMessage = "Unable to send a message"
		return streamError()
	} else {
		m.messages = append(m.messages, newMessage)
	}
	m.refreshMessagesView()

	ctx, cancel := context.WithCancel(context.Background())
	// ch := client.ChatCompletion(ctx, m.currentSession.ID, textareaValue, m.mode, img_bytes)
	ch := client.ChatCompletion(ctx, m.currentSession.ID, "", m.mode, [][]byte{})

	m.queue = []queue{}

	m.cancelStream = cancel
	m.streamCh = ch
	m.textarea.SetValue("")
	m.textarea.Reset()
	return tea.Batch(waitForMessages(ch), m.spinner.Tick)
}

func (m *model) cancelActiveGeneration() {
	m.queue = m.queue[:0]
	m.isGenerating = false
	m.cancelStream()
	m.cancelStream = nil
	m.streamCh = nil
	m.syncLayout()
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data: models.EncodeMessageData(models.StoredMessageData{
			Role:    "assistant",
			Content: "*Generation stopped.*",
		}),
	})
	m.refreshMessagesView()
	m.syncLayout()
	m.showEscMsg = false
	m.lastEsc = time.Time{}
}

func (m *model) runNextQueuedPrompt() tea.Cmd {
	if len(m.queue) == 0 {
		return nil
	}
	nextPrompt := m.queue[0]
	m.queue = m.queue[1:]
	return m.beginGeneration(nextPrompt.prompt)
}
