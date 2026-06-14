package views

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
	"golang.design/x/clipboard"
)

func parseQuestions(content string) []questionItem {
	var args map[string]any
	if err := json.Unmarshal([]byte(content), &args); err != nil {
		return nil
	}
	rawQuestions, ok := args["question"].([]any)
	if !ok {
		return nil
	}
	var questions []questionItem
	for _, item := range rawQuestions {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		q := questionItem{}
		q.question, _ = obj["question"].(string)
		if rawOpts, ok := obj["options"].([]any); ok {
			for _, o := range rawOpts {
				if s, ok := o.(string); ok {
					q.options = append(q.options, s)
				}
			}
		}
		if q.question != "" {
			questions = append(questions, q)
		}
	}
	return questions
}

func (m model) renderQueueList() string {
	header := styleTodoTitle.Render("Queue")
	lines := []string{header}
	if len(m.queue) == 0 {
		lines = append(lines, styleOptionNormal.Render(" no models configured"))
		return strings.Join(lines, "\n")
	}
	for i, ent := range m.queue {
		label := ent
		lines = append(lines, styleOptionNormal.Render(" "+strconv.Itoa(i+1)+".) "+label.prompt))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderQuestionUI() string {
	if !m.questionMode || m.questionIdx >= len(m.questions) {
		return ""
	}
	q := m.questions[m.questionIdx]
	var lines []string
	header := fmt.Sprintf("? (%d/%d) %s", m.questionIdx+1, len(m.questions), q.question)
	lines = append(lines, styleQuestionHeader.Render(header))
	for i, opt := range q.options {
		if i == m.questionSelected {
			lines = append(lines, styleOptionSelected.Render("  ▸ "+opt))
		} else {
			lines = append(lines, styleOptionNormal.Render("    "+opt))
		}
	}
	return strings.Join(lines, "\n")
}

func (m model) questionUIHeight() int {
	if !m.questionMode || m.questionIdx >= len(m.questions) {
		return 0
	}
	h := 1
	h += len(m.questions[m.questionIdx].options)
	return h
}

func (m model) handleQuestionInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.questionMode = false
		m.textarea.Placeholder = "Send a message..."
		m.textarea.Focus()
		m.syncLayout()
		return m, nil
	case "enter":
		q := m.questions[m.questionIdx]
		if len(q.options) > 0 {
			m.questionAnswers[m.questionIdx] = q.options[m.questionSelected]
		} else {
			m.questionAnswers[m.questionIdx] = m.textarea.Value()
			m.textarea.SetValue("")
			m.textarea.Reset()
		}
		m.questionIdx++
		if m.questionIdx >= len(m.questions) {
			return m.submitQuestionAnswers()
		}
		m.questionSelected = 0
		next := m.questions[m.questionIdx]
		if len(next.options) > 0 {
			m.textarea.Placeholder = "↑↓ select · Enter confirm"
			m.textarea.Blur()
		} else {
			m.textarea.Placeholder = "Type your answer..."
			m.textarea.Focus()
		}
		m.syncLayout()
		return m, nil
	case "up":
		if len(m.questions[m.questionIdx].options) > 0 && m.questionSelected > 0 {
			m.questionSelected--
		}
		return m, nil
	case "down":
		q := m.questions[m.questionIdx]
		if len(q.options) > 0 && m.questionSelected < len(q.options)-1 {
			m.questionSelected++
		}
		return m, nil
	default:
		if len(m.questions[m.questionIdx].options) == 0 {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

func (m model) handleApiKeyWin(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":

		client.SetApiKey(m.modelsList[m.modelsListIndex], m.textarea.Value())
		m.textarea.SetValue("")
		m.enter_api_win = false
		m.textarea.Placeholder = "Send a message..."
		m.textarea.Focus()
		m.syncLayout()
		return m, nil
	case "esc", "ctrl+c":
		m.enter_api_win = false
		m.textarea.SetValue("")
		m.textarea.Placeholder = "Send a message..."
		m.textarea.Focus()
		m.syncLayout()
		return m, nil
	case "ctrl+v", "meta+v":
		curVal := m.textarea.Value()
		err := clipboard.Init()
		if err != nil {
			panic(err)
		}
		textBytes := clipboard.Read(clipboard.FmtText)
		m.textarea.SetValue(curVal + string(textBytes))
		return m, nil

	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}

func (m model) submitQuestionAnswers() (tea.Model, tea.Cmd) {
	m.questionMode = false
	m.textarea.Placeholder = "Send a message..."
	m.textarea.Focus()

	var parts []string
	for i, q := range m.questions {
		parts = append(parts, fmt.Sprintf("Q: %s\nA: %s", q.question, m.questionAnswers[i]))
	}
	answer := strings.Join(parts, "\n\n")
	return m, tea.Batch(refreshGitStatusCmd(m.gitStatusDirectory()), m.beginGeneration(answer))
}
