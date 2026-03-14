package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type headersEditModel struct {
	textarea  textarea.Model
	statusMsg string
}

func newHeadersEditModel(headers map[string]string) *headersEditModel {
	ta := textarea.New()
	ta.Placeholder = "Key: Value (one per line)"
	ta.SetValue(formatHeaders(headers))
	ta.Focus()
	ta.SetWidth(60)
	ta.SetHeight(10)

	return &headersEditModel{textarea: ta}
}

func parseHeaders(text string) (map[string]string, error) {
	headers := make(map[string]string)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			return nil, fmt.Errorf("invalid header line (missing ':'): %q", line)
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("empty header key in line: %q", line)
		}
		headers[key] = value
	}
	return headers, nil
}

func formatHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + ": " + headers[k])
	}
	return sb.String()
}

func (m Model) updateHeadersEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	he := m.headersEdit
	if he == nil {
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
			headers, err := parseHeaders(he.textarea.Value())
			if err != nil {
				he.statusMsg = fmt.Sprintf("Parse error: %v", err)
				return m, nil
			}
			if m.profileForm != nil {
				m.profileForm.profile.CustomHeaders = headers
			}
			m.headersEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.headersEdit = nil
			m.returnToProfileForm()
			return m, nil
		}
	}

	var cmd tea.Cmd
	he.textarea, cmd = he.textarea.Update(msg)
	return m, cmd
}

func (m Model) viewHeadersEdit() string {
	he := m.headersEdit
	if he == nil {
		return ""
	}

	s := titleStyle.Render("Edit Custom Headers") + "\n\n"
	s += he.textarea.View() + "\n"

	if he.statusMsg != "" {
		s += "\n" + dimStyle.Render(he.statusMsg)
	}

	hints := []string{"Ctrl+S Save", "Esc Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}
