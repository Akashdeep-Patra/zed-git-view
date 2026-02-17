package components

import (
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// RenderScrollbar returns a vertical scrollbar track of the given height.
// It shows a thumb (filled block) proportional to the visible portion,
// positioned according to the scroll percentage.
//
// Returns an empty string if all content fits (no scrolling needed).
//
//	Parameters:
//	  styles     – application styles (for theming)
//	  height     – total height of the scrollbar track (rows)
//	  totalLines – total number of content lines
//	  visibleH   – number of lines visible at once
//	  scrollPct  – current scroll position as 0.0–1.0
func RenderScrollbar(styles ui.Styles, height, totalLines, visibleH int, scrollPct float64) string {
	if totalLines <= visibleH || height < 1 {
		return ""
	}

	t := styles.Theme

	// Thumb size: proportional to visible/total, min 1 row.
	thumbSize := int(float64(height) * float64(visibleH) / float64(totalLines))
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > height {
		thumbSize = height
	}

	// Thumb position.
	maxOffset := height - thumbSize
	thumbStart := int(scrollPct * float64(maxOffset))
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart > maxOffset {
		thumbStart = maxOffset
	}

	thumbStyle := lipgloss.NewStyle().Foreground(t.Primary)
	trackStyle := lipgloss.NewStyle().Foreground(t.Border)

	thumbChar := "█"
	trackChar := "░"

	var b strings.Builder
	b.Grow(height * 4)
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i >= thumbStart && i < thumbStart+thumbSize {
			b.WriteString(thumbStyle.Render(thumbChar))
		} else {
			b.WriteString(trackStyle.Render(trackChar))
		}
	}
	return b.String()
}
