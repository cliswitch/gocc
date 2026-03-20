package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

type proxyEditModel struct {
	proxy  config.Proxy
	inputs []textinput.Model // http_proxy, https_proxy, no_proxy
	focus  int
}

func newProxyEditModel(p config.Proxy) *proxyEditModel {
	pe := &proxyEditModel{proxy: p}
	pe.inputs = make([]textinput.Model, 3)
	pe.inputs[0] = newTextInput(p.HTTPProxy)
	pe.inputs[1] = newTextInput(p.HTTPSProxy)
	pe.inputs[2] = newTextInput(p.NoProxy)
	pe.inputs[0].Focus()
	return pe
}

func (pe *proxyEditModel) updateFocus() {
	for i := range pe.inputs {
		pe.inputs[i].Blur()
	}
	pe.inputs[pe.focus].Focus()
}

func (pe *proxyEditModel) applyToProxy() config.Proxy {
	return config.Proxy{
		HTTPProxy:  pe.inputs[0].Value(),
		HTTPSProxy: pe.inputs[1].Value(),
		NoProxy:    pe.inputs[2].Value(),
	}
}

func (m Model) updateProxyEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	pe := m.proxyEdit
	if pe == nil {
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
			if m.profileForm != nil {
				m.profileForm.profile.Proxy = pe.applyToProxy()
			}
			m.proxyEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.proxyEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "tab", "down":
			pe.focus = (pe.focus + 1) % len(pe.inputs)
			pe.updateFocus()
			return m, nil

		case "shift+tab", "up":
			pe.focus = (pe.focus - 1 + len(pe.inputs)) % len(pe.inputs)
			pe.updateFocus()
			return m, nil
		}
	}

	var cmd tea.Cmd
	pe.inputs[pe.focus], cmd = pe.inputs[pe.focus].Update(msg)
	return m, cmd
}

func (m Model) viewProxyEdit() string {
	pe := m.proxyEdit
	if pe == nil {
		return ""
	}

	s := titleStyle.Render("Edit Proxies") + "\n\n"

	labels := []string{"http_proxy", "https_proxy", "no_proxy"}
	for i, label := range labels {
		focused := pe.focus == i
		var content string
		if focused {
			content = pe.inputs[i].View()
		} else {
			v := pe.inputs[i].Value()
			if v == "" {
				content = dimStyle.Render("(empty)")
			} else {
				content = v
			}
		}
		ls := normalStyle
		indicator := "  "
		if focused {
			ls = selectedStyle
			indicator = "> "
		}
		s += "  " + ls.Render(fmt.Sprintf("%-*s", labelWidth, label)) + indicator + content + "\n"
	}

	hints := []string{"tab/↓ next", "shift+tab/↑ prev", "ctrl+s save", "esc cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}
