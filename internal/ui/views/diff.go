package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffView shows the diff for all changes or a selected file.
type DiffView struct {
	gitSvc     git.Service
	styles     ui.Styles
	width      int
	height     int
	vp         viewport.Model
	loaded     bool
	rawDiff    string
	sideBySide bool
}

// NewDiffView creates a new DiffView.
func NewDiffView(gitSvc git.Service, styles ui.Styles) *DiffView {
	return &DiffView{
		gitSvc: gitSvc,
		styles: styles,
		vp:     viewport.New(0, 0),
	}
}

func (v *DiffView) Init() tea.Cmd { return v.refresh() }

func (v *DiffView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.vp.Width = w
	v.vp.Height = h - 2
}

type diffResultMsg struct{ diff string }

func (v *DiffView) refresh() tea.Cmd {
	return func() tea.Msg {
		unstaged, err := v.gitSvc.Diff(false, "")
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		staged, err := v.gitSvc.Diff(true, "")
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		combined := ""
		if staged != "" {
			combined += "=== STAGED CHANGES ===\n\n" + staged + "\n"
		}
		if unstaged != "" {
			combined += "=== UNSTAGED CHANGES ===\n\n" + unstaged
		}
		if combined == "" {
			combined = "No changes"
		}
		return diffResultMsg{diff: combined}
	}
}

func (v *DiffView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case diffResultMsg:
		v.loaded = true
		v.rawDiff = msg.diff
		v.renderDiff()
		v.vp.GotoTop()
		return v, nil

	case common.RefreshMsg:
		return v, v.refresh()

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.vp.ScrollUp(3)
			return v, nil
		case tea.MouseButtonWheelDown:
			v.vp.ScrollDown(3)
			return v, nil
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return v, v.refresh()
		case "v": // Toggle side-by-side
			v.sideBySide = !v.sideBySide
			v.renderDiff()
			return v, nil
		}
	}

	var cmd tea.Cmd
	v.vp, cmd = v.vp.Update(msg)
	return v, cmd
}

func (v *DiffView) renderDiff() {
	if v.sideBySide {
		v.vp.SetContent(components.RenderSideBySideDiff(v.styles, v.rawDiff, v.width))
	} else {
		v.vp.SetContent(renderDiffColored(v.styles, v.rawDiff))
	}
}

func (v *DiffView) View() string {
	if !v.loaded {
		return ui.PlaceCentre(v.width, v.height,
			lipgloss.NewStyle().Foreground(v.styles.Theme.TextMuted).Render("Loading diff..."))
	}

	mode := "inline"
	if v.sideBySide {
		mode = "side-by-side"
	}
	hint := v.styles.Muted.Render("  [" + mode + "]  v toggle mode  r refresh")
	return v.vp.View() + "\n" + hint
}

func (v *DiffView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "↑/↓", Desc: "Scroll"},
		{Key: "ctrl+d/u", Desc: "Page down/up"},
		{Key: "v", Desc: "Toggle side-by-side"},
		{Key: "r", Desc: "Refresh"},
	}
}

func (v *DiffView) InputCapture() bool { return false }

