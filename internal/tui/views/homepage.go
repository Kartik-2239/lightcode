package views

import (
	"context"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/api"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
	"github.com/Kartik-2239/lightcode/internal/tui/components"
	"github.com/aymanbagabas/go-nativeclipboard"
	"github.com/charmbracelet/x/term"
	"golang.design/x/clipboard"
)

const textareaPrompt = "❯ "

func LauchHomePage() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}

type questionItem struct {
	question string
	options  []string
}

type queue struct {
	prompt string
	imgs   [][]byte
}

type kittyPreview struct {
	id   int
	cols int
	rows int
}

type model struct {
	viewport           viewport.Model
	islistSessionWin   bool
	islistCommandsWin  bool
	listSession        components.Model
	listCommands       components.ModelCmdList
	listModels         components.ModelModelsList
	sessions           []models.Session
	currentSession     models.Session
	messages           []models.Message
	completeMessages   []models.Message
	textarea           textarea.Model
	pasteCounter       int
	pastedTexts        map[int]string
	imgPasteCounter    int
	pastedImgs         map[int][]byte
	pastedImgPreviews  map[int]kittyPreview
	senderStyle        lipgloss.Style
	err                error
	cache              map[int]string
	cacheIndex         int
	streamCh           chan models.StoredMessageData
	width              int
	height             int
	bashMode           bool
	streamch           chan models.StoredMessageData
	cancelStream       context.CancelFunc
	spinner            spinner.Model
	isGenerating       bool
	lastEsc            time.Time
	showEscMsg         bool
	questionMode       bool
	questions          []questionItem
	questionIdx        int
	questionAnswers    []string
	questionSelected   int
	todoList           []models.ToDo
	modes              []string
	mode               string
	modelsList         []api.ModelInfo
	isModelsListWin    bool
	isLoginProviderWin bool
	loginProviders     []loginProvider
	loginProviderIndex int
	loginAction        string
	isEffortWin        bool
	effortOptions      []effortOption
	effortIndex        int
	modelsListIndex    int
	isCompacting       bool
	queue              []queue
	currentContextSize int64
	gitStatus          statusLineGitInfo
	enter_api_win      bool
	isError            bool
	errorMessage       string
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "enter to send message · / commands"
	ta.SetVirtualCursor(true)
	ta.Focus()
	ta.Prompt = ""

	// ta.SetPromptFunc(1, func(pi textarea.PromptInfo) string {
	// 	if pi.LineNumber == 0 {
	// 		return "❯ "
	// 	}
	// 	return " "
	// })

	ta.CharLimit = 32000

	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()

	// s.Focused.Base = s.Focused.Base.
	// 	Border(lipgloss.NormalBorder()).
	// 	BorderTop(true).
	// 	BorderBottom(true).
	// 	BorderRight(false).
	// 	BorderLeft(false)

	width, height := 80, 24
	if w, h, err := term.GetSize(os.Stdout.Fd()); err == nil {
		width, height = w, h
	}

	ta.SetWidth(max(width-lipgloss.Width(textareaPrompt), 1))
	ta.SetHeight(1)
	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height-ta.Height()))
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	ta.KeyMap.InsertNewline.SetEnabled(false)
	sessions, err := client.ListSession()
	if err != nil {
		fmt.Println("Error listing sessions")
		os.Exit(1)
	}
	sessionItems := make([]list.Item, len(sessions))
	for i, s := range sessions {
		sessionItems[i] = components.NewItem(s.Title, s.Directory)
	}

	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	modelsList, err := loadModelsList()
	if err != nil {
		modelsList = []api.ModelInfo{}
	}

	// currentModel, err := config.GetCurrentModel()
	currentModel, err := client.GetCurrentModel()
	currentModelIndex := 0
	if err == nil {
		for i, model := range modelsList {
			if model.Model == currentModel.Model {
				currentModelIndex = i
				break
			}
		}
	}

	m := model{
		textarea:           ta,
		pasteCounter:       0,
		pastedTexts:        make(map[int]string),
		messages:           []models.Message{},
		cacheIndex:         0,
		cache:              make(map[int]string),
		viewport:           vp,
		islistSessionWin:   false,
		islistCommandsWin:  false,
		bashMode:           false,
		listSession:        components.LaunchSessionList(sessionItems),
		listCommands:       components.LaunchCommandList(),
		listModels:         components.LaunchModelsList(),
		sessions:           sessions,
		senderStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:                nil,
		width:              width,
		spinner:            spin,
		isGenerating:       false,
		lastEsc:            time.Now(),
		mode:               "chat",
		modes:              []string{"chat", "plan", "assistant"},
		modelsList:         modelsList,
		isModelsListWin:    false,
		loginProviders:     defaultLoginProviders(),
		isLoginProviderWin: false,
		loginProviderIndex: 0,
		loginAction:        "login",
		effortOptions:      defaultEffortOptions(),
		isEffortWin:        false,
		effortIndex:        0,
		modelsListIndex:    currentModelIndex,
		imgPasteCounter:    0,
		pastedImgs:         make(map[int][]byte),
		pastedImgPreviews:  make(map[int]kittyPreview),
		currentContextSize: 0,
		gitStatus:          defaultStatusLineGitInfo("."),
		enter_api_win:      false,
	}
	if len(modelsList) == 0 {
		m.messages = append(m.messages, models.Message{
			SessionID: m.currentSession.ID,
			Data: models.EncodeMessageData(models.StoredMessageData{
				Role:    "assistant",
				Content: "No provider is logged in. Run `/login` to select and authenticate a provider.",
			}),
		})
		m.refreshMessagesView()
	}
	m.listModels.Refresh(modelsList)
	m.syncLayout()
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, refreshGitStatusCmd(m.gitStatusDirectory()))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncLayout()

		if len(m.messages) > 0 {
			m.viewport.SetContent(renderMessages(m.messages, m.width))
		}
		// m.viewport.GotoBottom()
	case tea.KeyPressMsg:
		if m.islistSessionWin {
			updatedModel, cmd := m.listSession.Update(msg)
			m.listSession = updatedModel.(components.Model)
			switch msg.String() {
			case "enter":
				curIdx := m.listSession.Current()
				if curIdx < 0 || curIdx >= len(m.sessions) {
					return m, cmd
				}
				m.currentSession = m.sessions[curIdx]
				sessionData, err := client.GetSessionData(m.currentSession.ID)
				if err != nil {
					fmt.Println("Failed to get session data.")
					os.Exit(1)
				}
				if len(sessionData) > 100 {
					m.messages = sessionData[len(sessionData)-100:]
				} else {
					m.messages = sessionData
				}
				m.completeMessages = sessionData

				//m.todoList = client.GetCurrentTodoList(m.currentSession.ID)
				m.islistSessionWin = false
				contextSize, err := client.GetContextSize(m.currentSession.ID)
				if err != nil {
					m.currentContextSize = 0
				} else {
					m.currentContextSize = contextSize
				}

				m.syncLayout()
				m.viewport.SetContent(renderMessages(m.messages, m.width))
				m.viewport.GotoBottom()
				return m, tea.Batch(cmd, refreshGitStatusCmd(m.gitStatusDirectory()))
			case "esc":
				m.islistSessionWin = false
				return m, nil
			}
			return m, cmd
		}
		if m.questionMode {
			return m.handleQuestionInput(msg)
		}
		if m.enter_api_win {
			return m.handleApiKeyWin(msg)
		}
		if m.isModelsListWin {
			return m.handleModelsListInput(msg)
		}
		if m.isLoginProviderWin {
			return m.handleLoginProviderInput(msg)
		}
		if m.isEffortWin {
			return m.handleEffortInput(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			if m.islistCommandsWin {
				m.islistCommandsWin = false
				m.syncLayout()
				return m, nil
			}
			if len(m.textarea.Value()) > 0 {
				m.textarea.Reset()
				return m, nil
			}
			return m, tea.Sequence(tea.Printf("Resume session with lightcode -r %s", m.currentSession.ID), tea.Quit)
		case "esc":
			if m.islistCommandsWin {
				m.islistCommandsWin = false
				m.listCommands.Filter("")
				m.syncLayout()
				return m, nil
			}
			if m.streamCh != nil && m.cancelStream != nil && time.Since(m.lastEsc) < 500*time.Millisecond {
				m.cancelActiveGeneration()
				return m, nil
			}
			m.lastEsc = time.Now()
			m.showEscMsg = true
			return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return clearEscMsg()
			})
			// m.sessions = client.ListSession()
			// m.islistSessionWin = true

		case "ctrl+v", "meta+v":
			curVal := m.textarea.Value()
			err := clipboard.Init()
			if err != nil {
				panic(err)
			}
			// textBytes := clipboard.Read(clipboard.FmtText)
			// imgBytes := clipboard.Read(clipboard.FmtImage)
			textBytes, textErr := nativeclipboard.Text.Read()
			imgBytes, imgErr := nativeclipboard.Image.Read()
			if textErr != nil && imgErr != nil {
				m.isError = true
				m.errorMessage = textErr.Error() + imgErr.Error()
				return m, nil
			}
			nextVal := curVal
			var cmd tea.Cmd
			if imgBytes != nil {
				m.imgPasteCounter++
				m.pastedImgs[m.imgPasteCounter] = imgBytes
				preview, upload := buildKittyPreviewUpload(imgBytes, m.imgPasteCounter, m.width, 10, os.Getenv("TMUX") != "")
				if preview.id > 0 && upload != "" {
					m.pastedImgPreviews[m.imgPasteCounter] = preview
					cmd = tea.Raw(upload)
				}
				placeholder := fmt.Sprintf("[pasted img #%d]", m.imgPasteCounter)
				nextVal = strings.TrimSpace(nextVal + " " + placeholder)
			}
			pasteValue := string(textBytes)
			if pasteValue == "" {
				m.textarea.SetValue(nextVal)
			} else if strings.Count(pasteValue, "\n") > 1 {
				m.pasteCounter++
				m.pastedTexts[m.pasteCounter] = pasteValue
				placeholder := fmt.Sprintf("[pasted text #%d]", m.pasteCounter)
				m.textarea.InsertString(strings.TrimSpace(placeholder))
				// m.textarea.SetValue()
			} else {
				// m.textarea.SetValue(nextVal + pasteValue)
				m.textarea.InsertString(strings.TrimSpace(pasteValue))
			}

			if len(strings.Split(curVal, "\n")) > len(strings.Split(m.textarea.Value(), "\n")) {
				if m.textarea.Height()-1 >= 1 {
					m.textarea.SetHeight(m.textarea.Height() - 1)
				}
			}
			if len(strings.Split(curVal, "\n")) < len(strings.Split(m.textarea.Value(), "\n")) {
				m.textarea.SetHeight(m.textarea.Height() + 1)
			}
			m.resizeTextareaToContent()
			m.syncLayout()
			return m, cmd
		// case "shift+enter":
		// 	curVal := m.textarea.Value()
		// 	m.textarea.SetValue(curVal + "\n")
		// 	m.adjustTextareaHeight()
		// 	return m, nil
		case "enter":
			if m.enter_api_win {
				return m, nil
			}
			if m.islistCommandsWin {
				m.cacheIndex++
				curCommand := m.listCommands.Current()
				m.islistCommandsWin = false
				m.listCommands.Filter("")
				if curCommand == "" {
					m.ensureCurrentSession(m.textarea.Value())
					if m.isGenerating {
						m.queue = append(m.queue, queue{prompt: m.textarea.Value()})
						m.textarea.SetValue("")
						m.syncLayout()
						return m, refreshGitStatusCmd(m.gitStatusDirectory())
					}
					val := m.textarea.Value()
					m.textarea.SetValue("")
					m.syncLayout()
					return m, tea.Batch(refreshGitStatusCmd(m.gitStatusDirectory()), m.beginGeneration(val))
				}
				m.syncLayout()
				if len(curCommand) > 1 {
					cmd := CmdHandler("/"+curCommand, &m)
					m.textarea.SetValue("")
					return m, cmd
				}
				return m, nil
			}
			if strings.HasPrefix(m.textarea.Value(), "/") {
				cmd := CmdHandler(m.textarea.Value(), &m)
				return m, cmd
			}
			m.ensureCurrentSession(m.textarea.Value())
			if m.isGenerating == true {
				// value, img_bytes := createPrompt(strings.Trim(m.textarea.Value(), "\n"), &m)
				m.queue = append(m.queue, queue{prompt: m.textarea.Value()})
				m.textarea.SetValue("")
				m.syncLayout()
				return m, refreshGitStatusCmd(m.gitStatusDirectory())
			}
			val := m.textarea.Value()
			m.textarea.SetValue("")
			m.syncLayout()
			return m, tea.Batch(refreshGitStatusCmd(m.gitStatusDirectory()), m.beginGeneration(val))
		case "shift+enter":
			m.textarea.SetValue(m.textarea.Value() + "\n")
			m.resizeTextareaToContent()
			m.syncLayout()
			return m, nil
		case "up", "down":
			if m.islistCommandsWin {
				var cmd tea.Cmd
				updatedModel, cmd := m.listCommands.Update(msg)
				m.listCommands = updatedModel.(components.ModelCmdList)
				return m, cmd
			}

			var cmd tea.Cmd
			if len(m.completeMessages) > 100 && len(m.messages) > 0 {
				curStart := len(m.completeMessages) - len(m.messages)
				for i := range m.completeMessages {
					if m.completeMessages[i].ID == m.messages[0].ID {
						curStart = i
						break
					}
				}
				size := 75
				step := 35
				top_offset := 15
				if msg.String() == "up" && m.viewport.YOffset() < top_offset && curStart > 0 {
					targetIdx := max(0, curStart-step)
					addedLines := strings.Count(renderMessages(m.completeMessages[targetIdx:curStart], m.width), "\n") - strings.Count(mascot(), "\n")
					m.messages = m.completeMessages[targetIdx:min(targetIdx+size, len(m.completeMessages))]
					m.viewport.SetContent(renderMessages(m.messages, m.width))
					m.viewport.SetYOffset(m.viewport.YOffset() + addedLines)
				} else if msg.String() == "down" && m.viewport.TotalLineCount()-(m.viewport.YOffset()+m.viewport.VisibleLineCount()) < top_offset && curStart+len(m.messages) < len(m.completeMessages) {
					targetIdx := min(curStart+step, len(m.completeMessages)-size)
					removedLines := strings.Count(renderMessages(m.completeMessages[curStart:targetIdx], m.width), "\n") - strings.Count(mascot(), "\n")
					m.messages = m.completeMessages[targetIdx:min(targetIdx+size, len(m.completeMessages))]
					m.viewport.SetContent(renderMessages(m.messages, m.width))
					m.viewport.SetYOffset(max(0, m.viewport.YOffset()-removedLines))
				}
			}

			m.viewport, cmd = m.viewport.Update(msg)

			return m, cmd
		case "tab":
			for i, v := range m.modes {
				if v == m.mode {
					m.mode = m.modes[(i+1)%len(m.modes)]
					break
				}
			}
			return m, nil

		case "/":
			m.listCommands.Reset()
			if m.islistCommandsWin {
				m.cache[m.cacheIndex] = m.textarea.Value()
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				m.listCommands.Filter(strings.TrimPrefix(m.textarea.Value(), "/"))
				return m, cmd
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			wasListCommandsOpen := m.islistCommandsWin
			if len(m.textarea.Value()) == 1 {
				m.listCommands.Filter("")
				m.islistCommandsWin = true
			}
			// if m.isGenerating {
			// 	return m, tea.Batch(cmd, waitForMessages(m.streamCh))
			// }
			if !wasListCommandsOpen && m.islistCommandsWin {
				m.syncLayout()
			}
			return m, cmd
		default:
			if m.islistCommandsWin {
				m.cache[m.cacheIndex] = m.textarea.Value()
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				val := m.textarea.Value()
				if !strings.HasPrefix(val, "/") {
					m.islistCommandsWin = false
					m.listCommands.Filter("")
					m.syncLayout()
				} else {
					m.listCommands.Filter(strings.TrimPrefix(val, "/"))
				}
				return m, cmd
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			m.resizeTextareaToContent()
			m.syncLayout()
			// m.adjustTextareaHeight()

			previousBashMode := m.bashMode
			if previousBashMode != m.bashMode {
				m.syncLayout()
			}
			return m, cmd
		}
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.viewport.SetYOffset(m.viewport.YOffset() - 1)
		case tea.MouseWheelDown:
			m.viewport.SetYOffset(m.viewport.YOffset() + 1)
		}
		m.syncLayout()
	case tea.MouseClickMsg:
		switch msg.Button {
		case tea.MouseLeft:
		case tea.MouseRight:
		}
	// case tea.MouseReleaseMsg:

	// case tea.MouseMotionMsg:

	case spinner.TickMsg:
		if !m.isGenerating && !m.isCompacting {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case streamMessageMsg:
		if msg.Role == "error" {
			m.isGenerating = false
			m.isError = true
			m.errorMessage = msg.Content
			return m, nil
		}
		m.messages = append(m.messages, models.Message{
			SessionID: m.currentSession.ID,
			// ID:        fmt.Sprintf("%s-assistant-%d", m.currentSession.ID, len(m.messages)),
			Data: models.EncodeMessageData(models.StoredMessageData(msg)),
		})

		m.refreshMessagesView()
		m.syncLayout()
		var refreshGit tea.Cmd
		if msg.Role == "tool_call" && shouldRefreshGitAfterToolCall(models.StoredMessageData(msg)) {
			refreshGit = refreshGitStatusCmd(m.gitStatusDirectory())
		}
		if msg.Role == "question" {
			questions := parseQuestions(msg.Content)
			if len(questions) > 0 {
				m.questionMode = true
				m.questions = questions
				m.questionIdx = 0
				m.questionAnswers = make([]string, len(questions))
				m.questionSelected = 0
				m.textarea.SetValue("")
				m.textarea.Reset()
				if len(questions[0].options) > 0 {
					m.textarea.Placeholder = "↑↓ select · Enter confirm"
					m.textarea.Blur()
				} else {
					m.textarea.Placeholder = "Type your answer..."
					m.textarea.Focus()
				}
				m.isGenerating = false
				m.syncLayout()
			}
		}
		return m, tea.Batch(refreshGit, waitForMessages(m.streamCh))

	case streamDoneMsg:
		m.streamCh = nil
		m.isGenerating = false
		m.syncLayout()
		if m.currentSession.ID != "" {
			// m.todoList = client.GetCurrentTodoList(m.currentSession.ID)
			m.messages = withoutEphemeralTodoStatus(m.messages)
			if len(m.todoList) != 0 {
				m.messages = append(m.messages, models.Message{
					SessionID: m.currentSession.ID,
					Data: models.EncodeMessageData(models.StoredMessageData{
						Role:    "todo_status",
						Content: models.EncodeToDoList(m.todoList),
					}),
				})
			}
			// temp will turn this into a function
			if len(m.queue) > 0 {
				return m, tea.Batch(refreshGitStatusCmd(m.gitStatusDirectory()), m.runNextQueuedPrompt())
			}
			if len(m.queue) == 0 {
				contextSize, err := client.GetContextSize(m.currentSession.ID)
				if err != nil {
					m.currentContextSize = 0
				} else {
					m.currentContextSize = contextSize
				}
				m.syncLayout()
				return m, refreshGitStatusCmd(m.gitStatusDirectory())
			}

		}
		m.refreshMessagesView()
		m.syncLayout()
		return m, nil

	case gitStatusMsg:
		if m.isCurrentGitStatus(msg.status) {
			m.gitStatus = msg.status
		}
		return m, nil

	case compactMemoryDoneMsg:
		m.isCompacting = false
		m.isGenerating = false
		if msg.err != nil {
			if m.currentSession.ID == msg.sessionID {
				appendCommandStatusMessage(&m, fmt.Sprintf("Compaction failed: %s", msg.err.Error()))
			}
			return m, nil
		}
		if m.currentSession.ID != msg.sessionID {
			return m, nil
		}
		messages, err := client.GetSessionData(m.currentSession.ID)
		if err != nil {
			fmt.Println("Failed to get session data")
			os.Exit(1)
		}
		m.messages = messages
		m.currentContextSize = msg.contextSize
		m.messages = append(m.messages, models.Message{
			SessionID: m.currentSession.ID,
			Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: "Compacted context"}),
		})
		m.refreshMessagesView()
		m.syncLayout()
		return m, nil

	case loginFinishedMsg:
		return m.handleLoginFinished(msg)

	case copilotLoginStartedMsg:
		return m.handleCopilotLoginStarted(msg)

	case logoutFinishedMsg:
		return m.handleLogoutFinished(msg)

	case clearEscMsgMsg:
		m.showEscMsg = false
		return m, nil
	}

	return m, nil
}
