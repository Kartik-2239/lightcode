package views

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func createPrompt(value string, m *model) (string, [][]byte) {
	re := regexp.MustCompile(`\[pasted text #(\d+)\]`)
	textareaValue := re.ReplaceAllStringFunc(value, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		var idx int
		fmt.Sscanf(sub[1], "%d", &idx)
		if real, ok := m.pastedTexts[idx]; ok {
			return real
		}
		return match
	})
	re2 := regexp.MustCompile(`\[pasted img #(\d+)\]`)
	imgBytes := make([][]byte, 0, len(re2.FindAllString(value, -1)))
	textareaValue = re2.ReplaceAllStringFunc(textareaValue, func(match string) string {
		sub := re2.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}

		idx, err := strconv.Atoi(sub[1])
		if err != nil {
			return match
		}
		if img, ok := m.pastedImgs[idx]; ok {
			imgBytes = append(imgBytes, img)
			return fmt.Sprintf("[pasted img #%d]", idx)
		}
		return match
	})

	textareaValue = strings.Join(strings.Fields(textareaValue), " ")
	return textareaValue, imgBytes
}

func (m model) currentKittyPreview() ([]kittyPreview, bool) {
	re := regexp.MustCompile(`\[pasted img #(\d+)\]`)
	matches := re.FindAllStringSubmatch(m.textarea.Value(), -1)
	previews := make([]kittyPreview, 0, len(matches))
	for i := len(matches) - 1; i >= 0; i-- {
		idx, err := strconv.Atoi(matches[i][1])
		if err != nil {
			continue
		}
		preview, ok := m.pastedImgPreviews[idx]
		if ok && preview.id > 0 && preview.cols > 0 && preview.rows > 0 {
			// preview = append(previews, preview)
			previews = append(previews, preview)
			// return preview, true
		}
	}
	if len(previews) > 0 {
		return previews, true
	}
	return []kittyPreview{}, false
}

func (m *model) clearPastedInput() {
	m.pasteCounter = 0
	m.pastedTexts = make(map[int]string)
	m.imgPasteCounter = 0
	m.pastedImgs = make(map[int][]byte)
	m.pastedImgPreviews = make(map[int]kittyPreview)
}
