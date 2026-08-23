package views

import (
	"bytes"
	"hash/fnv"
	"image"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

func buildKittyPreviewUpload(data []byte, pasteIndex, maxCols, maxRows int) (kittyPreview, string) {
	tmux := os.Getenv("TMUX") != ""
	if len(data) == 0 || pasteIndex <= 0 || maxCols <= 0 || maxRows <= 0 {
		return kittyPreview{}, ""
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil || img == nil {
		return kittyPreview{}, ""
	}

	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	if imgWidth <= 0 || imgHeight <= 0 {
		return kittyPreview{}, ""
	}

	cols, rows := kittyPreviewGrid(imgWidth, imgHeight, maxCols, maxRows)
	if cols <= 0 || rows <= 0 {
		return kittyPreview{}, ""
	}

	imgID := kittyImageID(pasteIndex, data)
	var buf strings.Builder
	err = kitty.EncodeGraphics(&buf, img, &kitty.Options{
		ID:               imgID,
		Action:           kitty.TransmitAndPut,
		Transmission:     kitty.Direct,
		Format:           kitty.RGBA,
		ImageWidth:       imgWidth,
		ImageHeight:      imgHeight,
		Columns:          cols,
		Rows:             rows,
		VirtualPlacement: true,
		Quite:            2,
		Chunk:            true,
		ChunkFormatter: func(chunk string) string {
			if tmux {
				return ansi.TmuxPassthrough(chunk)
			}
			return chunk
		},
	})
	if err != nil {
		return kittyPreview{}, ""
	}

	return kittyPreview{id: imgID, cols: cols, rows: rows}, buf.String()
}

func kittyPreviewGrid(imgWidth, imgHeight, maxCols, maxRows int) (int, int) {
	if imgWidth <= 0 || imgHeight <= 0 || maxCols <= 0 || maxRows <= 0 {
		return 0, 0
	}

	rows := maxRows
	cols := int(math.Round(float64(imgWidth) * float64(rows) / float64(imgHeight)))
	if cols < 1 {
		cols = 1
	}
	if cols > maxCols {
		cols = maxCols
		rows = int(math.Round(float64(imgHeight) * float64(cols) / float64(imgWidth)))
		if rows < 1 {
			rows = 1
		}
		if rows > maxRows {
			rows = maxRows
		}
	}

	return cols, rows
}

func kittyImageID(pasteIndex int, data []byte) int {
	if pasteIndex > 0 && pasteIndex < 256 {
		return pasteIndex
	}

	h := fnv.New32a()
	_, _ = h.Write(data)
	id := int(h.Sum32() & 0x00ffffff)
	if id == 0 {
		return 1
	}
	return id
}

func renderKittyPlaceholders(imgID, cols, rows int) string {
	if imgID <= 0 || cols <= 0 || rows <= 0 {
		return ""
	}

	fgStyle := ansi.NewStyle().ForegroundColor(ansi.IndexedColor(uint8(imgID & 0xff))).String()
	imageMSB := (imgID >> 24) & 0xff

	var buf strings.Builder
	for y := range rows {
		for x := range cols {
			buf.WriteString(fgStyle)
			buf.WriteRune(kitty.Placeholder)
			buf.WriteRune(kitty.Diacritic(y))
			buf.WriteRune(kitty.Diacritic(x))
			if imageMSB > 0 {
				buf.WriteRune(kitty.Diacritic(imageMSB))
			}
		}
		buf.WriteString(ansi.ResetStyle)
		if y < rows-1 {
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}
