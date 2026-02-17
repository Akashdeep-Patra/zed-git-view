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

// defaultLogLimit controls how many commits to load. Kept modest to ensure
// fast rendering even on repos with thousands of branches/tags.
const defaultLogLimit = 100

// LogView shows the commit log with an ASCII graph.
type LogView struct {
	gitSvc  git.Service
	styles  ui.Styles
	width   int
	height  int
	entries []git.GraphEntry
	commits []git.Commit // flat list for cursor navigation
	cursor  int
	vp      viewport.Model

	// Detail pane.
	showDetail bool
	detailVP   viewport.Model
}

// NewLogView creates a new LogView.
func NewLogView(gitSvc git.Service, styles ui.Styles) *LogView {
	return &LogView{
		gitSvc: gitSvc,
		styles: styles,
		vp:     viewport.New(0, 0),
	}
}

func (v *LogView) Init() tea.Cmd { return v.refresh() }

func (v *LogView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.vp.Width = w
	v.vp.Height = h - 2
	v.detailVP.Width = w / 2
	v.detailVP.Height = h - 2
}

type logResultMsg struct {
	entries []git.GraphEntry
	commits []git.Commit
}

type commitDetailMsg struct {
	commit *git.Commit
	diff   string
}

func (v *LogView) refresh() tea.Cmd {
	return func() tea.Msg {
		entries, err := v.gitSvc.LogGraph(defaultLogLimit)
		if err != nil {
			// Fall back to non-graph log.
			commits, err2 := v.gitSvc.Log(defaultLogLimit)
			if err2 != nil {
				return common.ErrMsg{Err: err2}
			}
			return logResultMsg{commits: commits}
		}
		var commits []git.Commit
		for _, e := range entries {
			if e.Commit != nil {
				commits = append(commits, *e.Commit)
			}
		}
		return logResultMsg{entries: entries, commits: commits}
	}
}

func (v *LogView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case logResultMsg:
		v.entries = msg.entries
		v.commits = msg.commits
		if v.cursor >= len(v.commits) && len(v.commits) > 0 {
			v.cursor = len(v.commits) - 1
		}
		v.rebuildContent()
		return v, nil

	case commitDetailMsg:
		v.showDetail = true
		v.detailVP = viewport.New(v.width/2, v.height-2)
		v.detailVP.SetContent(v.renderCommitDetail(msg.commit, msg.diff))
		return v, nil

	case common.RefreshMsg:
		return v, v.refresh()

	case tea.MouseMsg:
		return v.handleMouse(msg)

	case tea.KeyMsg:
		return v.handleKey(msg)
	}

	var cmd tea.Cmd
	if v.showDetail {
		v.detailVP, cmd = v.detailVP.Update(msg)
	}
	return v, cmd
}

func (v *LogView) handleMouse(msg tea.MouseMsg) (common.View, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if v.showDetail {
			v.detailVP.ScrollUp(3)
		} else {
			v.vp.ScrollUp(3)
			if v.cursor > 0 {
				v.cursor--
				v.rebuildContent()
			}
		}
	case tea.MouseButtonWheelDown:
		if v.showDetail {
			v.detailVP.ScrollDown(3)
		} else {
			v.vp.ScrollDown(3)
			if v.cursor < len(v.commits)-1 {
				v.cursor++
				v.rebuildContent()
			}
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			break
		}
		if v.showDetail {
			break
		}
		// Rough click-to-select: compute item from Y position.
		contentY := msg.Y - 2 // tab bar height
		if contentY >= 0 && contentY < len(v.commits) {
			v.cursor = contentY
			v.rebuildContent()
		}
	}
	return v, nil
}

func (v *LogView) handleKey(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.commits)-1 {
			v.cursor++
			v.rebuildContent()
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
			v.rebuildContent()
		}
	case "g", "home":
		v.cursor = 0
		v.rebuildContent()
	case "G", "end":
		if len(v.commits) > 0 {
			v.cursor = len(v.commits) - 1
			v.rebuildContent()
		}
	case "enter", "d":
		if v.cursor < len(v.commits) {
			c := v.commits[v.cursor]
			return v, v.loadDetail(c.Hash)
		}
	case "y":
		if v.cursor < len(v.commits) {
			return v, common.CmdInfo("Copied: " + v.commits[v.cursor].ShortHash)
		}
	case "esc":
		v.showDetail = false
	case "ctrl+d", "pgdown":
		v.vp.HalfPageDown()
	case "ctrl+u", "pgup":
		v.vp.HalfPageUp()
	}
	return v, nil
}

