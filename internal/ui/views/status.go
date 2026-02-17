package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Section identifiers ─────────────────────────────────────────────────────

type statusSection int

const (
	sectionStaged statusSection = iota
	sectionUnstaged
	sectionUntracked
	sectionConflicts
)

// ── Focus pane ──────────────────────────────────────────────────────────────

type focusPane int

const (
	focusFileList focusPane = iota
	focusDiffPane
)

// ── StatusView ──────────────────────────────────────────────────────────────

// StatusView is the primary working-tree view.
//
// Layout (when diff is visible):
//
//	┌─ Files ─────────┐┌─ Diff Preview ────────┐
//	│ ▸ M path/to/f… ││ @@ -1,3 +1,5 @@       │
//	│   A new_file.go ││ +added line            │
//	│   ? untracked   ││ -removed line          │
//	│                 ││                        │
//	└─────────────────┘└────────────────────────┘
//	 s stage  u unstage  c commit  d diff  x discard
type StatusView struct {
	gitSvc git.Service
	styles ui.Styles
	width  int
	height int
	status *git.StatusResult
	cursor int
	items  []statusItem

	// Focus pane.
	focus focusPane

	// Commit mode.
	commitTA   textarea.Model
	commitMode bool

	// Diff preview (inline, always visible in right pane).
	diffVP      viewport.Model
	diffContent string
	diffPath    string // path of the file whose diff is shown
	diffStaged  bool

	// Cached scroll state from last render — used by mouse click handler
	// so the hit-test exactly matches what's drawn on screen.
	lastScrollStart int
	lastListH       int
	lastListYOffset int // absolute terminal Y where the list area begins
}

type statusItem struct {
	file    git.FileStatus
	section statusSection
}

// ── Constructor ─────────────────────────────────────────────────────────────

func NewStatusView(gitSvc git.Service, styles ui.Styles) *StatusView {
	ta := textarea.New()
	ta.Placeholder = "Commit message..."
	ta.CharLimit = 0
	ta.SetWidth(60)
	ta.SetHeight(3)

	return &StatusView{
		gitSvc:   gitSvc,
		styles:   styles,
		status:   &git.StatusResult{},
		diffVP:   viewport.New(0, 0),
		commitTA: ta,
	}
}

// ── Init / SetSize ──────────────────────────────────────────────────────────

func (v *StatusView) Init() tea.Cmd { return v.refresh() }

func (v *StatusView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.commitTA.SetWidth(width - 6)

	// Diff pane takes ~60% of width.
	diffW := v.diffPaneWidth()
	v.diffVP.Width = diffW - 4   // -4 for border + padding
	v.diffVP.Height = height - 4 // -4 for title + command bar
}

func (v *StatusView) filePaneWidth() int {
	if v.width < 60 {
		return v.width // too narrow for split
	}
	return v.width * 2 / 5
}

func (v *StatusView) diffPaneWidth() int {
	if v.width < 60 {
		return 0
	}
	return v.width - v.filePaneWidth()
}

// ── Messages ────────────────────────────────────────────────────────────────

type (
	statusResultMsg struct{ status *git.StatusResult }
	diffPreviewMsg  struct{ diff string }
)

func (v *StatusView) refresh() tea.Cmd {
	return func() tea.Msg {
		status, err := v.gitSvc.Status()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return statusResultMsg{status: status}
	}
}

// ── Update ──────────────────────────────────────────────────────────────────

func (v *StatusView) Update(msg tea.Msg) (common.View, tea.Cmd) {
	switch msg := msg.(type) {
	case statusResultMsg:
		v.status = msg.status
		v.rebuildItems()
		if v.cursor >= len(v.items) && len(v.items) > 0 {
			v.cursor = len(v.items) - 1
		}
		// Auto-load diff for the selected file.
		return v, v.autoLoadDiff()

	case diffPreviewMsg:
		v.diffContent = msg.diff
		colored := renderDiffColored(v.styles, msg.diff)
		v.diffVP.SetContent(colored)
		v.diffVP.GotoTop()
		return v, nil

	case common.RefreshMsg:
		return v, v.refresh()

	case tea.MouseMsg:
		return v.handleMouse(msg)

	case tea.KeyMsg:
		if v.commitMode {
			return v.updateCommitMode(msg)
		}
		return v.updateNormal(msg)
	}

	if v.commitMode {
		var cmd tea.Cmd
		v.commitTA, cmd = v.commitTA.Update(msg)
		return v, cmd
	}
	return v, nil
}

