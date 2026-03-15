package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

type fallbackCandidate struct {
	ID       string
	Name     string
	Selected bool
}

type fallbackEditModel struct {
	profileID  string // the profile being edited (exclude from candidates)
	candidates []fallbackCandidate
	nameMap    map[string]string // id -> name for O(1) lookup
	order      []string          // ordered list of selected profile IDs
	cursor     int
	section    int // 0=candidates list, 1=order list
}

func newFallbackEditModel(profileID string, currentChain []string, allProfiles []DisplayProfile) *fallbackEditModel {
	selected := make(map[string]bool)
	for _, id := range currentChain {
		selected[id] = true
	}

	var candidates []fallbackCandidate
	for _, p := range allProfiles {
		if p.ID == config.NativeProfileID || p.ID == profileID {
			continue
		}
		candidates = append(candidates, fallbackCandidate{
			ID:       p.ID,
			Name:     p.Name,
			Selected: selected[p.ID],
		})
	}

	// Build name lookup map and use selected map for O(1) chain filtering.
	nameMap := make(map[string]string, len(candidates))
	for _, c := range candidates {
		nameMap[c.ID] = c.Name
	}

	order := make([]string, 0, len(currentChain))
	for _, id := range currentChain {
		if selected[id] {
			order = append(order, id)
		}
	}

	return &fallbackEditModel{
		profileID:  profileID,
		candidates: candidates,
		nameMap:    nameMap,
		order:      order,
	}
}

func (fe *fallbackEditModel) totalItems() int {
	if fe.section == 0 {
		return len(fe.candidates)
	}
	return len(fe.order)
}

func (fe *fallbackEditModel) toggleCandidate() {
	if fe.section != 0 || fe.cursor >= len(fe.candidates) {
		return
	}
	c := &fe.candidates[fe.cursor]
	c.Selected = !c.Selected
	if c.Selected {
		fe.order = append(fe.order, c.ID)
	} else {
		// Remove from order
		newOrder := make([]string, 0, len(fe.order))
		for _, id := range fe.order {
			if id != c.ID {
				newOrder = append(newOrder, id)
			}
		}
		fe.order = newOrder
	}
}

func (fe *fallbackEditModel) moveOrder(delta int) {
	if fe.section != 1 || len(fe.order) < 2 {
		return
	}
	newIdx := fe.cursor + delta
	if newIdx < 0 || newIdx >= len(fe.order) {
		return
	}
	fe.order[fe.cursor], fe.order[newIdx] = fe.order[newIdx], fe.order[fe.cursor]
	fe.cursor = newIdx
}

func (fe *fallbackEditModel) candidateName(id string) string {
	if name, ok := fe.nameMap[id]; ok {
		return name
	}
	return id
}

func (m Model) updateFallbackEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	fe := m.fallbackEdit
	if fe == nil {
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
				m.profileForm.profile.FallbackChain = make([]string, len(fe.order))
				copy(m.profileForm.profile.FallbackChain, fe.order)
			}
			m.fallbackEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "esc":
			m.fallbackEdit = nil
			m.returnToProfileForm()
			return m, nil

		case "up", "k":
			if fe.cursor > 0 {
				fe.cursor--
			}

		case "down", "j":
			total := fe.totalItems()
			if fe.cursor < total-1 {
				fe.cursor++
			}

		case "tab":
			// Switch between candidates and order sections
			if fe.section == 0 && len(fe.order) > 0 {
				fe.section = 1
				fe.cursor = 0
			} else {
				fe.section = 0
				fe.cursor = 0
			}

		case " ":
			fe.toggleCandidate()

		case "K":
			if fe.section == 1 {
				fe.moveOrder(-1)
			}

		case "J":
			if fe.section == 1 {
				fe.moveOrder(1)
			}
		}
	}

	return m, nil
}

func (m Model) viewFallbackEdit() string {
	fe := m.fallbackEdit
	if fe == nil {
		return ""
	}

	s := titleStyle.Render("Edit Fallback Chain") + "\n\n"

	// Candidates section
	active := ""
	if fe.section == 0 {
		active = " (active)"
	}
	s += sectionTitleStyle.Render("Candidates"+active) + "\n"
	for i, c := range fe.candidates {
		prefix := "  "
		style := normalStyle
		if fe.section == 0 && i == fe.cursor {
			prefix = "> "
			style = selectedStyle
		}
		check := "[ ]"
		if c.Selected {
			check = "[x]"
		}
		s += prefix + check + " " + style.Render(c.Name) + "\n"
	}
	if len(fe.candidates) == 0 {
		s += "  " + dimStyle.Render("(no candidates available)") + "\n"
	}

	s += "\n"

	// Order section
	active = ""
	if fe.section == 1 {
		active = " (active)"
	}
	s += sectionTitleStyle.Render("Fallback Order"+active) + "\n"
	for i, id := range fe.order {
		prefix := "  "
		style := normalStyle
		if fe.section == 1 && i == fe.cursor {
			prefix = "> "
			style = selectedStyle
		}
		name := fe.candidateName(id)
		s += prefix + style.Render(fmt.Sprintf("%d. %s", i+1, name)) + "\n"
	}
	if len(fe.order) == 0 {
		s += "  " + dimStyle.Render("(none selected)") + "\n"
	}

	hints := []string{"↑↓ Navigate", "Space Toggle", "Tab Switch Section", "K/J Reorder", "Ctrl+S Save", "Esc Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}
