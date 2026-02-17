// Package ui provides shared TUI styling, layout helpers, and theme definitions.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PlaceCentre centres content both horizontally and vertically within the given dimensions.
func PlaceCentre(width, height int, content string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// Truncate truncates s to maxLen runes, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// PadRight pads s with spaces to the given width.
func PadRight(s string, width int) string {
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// RenderKeyValue renders a "key: value" pair with styles.
func RenderKeyValue(styles Styles, key, value string) string {
	return styles.KeyBind.Render(key) + " " + styles.KeyDesc.Render(value)
}

// JoinHorizontal joins items horizontally with a separator.
func JoinHorizontal(sep string, items ...string) string {
	var filtered []string
	for _, item := range items {
		if item != "" {
			filtered = append(filtered, item)
		}
	}
	return strings.Join(filtered, sep)
}
