package components

import (
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// TabInfo describes a single tab for rendering.
type TabInfo struct {
	Name     string
	Icon     string
	Shortcut string
	Active   bool
	Group    string
}

// RenderTabs renders a modern tab bar with pill-style active indicator,
// icon+name labels, underlined shortcut hints, and group separators.
//
// Layout:  ┃  ● Status  ± Diff  ◆ Log  │  ⑂ Branches  ⇄ Remotes  ⊟ Stash  │  …  ┃
//
// The active tab gets a filled pill background + accent underline.
// Shortcuts are shown as dim superscript hints.
func RenderTabs(styles ui.Styles, tabs []TabInfo, width int) string {
	t := styles.Theme

	// ── Style definitions ────────────────────────────────────────────

	// Active tab: filled pill with primary accent.
	activePill := lipgloss.NewStyle().
		Foreground(t.Primary).
		Background(t.Surface).
		Bold(true).
		Padding(0, 1).
		BorderBottom(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBottomForeground(t.Primary)

	// Inactive tab: subtle, no background.
	inactivePill := lipgloss.NewStyle().
		Foreground(t.TextMuted).
		Padding(0, 1)

	// Shortcut hint styling (dimmed, appears after the name).
	shortcutActive := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(false).
		Faint(true)

	shortcutInactive := lipgloss.NewStyle().
		Foreground(t.TextSubtle).
		Faint(true)

	// Group separator: thin vertical bar.
	groupSep := lipgloss.NewStyle().
		Foreground(t.Border).
		Padding(0, 1).
		Render("│")

	// ── Build tab items ──────────────────────────────────────────────

	var rendered []string
	prevGroup := ""

	for _, tab := range tabs {
		// Insert group separator between different groups.
		if prevGroup != "" && tab.Group != prevGroup {
			rendered = append(rendered, groupSep)
		}
		prevGroup = tab.Group

		// Build the label: "icon Name (shortcut)"
		label := tab.Icon + " " + tab.Name
		var shortcut string
		if tab.Active {
			shortcut = shortcutActive.Render(" " + tab.Shortcut)
		} else {
			shortcut = shortcutInactive.Render(" " + tab.Shortcut)
		}

		if tab.Active {
			rendered = append(rendered, activePill.Render(label)+shortcut)
		} else {
			rendered = append(rendered, inactivePill.Render(label)+shortcut)
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, rendered...)

	// Navigation hint on the right side.
	hint := lipgloss.NewStyle().
		Foreground(t.TextSubtle).
		Faint(true).
		Render("tab/shift+tab  ?help")

	rowW := lipgloss.Width(row)
	hintW := lipgloss.Width(hint)
	gap := width - rowW - hintW - 3
	if gap < 0 {
		gap = 0
		hint = "" // drop hint if no room
	}

	fullRow := row + strings.Repeat(" ", gap) + hint

	// Wrap in the tab bar container.
	bar := lipgloss.NewStyle().
		Background(t.Bg).
		Width(width).
		Padding(0, 1).
		Render(fullRow)

	// Separator line below the tab bar.
	sep := lipgloss.NewStyle().
		Foreground(t.Border).
		Width(width).
		Render(strings.Repeat("─", width))

	return lipgloss.JoinVertical(lipgloss.Left, bar, sep)
}
