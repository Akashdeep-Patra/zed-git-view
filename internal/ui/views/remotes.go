package views

import (
	"fmt"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RemoteView manages remotes and push/pull/fetch operations.
type RemoteView struct {
	gitSvc  git.Service
	styles  ui.Styles
	width   int
	height  int
	remotes []git.Remote
	cursor  int
	loading bool
}

type (
	remoteListMsg   struct{ remotes []git.Remote }
	remoteOpDoneMsg struct{ info string }
)

// NewRemoteView creates a new RemoteView.
func NewRemoteView(gitSvc git.Service, styles ui.Styles) *RemoteView {
	return &RemoteView{gitSvc: gitSvc, styles: styles}
}

func (v *RemoteView) Init() tea.Cmd { return v.refresh() }

func (v *RemoteView) SetSize(w, h int) { v.width = w; v.height = h }

func (v *RemoteView) refresh() tea.Cmd {
	return func() tea.Msg {
		remotes, err := v.gitSvc.Remotes()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return remoteListMsg{remotes: remotes}
	}
}

func (v *RemoteView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case remoteListMsg:
		v.remotes = msg.remotes
		v.loading = false
		if v.cursor >= len(v.remotes) && len(v.remotes) > 0 {
			v.cursor = len(v.remotes) - 1
		}
		return v, nil

	case remoteOpDoneMsg:
		v.loading = false
		return v, tea.Batch(
			common.CmdInfo(msg.info),
			common.CmdRefresh,
		)

	case common.RefreshMsg:
		return v, v.refresh()

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if v.cursor > 0 {
				v.cursor--
			}
		case tea.MouseButtonWheelDown:
			if v.cursor < len(v.remotes)-1 {
				v.cursor++
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				idx := msg.Y - 2 - 2
				if idx >= 0 && idx < len(v.remotes) {
					v.cursor = idx
				}
			}
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *RemoteView) handleKey(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.remotes)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "f": // Fetch
		if r, ok := v.currentRemote(); ok {
			v.loading = true
			return v, v.fetch(r.Name)
		}
	case "F": // Fetch all
		v.loading = true
		return v, v.fetchAll()
	case "p": // Pull
		if r, ok := v.currentRemote(); ok {
			v.loading = true
			head, _ := v.gitSvc.Head()
			return v, v.pull(r.Name, head)
		}
	case "P": // Push
		if r, ok := v.currentRemote(); ok {
			v.loading = true
			head, _ := v.gitSvc.Head()
			return v, v.push(r.Name, head)
		}
	}
	return v, nil
}

func (v *RemoteView) fetch(remote string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Fetch(remote); err != nil {
			return common.ErrMsg{Err: err}
		}
		return remoteOpDoneMsg{info: "Fetched from " + remote}
	}
}

func (v *RemoteView) fetchAll() tea.Cmd {
	return func() tea.Msg {
		for _, r := range v.remotes {
			if err := v.gitSvc.Fetch(r.Name); err != nil {
				return common.ErrMsg{Err: err}
			}
		}
		return remoteOpDoneMsg{info: "Fetched from all remotes"}
	}
}

func (v *RemoteView) pull(remote, branch string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Pull(remote, branch); err != nil {
			return common.ErrMsg{Err: err}
		}
		return remoteOpDoneMsg{info: fmt.Sprintf("Pulled %s from %s", branch, remote)}
	}
}

func (v *RemoteView) push(remote, branch string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Push(remote, branch, false); err != nil {
			return common.ErrMsg{Err: err}
		}
		return remoteOpDoneMsg{info: fmt.Sprintf("Pushed %s to %s", branch, remote)}
	}
}

func (v *RemoteView) View() string {
	t := v.styles.Theme
	if len(v.remotes) == 0 {
		return ui.PlaceCentre(v.width, v.height,
			lipgloss.NewStyle().Foreground(t.TextMuted).Render("No remotes configured"))
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(t.Remote).Bold(true).
		Render(fmt.Sprintf("  Remotes (%d)", len(v.remotes))) + "\n\n")

	for i, r := range v.remotes {
		name := lipgloss.NewStyle().Foreground(t.Remote).Bold(true).Render(r.Name)
		fetch := v.styles.Muted.Render("fetch: " + r.FetchURL)
		push := v.styles.Muted.Render("push:  " + r.PushURL)
		line := name + "\n      " + fetch + "\n      " + push

		if i == v.cursor {
			b.WriteString(v.styles.ListSelected.Render("▸ "+line) + "\n\n")
		} else {
			b.WriteString("  " + line + "\n\n")
		}
	}

	if v.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Warning).Render("  Working...") + "\n")
	}

	b.WriteString(v.styles.Muted.Render("  f fetch  F fetch all  p pull  P push"))
	return b.String()
}

func (v *RemoteView) currentRemote() (git.Remote, bool) {
	if v.cursor < 0 || v.cursor >= len(v.remotes) {
		return git.Remote{}, false
	}
	return v.remotes[v.cursor], true
}

func (v *RemoteView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "f", Desc: "Fetch from remote"},
		{Key: "F", Desc: "Fetch all remotes"},
		{Key: "p", Desc: "Pull"},
		{Key: "P", Desc: "Push"},
	}
}

func (v *RemoteView) InputCapture() bool { return false }
