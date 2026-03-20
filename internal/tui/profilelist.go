package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cliswitch/gocc/internal/config"
)

var (
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	normalStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).MarginBottom(1)
	sectionTitleStyle = titleStyle.MarginBottom(0)
)

// totalItems returns the number of selectable items (profiles + config row).
func (m Model) totalItems() int {
	return len(m.profiles) + 1 // +1 for Config item
}

func (m Model) updateProfileList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		total := m.totalItems()
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < total-1 {
				m.cursor++
			}

		case "enter":
			if m.cursor < len(m.profiles) {
				// Launch selected profile
				m.chosen = m.profiles[m.cursor].ID
				m.quitting = true
				return m, tea.Quit
			}
			// Config item
			return m.enterGlobalConfig()

		case "a":
			return m.enterProfileAdd()

		case "e":
			if m.cursor < len(m.profiles) {
				return m.enterProfileEdit(m.profiles[m.cursor].ID)
			}

		case "d":
			if m.cursor < len(m.profiles) {
				p := m.profiles[m.cursor]
				if p.ID == config.NativeProfileID {
					m.statusMsg = "Cannot delete native profile"
					return m, nil
				}
				m.deleteTarget = &p
				m.mode = ModeDeleteConfirm
				return m, nil
			}

		case "y":
			if m.cursor < len(m.profiles) {
				p := m.profiles[m.cursor]
				if p.ID == config.NativeProfileID {
					m.statusMsg = "Cannot copy native profile"
					return m, nil
				}
				if m.callbacks.CopyProfile != nil {
					if err := m.callbacks.CopyProfile(p.ID); err != nil {
						m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
					} else {
						m.reloadProfiles()
						m.statusMsg = "Profile copied"
					}
				}
			}

		case "r":
			m.reloadProfiles()
			m.statusMsg = "Reloaded"

		case "K":
			if m.cursor < len(m.profiles) && m.callbacks.MoveProfile != nil {
				p := m.profiles[m.cursor]
				if err := m.callbacks.MoveProfile(p.ID, -1); err == nil {
					m.reloadProfiles()
					if m.cursor > 0 {
						m.cursor--
					}
				}
			}

		case "J":
			if m.cursor < len(m.profiles) && m.callbacks.MoveProfile != nil {
				p := m.profiles[m.cursor]
				if err := m.callbacks.MoveProfile(p.ID, 1); err == nil {
					m.reloadProfiles()
					if m.cursor < len(m.profiles)-1 {
						m.cursor++
					}
				}
			}
		}
	}
	return m, nil
}

func (m *Model) reloadProfiles() {
	if m.callbacks.Reload != nil {
		if profiles, err := m.callbacks.Reload(); err == nil {
			m.profiles = profiles
			if m.cursor >= m.totalItems() {
				m.cursor = m.totalItems() - 1
			}
		}
	}
}

func (m Model) enterProfileAdd() (tea.Model, tea.Cmd) {
	newProfile := config.Profile{
		ID:       config.GenerateProfileID(),
		Protocol: config.ProtocolOpenAI,
	}
	m.profileForm = newProfileFormModel(newProfile, true, m.callbacks)
	m.mode = ModeProfileAdd
	return m, nil
}

func (m Model) enterProfileEdit(id string) (tea.Model, tea.Cmd) {
	if m.callbacks.GetProfile == nil {
		return m, nil
	}
	p, ok := m.callbacks.GetProfile(id)
	if !ok {
		m.statusMsg = "Profile not found"
		return m, nil
	}
	m.profileForm = newProfileFormModel(p, false, m.callbacks)
	m.mode = ModeProfileEdit
	return m, nil
}

func (m Model) enterGlobalConfig() (tea.Model, tea.Cmd) {
	var g config.GlobalConfig
	if m.callbacks.GetGlobal != nil {
		g = m.callbacks.GetGlobal()
	}
	m.globalConfig = newGlobalConfigModel(g)
	m.mode = ModeGlobalConfig
	return m, nil
}

func (m Model) viewProfileList() string {
	s := titleStyle.Render("GoCC - Profile Selector") + "\n\n"

	for i, p := range m.profiles {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		proto := dimStyle.Render(fmt.Sprintf("(%s)", p.Protocol))
		s += cursor + style.Render(p.Name) + " " + proto + "\n"
	}

	// Separator + Config item
	s += "\n" + dimStyle.Render("───────────────────") + "\n"
	configIdx := len(m.profiles)
	cursor := "  "
	style := normalStyle
	if m.cursor == configIdx {
		cursor = "> "
		style = selectedStyle
	}
	s += cursor + style.Render("Config") + "\n"

	// Status message
	if m.statusMsg != "" {
		s += "\n" + dimStyle.Render(m.statusMsg)
	}

	hints := []string{
		"↑↓ Navigate", "Enter Launch", "a Add", "e Edit",
		"d Delete", "y Copy", "K/J Reorder", "r Reload", "q Quit",
	}
	s += "\n" + renderStatusBar(hints)
	return s
}