// autoLoadDiff loads the diff for the currently selected file.
func (v *StatusView) autoLoadDiff() tea.Cmd {
	item, ok := v.currentItem()
	if !ok {
		return nil
	}
	// Skip if we already have the diff for this exact file+staged combo.
	staged := item.section == sectionStaged
	if item.file.Path == v.diffPath && staged == v.diffStaged {
		return nil
	}
	return v.loadDiffPreview(item)
}

// ── Mouse handler ───────────────────────────────────────────────────────────

func (v *StatusView) handleMouse(msg tea.MouseMsg) (common.View, tea.Cmd) {
	fpw := v.filePaneWidth()

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if msg.X < fpw {
			if v.cursor > 0 {
				v.cursor--
				return v, v.autoLoadDiff()
			}
		} else {
			v.diffVP.ScrollUp(3)
		}
	case tea.MouseButtonWheelDown:
		if msg.X < fpw {
			if v.cursor < len(v.items)-1 {
				v.cursor++
				return v, v.autoLoadDiff()
			}
		} else {
			v.diffVP.ScrollDown(3)
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			break
		}
		if v.commitMode {
			break
		}
		if msg.X < fpw {
			v.focus = focusFileList
			clickedItem := v.itemAtY(msg.Y)
			if clickedItem >= 0 && clickedItem < len(v.items) {
				v.cursor = clickedItem
				// Force-load diff on click (bypass dedup check).
				item := v.items[clickedItem]
				return v, v.loadDiffPreview(item)
			}
		} else {
			v.focus = focusDiffPane
		}
	}
	return v, nil
}

// ── Keyboard handler ────────────────────────────────────────────────────────

func (v *StatusView) updateNormal(msg tea.KeyMsg) (common.View, tea.Cmd) {
	// If diff pane is focused, handle scroll keys there.
	if v.focus == focusDiffPane {
		switch msg.String() {
		case "j", "down":
			v.diffVP.ScrollDown(1)
			return v, nil
		case "k", "up":
			v.diffVP.ScrollUp(1)
			return v, nil
		case "ctrl+d", "pgdown":
			v.diffVP.HalfPageDown()
			return v, nil
		case "ctrl+u", "pgup":
			v.diffVP.HalfPageUp()
			return v, nil
		case "g", "home":
			v.diffVP.GotoTop()
			return v, nil
		case "G", "end":
			v.diffVP.GotoBottom()
			return v, nil
		case "tab":
			v.focus = focusFileList
			return v, nil
		case "esc":
			v.focus = focusFileList
			return v, nil
		}
	}

	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.items)-1 {
			v.cursor++
			return v, v.autoLoadDiff()
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
			return v, v.autoLoadDiff()
		}
	case "g", "home":
		v.cursor = 0
		return v, v.autoLoadDiff()
	case "G", "end":
		if len(v.items) > 0 {
			v.cursor = len(v.items) - 1
			return v, v.autoLoadDiff()
		}
	case "ctrl+d", "pgdown":
		v.cursor = min(v.cursor+v.pageSize(), len(v.items)-1)
		return v, v.autoLoadDiff()
	case "ctrl+u", "pgup":
		v.cursor = max(v.cursor-v.pageSize(), 0)
		return v, v.autoLoadDiff()
	case "tab":
		if v.diffPaneWidth() > 0 {
			v.focus = focusDiffPane
		}
		return v, nil
	case "s":
		if item, ok := v.currentItem(); ok {
			return v, v.stageFile(item)
		}
	case "S":
		return v, v.stageAllFiles()
	case "u":
		if item, ok := v.currentItem(); ok {
			return v, v.unstageFile(item)
		}
	case "U":
		return v, v.unstageAllFiles()
	case "x":
		if item, ok := v.currentItem(); ok {
			return v, v.discardFile(item)
		}
	case "c":
		v.commitMode = true
		v.commitTA.Reset()
		v.commitTA.Focus()
		return v, v.commitTA.Focus()
	case "d", "enter":
		// Diff is already shown; pressing d/enter could toggle focus.
		if v.diffPaneWidth() > 0 {
			v.focus = focusDiffPane
		}
		return v, nil
	}
	return v, nil
}

