package views

import (
	"strings"
)

// activeMention finds the "@token" immediately to the left of the cursor on its
// line. A token starts at an '@' that is at line start or preceded by
// whitespace, and runs to the cursor with no embedded whitespace. It returns the
// token text (without '@'), the absolute byte offset of the '@' and of the
// cursor within value, and whether a valid token was found.
func activeMention(value string, line, col int) (token string, atAbs, curAbs int, ok bool) {
	lines := strings.Split(value, "\n")
	if line < 0 || line >= len(lines) {
		return "", 0, 0, false
	}
	cur := lines[line]
	if col > len(cur) {
		col = len(cur)
	}
	lineStart := 0
	for i := 0; i < line; i++ {
		lineStart += len(lines[i]) + 1
	}
	curAbs = lineStart + col

	before := cur[:col]
	at := strings.LastIndex(before, "@")
	if at == -1 {
		return "", 0, 0, false
	}
	if at > 0 {
		if prev := before[at-1]; prev != ' ' && prev != '\t' {
			return "", 0, 0, false
		}
	}
	tok := before[at+1:]
	if strings.ContainsAny(tok, " \t") {
		return "", 0, 0, false
	}
	return tok, lineStart + at, curAbs, true
}

// updateMentionState opens/filters/closes the @file picker based on the token
// under the cursor. The file index is built lazily on first use and cached.
func (m *model) updateMentionState() {
	tok, _, _, ok := activeMention(m.textarea.Value(), m.textarea.Line(), m.textarea.Column())
	if !ok {
		m.isFileListWin = false
		return
	}
	if !m.fileIndexBuilt {
		m.fileIndex = buildFileIndex()
		m.fileList.SetItems(m.fileIndex)
		m.fileIndexBuilt = true
	}
	m.fileList.Filter(tok)
	if m.fileList.Current() == "" {
		m.isFileListWin = false
		return
	}
	m.isFileListWin = true
}

// acceptFileMention replaces the active "@token" with the highlighted path and
// records it as a mention. This is the only place a mention is recorded.
func (m *model) acceptFileMention() {
	m.isFileListWin = false
	path := m.fileList.Current()
	if path == "" {
		return
	}
	value := m.textarea.Value()
	_, atAbs, curAbs, ok := activeMention(value, m.textarea.Line(), m.textarea.Column())
	if !ok {
		return
	}

	insert := "@" + path
	if strings.ContainsAny(path, " \t") {
		insert = "@\"" + path + "\""
	}
	insert += " "

	newValue := value[:atAbs] + insert + value[curAbs:]
	m.textarea.SetValue(newValue)

	// Reposition the cursor to just after the inserted mention (SetValue leaves
	// it at the end of the whole value).
	insEnd := atAbs + len(insert)
	if insEnd > len(newValue) {
		insEnd = len(newValue)
	}
	targetLine := strings.Count(newValue[:insEnd], "\n")
	totalLines := strings.Count(newValue, "\n")
	for i := 0; i < totalLines-targetLine; i++ {
		m.textarea.CursorUp()
	}
	m.textarea.SetCursorColumn(insEnd - (strings.LastIndex(newValue[:insEnd], "\n") + 1))

}

// extractMentionsFromText parses @path tokens from prompt text. A token starts
// at an '@' that is at text start or preceded by whitespace (so email@host won't
// match), and runs until the next whitespace. Quoted form @"path with spaces" is
// also supported. Results are deduped in order of first appearance.
func extractMentionsFromText(text string) []string {
	var out []string
	seen := map[string]bool{}
	runes := []rune(text)
	for i, r := range runes {
		if r != '@' {
			continue
		}
		if i > 0 {
			prev := runes[i-1]
			if prev != ' ' && prev != '\t' && prev != '\n' {
				continue
			}
		}
		j := i + 1
		var path string
		if j < len(runes) && runes[j] == '"' {
			j++
			start := j
			for j < len(runes) && runes[j] != '"' {
				j++
			}
			path = string(runes[start:j])
		} else {
			start := j
			for j < len(runes) && runes[j] != ' ' && runes[j] != '\t' && runes[j] != '\n' {
				j++
			}
			path = string(runes[start:j])
		}
		if path != "" && !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	return out
}
