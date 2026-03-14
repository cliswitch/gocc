package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type argsEditModel struct {
	textarea  textarea.Model
	statusMsg string
}

func newArgsEditModel(args []string) *argsEditModel {
	ta := textarea.New()
	ta.Placeholder = "One argument per line"
	ta.SetValue(strings.Join(args, "\n"))
	ta.Focus()
	ta.SetWidth(60)
	ta.SetHeight(10)

	return &argsEditModel{textarea: ta}
}

func parseArgs(text string) []string {
	var args []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			args = append(args, line)
		}
	}
	return args
}

func (m Model) updateArgsEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	ae := m.argsEdit
	if ae == nil {
		m.returnToProfileForm()
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			args := parseArgs(ae.textarea.Value())
			if m.profileForm != nil {
				m.profileForm.profile.ClaudeArgs = args
			}
			m.argsEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.argsEdit = nil
			m.returnToProfileForm()
			return m, nil
		}
	}

	var cmd tea.Cmd
	ae.textarea, cmd = ae.textarea.Update(msg)
	return m, cmd
}

func (m Model) viewArgsEdit() string {
	ae := m.argsEdit
	if ae == nil {
		return ""
	}

	s := titleStyle.Render("Edit Claude Args") + "\n\n"
	s += ae.textarea.View() + "\n"

	if ae.statusMsg != "" {
		s += "\n" + dimStyle.Render(ae.statusMsg)
	}

	hints := []string{"Ctrl+S Save", "Esc Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}

func envSummary(env map[string]string) string {
	if len(env) == 0 {
		return dimStyle.Render("(empty)")
	}
	return fmt.Sprintf("%d var(s)", len(env))
}

func argsSummary(args []string) string {
	if len(args) == 0 {
		return dimStyle.Render("(none)")
	}
	return fmt.Sprintf("%d arg(s)", len(args))
}