func (v *StatusView) updateCommitMode(msg tea.KeyMsg) (common.View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.commitMode = false
		v.commitTA.Blur()
		return v, nil
	case "ctrl+s":
		message := strings.TrimSpace(v.commitTA.Value())
		if message == "" {
			return v, common.CmdErr(fmt.Errorf("commit message cannot be empty"))
		}
		v.commitMode = false
		v.commitTA.Blur()
		return v, v.doCommit(message)
	}
	var cmd tea.Cmd
	v.commitTA, cmd = v.commitTA.Update(msg)
	return v, cmd
}

// ── Actions ─────────────────────────────────────────────────────────────────

func (v *StatusView) stageFile(item statusItem) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Stage(item.file.Path); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StatusView) stageAllFiles() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.StageAll(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StatusView) unstageFile(item statusItem) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Unstage(item.file.Path); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StatusView) unstageAllFiles() tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.UnstageAll(); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StatusView) discardFile(item statusItem) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Discard(item.file.Path); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StatusView) doCommit(message string) tea.Cmd {
	return func() tea.Msg {
		if err := v.gitSvc.Commit(message); err != nil {
			return common.ErrMsg{Err: err}
		}
		return common.CmdRefresh()
	}
}

func (v *StatusView) loadDiffPreview(item statusItem) tea.Cmd {
	staged := item.section == sectionStaged
	path := item.file.Path
	v.diffPath = path
	v.diffStaged = staged
	return func() tea.Msg {
		diff, err := v.gitSvc.Diff(staged, path)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		if diff == "" {
			diff = "(no diff — file may be untracked or binary)"
		}
		return diffPreviewMsg{diff: diff}
	}
}

// ── View ────────────────────────────────────────────────────────────────────

func (v *StatusView) View() string {
	if v.commitMode {
		return v.viewCommit()
	}

	// Reserve 2 lines at the bottom for the persistent command bar.
	cmdBar := v.renderCommandBar()
	cmdBarH := lipgloss.Height(cmdBar)
	contentH := v.height - cmdBarH

	// Build the file list pane.
	filePane := v.renderFilePane(contentH)

	// Build diff preview pane (if terminal is wide enough).
	var mainContent string
	if dpw := v.diffPaneWidth(); dpw > 0 {
		diffPane := v.renderDiffPane(contentH, dpw)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, filePane, diffPane)
	} else {
		mainContent = filePane
	}

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, cmdBar)
}

// ── File pane ───────────────────────────────────────────────────────────────

