package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// textareaEditor is a reusable textarea-based sub-model for editing
// text that gets parsed on save (e.g. env vars, headers, extra body, args).
type textareaEditor struct {
	textarea  textarea.Model
	statusMsg string
	title     string
	onSave    func(text string) error // parse + save; error shown as statusMsg
	onCancel  func()                  // restore mode on cancel
}

func newTextareaEditor(title, placeholder, initialValue string,
	onSave func(string) error, onCancel func()) *textareaEditor {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.SetValue(initialValue)
	ta.Focus()
	ta.SetWidth(editorWidth)
	ta.SetHeight(editorHeight)

	return &textareaEditor{
		textarea: ta,
		title:    title,
		onSave:   onSave,
		onCancel: onCancel,
	}
}

// update handles key messages. Returns quit=true for ctrl+c,
// done=true when the editor should be dismissed (save or cancel).
func (e *textareaEditor) update(msg tea.Msg) (quit bool, done bool, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return true, false, nil

		case "ctrl+s":
			if err := e.onSave(e.textarea.Value()); err != nil {
				e.statusMsg = fmt.Sprintf("Parse error: %v", err)
				return false, false, nil
			}
			return false, true, nil

		case "esc":
			e.onCancel()
			return false, true, nil
		}
	}

	e.textarea, cmd = e.textarea.Update(msg)
	return false, false, cmd
}

func (e *textareaEditor) view() string {
	s := titleStyle.Render(e.title) + "\n\n"
	s += e.textarea.View() + "\n"

	if e.statusMsg != "" {
		s += "\n" + dimStyle.Render(e.statusMsg)
	}

	hints := []string{"Ctrl+S Save", "Esc Cancel"}
	s += "\n" + renderStatusBar(hints)
	return s
}

// updateTextareaEdit dispatches to the shared textareaEditor.
func (m Model) updateTextareaEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.textEditor == nil {
		m.returnToProfileForm()
		return m, nil
	}

	quit, done, cmd := m.textEditor.update(msg)
	if quit {
		m.quitting = true
		return m, tea.Quit
	}
	if done {
		m.textEditor = nil
	}
	return m, cmd
}

func (m Model) viewTextareaEdit() string {
	if m.textEditor == nil {
		return ""
	}
	return m.textEditor.view()
}
