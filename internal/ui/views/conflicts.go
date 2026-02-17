package views

import (
	"fmt"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConflictView helps resolve merge conflicts.
type ConflictView struct {
	gitSvc   git.Service
	styles   ui.Styles
	width    int
	height   int
	files    []string
	cursor   int
	diffVP   viewport.Model
	showDiff bool
}

type (
	conflictFilesMsg struct{ files []string }
	conflictDiffMsg  struct{ diff string }
)

// NewConflictView creates a new ConflictView.
func NewConflictView(gitSvc git.Service, styles ui.Styles) *ConflictView {
	return &ConflictView{gitSvc: gitSvc, styles: styles}
}

func (v *ConflictView) Init() tea.Cmd { return v.refresh() }

func (v *ConflictView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.diffVP.Width = w / 2
	v.diffVP.Height = h - 2
}

func (v *ConflictView) refresh() tea.Cmd {
	return func() tea.Msg {
		files, err := v.gitSvc.ConflictFiles()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return conflictFilesMsg{files: files}
	}
}

func (v *ConflictView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case conflictFilesMsg:
		v.files = msg.files
		if v.cursor >= len(v.files) && len(v.files) > 0 {
			v.cursor = len(v.files) - 1
		}
		return v, nil

	case conflictDiffMsg:
		v.showDiff = true
		v.diffVP = viewport.New(v.width/2, v.height-2)
		v.diffVP.SetContent(renderDiffColored(v.styles, msg.diff))
		return v, nil

	case common.RefreshMsg:
		return v, v.refresh()

	case tea.KeyMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *ConflictView) handleKey(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.files)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "m": // Mark resolved
		if v.cursor < len(v.files) {
			return v, v.markResolved(v.files[v.cursor])
		}
	case "d", "enter": // Show diff
		if v.cursor < len(v.files) {
			return v, v.showConflictDiff(v.files[v.cursor])
		}
	case "esc":
		v.showDiff = false
	}
	return v, nil
}

func (v *ConflictView) markResolved(path string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.MarkResolved(path); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *ConflictView) showConflictDiff(path string) tea.Cmd {
	return func() tea.Msg {
		diff, err := v.gitSvc.Diff(false, path)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return conflictDiffMsg{diff: diff}
	}
}

func (v *ConflictView) View() string {
	t := v.styles.Theme
	if len(v.files) == 0 {
		return ui.PlaceCentre(v.width, v.height,
			lipgloss.NewStyle().Foreground(t.Success).Render("No merge conflicts"))
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(t.Conflict).Bold(true).
		Render(fmt.Sprintf("  Conflicts (%d)", len(v.files))) + "\n\n")

	for i, f := range v.files {
		icon := lipgloss.NewStyle().Foreground(t.Conflict).Render("[U] ")
		line := icon + v.styles.FileConflict.Render(f)
		if i == v.cursor {
			b.WriteString(v.styles.ListSelected.Render("▸ "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n" + v.styles.Muted.Render("  m mark resolved  d/enter show diff"))

	left := b.String()
	if v.showDiff {
		right := v.styles.Panel.Width(v.width/2 - 2).Height(v.height - 2).
			Render(v.diffVP.View())
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	return left
}

func (v *ConflictView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "m", Desc: "Mark resolved"},
		{Key: "d / enter", Desc: "Show diff"},
	}
}

func (v *ConflictView) InputCapture() bool { return false }
