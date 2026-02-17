package views

import (
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

// BisectView manages interactive git bisect.
type BisectView struct {
	gitSvc    git.Service
	styles    ui.Styles
	width     int
	height    int
	active    bool // whether bisect is in progress
	log       string
	logVP     viewport.Model
	inputMode bool
	inputStep int // 0=bad, 1=good
	badInput  textinput.Model
	goodInput textinput.Model
}

type bisectLogMsg struct{ log string }

// NewBisectView creates a new BisectView.
func NewBisectView(gitSvc git.Service, styles ui.Styles) *BisectView {
	bad := textinput.New()
	bad.Placeholder = "bad commit (e.g. HEAD)"
	bad.CharLimit = 100
	bad.Width = 40

	good := textinput.New()
	good.Placeholder = "good commit (e.g. v1.0.0)"
	good.CharLimit = 100
	good.Width = 40

	return &BisectView{
		gitSvc:    gitSvc,
		styles:    styles,
		badInput:  bad,
		goodInput: good,
		logVP:     viewport.New(0, 0),
	}
}

func (v *BisectView) Init() tea.Cmd { return v.loadLog() }

func (v *BisectView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.logVP.Width = w - 4
	v.logVP.Height = h - 10
}

func (v *BisectView) loadLog() tea.Cmd {
	return func() tea.Msg {
		log, err := v.gitSvc.BisectLog()
		if err != nil {
			return bisectLogMsg{}
		}
		return bisectLogMsg{log: log}
	}
}

func (v *BisectView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case bisectLogMsg:
		v.log = msg.log
		v.active = msg.log != ""
		v.logVP.SetContent(msg.log)
		return v, nil
	case common.RefreshMsg:
		return v, v.loadLog()
	case tea.KeyMsg:
		if v.inputMode {
			return v.updateInput(msg)
		}
		return v.handleKey(msg)
	}
	var cmd tea.Cmd
	v.logVP, cmd = v.logVP.Update(msg)
	return v, cmd
}

func (v *BisectView) handleKey(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "b": // Start bisect
		if !v.active {
			v.inputMode = true
			v.inputStep = 0
			v.badInput.Reset()
			v.goodInput.Reset()
			v.badInput.Focus()
			return v, v.badInput.Focus()
		}
	case "g": // Good
		if v.active {
			return v, v.bisectGood()
		}
	case "B": // Bad
		if v.active {
			return v, v.bisectBad()
		}
	case "R": // Reset
		if v.active {
			return v, v.bisectReset()
		}
	}
	return v, nil
}

func (v *BisectView) updateInput(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.inputMode = false
		return v, nil
	case "enter":
		if v.inputStep == 0 {
			v.inputStep = 1
			v.badInput.Blur()
			v.goodInput.Focus()
			return v, v.goodInput.Focus()
		}
		bad := strings.TrimSpace(v.badInput.Value())
		good := strings.TrimSpace(v.goodInput.Value())
		v.inputMode = false
		if bad == "" || good == "" {
			return v, common.CmdErr(nil)
		}
		return v, v.bisectStart(bad, good)
	case "tab":
		if v.inputStep == 0 {
			v.inputStep = 1
			v.badInput.Blur()
			v.goodInput.Focus()
			return v, v.goodInput.Focus()
		}
	}
	var cmd tea.Cmd
	if v.inputStep == 0 {
		v.badInput, cmd = v.badInput.Update(msg)
	} else {
		v.goodInput, cmd = v.goodInput.Update(msg)
	}
	return v, cmd
}

func (v *BisectView) bisectStart(bad, good string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.BisectStart(bad, good); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BisectView) bisectGood() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.BisectGood(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BisectView) bisectBad() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.BisectBad(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BisectView) bisectReset() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.BisectReset(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *BisectView) View() string {
	t := v.styles.Theme

	if v.inputMode {
		title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  Start Bisect")
		hint := v.styles.Muted.Render("  tab to switch field | enter to start | esc to cancel")
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			"  Bad commit:", "  "+v.badInput.View(), "",
			"  Good commit:", "  "+v.goodInput.View(), "",
			hint)
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("  Bisect")
	b.WriteString(title + "\n\n")

	if v.active {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Warning).Bold(true).
			Render("  BISECT IN PROGRESS") + "\n\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "g", "mark current as good") + "\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "B", "mark current as bad") + "\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "R", "reset bisect") + "\n")

		if v.log != "" {
			b.WriteString("\n  " + v.styles.Subtitle.Render("Bisect Log:") + "\n")
			b.WriteString(v.logVP.View())
		}
	} else {
		b.WriteString("  " + v.styles.Body.Render("No bisect in progress.") + "\n\n")
		b.WriteString("  " + ui.RenderKeyValue(v.styles, "b", "start bisect") + "\n")
	}

	return b.String()
}

func (v *BisectView) ShortHelp() []components.HelpEntry {
	if v.active {
		return []components.HelpEntry{
			{Key: "g", Desc: "Mark good"},
			{Key: "B", Desc: "Mark bad"},
			{Key: "R", Desc: "Reset bisect"},
		}
	}
	return []components.HelpEntry{
		{Key: "b", Desc: "Start bisect"},
	}
}

func (v *BisectView) InputCapture() bool { return v.inputMode }
