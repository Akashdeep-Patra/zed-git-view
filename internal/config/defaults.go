package config

// KeyBindings defines the mapping of actions to keys.
// Kept separate so it can later be made configurable via config file.
type KeyBindings struct {
	Quit           string
	Help           string
	Tab            string
	ShiftTab       string
	Up             string
	Down           string
	Enter          string
	Space          string
	Stage          string
	StageAll       string
	Unstage        string
	UnstageAll     string
	Discard        string
	Commit         string
	AmendCommit    string
	Push           string
	Pull           string
	Fetch          string
	Refresh        string
	Search         string
	CopyHash       string
	OpenInEditor   string
	ToggleSideDiff string
}

// DefaultKeyBindings returns the default key bindings.
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Quit:           "q",
		Help:           "?",
		Tab:            "tab",
		ShiftTab:       "shift+tab",
		Up:             "k",
		Down:           "j",
		Enter:          "enter",
		Space:          " ",
		Stage:          "s",
		StageAll:       "S",
		Unstage:        "u",
		UnstageAll:     "U",
		Discard:        "x",
		Commit:         "c",
		AmendCommit:    "C",
		Push:           "P",
		Pull:           "p",
		Fetch:          "f",
		Refresh:        "r",
		Search:         "/",
		CopyHash:       "y",
		OpenInEditor:   "e",
		ToggleSideDiff: "v",
	}
}
