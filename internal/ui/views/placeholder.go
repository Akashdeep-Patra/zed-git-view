package views

import (
	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PlaceholderView is a temporary stub for tabs not yet implemented.
type PlaceholderView struct {
	name   string
	styles ui.Styles
	width  int
	height int
}

// NewPlaceholderView creates a placeholder with the given tab name.
func NewPlaceholderView(name string, styles ui.Styles) *PlaceholderView {
	return &PlaceholderView{name: name, styles: styles}
}

func (v *PlaceholderView) Init() tea.Cmd                           { return nil }
func (v *PlaceholderView) Update(_ tea.Msg) (common.View, tea.Cmd) { return v, nil }
func (v *PlaceholderView) SetSize(w, h int)                        { v.width = w; v.height = h }
func (v *PlaceholderView) ShortHelp() []components.HelpEntry       { return nil }
func (v *PlaceholderView) InputCapture() bool                      { return false }

func (v *PlaceholderView) View() string {
	msg := lipgloss.NewStyle().Foreground(v.styles.Theme.TextMuted).
		Render(v.name + " — coming soon")
	return ui.PlaceCentre(v.width, v.height, msg)
}
