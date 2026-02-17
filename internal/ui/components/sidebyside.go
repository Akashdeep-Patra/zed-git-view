package components

import (
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// RenderSideBySideDiff renders a unified diff in side-by-side format.
func RenderSideBySideDiff(styles ui.Styles, diff string, totalWidth int) string {
	if diff == "" {
		return styles.Muted.Render("No diff content")
	}

	panelW := (totalWidth - 3) / 2 // 3 for separator
	if panelW < 20 {
		panelW = 20
	}

	lines := strings.Split(diff, "\n")
	var leftLines, rightLines []string

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"):
			// Header lines go on both sides.
			styled := styles.DiffHeader.Render(truncateTo(line, panelW))
			leftLines = append(leftLines, styled)
			rightLines = append(rightLines, styled)

		case strings.HasPrefix(line, "@@"):
			styled := styles.DiffHunkHeader.Render(truncateTo(line, panelW))
			leftLines = append(leftLines, styled)
			rightLines = append(rightLines, styled)

		case strings.HasPrefix(line, "-"):
			leftLines = append(leftLines, styles.DiffRemoved.Render(truncateTo(line, panelW)))
			rightLines = append(rightLines, strings.Repeat(" ", min(lipgloss.Width(line), panelW)))

		case strings.HasPrefix(line, "+"):
			leftLines = append(leftLines, strings.Repeat(" ", min(lipgloss.Width(line), panelW)))
			rightLines = append(rightLines, styles.DiffAdded.Render(truncateTo(line, panelW)))

		default:
			styled := styles.DiffContext.Render(truncateTo(line, panelW))
			leftLines = append(leftLines, styled)
			rightLines = append(rightLines, styled)
		}
	}

	// Pad to same length.
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	sep := lipgloss.NewStyle().Foreground(styles.Theme.Border).Render(" │ ")

	var b strings.Builder
	for i := range leftLines {
		l := padTo(leftLines[i], panelW)
		r := rightLines[i]
		b.WriteString(l + sep + r + "\n")
	}

	return b.String()
}

func truncateTo(s string, maxW int) string {
	runes := []rune(s)
	if len(runes) <= maxW {
		return s
	}
	if maxW <= 1 {
		return "…"
	}
	return string(runes[:maxW-1]) + "…"
}

func padTo(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
