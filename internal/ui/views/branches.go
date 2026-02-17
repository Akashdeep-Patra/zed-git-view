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

// BranchView manages branches.
type BranchView struct {
	gitSvc   git.Service
	styles   ui.Styles
	width    int
	height   int
	branches []git.Branch
	cursor   int

	// Input mode for creating/renaming.
	inputMode bool
	inputKind branchInputKind
	input     textinput.Model
	renameSrc string
}

type branchInputKind int

const (
	branchInputCreate branchInputKind = iota
	branchInputRename
)

type branchResultMsg struct{ branches []git.Branch }

// NewBranchView creates a new BranchView.
func NewBranchView(gitSvc git.Service, styles ui.Styles) *BranchView {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.Width = 40
	return &BranchView{gitSvc: gitSvc, styles: styles, input: ti}
}

func (v *BranchView) Init() tea.Cmd { return v.refresh() }

func (v *BranchView) SetSize(w, h int) { v.width = w; v.height = h }

func (v *BranchView) refresh() tea.Cmd {
	return func() tea.Msg {
		branches, err := v.gitSvc.Branches()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return branchResultMsg{branches: branches}
	}
}

func (v *BranchView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case branchResultMsg:
		v.branches = msg.branches
		if v.cursor >= len(v.branches) && len(v.branches) > 0 {
			v.cursor = len(v.branches) - 1
		}
		return v, nil
	case common.RefreshMsg:
		return v, v.refresh()
	case tea.MouseMsg:
		return v.handleMouse(msg)

	case tea.KeyMsg:
		if v.inputMode {
			return v.updateInput(msg)
		}
		return v.updateNormal(msg)
	}
	return v, nil
}

func (v *BranchView) handleMouse(msg tea.MouseMsg) (common.View, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if v.cursor > 0 {
			v.cursor--
		}
	case tea.MouseButtonWheelDown:
		if v.cursor < len(v.branches)-1 {
			v.cursor++
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress || v.inputMode {
			break
		}
		// Content starts at Y=2, header is 2 lines ("Branches (N)" + blank).
		idx := msg.Y - 2 - 2
		if idx >= 0 && idx < len(v.branches) {
			v.cursor = idx
		}
	}
	return v, nil
}

func (v *BranchView) updateNormal(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.branches)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "g", "home":
		v.cursor = 0
	case "G", "end":
		if len(v.branches) > 0 {
			v.cursor = len(v.branches) - 1
		}
	case "enter": // Switch
		if b, ok := v.currentBranch(); ok && !b.IsCurrent {
			return v, v.switchBranch(b.Name)
		}
	case "n": // New branch
		v.inputMode = true
		v.inputKind = branchInputCreate
		v.input.Placeholder = "new-branch-name"
		v.input.Reset()
		v.input.Focus()
		return v, v.input.Focus()
	case "R": // Rename
		if b, ok := v.currentBranch(); ok && !b.IsRemote {
			v.inputMode = true
			v.inputKind = branchInputRename
			v.renameSrc = b.Name
			v.input.Placeholder = b.Name
			v.input.Reset()
			v.input.Focus()
			return v, v.input.Focus()
		}
	case "D": // Delete
		if b, ok := v.currentBranch(); ok && !b.IsCurrent && !b.IsRemote {
			return v, v.deleteBranch(b.Name)
		}
	case "m": // Merge
		if b, ok := v.currentBranch(); ok && !b.IsCurrent {
			return v, v.mergeBranch(b.Name)
		}
	}
	return v, nil
}

func (v *BranchView) updateInput(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.inputMode = false
		v.input.Blur()
		return v, nil
	case "enter":
		name := strings.TrimSpace(v.input.Value())
		v.inputMode = false
		v.input.Blur()
		if name == "" {
			return v, nil
		}
		switch v.inputKind {
		case branchInputCreate:
			return v, v.createBranch(name)
		case branchInputRename:
			return v, v.renameBranch(v.renameSrc, name)
		}
		return v, nil
	}
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return v, cmd
}

func (v *BranchView) switchBranch(name string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.SwitchBranch(name); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BranchView) createBranch(name string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.CreateBranch(name); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BranchView) deleteBranch(name string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.DeleteBranch(name, false); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BranchView) mergeBranch(name string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.MergeBranch(name); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BranchView) renameBranch(oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.RenameBranch(oldName, newName); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BranchView) View() string {
	if v.inputMode {
		return v.viewInput()
	}
	return v.viewList()
}

func (v *BranchView) viewList() string {
	t := v.styles.Theme
	if len(v.branches) == 0 {
		return ui.PlaceCentre(v.width, v.height,
			lipgloss.NewStyle().Foreground(t.TextMuted).Render("No branches found"))
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(t.Primary).Bold(true).
		Render(fmt.Sprintf("  Branches (%d)", len(v.branches))) + "\n\n")

	for i, br := range v.branches {
		line := v.renderBranchLine(br)
		if i == v.cursor {
			b.WriteString(v.styles.ListSelected.Render("▸ "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n" + v.styles.Muted.Render("  enter switch  n new  R rename  D delete  m merge"))
	return b.String()
}

func (v *BranchView) renderBranchLine(br git.Branch) string {
	t := v.styles.Theme
	var parts []string

	switch {
	case br.IsCurrent:
		parts = append(parts, lipgloss.NewStyle().Foreground(t.BranchHead).Bold(true).Render("* "+br.Name))
	case br.IsRemote:
		parts = append(parts, v.styles.RemoteName.Render(br.Name))
	default:
		parts = append(parts, v.styles.BranchName.Render(br.Name))
	}

	parts = append(parts, v.styles.CommitHash.Render(br.Hash))

	if br.Upstream != "" {
		track := br.Upstream
		if br.Ahead > 0 || br.Behind > 0 {
			track += fmt.Sprintf(" [+%d/-%d]", br.Ahead, br.Behind)
		}
		parts = append(parts, v.styles.Muted.Render(track))
	}

	parts = append(parts, v.styles.Muted.Render(ui.Truncate(br.Subject, 40)))

	return strings.Join(parts, "  ")
}

func (v *BranchView) viewInput() string {
	t := v.styles.Theme
	var title string
	switch v.inputKind {
	case branchInputCreate:
		title = "Create New Branch"
	case branchInputRename:
		title = "Rename Branch: " + v.renameSrc
	}
	titleStr := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  " + title)
	hint := v.styles.Muted.Render("  enter to confirm | esc to cancel")
	return lipgloss.JoinVertical(lipgloss.Left, titleStr, "", "  "+v.input.View(), "", hint)
}

func (v *BranchView) currentBranch() (git.Branch, bool) {
	if v.cursor < 0 || v.cursor >= len(v.branches) {
		return git.Branch{}, false
	}
	return v.branches[v.cursor], true
}

func (v *BranchView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "enter", Desc: "Switch branch"},
		{Key: "n", Desc: "New branch"},
		{Key: "R", Desc: "Rename branch"},
		{Key: "D", Desc: "Delete branch"},
		{Key: "m", Desc: "Merge into current"},
	}
}
