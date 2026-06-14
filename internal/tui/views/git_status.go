package views

import tea "charm.land/bubbletea/v2"

type gitStatusMsg struct {
	status statusLineGitInfo
}

func refreshGitStatusCmd(directory string) tea.Cmd {
	return func() tea.Msg {
		return gitStatusMsg{status: loadStatusLineGitInfo(directory)}
	}
}

func (m model) gitStatusDirectory() string {
	if m.currentSession.Directory != "" && m.currentSession.Directory != "." {
		return m.currentSession.Directory
	}
	return "."
}

func (m model) isCurrentGitStatus(status statusLineGitInfo) bool {
	return status.Directory == expandWorkingDir(m.gitStatusDirectory())
}