func (v *StatusView) renderFilePane(height int) string {
	t := v.styles.Theme
	fpw := v.filePaneWidth()

	if v.status.TotalCount() == 0 {
		empty := lipgloss.NewStyle().
			Foreground(t.TextMuted).
			Width(fpw).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("✓ Working tree clean")
		return empty
	}

	// ── Title row ────────────────────────────────────────────
	total := v.status.TotalCount()
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(t.TextMuted)
	title := titleStyle.Render("Files") + " " + countStyle.Render(fmt.Sprintf("(%d)", total))
	if v.focus == focusFileList {
		title += " " + lipgloss.NewStyle().Foreground(t.Primary).Faint(true).Render("●")
	}

	// ── Section definitions ──────────────────────────────────
	type sectionDef struct {
		icon  string
		name  string
		items []git.FileStatus
		sec   statusSection
		color lipgloss.Color
	}
	sections := []sectionDef{
		{"✚", "Staged", v.status.Staged, sectionStaged, t.Added},
		{"●", "Modified", v.status.Unstaged, sectionUnstaged, t.Modified},
		{"?", "Untracked", v.status.Untracked, sectionUntracked, t.Untracked},
		{"⚡", "Conflicts", v.status.Conflicts, sectionConflicts, t.Conflict},
	}

	// Available height for file list = height - 1 (title).
	listH := height - 1
	if listH < 2 {
		listH = 2
	}

	// Max path width = pane width - prefix chars (icon + indicator + spaces).
	maxPath := fpw - 8
	if maxPath < 10 {
		maxPath = 10
	}

	// ── Build visible lines with virtual scrolling ───────────
	// First, compute total line count and cursor's line position.
	totalLines := 0
	cursorLine := 0
	itemIdx := 0
	for _, sec := range sections {
		if len(sec.items) == 0 {
			continue
		}
		totalLines++ // section header
		for range sec.items {
			if itemIdx == v.cursor {
				cursorLine = totalLines
			}
			totalLines++
			itemIdx++
		}
	}

	// Scroll window.
	scrollStart := 0
	if totalLines > listH {
		scrollStart = cursorLine - listH/2
		if scrollStart < 0 {
			scrollStart = 0
		}
		if scrollStart+listH > totalLines {
			scrollStart = totalLines - listH
		}
	}
	scrollEnd := scrollStart + listH
	if scrollEnd > totalLines {
		scrollEnd = totalLines
	}

	// Cache scroll state for mouse hit-testing.
	// The list area begins at absolute Y = 2 (tab bar) + 1 (title row) = 3.
	v.lastScrollStart = scrollStart
	v.lastListH = listH
	v.lastListYOffset = 3 // tab bar (2) + title row (1)

	// ── Render only visible lines ────────────────────────────
	var buf strings.Builder
	buf.Grow(listH * (maxPath + 16))

	lineIdx := 0
	itemIdx = 0
	rendered := 0

	headerStyle := lipgloss.NewStyle().Bold(true)

	for _, sec := range sections {
		if len(sec.items) == 0 {
			continue
		}

		// Section header.
		if lineIdx >= scrollStart && lineIdx < scrollEnd {
			if rendered > 0 {
				buf.WriteByte('\n')
			}
			hdr := headerStyle.Foreground(sec.color).
				Render(fmt.Sprintf(" %s %s %d", sec.icon, sec.name, len(sec.items)))
			buf.WriteString(hdr)
			rendered++
		}
		lineIdx++

		// File items.
		for _, f := range sec.items {
			if lineIdx >= scrollStart && lineIdx < scrollEnd {
				if rendered > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(v.renderFileItem(f, itemIdx == v.cursor, sec.color, maxPath))
				rendered++
			}
			lineIdx++
			itemIdx++
		}
	}

	// ── Compose pane ─────────────────────────────────────────
	listContent := buf.String()

	// Scrollbar indicator.
	var scrollHint string
	if totalLines > listH {
		pct := 0
		if totalLines-listH > 0 {
			pct = scrollStart * 100 / (totalLines - listH)
		}
		scrollHint = lipgloss.NewStyle().Foreground(t.TextSubtle).
			Render(fmt.Sprintf(" %d%% ", pct))
	}

	titleRow := lipgloss.NewStyle().Width(fpw).Render(
		" " + title + strings.Repeat(" ", max(0, fpw-lipgloss.Width(title)-lipgloss.Width(scrollHint)-2)) + scrollHint,
	)

	pane := lipgloss.JoinVertical(lipgloss.Left, titleRow, listContent)
	return lipgloss.NewStyle().Width(fpw).Height(height).Render(pane)
}

// renderFileItem renders a single file entry.
//
//	▸ M path/to/file.go     (selected, colored)
//	  A new_file.go          (normal, colored)
func (v *StatusView) renderFileItem(f git.FileStatus, selected bool, sectionColor lipgloss.Color, maxPath int) string {
	t := v.styles.Theme

	// Status indicator: single colored letter.
	code := f.Worktree
	if f.IsStaged {
		code = f.Staging
	}
	indicator := statusIndicator(code)
	indicatorColor := v.statusColor(code, sectionColor)

	// Path: show only filename for deep paths, full for short ones.
	path := f.Path
	if f.OrigPath != "" {
		path = f.Path + " ← " + filepath.Base(f.OrigPath)
	}
	if len(path) > maxPath {
		// Show "dir/…/filename" for long paths.
		dir := filepath.Dir(f.Path)
		base := filepath.Base(f.Path)
		available := maxPath - len(base) - 4 // 4 for "/…/"
		if available > 0 && dir != "." {
			path = ui.Truncate(dir, available) + "/" + base
		} else {
			path = ui.Truncate(path, maxPath)
		}
	}

	indicatorStyled := lipgloss.NewStyle().Foreground(indicatorColor).Bold(true).Render(indicator)
	pathStyled := lipgloss.NewStyle().Foreground(t.Text).Render(path)

	if selected {
		cursor := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("▸")
		line := fmt.Sprintf("%s %s %s", cursor, indicatorStyled, pathStyled)
		return lipgloss.NewStyle().Background(t.SurfaceHover).Render(" " + line)
	}

	return fmt.Sprintf("   %s %s", indicatorStyled, pathStyled)
}

// ── Diff pane ───────────────────────────────────────────────────────────────

