package components

import (
	"fmt"
	"path/filepath"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarData carries the info displayed in the bottom status bar.
type StatusBarData struct {
	Branch   string
	Ahead    int
	Behind   int
	Clean    bool
	Merging  bool
	Rebasing bool
	Message  string // transient info/error message
	IsError  bool
	RepoRoot string
}

// RenderStatusBar renders the bottom status bar with a modern, informative layout.
//
// Layout: [ branch  ↑2 ↓1  clean/MERGING/REBASING ]  ←gap→  [ message | repo name ]
func RenderStatusBar(styles ui.Styles, data StatusBarData, width int) string {
	t := styles.Theme

	// ── Left side: branch + sync + state ─────────────────────

	branchIcon := lipgloss.NewStyle().Foreground(t.BranchHead).Render("")
	branchName := lipgloss.NewStyle().Foreground(t.BranchHead).Bold(true).Render(" " + data.Branch)
	branch := branchIcon + branchName

	var sync string
	if data.Ahead > 0 || data.Behind > 0 {
		parts := ""
		if data.Ahead > 0 {
			parts += fmt.Sprintf("↑%d", data.Ahead)
		}
		if data.Behind > 0 {
			if parts != "" {
				parts += " "
			}
			parts += fmt.Sprintf("↓%d", data.Behind)
		}
		sync = "  " + lipgloss.NewStyle().Foreground(t.Warning).Render(parts)
	}

	var state string
	switch {
	case data.Merging:
		state = "  " + lipgloss.NewStyle().Foreground(t.TextInverse).Background(t.Warning).
			Bold(true).Padding(0, 1).Render("MERGING")
	case data.Rebasing:
		state = "  " + lipgloss.NewStyle().Foreground(t.TextInverse).Background(t.Warning).
			Bold(true).Padding(0, 1).Render("REBASING")
	case data.Clean:
		state = "  " + lipgloss.NewStyle().Foreground(t.Success).Render("✓ clean")
	default:
		state = "  " + lipgloss.NewStyle().Foreground(t.Modified).Render("● modified")
	}

	left := branch + sync + state

	// ── Right side: message or repo name ─────────────────────

	var right string
	if data.Message != "" {
		fg := t.Info
		if data.IsError {
			fg = t.Error
		}
		right = lipgloss.NewStyle().Foreground(fg).Render(data.Message)
	} else if data.RepoRoot != "" {
		repoName := filepath.Base(data.RepoRoot)
		right = lipgloss.NewStyle().Foreground(t.TextSubtle).Render(" " + repoName)
	}

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW - 2
	if gap < 0 {
		gap = 1
	}

	content := left + lipgloss.NewStyle().Width(gap).Render("") + right

	return styles.StatusBar.Width(width).Render(content)
}
