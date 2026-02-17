package views

import (
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RebaseView handles interactive rebase operations.
type RebaseView struct {
	gitSvc    git.Service
	styles    ui.Styles
	width     int
	height    int
	rebasing  bool
	inputMode bool
	input     textinput.Model
}

// NewRebaseView creates a new RebaseView.
func NewRebaseView(gitSvc git.Service, styles ui.Styles) *RebaseView {
	ti := textinput.New()
	ti.Placeholder = "commit hash or branch (e.g. HEAD~3, main)"
	ti.CharLimit = 100
	ti.Width = 50
	return &RebaseView{gitSvc: gitSvc, styles: styles, input: ti}
}

func (v *RebaseView) Init() tea.Cmd {
	v.rebasing = v.gitSvc.IsRebasing()
	return nil
}

func (v *RebaseView) SetSize(w, h int) { v.width = w; v.height = h }

func (v *RebaseView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case common.RefreshMsg:
		v.rebasing = v.gitSvc.IsRebasing()
		return v, nil
	case tea.KeyMsg:
		if v.inputMode {
			return v.updateInput(msg)
		}
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *RebaseView) handleKey(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "i": // Start interactive rebase
		v.inputMode = true
		v.input.Reset()
		v.input.Focus()
		return v, v.input.Focus()
	case "c": // Continue
		if v.rebasing {
			return v, v.rebaseContinue()
		}
	case "a": // Abort
		if v.rebasing {
			return v, v.rebaseAbort()
		}
	}
	return v, nil
}

func (v *RebaseView) updateInput(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.inputMode = false
		v.input.Blur()
		return v, nil
	case "enter":
		onto := strings.TrimSpace(v.input.Value())
		v.inputMode = false
		v.input.Blur()
		if onto == "" {
			return v, nil
		}
		return v, v.rebaseStart(onto)
	}
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return v, cmd
}

func (v *RebaseView) rebaseStart(onto string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.RebaseInteractive(onto); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *RebaseView) rebaseContinue() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.RebaseContinue(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *RebaseView) rebaseAbort() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.RebaseAbort(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *RebaseView) View() string {
	t := v.styles.Theme
	if v.inputMode {
		title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  Interactive Rebase")
		hint := v.styles.Muted.Render("  enter to start | esc to cancel")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "  Rebase onto:", "  "+v.input.View(), "", hint)
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  Rebase")
	b.WriteString(title + "\n\n")

	if v.rebasing {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Warning).Bold(true).
			Render("  REBASE IN PROGRESS") + "\n\n")
		b.WriteString("  " + v.styles.Muted.Render("Resolve conflicts, stage changes, then:") + "\n\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "c", "continue rebase") + "\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "a", "abort rebase") + "\n")
	} else {
		b.WriteString("  " + v.styles.Body.Render("No rebase in progress.") + "\n\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "i", "start interactive rebase") + "\n")
	}

	return b.String()
}

func (v *RebaseView) ShortHelp() []components.HelpEntry {
	if v.rebasing {
		return []components.HelpEntry{
			{Key: "c", Desc: "Continue rebase"},
			{Key: "a", Desc: "Abort rebase"},
		}
	}
	return []components.HelpEntry{
		{Key: "i", Desc: "Start interactive rebase"},
	}
}
