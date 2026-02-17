package ui

import "github.com/charmbracelet/lipgloss"

// Theme holds all colours for the application.
// Inspired by Zed's default dark palette (Catppuccin Mocha).
type Theme struct {
	Bg            lipgloss.Color
	Surface       lipgloss.Color
	SurfaceHover  lipgloss.Color
	Border        lipgloss.Color
	BorderFocused lipgloss.Color

	Text        lipgloss.Color
	TextMuted   lipgloss.Color
	TextSubtle  lipgloss.Color
	TextInverse lipgloss.Color

	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	Added     lipgloss.Color
	Modified  lipgloss.Color
	Deleted   lipgloss.Color
	Renamed   lipgloss.Color
	Conflict  lipgloss.Color
	Untracked lipgloss.Color

	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	CommitHash  lipgloss.Color
	BranchLocal lipgloss.Color
	BranchHead  lipgloss.Color
	Tag         lipgloss.Color
	Remote      lipgloss.Color
	Stash       lipgloss.Color

	GraphColors []lipgloss.Color
}

// DarkTheme returns the default Zed-inspired dark theme.
func DarkTheme() Theme {
	return Theme{
		Bg:            lipgloss.Color("#1e1e2e"),
		Surface:       lipgloss.Color("#282840"),
		SurfaceHover:  lipgloss.Color("#313152"),
		Border:        lipgloss.Color("#3b3b5c"),
		BorderFocused: lipgloss.Color("#7c7cf0"),

		Text:        lipgloss.Color("#cdd6f4"),
		TextMuted:   lipgloss.Color("#9399b2"),
		TextSubtle:  lipgloss.Color("#6c7086"),
		TextInverse: lipgloss.Color("#1e1e2e"),

		Primary:   lipgloss.Color("#89b4fa"),
		Secondary: lipgloss.Color("#b4befe"),
		Accent:    lipgloss.Color("#f5c2e7"),

		Added:     lipgloss.Color("#a6e3a1"),
		Modified:  lipgloss.Color("#f9e2af"),
		Deleted:   lipgloss.Color("#f38ba8"),
		Renamed:   lipgloss.Color("#89dceb"),
		Conflict:  lipgloss.Color("#fab387"),
		Untracked: lipgloss.Color("#9399b2"),

		Success: lipgloss.Color("#a6e3a1"),
		Warning: lipgloss.Color("#f9e2af"),
		Error:   lipgloss.Color("#f38ba8"),
		Info:    lipgloss.Color("#89b4fa"),

		CommitHash:  lipgloss.Color("#f9e2af"),
		BranchLocal: lipgloss.Color("#a6e3a1"),
		BranchHead:  lipgloss.Color("#89b4fa"),
		Tag:         lipgloss.Color("#f5c2e7"),
		Remote:      lipgloss.Color("#f38ba8"),
		Stash:       lipgloss.Color("#fab387"),

		GraphColors: []lipgloss.Color{
			"#89b4fa", "#a6e3a1", "#f5c2e7", "#f9e2af",
			"#89dceb", "#fab387", "#cba6f7", "#f38ba8",
		},
	}
}

// Styles holds pre-computed lipgloss styles derived from a Theme.
type Styles struct {
	Theme Theme

	// Layout
	TabBar    lipgloss.Style
	TabActive lipgloss.Style
	TabItem   lipgloss.Style
	Content   lipgloss.Style
	StatusBar lipgloss.Style
	HelpBar   lipgloss.Style

	// Panels
	Panel        lipgloss.Style
	PanelFocused lipgloss.Style
	PanelTitle   lipgloss.Style

	// List items
	ListItem     lipgloss.Style
	ListSelected lipgloss.Style
	ListDimmed   lipgloss.Style

	// Text
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Body     lipgloss.Style
	Muted    lipgloss.Style
	Bold     lipgloss.Style
	Code     lipgloss.Style
	KeyBind  lipgloss.Style
	KeyDesc  lipgloss.Style

	// Git file statuses
	FileAdded     lipgloss.Style
	FileModified  lipgloss.Style
	FileDeleted   lipgloss.Style
	FileRenamed   lipgloss.Style
	FileConflict  lipgloss.Style
	FileUntracked lipgloss.Style

	// Diff
	DiffAdded      lipgloss.Style
	DiffRemoved    lipgloss.Style
	DiffContext    lipgloss.Style
	DiffHeader     lipgloss.Style
	DiffHunkHeader lipgloss.Style
	DiffLineNum    lipgloss.Style

	// Commit / refs
	CommitHash lipgloss.Style
	CommitMsg  lipgloss.Style
	Author     lipgloss.Style
	Date       lipgloss.Style
	BranchName lipgloss.Style
	TagName    lipgloss.Style
	RemoteName lipgloss.Style

	// Dialogs
	Dialog       lipgloss.Style
	DialogTitle  lipgloss.Style
	DialogButton lipgloss.Style

	Spinner lipgloss.Style
}

