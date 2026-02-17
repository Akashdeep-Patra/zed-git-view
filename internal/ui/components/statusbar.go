package components

import (
	"fmt"
	"path/filepath"
	"strings"

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

// RenderStatusBar renders the bottom status bar with clear visual sections
// separated by dim vertical bars.
//
// Wide (>= 60):   main  │  ↑2 ↓1  │  ● modified              zed-git-view
// Medium (40-59):  main  │  ↑2 ↓1  │  ● modified
// Narrow (< 40):   main  │  ● modified
func RenderStatusBar(styles ui.Styles, data StatusBarData, width int) string {
	t := styles.Theme

	sepStyle := lipgloss.NewStyle().Foreground(t.Border).Faint(true)
	sep := sepStyle.Render(" │ ")

	// ── Left sections ────────────────────────────────────────────

	// Branch.
	branchStyle := lipgloss.NewStyle().Foreground(t.BranchHead).Bold(true)
	branchSection := " " + branchStyle.Render(" "+data.Branch)

	// Sync (only if non-zero and terminal is wide enough).
	var syncSection string
	if width >= 40 && (data.Ahead > 0 || data.Behind > 0) {
		syncStyle := lipgloss.NewStyle().Foreground(t.Warning)
		var parts []string
		if data.Ahead > 0 {
			parts = append(parts, fmt.Sprintf("↑%d", data.Ahead))
		}
		if data.Behind > 0 {
			parts = append(parts, fmt.Sprintf("↓%d", data.Behind))
		}
		syncSection = sep + syncStyle.Render(strings.Join(parts, " "))
	}

	// State.
	var stateSection string
	switch {
	case data.Merging:
		badge := lipgloss.NewStyle().
			Foreground(t.TextInverse).
			Background(t.Warning).
			Bold(true).
			Padding(0, 1).
			Render("MERGING")
		stateSection = sep + badge
	case data.Rebasing:
		badge := lipgloss.NewStyle().
			Foreground(t.TextInverse).
			Background(t.Warning).
			Bold(true).
			Padding(0, 1).
			Render("REBASING")
		stateSection = sep + badge
	case data.Clean:
		stateSection = sep + lipgloss.NewStyle().Foreground(t.Success).Render("✓ clean")
	default:
		stateSection = sep + lipgloss.NewStyle().Foreground(t.Modified).Render("● modified")
	}

	left := branchSection + syncSection + stateSection

	// ── Right section ────────────────────────────────────────────

	var right string
	if data.Message != "" {
		fg := t.Info
		if data.IsError {
			fg = t.Error
		}
		right = lipgloss.NewStyle().Foreground(fg).Render(data.Message) + " "
	} else if width >= 60 && data.RepoRoot != "" {
		repoName := filepath.Base(data.RepoRoot)
		right = lipgloss.NewStyle().Foreground(t.TextSubtle).Render(repoName) + " "
	}

	// ── Assemble ─────────────────────────────────────────────────

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW
	if gap < 0 {
		gap = 1
		right = "" // drop right side if no room
	}

	content := left + strings.Repeat(" ", gap) + right

	return styles.StatusBar.Width(width).Render(content)
}
