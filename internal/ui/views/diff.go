package views

import (
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
		{Key: "j/k", Desc: "Scroll"},
		{Key: "ctrl+d/u", Desc: "Page down/up"},
		{Key: "v", Desc: "Toggle side-by-side"},
		{Key: "r", Desc: "Refresh"},
	}
}

// renderDiffColored applies syntax colouring to a unified diff string.
func renderDiffColored(styles ui.Styles, diff string) string {
	if diff == "" {
		return styles.Muted.Render("No diff content")
	}
	var b strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			b.WriteString(styles.DiffHeader.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(styles.DiffHunkHeader.Render(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(styles.DiffAdded.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(styles.DiffRemoved.Render(line))
		case strings.HasPrefix(line, "diff "):
			b.WriteString(styles.DiffHeader.Render(line))
		case strings.HasPrefix(line, "index "):
			b.WriteString(styles.Muted.Render(line))
		case strings.HasPrefix(line, "==="):
			b.WriteString(styles.Title.Render(line))
		default:
			b.WriteString(styles.DiffContext.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}
