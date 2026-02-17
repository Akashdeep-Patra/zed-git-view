package components

import (
	"strings"
	"unicode/utf8"

	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// TabInfo describes a single tab for rendering.
type TabInfo struct {
	Name     string
	Icon     string
	Shortcut string
	Active   bool
	Group    string
}

// tabDisplayMode controls how tab labels are rendered.
type tabDisplayMode int

const (
	tabDisplayFull  tabDisplayMode = iota // "● Status"
	tabDisplayShort                       // "● Stat"
	tabDisplayIcon                        // "●"
)

// safeIconWidth returns a conservative width estimate for a Unicode icon.
// Many TUI icons (⑂ ⇄ ⊟ ↻ ⚡ ⌥ ◎ etc.) are rendered as double-width in
// some terminals even though runewidth reports them as single-width.
// We assume 2 cells for any non-ASCII rune to be safe.
func safeIconWidth(icon string) int {
	w := 0
	for _, r := range icon {
		if r < 128 {
			w++
		} else {
			w += 2 // conservative: assume double-width
		}
	}
	return w
}

// tabLabelWidth returns the visual width of a tab in the given display mode.
// Layout: " LABEL " (1 space + label + 1 space).
func tabLabelWidth(tab TabInfo, mode tabDisplayMode) int {
	iw := safeIconWidth(tab.Icon)
	switch mode {
	case tabDisplayFull:
		// " icon Name "
		return 1 + iw + 1 + utf8.RuneCountInString(tab.Name) + 1
	case tabDisplayShort:
		// " icon Nam "  (max 3 chars of name)
		nameLen := utf8.RuneCountInString(tab.Name)
		if nameLen > 3 {
			nameLen = 3
		}
		return 1 + iw + 1 + nameLen + 1
	default:
		// " icon "
		return 1 + iw + 1
	}
}

// shortName returns a truncated name (max 3 runes).
func shortName(name string) string {
	runes := []rune(name)
	if len(runes) <= 3 {
		return name
	}
	return string(runes[:3])
}

// layoutRowsWithMode greedily packs tabs into rows for the given mode.
// Returns the rows (as slices of tab indices) and the total width each row needs.
func layoutRowsWithMode(tabs []TabInfo, width int, mode tabDisplayMode) (rows [][]int, fits bool) {
	var currentRow []int
	currentW := 1 // left padding
	prevGroup := ""

	for i, tab := range tabs {
		tw := tabLabelWidth(tab, mode)
		sepW := 0
		if prevGroup != "" && tab.Group != prevGroup && len(currentRow) > 0 {
			sepW = 3 // " │ "
		}

		needed := sepW + tw
		if currentW+needed > width && len(currentRow) > 0 {
			rows = append(rows, currentRow)
			currentRow = nil
			currentW = 1
			sepW = 0
		}

		currentRow = append(currentRow, i)
		currentW += sepW + tw
		prevGroup = tab.Group
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// Consider it "fits" if we use 3 or fewer rows.
	return rows, len(rows) <= 3
}

// bestMode picks the best display mode that fits within maxRows (3).
func bestMode(tabs []TabInfo, width int) ([][]int, tabDisplayMode) {
	for _, mode := range []tabDisplayMode{tabDisplayFull, tabDisplayShort, tabDisplayIcon} {
		rows, fits := layoutRowsWithMode(tabs, width, mode)
		if fits {
			return rows, mode
		}
	}
	// Even icon-only doesn't fit in 3 rows — use icon anyway (best effort).
	rows, _ := layoutRowsWithMode(tabs, width, tabDisplayIcon)
	return rows, tabDisplayIcon
}

// TabBarRows returns the number of screen rows the tab bar will occupy
// (tab rows + 1 underline row).
func TabBarRows(tabs []TabInfo, width int) int {
	if width <= 0 || len(tabs) == 0 {
		return 2
	}
	rows, _ := bestMode(tabs, width)
	return len(rows) + 1
}

// RenderTabs renders a responsive tab bar that adapts to any width:
//   - Wide: icon + full name, single row
//   - Medium: icon + full name, wrapped to 2-3 rows
//   - Narrow: icon + abbreviated name (3 chars)
//   - Very narrow: icon only
//
// Active tab has a bold accent underline beneath its row.
func RenderTabs(styles ui.Styles, tabs []TabInfo, width int) string {
	t := styles.Theme
	rows, mode := bestMode(tabs, width)

	// ── Styles ───────────────────────────────────────────────────────

	activeStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(t.TextMuted)
	groupSepStyle := lipgloss.NewStyle().Foreground(t.Border)

	// ── Render each row ─────────────────────────────────────────────

	var renderedRows []string
	activeStart, activeEnd, activeRowIdx := -1, -1, -1
	lastRow := len(rows) - 1

	for rowIdx, indices := range rows {
		var row strings.Builder
		row.Grow(width + 32)
		row.WriteByte(' ') // left padding
		col := 1

		prevGroup := ""
		if len(indices) > 0 {
			prevGroup = tabs[indices[0]].Group
		}

		for j, idx := range indices {
			tab := tabs[idx]

			if j > 0 && tab.Group != prevGroup {
				sep := groupSepStyle.Render(" │ ")
				row.WriteString(sep)
				col += lipgloss.Width(sep)
			}
			prevGroup = tab.Group

			var label string
			switch mode {
			case tabDisplayFull:
				label = tab.Icon + " " + tab.Name
			case tabDisplayShort:
				label = tab.Icon + " " + shortName(tab.Name)
			default:
				label = tab.Icon
			}

			var styled string
			if tab.Active {
				styled = " " + activeStyle.Render(label) + " "
			} else {
				styled = " " + inactiveStyle.Render(label) + " "
			}

			w := lipgloss.Width(styled)
			if tab.Active {
				activeStart = col
				activeEnd = col + w
				activeRowIdx = rowIdx
			}

			row.WriteString(styled)
			col += w
		}

		rendered := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Background(t.Bg).
			Render(row.String())

		renderedRows = append(renderedRows, rendered)
	}

	// ── Single bottom underline ─────────────────────────────────────
	// Exactly ONE underline after all tab rows. The active tab is already
	// visually distinct (bold + primary). Adding inline underlines between
	// rows would change the rendered height and break contentHeight().
	thinChar := "─"
	boldChar := "━"
	borderStyle := lipgloss.NewStyle().Foreground(t.Border)
	accentStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)

	// Only accent the underline if the active tab is on the last row;
	// otherwise show a plain thin line (the active tab's bold+primary
	// text is sufficient visual indication on other rows).
	ulStart, ulEnd := -1, -1
	if activeRowIdx == lastRow {
		ulStart, ulEnd = activeStart, activeEnd
	}

	bottomUl := buildUnderline(width, ulStart, ulEnd, borderStyle, accentStyle, thinChar, boldChar)

	// Overlay a right-side hint.
	hintStyle := lipgloss.NewStyle().Foreground(t.TextSubtle).Faint(true)
	hint := hintStyle.Render("←/→  ?help")
	hintW := lipgloss.Width(hint)
	if hintW+4 < width {
		hintStart := width - hintW - 1
		var finalUl strings.Builder
		finalUl.WriteString(buildUnderline(hintStart, ulStart, ulEnd, borderStyle, accentStyle, thinChar, boldChar))
		finalUl.WriteByte(' ')
		finalUl.WriteString(hint)
		bottomUl = finalUl.String()
	}

	renderedRows = append(renderedRows, lipgloss.NewStyle().Width(width).Render(bottomUl))

	return lipgloss.JoinVertical(lipgloss.Left, renderedRows...)
}

// buildUnderline builds a width-wide underline string with a bold accent
// segment between activeStart..activeEnd and thin segments elsewhere.
func buildUnderline(width, activeStart, activeEnd int, borderSt, accentSt lipgloss.Style, thin, bold string) string {
	if activeStart < 0 || activeEnd < 0 {
		return borderSt.Render(strings.Repeat(thin, width))
	}
	// Clamp to width.
	if activeEnd > width {
		activeEnd = width
	}
	if activeStart > width {
		activeStart = width
	}

	var b strings.Builder
	b.Grow(width * 4)
	if activeStart > 0 {
		b.WriteString(borderSt.Render(strings.Repeat(thin, activeStart)))
	}
	seg := activeEnd - activeStart
	if seg > 0 {
		b.WriteString(accentSt.Render(strings.Repeat(bold, seg)))
	}
	if rem := width - activeEnd; rem > 0 {
		b.WriteString(borderSt.Render(strings.Repeat(thin, rem)))
	}
	return b.String()
}