// NewStyles builds all styles from the given theme.
func NewStyles(t Theme) Styles {
	s := Styles{Theme: t}

	s.TabBar = lipgloss.NewStyle().Padding(0, 1).Background(t.Surface)
	s.TabActive = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 2).
		Background(t.Bg).BorderBottom(true).BorderStyle(lipgloss.ThickBorder()).BorderBottomForeground(t.Primary)
	s.TabItem = lipgloss.NewStyle().Foreground(t.TextMuted).Padding(0, 2)
	s.Content = lipgloss.NewStyle().Padding(1, 2)
	s.StatusBar = lipgloss.NewStyle().Foreground(t.TextMuted).Background(t.Surface).Padding(0, 1)
	s.HelpBar = lipgloss.NewStyle().Foreground(t.TextSubtle).Padding(0, 1)

	s.Panel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(0, 1)
	s.PanelFocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.BorderFocused).Padding(0, 1)
	s.PanelTitle = lipgloss.NewStyle().Foreground(t.Text).Bold(true).Padding(0, 1)

	s.ListItem = lipgloss.NewStyle().Foreground(t.Text).PaddingLeft(2)
	s.ListSelected = lipgloss.NewStyle().Foreground(t.Text).Background(t.SurfaceHover).Bold(true).PaddingLeft(1)
	s.ListDimmed = lipgloss.NewStyle().Foreground(t.TextSubtle).PaddingLeft(2)

	s.Title = lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	s.Subtitle = lipgloss.NewStyle().Foreground(t.TextMuted).Bold(true)
	s.Body = lipgloss.NewStyle().Foreground(t.Text)
	s.Muted = lipgloss.NewStyle().Foreground(t.TextMuted)
	s.Bold = lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	s.Code = lipgloss.NewStyle().Foreground(t.Primary).Background(t.Surface).Padding(0, 1)
	s.KeyBind = lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	s.KeyDesc = lipgloss.NewStyle().Foreground(t.TextMuted)

	s.FileAdded = lipgloss.NewStyle().Foreground(t.Added)
	s.FileModified = lipgloss.NewStyle().Foreground(t.Modified)
	s.FileDeleted = lipgloss.NewStyle().Foreground(t.Deleted).Strikethrough(true)
	s.FileRenamed = lipgloss.NewStyle().Foreground(t.Renamed)
	s.FileConflict = lipgloss.NewStyle().Foreground(t.Conflict).Bold(true)
	s.FileUntracked = lipgloss.NewStyle().Foreground(t.Untracked)

	s.DiffAdded = lipgloss.NewStyle().Foreground(t.Added)
	s.DiffRemoved = lipgloss.NewStyle().Foreground(t.Deleted)
	s.DiffContext = lipgloss.NewStyle().Foreground(t.TextMuted)
	s.DiffHeader = lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	s.DiffHunkHeader = lipgloss.NewStyle().Foreground(t.Secondary).Italic(true)
	s.DiffLineNum = lipgloss.NewStyle().Foreground(t.TextSubtle).Width(5).Align(lipgloss.Right)

	s.CommitHash = lipgloss.NewStyle().Foreground(t.CommitHash)
	s.CommitMsg = lipgloss.NewStyle().Foreground(t.Text)
	s.Author = lipgloss.NewStyle().Foreground(t.Primary)
	s.Date = lipgloss.NewStyle().Foreground(t.TextMuted)
	s.BranchName = lipgloss.NewStyle().Foreground(t.BranchLocal).Bold(true)
	s.TagName = lipgloss.NewStyle().Foreground(t.Tag).Bold(true)
	s.RemoteName = lipgloss.NewStyle().Foreground(t.Remote)

	s.Dialog = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(t.Primary).Padding(1, 2).Width(60)
	s.DialogTitle = lipgloss.NewStyle().Foreground(t.Text).Bold(true).Align(lipgloss.Center)
	s.DialogButton = lipgloss.NewStyle().Foreground(t.TextInverse).Background(t.Primary).Padding(0, 3).Bold(true)

	s.Spinner = lipgloss.NewStyle().Foreground(t.Primary)

	return s
}

// DefaultStyles returns styles using the dark theme.
func DefaultStyles() Styles {
	return NewStyles(DarkTheme())
}