func (v *LogView) loadDetail(hash string) tea.Cmd {
	return func() tea.Msg {
		commit, diff, err := v.gitSvc.Show(hash)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return commitDetailMsg{commit: commit, diff: diff}
	}
}

func (v *LogView) View() string {
	if v.showDetail {
		left := v.vp.View()
		right := v.styles.Panel.Width(v.width/2 - 2).Height(v.height - 2).
			Render(v.detailVP.View())
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	return v.vp.View()
}

func (v *LogView) rebuildContent() {
	t := v.styles.Theme
	var b strings.Builder
	commitIdx := 0

	if len(v.entries) > 0 {
		for _, e := range v.entries {
			graphStyle := lipgloss.NewStyle().Foreground(t.GraphColors[commitIdx%len(t.GraphColors)])
			graph := graphStyle.Render(e.Graph)

			if e.Commit != nil {
				line := v.renderCommitLine(e.Commit, commitIdx == v.cursor)
				b.WriteString(graph + line + "\n")
				commitIdx++
			} else {
				b.WriteString(graph + "\n")
			}
		}
	} else {
		for i, c := range v.commits {
			b.WriteString(v.renderCommitLine(&c, i == v.cursor) + "\n")
		}
	}

	if len(v.commits) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(t.TextMuted).Render("  No commits found"))
	}

	b.WriteString("\n" + v.styles.Muted.Render("  enter/d detail  y copy hash  j/k navigate"))
	v.vp.SetContent(b.String())
}

func (v *LogView) renderCommitLine(c *git.Commit, selected bool) string {
	t := v.styles.Theme
	hash := v.styles.CommitHash.Render(c.ShortHash)
	subj := v.styles.CommitMsg.Render(ui.Truncate(c.Subject, 60))
	author := v.styles.Author.Render(c.Author)
	date := v.styles.Date.Render(c.RelDate)

	refs := v.renderRefs(c.Refs)

	line := fmt.Sprintf(" %s %s%s %s %s", hash, subj, refs, author, date)

	if selected {
		return lipgloss.NewStyle().Background(t.SurfaceHover).Bold(true).Render("▸" + line)
	}
	return " " + line
}

func (v *LogView) renderRefs(refs []git.Ref) string {
	if len(refs) == 0 {
		return ""
	}
	var parts []string
	for _, r := range refs {
		switch r.Type {
		case git.RefHead:
			parts = append(parts, v.styles.BranchName.Render(" "+r.Name))
		case git.RefBranch:
			parts = append(parts, v.styles.BranchName.Render(r.Name))
		case git.RefRemoteBranch:
			parts = append(parts, v.styles.RemoteName.Render(r.Remote+"/"+r.Name))
		case git.RefTag:
			parts = append(parts, v.styles.TagName.Render("  "+r.Name))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

func (v *LogView) renderCommitDetail(c *git.Commit, diff string) string {
	t := v.styles.Theme
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Commit Detail") + "\n\n")
	b.WriteString(v.styles.Muted.Render("Hash:    ") + v.styles.CommitHash.Render(c.Hash) + "\n")
	b.WriteString(v.styles.Muted.Render("Author:  ") + v.styles.Author.Render(c.Author+" <"+c.AuthorEmail+">") + "\n")
	b.WriteString(v.styles.Muted.Render("Date:    ") + v.styles.Date.Render(c.Date.Format("2006-01-02 15:04:05")) + "\n")

	if len(c.Parents) > 0 {
		b.WriteString(v.styles.Muted.Render("Parents: ") + v.styles.CommitHash.Render(strings.Join(c.Parents, " ")) + "\n")
	}
	if len(c.Refs) > 0 {
		b.WriteString(v.styles.Muted.Render("Refs:    ") + v.renderRefs(c.Refs) + "\n")
	}

	b.WriteString("\n" + v.styles.Bold.Render(c.Subject) + "\n")
	if c.Body != "" {
		b.WriteString("\n" + v.styles.Body.Render(c.Body) + "\n")
	}

	if diff != "" {
		b.WriteString("\n" + renderDiffColored(v.styles, diff))
	}

	return b.String()
}

func (v *LogView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate commits"},
		{Key: "enter / d", Desc: "Show commit detail"},
		{Key: "y", Desc: "Copy commit hash"},
		{Key: "home/end", Desc: "Top / bottom"},
		{Key: "esc", Desc: "Close detail"},
	}
}

func (v *LogView) InputCapture() bool { return false }
