package views

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/api"
)

func FormatK(n int64) string {
	if n >= 1000 {
		if n%1000 == 0 {
			return fmt.Sprintf("%dK", n/1000)
		}
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return strconv.FormatInt(n, 10)
}

func shortenDir(curDir string) string {
	home, _ := os.UserHomeDir()
	return strings.Replace(curDir, home, "~", 1)
}

type statusLineGitInfo struct {
	Directory string
	Branch    string
	Changes   string
}

func defaultStatusLineGitInfo(directory string) statusLineGitInfo {
	return statusLineGitInfo{
		Directory: expandWorkingDir(directory),
		Branch:    "No git",
		Changes:   "No git",
	}
}

func renderStatusLine(model api.ModelInfo, usedTokens int64, width int, git statusLineGitInfo) string {
	parts := []string{}
	if model.Model != "" {
		parts = append(parts, modelStatusName(model))
	} else {
		parts = append(parts, "No model")
	}
	parts = append(parts, git.Branch)
	parts = append(parts, git.Changes)
	parts = append(parts, FormatK(usedTokens)+" used")
	parts = append(parts, FormatK(contextWindowForModel(model))+" window")

	line := strings.Join(parts, " · ")
	if width > 0 && lipgloss.Width(line) > width {
		line = truncateStatusLine(line, width)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Render(line)
}

func modelStatusName(model api.ModelInfo) string {
	name := model.Model
	if effort := reasoningEffortLabel(model.ReasoningEffort); effort != "" {
		name += " " + effort
	}
	return name
}

func reasoningEffortLabel(effort string) string {
	switch effort {
	case "low", "medium", "high":
		return effort
	case "xhigh":
		return "extra high"
	default:
		return ""
	}
}

func loadStatusLineGitInfo(directory string) statusLineGitInfo {
	dir := expandWorkingDir(directory)
	return statusLineGitInfo{
		Directory: dir,
		Branch:    gitBranchLabel(dir),
		Changes:   gitChangesLabel(dir),
	}
}

func expandWorkingDir(directory string) string {
	dir := strings.TrimSpace(directory)
	if dir == "" || dir == "." {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	if dir == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(dir, "~/"))
		}
	}
	return dir
}

func gitBranchLabel(directory string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = directory
	branch, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(branch)) != "" {
		return strings.TrimSpace(string(branch))
	}
	cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = directory
	commit, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(commit)) != "" {
		return strings.TrimSpace(string(commit))
	}
	return "No git"
}

func gitChangesLabel(directory string) string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = directory
	out, err := cmd.Output()
	if err != nil {
		return "No git"
	}
	status := strings.TrimSpace(string(out))
	if status == "" {
		return "No changes"
	}
	return fmt.Sprintf("%d changes", len(strings.Split(status, "\n")))
}

func contextWindowForModel(model api.ModelInfo) int64 {
	if model.BaseUrl == "codex" && strings.HasPrefix(model.Model, "gpt-5") {
		return 258000
	}
	return 128000
}

func truncateStatusLine(line string, width int) string {
	if width <= 1 {
		return ""
	}
	runes := []rune(line)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func countWrappedLines(text string, width int, m *model) int {
	return len(wrapTextLines(text, width))
}

func wrapTextLines(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	var lines []string
	for _, logicalLine := range strings.Split(text, "\n") {
		if logicalLine == "" {
			lines = append(lines, "")
			continue
		}

		var current strings.Builder
		currentWidth := 0
		for _, r := range logicalLine {
			runeWidth := lipgloss.Width(string(r))
			if runeWidth == 0 {
				runeWidth = 1
			}
			if currentWidth+runeWidth > width {
				lines = append(lines, current.String())
				current.Reset()
				currentWidth = runeWidth
				current.WriteRune(r)
				continue
			}
			current.WriteRune(r)
			currentWidth += runeWidth
		}
		lines = append(lines, current.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func wrappedCursorPosition(text string, logicalLine int, column int, width int) (int, int) {
	if width <= 0 {
		return 0, 0
	}
	logicalLines := strings.Split(text, "\n")
	if logicalLine < 0 {
		logicalLine = 0
	}
	if logicalLine >= len(logicalLines) {
		logicalLine = len(logicalLines) - 1
	}

	y := 0
	for i := 0; i < logicalLine; i++ {
		y += len(wrapTextLines(logicalLines[i], width))
	}

	x := 0
	currentColumn := 0
	for _, r := range logicalLines[logicalLine] {
		if currentColumn >= column {
			break
		}
		runeWidth := lipgloss.Width(string(r))
		if runeWidth == 0 {
			runeWidth = 1
		}
		if x+runeWidth > width {
			y++
			x = 0
		}
		x += runeWidth
		currentColumn++
	}

	return x, y
}
