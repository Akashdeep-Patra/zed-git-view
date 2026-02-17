package views

import (
	"fmt"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StashView manages the stash list.
type StashView struct {
	gitSvc  git.Service
	styles  ui.Styles
	width   int
	height  int
	entries []git.StashEntry
	cursor  int

	// Save mode
	saving bool
	input  textinput.Model

	// Detail
	showDetail bool
	detailVP   viewport.Model
}

type (
	stashListMsg struct{ entries []git.StashEntry }
	stashDiffMsg struct{ diff string }
)

// NewStashView creates a new StashView.
func NewStashView(gitSvc git.Service, styles ui.Styles) *StashView {
	ti := textinput.New()
	ti.Placeholder = "stash message (optional)"
	ti.CharLimit = 200
	ti.Width = 50
	return &StashView{gitSvc: gitSvc, styles: styles, input: ti}
}

func (v *StashView) Init() tea.Cmd { return v.refresh() }

func (v *StashView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.detailVP.Width = w / 2
	v.detailVP.Height = h - 2
}

func (v *StashView) refresh() tea.Cmd {
	return func() tea.Msg {
		entries, err := v.gitSvc.StashList()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return stashListMsg{entries: entries}
	}
}

func (v *StashView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case stashListMsg:
		v.entries = msg.entries
		if v.cursor >= len(v.entries) && len(v.entries) > 0 {
			v.cursor = len(v.entries) - 1
		}
		return v, nil

	case stashDiffMsg:
		v.showDetail = true
		v.detailVP = viewport.New(v.width/2, v.height-2)
		v.detailVP.SetContent(renderDiffColored(v.styles, msg.diff))
		return v, nil

	case common.RefreshMsg:
		return v, v.refresh()

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if v.showDetail {
				v.detailVP.ScrollUp(3)
			} else if v.cursor > 0 {
				v.cursor--
			}
		case tea.MouseButtonWheelDown:
			if v.showDetail {
				v.detailVP.ScrollDown(3)
			} else if v.cursor < len(v.entries)-1 {
				v.cursor++
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress && !v.saving && !v.showDetail {
				idx := msg.Y - 2 - 2
				if idx >= 0 && idx < len(v.entries) {
					v.cursor = idx
				}
			}
		}
		return v, nil

	case tea.KeyMsg:
		if v.saving {
			return v.updateSaveMode(msg)
		}
		return v.updateNormal(msg)
	}
	return v, nil
}

func (v *StashView) updateNormal(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.entries)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "s": // Save/push new stash
		v.saving = true
		v.input.Reset()
		v.input.Focus()
		return v, v.input.Focus()
	case "p": // Pop
		if v.cursor < len(v.entries) {
			return v, v.stashPop(v.entries[v.cursor].Index)
		}
	case "a": // Apply
		if v.cursor < len(v.entries) {
			return v, v.stashApply(v.entries[v.cursor].Index)
		}
	case "D": // Drop
		if v.cursor < len(v.entries) {
			return v, v.stashDrop(v.entries[v.cursor].Index)
		}
	case "enter", "d": // Show diff
		if v.cursor < len(v.entries) {
			return v, v.stashShow(v.entries[v.cursor].Index)
		}
	case "esc":
		v.showDetail = false
	}
	return v, nil
}

func (v *StashView) updateSaveMode(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.saving = false
		v.input.Blur()
		return v, nil
	case "enter":
		message := strings.TrimSpace(v.input.Value())
		v.saving = false
		v.input.Blur()
		return v, v.stashSave(message)
	}
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return v, cmd
}

func (v *StashView) stashSave(message string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.StashSave(message); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StashView) stashPop(idx int) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.StashPop(idx); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StashView) stashApply(idx int) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.StashApply(idx); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StashView) stashDrop(idx int) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.StashDrop(idx); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StashView) stashShow(idx int) tea.Cmd {
	return func() tea.Msg {
		diff, err := v.gitSvc.StashShow(idx)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return stashDiffMsg{diff: diff}
	}
}

func (v *StashView) View() string {
	if v.saving {
		t := v.styles.Theme
		title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  Stash Save")
		hint := v.styles.Muted.Render("  enter to save | esc to cancel")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "  "+v.input.View(), "", hint)
	}

	left := v.viewList()
	if v.showDetail {
		right := v.styles.Panel.Width(v.width/2 - 2).Height(v.height - 2).
			Render(v.detailVP.View())
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	return left
}

func (v *StashView) viewList() string {
	t := v.styles.Theme
	if len(v.entries) == 0 {
		return ui.PlaceCentre(v.width, v.height,
			lipgloss.NewStyle().Foreground(t.TextMuted).Render("No stashes"))
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(t.Stash).Bold(true).
		Render(fmt.Sprintf("  Stash (%d)", len(v.entries))) + "\n\n")

	for i, e := range v.entries {
		idx := lipgloss.NewStyle().Foreground(t.CommitHash).Render(fmt.Sprintf("stash@{%d}", e.Index))
		msg := v.styles.Body.Render(ui.Truncate(e.Message, 50))
		branch := ""
		if e.Branch != "" {
			branch = v.styles.BranchName.Render(" on " + e.Branch)
		}
		line := idx + " " + msg + branch

		if i == v.cursor {
			b.WriteString(v.styles.ListSelected.Render("▸ "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n" + v.styles.Muted.Render("  s save  p pop  a apply  D drop  d/enter show diff"))
	return b.String()
}

func (v *StashView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "s", Desc: "Save stash"},
		{Key: "p", Desc: "Pop stash"},
		{Key: "a", Desc: "Apply stash"},
		{Key: "D", Desc: "Drop stash"},
		{Key: "d / enter", Desc: "Show stash diff"},
	}
}

func (v *StashView) InputCapture() bool { return false }
