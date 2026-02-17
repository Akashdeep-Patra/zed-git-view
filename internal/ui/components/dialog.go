package components

import (
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DialogKind specifies the type of dialog.
type DialogKind int

const (
	DialogConfirm DialogKind = iota
	DialogInput
)

// DialogResult is sent when the dialog is dismissed.
type DialogResult struct {
	Confirmed bool
	Value     string
	Tag       string // arbitrary tag to identify which dialog this was
}

// Dialog is a modal confirmation or input dialog.
type Dialog struct {
	Kind    DialogKind
	Title   string
	Message string
	Tag     string
	input   textinput.Model
	focused int // 0 = yes/input, 1 = no
	styles  ui.Styles
	visible bool
}

// NewConfirmDialog creates a Yes/No confirmation dialog.
func NewConfirmDialog(styles ui.Styles, title, message, tag string) Dialog {
	return Dialog{
		Kind:    DialogConfirm,
		Title:   title,
		Message: message,
		Tag:     tag,
		styles:  styles,
		visible: true,
	}
}

// NewInputDialog creates a text input dialog.
func NewInputDialog(styles ui.Styles, title, placeholder, tag string) Dialog {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50
	return Dialog{
		Kind:    DialogInput,
		Title:   title,
		Tag:     tag,
		input:   ti,
		styles:  styles,
		visible: true,
	}
}

// Visible returns whether the dialog is showing.
func (d Dialog) Visible() bool { return d.visible }

// Update handles key events for the dialog.
func (d Dialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			d.visible = false
			return d, func() tea.Msg { return DialogResult{Tag: d.Tag} }

		case "enter":
			d.visible = false
			if d.Kind == DialogInput {
				return d, func() tea.Msg {
					return DialogResult{Confirmed: true, Value: d.input.Value(), Tag: d.Tag}
				}
			}
			return d, func() tea.Msg {
				return DialogResult{Confirmed: d.focused == 0, Tag: d.Tag}
			}

		case "tab", "left", "right", "h", "l":
			if d.Kind == DialogConfirm {
				d.focused = 1 - d.focused
			}
		}
	}

	if d.Kind == DialogInput {
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return d, cmd
	}
	return d, nil
}

// View renders the dialog.
func (d Dialog) View() string {
	if !d.visible {
		return ""
	}
	t := d.styles.Theme

	title := lipgloss.NewStyle().Foreground(t.Text).Bold(true).Render(d.Title)
	var content string

	if d.Kind == DialogConfirm {
		message := lipgloss.NewStyle().Foreground(t.TextMuted).Render(d.Message)
		yes := "  Yes  "
		no := "  No   "
		activeBtn := lipgloss.NewStyle().Foreground(t.TextInverse).Background(t.Primary).Bold(true)
		inactiveBtn := lipgloss.NewStyle().Foreground(t.TextMuted).Background(t.Surface)
		if d.focused == 0 {
			yes = activeBtn.Render(yes)
			no = inactiveBtn.Render(no)
		} else {
			yes = inactiveBtn.Render(yes)
			no = activeBtn.Render(no)
		}
		buttons := lipgloss.JoinHorizontal(lipgloss.Top, yes, "  ", no)
		content = title + "\n\n" + message + "\n\n" + buttons
	} else {
		content = title + "\n\n" + d.input.View()
	}

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Width(56).
		Render(content)
}