// isGitDiffHeader reports whether the line is part of the per-file
// metadata header that Git emits before the actual unified diff hunks.
func isGitDiffHeader(line string) bool {
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

// parseHunkRange extracts (start, count) from a hunk range token like "+3,7".
func parseHunkRange(tok string) (int, int) {
	tok = strings.TrimLeft(tok, "+-")
	parts := strings.SplitN(tok, ",", 2)
	start, _ := strconv.Atoi(parts[0])
	count := 1
	if len(parts) == 2 {
		count, _ = strconv.Atoi(parts[1])
	}
	return start, count
}

// renderDiffColored renders a production-grade diff view with:
//   - Clean file headers (filename only, no raw git paths)
//   - No hunk headers (line numbers tell the story)
//   - Old/new line numbers in a gutter column
//   - Colored gutter indicators (green █ / red █) instead of +/-
//   - Subtle background tints on added/removed lines
//   - Stripped +/- prefixes from content
//
// Inspired by GitHub, VS Code, and GitKraken diff views.
func renderDiffColored(styles ui.Styles, diff string) string {
	if diff == "" {
		return styles.Muted.Render("No diff content")
	}

	t := styles.Theme
	sep := styles.DiffSeparator.Render("│")

	// Line number width (4 chars is enough for most files).
	const lnW = 4
	lnFmt := fmt.Sprintf("%%%dd", lnW)
	lnBlank := strings.Repeat(" ", lnW)

	var b strings.Builder
	inHeader := true

	// Track line numbers: old (left), new (right).
	oldLine, newLine := 0, 0
	fileCount := 0

	for _, line := range strings.Split(diff, "\n") {
		// ── Section title (=== STAGED CHANGES === etc.) ──────────
		if strings.HasPrefix(line, "===") {
			if fileCount > 0 {
				b.WriteByte('\n') // visual gap before section
			}
			b.WriteString(styles.Title.Render(line))
			b.WriteByte('\n')
			continue
		}

		// ── Git metadata header → enter header mode ─────────────
		if strings.HasPrefix(line, "diff --git ") {
			inHeader = true
			continue
		}

		if inHeader {
			if isGitDiffHeader(line) {
				continue
			}

			// --- a/path or +++ b/path → render clean file header.
			if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
				if strings.HasPrefix(line, "+++ ") {
					// Extract file path from "+++ b/path" or "+++ /dev/null".
					path := strings.TrimPrefix(line, "+++ ")
					path = strings.TrimPrefix(path, "b/")
					if path == "/dev/null" {
						path = "(deleted)"
					}

					if fileCount > 0 {
						b.WriteByte('\n')
					}
					fileCount++

					// File header: icon + filename.
					header := styles.DiffHeader.Render("  ▎ " + path)
					b.WriteString(header)
					b.WriteByte('\n')

					// Thin separator under file header.
					b.WriteString(styles.DiffSeparator.Render(
						strings.Repeat("─", lnW) + "┬" +
							strings.Repeat("─", lnW) + "┬" +
							strings.Repeat("─", 40)))
					b.WriteByte('\n')
				}
				continue
			}

			// First non-header line exits header mode.
			inHeader = false
		}

		// ── Hunk header → update line counters, don't render ────
		if strings.HasPrefix(line, "@@") {
			rest := strings.TrimPrefix(line, "@@")
			idx := strings.Index(rest, "@@")
			if idx >= 0 {
				rangesPart := strings.TrimSpace(rest[:idx])
				for _, tok := range strings.Fields(rangesPart) {
					if strings.HasPrefix(tok, "-") {
						oldLine, _ = parseHunkRange(tok)
					} else if strings.HasPrefix(tok, "+") {
						newLine, _ = parseHunkRange(tok)
					}
				}
			}
			// Render a subtle spacer between hunks (if not first hunk).
			spacer := styles.DiffSeparator.Render(
				lnBlank + "│" + lnBlank + "│" + "  ···")
			b.WriteString(spacer)
			b.WriteByte('\n')
			continue
		}

		// ── Diff content lines ──────────────────────────────────

		switch {
		case strings.HasPrefix(line, "+"):
			content := strings.TrimPrefix(line, "+")
			ln := fmt.Sprintf(lnFmt, newLine)
			b.WriteString(styles.DiffAddedLineNum.Render(lnBlank))
			b.WriteString(styles.DiffAddedGutter.Render("│"))
			b.WriteString(styles.DiffAddedLineNum.Render(ln))
			b.WriteString(styles.DiffAddedGutter.Render("│"))
			b.WriteString(styles.DiffAdded.Render(" " + content))
			newLine++

		case strings.HasPrefix(line, "-"):
			content := strings.TrimPrefix(line, "-")
			ln := fmt.Sprintf(lnFmt, oldLine)
			b.WriteString(styles.DiffRemovedLineNum.Render(ln))
			b.WriteString(styles.DiffRemovedGutter.Render("│"))
			b.WriteString(styles.DiffRemovedLineNum.Render(lnBlank))
			b.WriteString(styles.DiffRemovedGutter.Render("│"))
			b.WriteString(styles.DiffRemoved.Render(" " + content))
			oldLine++

		default:
			// Context line.
			if line == "" && oldLine == 0 && newLine == 0 {
				continue // skip empty lines outside of hunks
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
			b.WriteString(styles.DiffContextLineNum.Render(oldLn))
			b.WriteString(lipgloss.NewStyle().Foreground(t.Border).Render("│"))
			b.WriteString(styles.DiffContextLineNum.Render(newLn))
			b.WriteString(sep)
			b.WriteString(styles.DiffContext.Render(" " + line))
		}

		b.WriteByte('\n')
	}
	return b.String()
}