func (v *StatusView) renderDiffPane(height, width int) string {
	t := v.styles.Theme

	// Title.
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	title := titleStyle.Render("Preview")
	if v.diffPath != "" {
		fname := filepath.Base(v.diffPath)
		title += " " + lipgloss.NewStyle().Foreground(t.TextMuted).Render(fname)
	}
	if v.focus == focusDiffPane {
		title += " " + lipgloss.NewStyle().Foreground(t.Primary).Faint(true).Render("●")
	}

	// Border style depends on focus.
	borderColor := t.Border
	if v.focus == focusDiffPane {
		borderColor = t.BorderFocused
	}

	innerW := width - 4 // border + padding
	if innerW < 10 {
		innerW = 10
	}
	innerH := height - 3 // title + border top/bottom
	if innerH < 2 {
		innerH = 2
	}
	v.diffVP.Width = innerW
	v.diffVP.Height = innerH

	var content string
	if v.diffContent == "" {
		content = lipgloss.NewStyle().
			Foreground(t.TextSubtle).
			Width(innerW).Height(innerH).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Select a file to preview diff")
	} else {
		content = v.diffVP.View()
	}

	// Scroll indicator.
	scrollInfo := ""
	if v.diffVP.TotalLineCount() > v.diffVP.Height {
		pct := v.diffVP.ScrollPercent() * 100
		scrollInfo = lipgloss.NewStyle().Foreground(t.TextSubtle).
			Render(fmt.Sprintf("%.0f%%", pct))
	}

	titleBar := " " + title +
		strings.Repeat(" ", max(0, innerW-lipgloss.Width(title)-lipgloss.Width(scrollInfo)-1)) +
		scrollInfo

	pane := lipgloss.JoinVertical(lipgloss.Left, titleBar, content)

	return lipgloss.NewStyle().
		Width(width).Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderLeft(true).BorderRight(false).BorderTop(false).BorderBottom(false).
		BorderForeground(borderColor).
		Render(pane)
}

// ── Command bar (always visible) ────────────────────────────────────────────

func (v *StatusView) renderCommandBar() string {
	t := v.styles.Theme

	keyStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(t.TextMuted)
	sepStyle := lipgloss.NewStyle().Foreground(t.Border)

	sep := sepStyle.Render(" │ ")

	// Context-aware: show different hints based on focus and selection.
	var entries []string

	if v.focus == focusDiffPane {
		entries = []string{
			keyStyle.Render("j/k") + descStyle.Render(" scroll"),
			keyStyle.Render("tab") + descStyle.Render(" files"),
			keyStyle.Render("esc") + descStyle.Render(" back"),
		}
	} else {
		entries = []string{
			keyStyle.Render("j/k") + descStyle.Render(" nav"),
			keyStyle.Render("s") + descStyle.Render(" stage"),
			keyStyle.Render("u") + descStyle.Render(" unstage"),
			keyStyle.Render("S/U") + descStyle.Render(" all"),
			keyStyle.Render("x") + descStyle.Render(" discard"),
			keyStyle.Render("c") + descStyle.Render(" commit"),
		}
		if v.diffPaneWidth() > 0 {
			entries = append(entries, keyStyle.Render("tab")+descStyle.Render(" diff"))
		}
	}

	cmdLine := strings.Join(entries, sep)

	// Right-align position indicator.
	posInfo := ""
	if len(v.items) > 0 {
		posInfo = descStyle.Render(fmt.Sprintf("%d/%d", v.cursor+1, len(v.items)))
	}

	leftW := lipgloss.Width(cmdLine)
	rightW := lipgloss.Width(posInfo)
	gap := v.width - leftW - rightW - 3
	if gap < 0 {
		gap = 1
	}

	fullLine := " " + cmdLine + strings.Repeat(" ", gap) + posInfo + " "

	// Separator + command bar.
	divider := lipgloss.NewStyle().Foreground(t.Border).Width(v.width).
		Render(strings.Repeat("─", v.width))
	bar := lipgloss.NewStyle().
		Foreground(t.TextMuted).
		Background(t.Surface).
		Width(v.width).
		Render(fullLine)

	return lipgloss.JoinVertical(lipgloss.Left, divider, bar)
}

// ── Commit view ─────────────────────────────────────────────────────────────

