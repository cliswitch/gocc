package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

type globalConfigModel struct {
	global   config.GlobalConfig
	editing  bool // false=summary view, true=editing Claude Args textarea
	focus    int  // 0=Claude Args, 1=Custom Env
	textarea textarea.Model
}

func newGlobalConfigModel(g config.GlobalConfig) *globalConfigModel {
	ta := textarea.New()
	ta.Placeholder = "One argument per line"
	ta.SetValue(strings.Join(g.ClaudeArgs, "\n"))
	ta.SetWidth(60)
	ta.SetHeight(10)

	return &globalConfigModel{
		global:   g,
		textarea: ta,
	}
}

func (m Model) updateGlobalConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	gc := m.globalConfig
	if gc == nil {
		m.mode = ModeProfileList
		return m, nil
	}

	if gc.editing {
		return m.updateGlobalArgsEdit(msg)
	}

	// Summary view
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "tab", "down":
			gc.focus = (gc.focus + 1) % 2
			return m, nil

		case "shift+tab", "up":
			gc.focus = (gc.focus - 1 + 2) % 2
			return m, nil

		case "enter":
			switch gc.focus {
			case 0:
				gc.editing = true
				gc.textarea.Focus()
				return m, nil
			case 1:
				m.envEdit = newEnvEditModel(gc.global.CustomEnv)
				m.mode = ModeGlobalEnvEdit
				return m, nil
			}

		case "esc":
			m.mode = ModeProfileList
			m.globalConfig = nil
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateGlobalArgsEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	gc := m.globalConfig

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			gc.global.ClaudeArgs = parseArgs(gc.textarea.Value())
			if m.callbacks.SaveGlobal != nil {
				if err := m.callbacks.SaveGlobal(gc.global); err != nil {
					m.statusMsg = fmt.Sprintf("Save failed: %v", err)
					m.mode = ModeProfileList
					m.globalConfig = nil
					return m, nil
				}
			}
			gc.editing = false
			gc.textarea.Blur()
			return m, nil

		case "esc":
			gc.textarea.SetValue(strings.Join(gc.global.ClaudeArgs, "\n"))
			gc.editing = false
			gc.textarea.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	gc.textarea, cmd = gc.textarea.Update(msg)
	return m, cmd
}

func (m Model) viewGlobalConfig() string {
	gc := m.globalConfig
	if gc == nil {
		return ""
	}

	s := titleStyle.Render("Global Config") + "\n\n"

	if gc.editing {
		s += normalStyle.Render("Claude Args (one per line):") + "\n"
		s += gc.textarea.View() + "\n"

		hints := []string{"Ctrl+S Save", "Esc Cancel"}
		s += "\n" + renderStatusBar(hints)
	} else {
		argsSum := dimStyle.Render("(none)")
		if len(gc.global.ClaudeArgs) > 0 {
			argsSum = fmt.Sprintf("%d arg(s)", len(gc.global.ClaudeArgs))
		}
		s += renderAlignedField(0, gc.focus, 11, "Claude Args", argsSum)
		s += renderAlignedField(1, gc.focus, 11, "Custom Env", envSummary(gc.global.CustomEnv))

		hints := []string{"tab/↓ next", "shift+tab/↑ prev", "Enter Edit", "Esc Back"}
		s += "\n" + renderStatusBar(hints)
	}

	return s
}

// Delete confirm

func (m Model) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "y":
			if m.deleteTarget != nil && m.callbacks.DeleteProfile != nil {
				if err := m.callbacks.DeleteProfile(m.deleteTarget.ID); err != nil {
					m.statusMsg = fmt.Sprintf("Delete failed: %v", err)
				} else {
					m.reloadProfiles()
					m.statusMsg = "Profile deleted"
				}
			}
			m.deleteTarget = nil
			m.mode = ModeProfileList
			return m, nil

		case "n", "esc":
			m.deleteTarget = nil
			m.mode = ModeProfileList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) viewDeleteConfirm() string {
	name := "(unknown)"
	if m.deleteTarget != nil {
		name = m.deleteTarget.Name
	}

	s := titleStyle.Render("Delete Profile") + "\n\n"
	s += normalStyle.Render(fmt.Sprintf("Delete %q? (y/n)", name)) + "\n"

	hints := []string{"y Confirm", "n Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}
