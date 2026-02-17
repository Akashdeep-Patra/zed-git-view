package components

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// isGitDiffHeaderSBS duplicates the header check for the side-by-side renderer
// (avoids a cross-package dependency on views).
func isGitDiffHeaderSBS(line string) bool {
	switch {
	case strings.HasPrefix(line, "diff --git "),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "old mode "),
		strings.HasPrefix(line, "new mode "),
		strings.HasPrefix(line, "new file mode "),
		strings.HasPrefix(line, "deleted file mode "),
		strings.HasPrefix(line, "similarity index "),
		strings.HasPrefix(line, "rename from "),
		strings.HasPrefix(line, "rename to "),
		strings.HasPrefix(line, "copy from "),
		strings.HasPrefix(line, "copy to "):
		return true
	}
	return false
}

// RenderSideBySideDiff renders a unified diff in side-by-side format with
// line numbers, gutter indicators, and clean styling.
func RenderSideBySideDiff(styles ui.Styles, diff string, totalWidth int) string {
	if diff == "" {
		return styles.Muted.Render("No diff content")
	}

	const lnW = 4
	lnFmt := fmt.Sprintf("%%%dd", lnW)
	lnBlank := strings.Repeat(" ", lnW)

	// Each panel: lineNum(4) + gutter(1) + content
	panelW := (totalWidth - 3) / 2 // 3 for center separator
	if panelW < 20 {
		panelW = 20
	}
	contentW := panelW - lnW - 1 // -lineNum -gutter
	if contentW < 10 {
		contentW = 10
	}

	lines := strings.Split(diff, "\n")
	var leftLines, rightLines []string

	inHeader := true
	oldLine, newLine := 0, 0

	for _, line := range lines {
		// Section titles.
		if strings.HasPrefix(line, "===") {
			styled := styles.Title.Render(truncateTo(line, panelW))
			leftLines = append(leftLines, styled)
			rightLines = append(rightLines, padTo("", panelW))
			continue
		}

		// Git metadata header.
		if strings.HasPrefix(line, "diff --git ") {
			inHeader = true
			continue
		}

		if inHeader {
			if isGitDiffHeaderSBS(line) {
				continue
			}
			if strings.HasPrefix(line, "--- ") {
				continue
			}
			if strings.HasPrefix(line, "+++ ") {
				path := strings.TrimPrefix(line, "+++ ")
				path = strings.TrimPrefix(path, "b/")
				if path == "/dev/null" {
					path = "(deleted)"
				}
				header := styles.DiffHeader.Render("  ▎ " + truncateTo(path, totalWidth-6))
				leftLines = append(leftLines, header)
				rightLines = append(rightLines, padTo("", panelW))
				continue
			}
			inHeader = false
		}

		// Hunk header → update counters, render spacer.
		if strings.HasPrefix(line, "@@") {
			rest := strings.TrimPrefix(line, "@@")
			if idx := strings.Index(rest, "@@"); idx >= 0 {
				for _, tok := range strings.Fields(rest[:idx]) {
					if strings.HasPrefix(tok, "-") {
						oldLine, _ = parseHunkRangeSBS(tok)
					} else if strings.HasPrefix(tok, "+") {
						newLine, _ = parseHunkRangeSBS(tok)
					}
				}
			}
			spacer := styles.DiffSeparator.Render(lnBlank + "│ ···")
			leftLines = append(leftLines, spacer)
			rightLines = append(rightLines, spacer)
			continue
		}

		switch {
		case strings.HasPrefix(line, "-"):
			content := strings.TrimPrefix(line, "-")
			ln := fmt.Sprintf(lnFmt, oldLine)
			left := styles.DiffRemovedLineNum.Render(ln) +
				styles.DiffRemovedGutter.Render("│") +
				styles.DiffRemoved.Render(padTo(truncateTo(" "+content, contentW), contentW))
			leftLines = append(leftLines, left)
			rightLines = append(rightLines, padTo("", panelW))
			oldLine++

		case strings.HasPrefix(line, "+"):
			content := strings.TrimPrefix(line, "+")
			ln := fmt.Sprintf(lnFmt, newLine)
			right := styles.DiffAddedLineNum.Render(ln) +
				styles.DiffAddedGutter.Render("│") +
				styles.DiffAdded.Render(padTo(truncateTo(" "+content, contentW), contentW))
			leftLines = append(leftLines, padTo("", panelW))
			rightLines = append(rightLines, right)
			newLine++

		default:
			if line == "" && oldLine == 0 && newLine == 0 {
				continue
			}
			oldLn := lnBlank
			newLn := lnBlank
			if oldLine > 0 {
				oldLn = fmt.Sprintf(lnFmt, oldLine)
				oldLine++
			}
			if newLine > 0 {
				newLn = fmt.Sprintf(lnFmt, newLine)
				newLine++
			}
			sep := lipgloss.NewStyle().Foreground(styles.Theme.Border).Render("│")
			left := styles.DiffContextLineNum.Render(oldLn) + sep +
				styles.DiffContext.Render(padTo(truncateTo(" "+line, contentW), contentW))
			right := styles.DiffContextLineNum.Render(newLn) + sep +
				styles.DiffContext.Render(padTo(truncateTo(" "+line, contentW), contentW))
			leftLines = append(leftLines, left)
			rightLines = append(rightLines, right)
		}
	}

	// Pad to same length.
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	centerSep := lipgloss.NewStyle().Foreground(styles.Theme.Border).Render(" │ ")

	var b strings.Builder
	for i := range leftLines {
		l := padTo(leftLines[i], panelW)
		r := rightLines[i]
		b.WriteString(l + centerSep + r + "\n")
	}

	return b.String()
}

func parseHunkRangeSBS(tok string) (int, int) {
	tok = strings.TrimLeft(tok, "+-")
	parts := strings.SplitN(tok, ",", 2)
	start := 0
	count := 1
	if s, err := strconv.Atoi(parts[0]); err == nil {
		start = s
	}
	if len(parts) == 2 {
		if c, err := strconv.Atoi(parts[1]); err == nil {
			count = c
		}
	}
	return start, count
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
