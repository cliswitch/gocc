package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var statusStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	BorderTop(true).
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("238")).
	PaddingLeft(1)

func renderStatusBar(hints []string) string {
	return statusStyle.Render(joinHints(hints))
}

func joinHints(hints []string) string {
	return strings.Join(hints, "  ")
}

// renderAlignedField renders a form field with right-aligned label.
// The ">" prefix is shown only when focusIdx == currentFocus.
func renderAlignedField(focusIdx, currentFocus, maxLabelWidth int, name, content string) string {
	padded := fmt.Sprintf("%*s", maxLabelWidth, name)
	style := normalStyle
	prefix := "  "
	if focusIdx == currentFocus {
		style = selectedStyle
		prefix = "> "
	}
	return prefix + style.Render(padded+":") + " " + content + "\n"
}
