package common

import (
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Tab identifiers ─────────────────────────────────────────────────────────

// TabID identifies which view/tab is active.
type TabID int

const (
	TabStatus TabID = iota
	TabLog
	TabDiff
	TabBranches
	TabStash
	TabRemotes
	TabRebase
	TabConflicts
	TabWorktrees
	TabBisect
)

// TabMeta describes a tab for display purposes.
type TabMeta struct {
	ID       TabID
	Name     string // Display name shown in the tab bar.
	Icon     string // Unicode icon (nerdfont-free, works in all terminals).
	Shortcut string // Mnemonic shortcut hint displayed in the tab (e.g., "s").
	Group    string // Logical group: "core", "history", "branch", "advanced".
}

// AllTabs is the ordered list of all tabs, grouped logically.
// Navigation: Tab/Shift+Tab cycles through them, or use the shortcut key.
var AllTabs = []TabMeta{
	// ── Core (everyday workflow) ──────────────────────────────
	{TabStatus, "Status", "●", "s", "core"},
	{TabDiff, "Diff", "±", "d", "core"},
	{TabLog, "Log", "◆", "l", "core"},

	// ── Branch & Sync ────────────────────────────────────────
	{TabBranches, "Branches", "⑂", "b", "branch"},
	{TabRemotes, "Remotes", "⇄", "m", "branch"},
	{TabStash, "Stash", "⊟", "t", "branch"},

	// ── Advanced Git operations ──────────────────────────────
	{TabRebase, "Rebase", "↻", "e", "advanced"},
	{TabConflicts, "Conflicts", "⚡", "x", "advanced"},
	{TabWorktrees, "Worktrees", "⌥", "w", "advanced"},
	{TabBisect, "Bisect", "◎", "i", "advanced"},
}

// ── Custom messages ─────────────────────────────────────────────────────────

// RefreshMsg signals views to reload data.
type RefreshMsg struct{}

// ErrMsg carries an error to be displayed.
type ErrMsg struct{ Err error }

// InfoMsg carries an informational message.
type InfoMsg struct{ Text string }

// SwitchTabMsg requests a tab switch.
type SwitchTabMsg struct{ Tab TabID }

// ToggleHelpMsg toggles the help overlay.
type ToggleHelpMsg struct{}

// CmdRefresh returns a RefreshMsg (use as return from tea.Cmd).
func CmdRefresh() tea.Msg { return RefreshMsg{} }

// CmdErr creates a tea.Cmd that sends an ErrMsg.
func CmdErr(err error) tea.Cmd {
	return func() tea.Msg { return ErrMsg{Err: err} }
}

// CmdInfo creates a tea.Cmd that sends an InfoMsg.
func CmdInfo(text string) tea.Cmd {
	return func() tea.Msg { return InfoMsg{Text: text} }
}

// ── View interface ──────────────────────────────────────────────────────────

// View is the interface every tab view must implement.
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
	SetSize(width, height int)
	ShortHelp() []components.HelpEntry

	// InputCapture returns true when the view is in a text-input mode
	// (e.g. commit message editing) and wants to capture arrow keys,
	// letters, etc. instead of letting the app handle them for tab
	// switching.
	InputCapture() bool
}
