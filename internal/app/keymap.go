package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the global keybindings used across the application.
// Tab switching uses mnemonic single-key shortcuts that match the tab's
// first letter (or a memorable alternative when there's a conflict).
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding
	Refresh  key.Binding
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Enter    key.Binding
	Back     key.Binding

	// Mnemonic tab shortcuts — each maps to the shortcut shown in the tab bar.
	// These are only active when no view is capturing text input.
	TabStatus    key.Binding // s (but note: views may also use 's' for stage)
	TabDiff      key.Binding // d
	TabLog       key.Binding // l
	TabBranches  key.Binding // b
	TabRemotes   key.Binding // m  (r is taken by refresh)
	TabStash     key.Binding // t
	TabRebase    key.Binding // e
	TabConflicts key.Binding // x
	TabWorktrees key.Binding // w
	TabBisect    key.Binding // i
}

// DefaultKeyMap returns the default keybindings.
// Tab shortcuts use Alt+key so they never conflict with view-level keys.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		NextTab:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		PrevTab:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
		Refresh:  key.NewBinding(key.WithKeys("r", "ctrl+r"), key.WithHelp("r", "refresh")),
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "down")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
		Home:     key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "top")),
		End:      key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "bottom")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),

		// Alt+key tab shortcuts — never conflict with view-level bindings.
		TabStatus:    key.NewBinding(key.WithKeys("alt+s"), key.WithHelp("alt+s", "status")),
		TabDiff:      key.NewBinding(key.WithKeys("alt+d"), key.WithHelp("alt+d", "diff")),
		TabLog:       key.NewBinding(key.WithKeys("alt+l"), key.WithHelp("alt+l", "log")),
		TabBranches:  key.NewBinding(key.WithKeys("alt+b"), key.WithHelp("alt+b", "branches")),
		TabRemotes:   key.NewBinding(key.WithKeys("alt+m"), key.WithHelp("alt+m", "remotes")),
		TabStash:     key.NewBinding(key.WithKeys("alt+t"), key.WithHelp("alt+t", "stash")),
		TabRebase:    key.NewBinding(key.WithKeys("alt+e"), key.WithHelp("alt+e", "rebase")),
		TabConflicts: key.NewBinding(key.WithKeys("alt+x"), key.WithHelp("alt+x", "conflicts")),
		TabWorktrees: key.NewBinding(key.WithKeys("alt+w"), key.WithHelp("alt+w", "worktrees")),
		TabBisect:    key.NewBinding(key.WithKeys("alt+i"), key.WithHelp("alt+i", "bisect")),
	}
}