func (v *StatusView) viewCommit() string {
	t := v.styles.Theme
	title := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(" Commit")
	info := v.styles.Muted.Render(fmt.Sprintf(" %d file(s) staged", len(v.status.Staged)))
	ta := " " + v.commitTA.View()

	// Command bar for commit mode.
	keyStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(t.TextMuted)
	hint := " " + keyStyle.Render("ctrl+s") + descStyle.Render(" commit") + "  " +
		keyStyle.Render("esc") + descStyle.Render(" cancel")

	divider := lipgloss.NewStyle().Foreground(t.Border).Width(v.width).
		Render(strings.Repeat("─", v.width))
	cmdBar := lipgloss.NewStyle().Background(t.Surface).Width(v.width).Render(hint)

	top := lipgloss.JoinVertical(lipgloss.Left, title, "", info, "", ta)
	topH := v.height - 2 // reserve for command bar
	topPadded := lipgloss.NewStyle().Width(v.width).Height(topH).Render(top)

	return lipgloss.JoinVertical(lipgloss.Left, topPadded, divider, cmdBar)
}

// ── Data helpers ─────────────────────────────────────────────────────────────

func (v *StatusView) rebuildItems() {
	total := v.status.TotalCount()
	if cap(v.items) < total {
		v.items = make([]statusItem, 0, total)
	} else {
		v.items = v.items[:0]
	}
	for _, f := range v.status.Staged {
		v.items = append(v.items, statusItem{file: f, section: sectionStaged})
	}
	for _, f := range v.status.Unstaged {
		v.items = append(v.items, statusItem{file: f, section: sectionUnstaged})
	}
	for _, f := range v.status.Untracked {
		v.items = append(v.items, statusItem{file: f, section: sectionUntracked})
	}
	for _, f := range v.status.Conflicts {
		v.items = append(v.items, statusItem{file: f, section: sectionConflicts})
	}
}

func (v *StatusView) currentItem() (statusItem, bool) {
	if v.cursor < 0 || v.cursor >= len(v.items) {
		return statusItem{}, false
	}
	return v.items[v.cursor], true
}

func (v *StatusView) pageSize() int {
	ps := v.height - 6
	if ps < 1 {
		ps = 1
	}
	return ps
}

// itemAtY maps a terminal Y coordinate to an item index.
// Uses the cached scroll state from the last render pass so the hit-test
// exactly matches what's visually on screen.
func (v *StatusView) itemAtY(y int) int {
	// Convert absolute Y to a position within the list area.
	listRow := y - v.lastListYOffset
	if listRow < 0 || listRow >= v.lastListH {
		return -1
	}

	// The target line in the virtual list.
	targetLine := v.lastScrollStart + listRow

	// Walk through sections to find which item is at this line.
	lineIdx := 0
	itemIdx := 0
	sectionItems := [][]git.FileStatus{
		v.status.Staged,
		v.status.Unstaged,
		v.status.Untracked,
		v.status.Conflicts,
	}

	for _, items := range sectionItems {
		if len(items) == 0 {
			continue
		}
		// Section header occupies this line.
		if lineIdx == targetLine {
			return -1 // clicked on a header, not a file
		}
		lineIdx++

		for range items {
			if lineIdx == targetLine {
				return itemIdx
			}
			lineIdx++
			itemIdx++
		}
	}
	return -1
}

// ── Status indicators ───────────────────────────────────────────────────────

// statusIndicator returns a clean single-character status marker.
func statusIndicator(code git.StatusCode) string {
	switch code {
	case git.StatusAdded:
		return "A"
	case git.StatusModified:
		return "M"
	case git.StatusDeleted:
		return "D"
	case git.StatusRenamed:
		return "R"
	case git.StatusCopied:
		return "C"
	case git.StatusUnmerged:
		return "U"
	case git.StatusUntracked:
		return "?"
	default:
		return " "
	}
}

func (v *StatusView) statusColor(code git.StatusCode, fallback lipgloss.Color) lipgloss.Color {
	t := v.styles.Theme
	switch code {
	case git.StatusAdded:
		return t.Added
	case git.StatusModified:
		return t.Modified
	case git.StatusDeleted:
		return t.Deleted
	case git.StatusRenamed:
		return t.Renamed
	case git.StatusUnmerged:
		return t.Conflict
	case git.StatusUntracked:
		return t.Untracked
	default:
		return fallback
	}
}

func (v *StatusView) ShortHelp() []components.HelpEntry {
	return []components.HelpEntry{
		{Key: "j/k", Desc: "Navigate files"},
		{Key: "s / S", Desc: "Stage file / all"},
		{Key: "u / U", Desc: "Unstage file / all"},
		{Key: "x", Desc: "Discard changes"},
		{Key: "c", Desc: "Commit"},
		{Key: "tab", Desc: "Switch file/diff pane"},
		{Key: "d/enter", Desc: "Focus diff"},
	}
}
