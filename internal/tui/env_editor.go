package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type envEditModel struct {
	textarea  textarea.Model
	statusMsg string
}

func newEnvEditModel(env map[string]string) *envEditModel {
	ta := textarea.New()
	ta.Placeholder = "KEY=VALUE (one per line)"
	ta.SetValue(formatEnvVars(env))
	ta.Focus()
	ta.SetWidth(60)
	ta.SetHeight(10)

	return &envEditModel{textarea: ta}
}

func parseEnvVars(text string) (map[string]string, error) {
	env := make(map[string]string)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("invalid env line (missing '='): %q", line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("empty env key in line: %q", line)
		}
		env[key] = value
	}
	return env, nil
}

func formatEnvVars(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + "=" + env[k])
	}
	return sb.String()
}

func (m Model) updateEnvEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	ee := m.envEdit
	if ee == nil {
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
			env, err := parseEnvVars(ee.textarea.Value())
			if err != nil {
				ee.statusMsg = fmt.Sprintf("Parse error: %v", err)
				return m, nil
			}
			if m.profileForm != nil {
				m.profileForm.profile.CustomEnv = env
			}
			m.envEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.envEdit = nil
			m.returnToProfileForm()
			return m, nil
		}
	}

	var cmd tea.Cmd
	ee.textarea, cmd = ee.textarea.Update(msg)
	return m, cmd
}

func (m Model) updateGlobalEnvEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	ee := m.envEdit
	if ee == nil {
		m.mode = ModeGlobalConfig
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			env, err := parseEnvVars(ee.textarea.Value())
			if err != nil {
				ee.statusMsg = fmt.Sprintf("Parse error: %v", err)
				return m, nil
			}
			if m.globalConfig != nil {
				m.globalConfig.global.CustomEnv = env
				if m.callbacks.SaveGlobal != nil {
					if err := m.callbacks.SaveGlobal(m.globalConfig.global); err != nil {
						ee.statusMsg = fmt.Sprintf("Save failed: %v", err)
						return m, nil
					}
				}
			}
			m.envEdit = nil
			m.mode = ModeGlobalConfig
			return m, nil

		case "esc":
			m.envEdit = nil
			m.mode = ModeGlobalConfig
			return m, nil
		}
	}

	var cmd tea.Cmd
	ee.textarea, cmd = ee.textarea.Update(msg)
	return m, cmd
}

func (m Model) viewEnvEdit() string {
	ee := m.envEdit
	if ee == nil {
		return ""
	}

	s := titleStyle.Render("Edit Custom Env Vars") + "\n\n"
	s += ee.textarea.View() + "\n"

	if ee.statusMsg != "" {
		s += "\n" + dimStyle.Render(ee.statusMsg)
	}

	hints := []string{"Ctrl+S Save", "Esc Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}
