package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type extraBodyEditModel struct {
	textarea  textarea.Model
	statusMsg string
}

func newExtraBodyEditModel(extraBody map[string]any) *extraBodyEditModel {
	ta := textarea.New()
	ta.Placeholder = "field_name: json_value (one per line)"
	ta.SetValue(formatExtraBody(extraBody))
	ta.Focus()
	ta.SetWidth(60)
	ta.SetHeight(10)

	return &extraBodyEditModel{textarea: ta}
}

func parseExtraBody(text string) (map[string]any, error) {
	result := make(map[string]any)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx < 0 {
			return nil, fmt.Errorf("invalid line (missing ': '): %q", line)
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" {
			return nil, fmt.Errorf("empty key in line: %q", line)
		}
		valStr := line[idx+2:]

		var val any
		if err := json.Unmarshal([]byte(valStr), &val); err != nil {
			// Values that look like JSON structures must parse correctly
			if len(valStr) > 0 && (valStr[0] == '{' || valStr[0] == '[' || valStr[0] == '"') {
				return nil, fmt.Errorf("invalid JSON value for key %q: %s", key, valStr)
			}
			// Bare string: treat as string value
			val = valStr
		}
		result[key] = val
	}
	return result, nil
}

func formatExtraBody(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		b, _ := json.Marshal(m[k])
		sb.WriteString(k + ": " + string(b))
	}
	return sb.String()
}

func extraBodySummary(eb map[string]any) string {
	if len(eb) == 0 {
		return dimStyle.Render("(empty)")
	}
	return fmt.Sprintf("%d field(s)", len(eb))
}

func (m Model) updateExtraBodyEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	eb := m.extraBodyEdit
	if eb == nil {
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
			extraBody, err := parseExtraBody(eb.textarea.Value())
			if err != nil {
				eb.statusMsg = fmt.Sprintf("Parse error: %v", err)
				return m, nil
			}
			if m.profileForm != nil {
				m.profileForm.profile.ExtraBody = extraBody
			}
			m.extraBodyEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.extraBodyEdit = nil
			m.returnToProfileForm()
			return m, nil
		}
	}

	var cmd tea.Cmd
	eb.textarea, cmd = eb.textarea.Update(msg)
	return m, cmd
}

func (m Model) viewExtraBodyEdit() string {
	eb := m.extraBodyEdit
	if eb == nil {
		return ""
	}

	s := titleStyle.Render("Edit Extra Body") + "\n\n"
	s += eb.textarea.View() + "\n"

	if eb.statusMsg != "" {
		s += "\n" + dimStyle.Render(eb.statusMsg)
	}

	hints := []string{"Ctrl+S Save", "Esc Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}
