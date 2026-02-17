package app

import (
	"time"

	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/config"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the top-level Bubbletea model that orchestrates tabs and views.
type Model struct {
	git       git.Service
	cfg       *config.Config
	styles    ui.Styles
	keys      KeyMap
	width     int
	height    int
	activeTab common.TabID
	views     map[common.TabID]common.View
	showHelp  bool
	statusMsg string
	statusErr bool
	statusExp time.Time
	dialog    *components.Dialog

	// Cached status bar data — refreshed via tea.Cmd, never computed in View().
	barData components.StatusBarData

	// viewStale tracks which views need a re-init on next switch.
	viewStale map[common.TabID]bool

	// tabLayout caches the pixel positions of each tab for mouse hit-testing.
	// Rebuilt every render cycle (cheap — just len(AllTabs) iterations).
	tabLayout []tabHitZone
}

// tabHitZone maps a screen X range to a tab ID for mouse clicking.
type tabHitZone struct {
	ID    common.TabID
	Start int // inclusive X
	End   int // exclusive X
}

// statusBarMsg carries refreshed status bar data from a background command.
type statusBarMsg struct {
	data components.StatusBarData
}

// New creates a new application model.
func New(gitSvc git.Service, cfg *config.Config, views map[common.TabID]common.View) Model {
	return Model{
		git:       gitSvc,
		cfg:       cfg,
		styles:    ui.DefaultStyles(),
		keys:      DefaultKeyMap(),
		activeTab: common.TabStatus,
		views:     views,
		barData:   components.StatusBarData{RepoRoot: gitSvc.RepoRoot()},
		viewStale: make(map[common.TabID]bool),
	}
}

// Init initialises the active view and triggers the first status bar refresh.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.refreshStatusBar()}
	if v, ok := m.views[m.activeTab]; ok {
		if cmd := v.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// refreshStatusBar runs git queries in the background and returns a statusBarMsg.
func (m Model) refreshStatusBar() tea.Cmd {
	svc := m.git
	return func() tea.Msg {
		data := components.StatusBarData{RepoRoot: svc.RepoRoot()}
		if head, err := svc.Head(); err == nil {
			data.Branch = head
		}
		data.Ahead, data.Behind, _ = svc.AheadBehind()
		data.Clean, _ = svc.IsClean()
		data.Merging = svc.IsMerging()
		data.Rebasing = svc.IsRebasing()
		return statusBarMsg{data: data}
	}
}

// Update processes messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Dialog has exclusive input when visible.
	if m.dialog != nil && m.dialog.Visible() {
		d, cmd := m.dialog.Update(msg)
		m.dialog = &d
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentH := m.contentHeight()
		for _, v := range m.views {
			v.SetSize(m.width, contentH)
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.Refresh):
			return m, m.triggerRefresh()
		case key.Matches(msg, m.keys.NextTab):
			m.cycleTab(1)
			return m, m.initActiveView()
		case key.Matches(msg, m.keys.PrevTab):
			m.cycleTab(-1)
			return m, m.initActiveView()

		// ── Mnemonic tab shortcuts (Alt+key) ─────────────────────
		case key.Matches(msg, m.keys.TabStatus):
			return m, m.switchTo(common.TabStatus)
		case key.Matches(msg, m.keys.TabDiff):
			return m, m.switchTo(common.TabDiff)
		case key.Matches(msg, m.keys.TabLog):
			return m, m.switchTo(common.TabLog)
		case key.Matches(msg, m.keys.TabBranches):
			return m, m.switchTo(common.TabBranches)
		case key.Matches(msg, m.keys.TabRemotes):
			return m, m.switchTo(common.TabRemotes)
		case key.Matches(msg, m.keys.TabStash):
			return m, m.switchTo(common.TabStash)
		case key.Matches(msg, m.keys.TabRebase):
			return m, m.switchTo(common.TabRebase)
		case key.Matches(msg, m.keys.TabConflicts):
			return m, m.switchTo(common.TabConflicts)
		case key.Matches(msg, m.keys.TabWorktrees):
			return m, m.switchTo(common.TabWorktrees)
		case key.Matches(msg, m.keys.TabBisect):
			return m, m.switchTo(common.TabBisect)

		case key.Matches(msg, m.keys.Back):
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
		}
		// Keys not handled globally are forwarded to the active view below.

	case statusBarMsg:
		m.barData = msg.data
		return m, nil

	case common.RefreshMsg:
		// Only refresh the ACTIVE view + status bar. Inactive views will
		// reload when the user switches to them (lazy init). This prevents
		// spawning N*git-commands for N views on every filesystem event.
		if v, ok := m.views[m.activeTab]; ok {
			updated, cmd := v.Update(msg)
			m.views[m.activeTab] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Mark all OTHER views as stale so they reload on next switch.
		for id := range m.views {
			if id != m.activeTab {
				m.viewStale[id] = true
			}
		}
		cmds = append(cmds, m.refreshStatusBar())
		return m, tea.Batch(cmds...)

	case common.ErrMsg:
		m.statusMsg = msg.Err.Error()
		m.statusErr = true
		m.statusExp = time.Now().Add(5 * time.Second)
		return m, nil

	case common.InfoMsg:
		m.statusMsg = msg.Text
		m.statusErr = false
		m.statusExp = time.Now().Add(3 * time.Second)
		return m, nil

	case common.SwitchTabMsg:
		return m, m.switchTo(msg.Tab)

	case components.DialogResult:
		m.dialog = nil
	}

	// Forward unhandled messages to the active view.
	if v, ok := m.views[m.activeTab]; ok {
		updated, cmd := v.Update(msg)
		m.views[m.activeTab] = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the entire UI. This is a pure function — no I/O.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.showHelp {
		sections := components.GlobalHelpEntries()
		tabName := ""
		for _, t := range common.AllTabs {
			if t.ID == m.activeTab {
				tabName = t.Name
				break
			}
		}
		if v, ok := m.views[m.activeTab]; ok && tabName != "" {
			sections[tabName] = v.ShortHelp()
		}
		return components.RenderHelp(m.styles, "Keyboard Shortcuts", sections, m.width, m.height)
	}

	tabInfos := m.buildTabInfos()
	tabBar := components.RenderTabs(m.styles, tabInfos, m.width)

	// Rebuild tab hit zones for mouse click detection.
	m.rebuildTabLayout(tabInfos)

	content := ""
	if v, ok := m.views[m.activeTab]; ok {
		content = v.View()
	}

	contentH := m.contentHeight()
	content = lipgloss.NewStyle().Width(m.width).Height(contentH).Render(content)

	barData := m.barData
	if m.statusMsg != "" && time.Now().Before(m.statusExp) {
		barData.Message = m.statusMsg
		barData.IsError = m.statusErr
	}
	statusBar := components.RenderStatusBar(m.styles, barData, m.width)

	screen := lipgloss.JoinVertical(lipgloss.Left, tabBar, content, statusBar)

	if m.dialog != nil && m.dialog.Visible() {
		overlay := m.dialog.View()
		screen = ui.PlaceCentre(m.width, m.height, overlay)
	}

	return screen
}

func (m Model) contentHeight() int {
	return m.height - 4
}

func (m *Model) cycleTab(delta int) {
	n := len(common.AllTabs)
	cur := m.tabIndex()
	next := (cur + delta + n) % n
	m.activeTab = common.AllTabs[next].ID
}

// tabIndex returns the index of the active tab in AllTabs.
func (m Model) tabIndex() int {
	for i, t := range common.AllTabs {
		if t.ID == m.activeTab {
			return i
		}
	}
	return 0
}

// switchTo changes the active tab and lazily initialises the target view.
func (m *Model) switchTo(tab common.TabID) tea.Cmd {
	m.activeTab = tab
	delete(m.viewStale, tab)
	return m.initActiveView()
}

// initActiveView calls Init on the current tab to load its data.
func (m Model) initActiveView() tea.Cmd {
	if v, ok := m.views[m.activeTab]; ok {
		return v.Init()
	}
	return nil
}

// triggerRefresh refreshes the active view and the status bar.
func (m Model) triggerRefresh() tea.Cmd {
	var cmds []tea.Cmd
	if v, ok := m.views[m.activeTab]; ok {
		updated, cmd := v.Update(common.RefreshMsg{})
		m.views[m.activeTab] = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	cmds = append(cmds, m.refreshStatusBar())
	return tea.Batch(cmds...)
}

// handleMouse processes mouse events: tab clicks, scroll wheel, and click-through.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Tab bar occupies the first 2 rows (bar + separator).
	const tabBarHeight = 2

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		// Scroll wheel in tab bar area cycles tabs.
		if msg.Y < tabBarHeight {
			if msg.Button == tea.MouseButtonWheelUp {
				m.cycleTab(-1)
			} else {
				m.cycleTab(1)
			}
			return m, m.initActiveView()
		}
		// Forward scroll wheel to active view.
		if v, ok := m.views[m.activeTab]; ok {
			updated, cmd := v.Update(msg)
			m.views[m.activeTab] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			break
		}

		// Click in tab bar — switch tabs.
		if msg.Y < tabBarHeight {
			if tab, ok := m.tabAtX(msg.X); ok && tab != m.activeTab {
				return m, m.switchTo(tab)
			}
			return m, nil
		}

		// Click in content area — forward to active view.
		if v, ok := m.views[m.activeTab]; ok {
			updated, cmd := v.Update(msg)
			m.views[m.activeTab] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// rebuildTabLayout recomputes tab hit zones from the rendered tab infos.
// This runs during View() so it's always in sync with what's displayed.
func (m *Model) rebuildTabLayout(tabs []components.TabInfo) {
	m.tabLayout = m.tabLayout[:0]

	// Match the rendering logic in components/tabs.go:
	// - 1 char left padding from the bar
	// - Each tab: " icon Name " (1 pad + icon + space + name + 1 pad) + " shortcut"
	// - Group separators: " │ " = 3 chars
	offset := 1 // bar left padding
	prevGroup := ""

	for _, tab := range tabs {
		if prevGroup != "" && tab.Group != prevGroup {
			offset += 3 // group separator " │ "
		}
		prevGroup = tab.Group

		// Tab pill: padding(0,1) → 1 left + content + 1 right.
		// Content: "icon space name" = len(icon) + 1 + len(name)
		iconW := lipgloss.Width(tab.Icon)
		pillContentW := iconW + 1 + len(tab.Name)
		pillW := pillContentW + 2 // +2 for Padding(0,1) on each side

		// Shortcut: " s" = 2 chars
		shortcutW := 1 + len(tab.Shortcut)

		totalW := pillW + shortcutW

		// Find which TabID this corresponds to.
		var tabID common.TabID
		for _, t := range common.AllTabs {
			if t.Name == tab.Name {
				tabID = t.ID
				break
			}
		}

		m.tabLayout = append(m.tabLayout, tabHitZone{
			ID:    tabID,
			Start: offset,
			End:   offset + totalW,
		})

		offset += totalW
	}
}

// tabAtX determines which tab was clicked given an X coordinate.
func (m Model) tabAtX(x int) (common.TabID, bool) {
	for _, zone := range m.tabLayout {
		if x >= zone.Start && x < zone.End {
			return zone.ID, true
		}
	}
	return 0, false
}

func (m Model) buildTabInfos() []components.TabInfo {
	infos := make([]components.TabInfo, len(common.AllTabs))
	for i, t := range common.AllTabs {
		infos[i] = components.TabInfo{
			Name:     t.Name,
			Icon:     t.Icon,
			Shortcut: t.Shortcut,
			Active:   t.ID == m.activeTab,
			Group:    t.Group,
		}
	}
	return infos
}
