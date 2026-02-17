package views

import (
	"fmt"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorktreeView manages linked working trees.
type WorktreeView struct {
	gitSvc    git.Service
	styles    ui.Styles
	width     int
	height    int
	worktrees []git.Worktree
	cursor    int
	adding    bool
	pathInput textinput.Model
	brInput   textinput.Model
	inputStep int // 0=path, 1=branch
}

type worktreeListMsg struct{ wts []git.Worktree }

// NewWorktreeView creates a new WorktreeView.
func NewWorktreeView(gitSvc git.Service, styles ui.Styles) *WorktreeView {
	pi := textinput.New()
	pi.Placeholder = "/path/to/worktree"
	pi.CharLimit = 200
	pi.Width = 50

	bi := textinput.New()
	bi.Placeholder = "branch name (optional)"
	bi.CharLimit = 100
	bi.Width = 50

	return &WorktreeView{gitSvc: gitSvc, styles: styles, pathInput: pi, brInput: bi}
}

func (v *WorktreeView) Init() tea.Cmd { return v.refresh() }

func (v *WorktreeView) SetSize(w, h int) { v.width = w; v.height = h }

func (v *WorktreeView) refresh() tea.Cmd {
	return func() tea.Msg {
		wts, err := v.gitSvc.WorktreeList()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return worktreeListMsg{wts: wts}
	}
}

func (v *WorktreeView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case worktreeListMsg:
		v.worktrees = msg.wts
		if v.cursor >= len(v.worktrees) && len(v.worktrees) > 0 {
			v.cursor = len(v.worktrees) - 1
		}
		return v, nil
	case common.RefreshMsg:
		return v, v.refresh()
	case tea.KeyMsg:
		if v.adding {
			return v.updateAdd(msg)
		}
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *WorktreeView) handleKey(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.worktrees)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "n": // Add worktree
		v.adding = true
		v.inputStep = 0
		v.pathInput.Reset()
		v.brInput.Reset()
		v.pathInput.Focus()
		return v, v.pathInput.Focus()
	case "D": // Remove
		if v.cursor > 0 && v.cursor < len(v.worktrees) {
			return v, v.removeWorktree(v.worktrees[v.cursor].Path)
		}
	}
	return v, nil
}

func (v *WorktreeView) updateAdd(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.adding = false
		return v, nil
	case "enter":
		if v.inputStep == 0 {
			v.inputStep = 1
			v.pathInput.Blur()
			v.brInput.Focus()
			return v, v.brInput.Focus()
		}
		path := strings.TrimSpace(v.pathInput.Value())
		branch := strings.TrimSpace(v.brInput.Value())
		v.adding = false
		if path == "" {
			return v, nil
		}
		return v, v.addWorktree(path, branch)
	case "tab":
		if v.inputStep == 0 {
			v.inputStep = 1
			v.pathInput.Blur()
			v.brInput.Focus()
			return v, v.brInput.Focus()
		}
	}
	var cmd tea.Cmd
	if v.inputStep == 0 {
		v.pathInput, cmd = v.pathInput.Update(msg)
	} else {
		v.brInput, cmd = v.brInput.Update(msg)
	}
	return v, cmd
}

func (v *WorktreeView) addWorktree(path, branch string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.WorktreeAdd(path, branch); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *WorktreeView) removeWorktree(path string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.WorktreeRemove(path); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *WorktreeView) View() string {
	t := v.styles.Theme

	if v.adding {
		title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  Add Worktree")
		pathLabel := v.styles.Body.Render("  Path:")
		brLabel := v.styles.Body.Render("  Branch:")
		hint := v.styles.Muted.Render("  tab to switch field | enter to confirm | esc to cancel")
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "", pathLabel, "  "+v.pathInput.View(), "", brLabel, "  "+v.brInput.View(), "", hint)
	}

	if len(v.worktrees) == 0 {
		return ui.PlaceCentre(v.width, v.height,
			lipgloss.NewStyle().Foreground(t.TextMuted).Render("No worktrees"))
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(t.Primary).Bold(true).
		Render(fmt.Sprintf("  Worktrees (%d)", len(v.worktrees))) + "\n\n")

	for i, wt := range v.worktrees {
		path := v.styles.Body.Render(wt.Path)
		branch := ""
		if wt.Branch != "" {
			branch = v.styles.BranchName.Render(" [" + wt.Branch + "]")
		}
		head := v.styles.CommitHash.Render(ui.Truncate(wt.Head, 8))
		bare := ""
		if wt.Bare {
			bare = v.styles.Muted.Render(" (bare)")
		}
		line := path + branch + " " + head + bare

		if i == v.cursor {
			b.WriteString(v.styles.ListSelected.Render("▸ "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n" + v.styles.Muted.Render("  n add worktree  D remove"))
	return b.String()
}

func (v *WorktreeView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "n", Desc: "Add worktree"},
		{Key: "D", Desc: "Remove worktree"},
	}
}

func (v *WorktreeView) InputCapture() bool { return false }
