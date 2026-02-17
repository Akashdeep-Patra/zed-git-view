package components

import (
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// HelpEntry is a single key-description pair for the help overlay.
type HelpEntry struct {
	Key  string
	Desc string
}

// RenderHelp renders a full-screen help overlay.
func RenderHelp(styles ui.Styles, title string, sections map[string][]HelpEntry, width, height int) string {
	t := styles.Theme

	titleStr := lipgloss.NewStyle().
		Foreground(t.Primary).Bold(true).
		Align(lipgloss.Center).
		Width(width - 4).
		Render(title)

	var body strings.Builder
	body.WriteString(titleStr + "\n\n")

	sectionStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Underline(true)
	keyStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Width(16).Align(lipgloss.Right)
	descStyle := lipgloss.NewStyle().Foreground(t.Text)

	// Deterministic order from a predefined list.
	order := []string{"Navigation", "Tabs", "Status", "Staging", "Diff", "Branches", "Stash", "Remotes", "Rebase", "Bisect", "General"}
	for _, section := range order {
		entries, ok := sections[section]
		if !ok || len(entries) == 0 {
			continue
		}
		body.WriteString(sectionStyle.Render(section) + "\n")
		for _, e := range entries {
			body.WriteString("  " + keyStyle.Render(e.Key) + "  " + descStyle.Render(e.Desc) + "\n")
		}
		body.WriteString("\n")
	}

	content := body.String()

	overlay := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Width(min(70, width-4)).
		MaxHeight(height - 2).
		Render(content)

	return ui.PlaceCentre(width, height, overlay)
}

// GlobalHelpEntries returns the help entries for global keybindings.
func GlobalHelpEntries() map[string][]HelpEntry {
	return map[string][]HelpEntry{
		"Navigation": {
			{Key: "j / ↓", Desc: "Move down"},
			{Key: "k / ↑", Desc: "Move up"},
			{Key: "g / Home", Desc: "Go to top"},
			{Key: "G / End", Desc: "Go to bottom"},
			{Key: "pgup / ctrl+u", Desc: "Page up"},
			{Key: "pgdn / ctrl+d", Desc: "Page down"},
			{Key: "enter", Desc: "Confirm / expand"},
			{Key: "esc", Desc: "Back / cancel"},
		},
		"Tabs": {
			{Key: "tab", Desc: "Next tab"},
			{Key: "shift+tab", Desc: "Previous tab"},
			{Key: "scroll on bar", Desc: "Cycle tabs (mouse)"},
			{Key: "alt+s", Desc: "Status"},
			{Key: "alt+d", Desc: "Diff"},
			{Key: "alt+l", Desc: "Log"},
			{Key: "alt+b", Desc: "Branches"},
			{Key: "alt+m", Desc: "Remotes"},
			{Key: "alt+t", Desc: "Stash"},
			{Key: "alt+e", Desc: "Rebase"},
			{Key: "alt+x", Desc: "Conflicts"},
			{Key: "alt+w", Desc: "Worktrees"},
			{Key: "alt+i", Desc: "Bisect"},
		},
		"General": {
			{Key: "r", Desc: "Refresh data"},
			{Key: "?", Desc: "Toggle this help"},
			{Key: "q / ctrl+c", Desc: "Quit"},
		},
	}
}
