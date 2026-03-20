package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cliswitch/gocc/internal/config"
)

// ── key helpers ──────────────────────────────────────────────────────────────

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func keyType(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// updateModel is a convenience wrapper that casts the returned tea.Model back
// to the concrete Model type so tests can inspect fields directly.
func updateModel(m Model, msg tea.Msg) Model {
	got, _ := m.Update(msg)
	return got.(Model)
}

// ── test fixtures ─────────────────────────────────────────────────────────────

func newTestProfiles() []DisplayProfile {
	return []DisplayProfile{
		{ID: "prof1", Name: "Profile 1", Protocol: config.ProtocolOpenAI},
		{ID: "prof2", Name: "Profile 2", Protocol: config.ProtocolAnthropic},
	}
}

func newTestModel() Model {
	return NewModel(newTestProfiles(), 0, Callbacks{})
}

// ── Profile list: navigation ──────────────────────────────────────────────────

func TestProfileListDownIncrementsCursor(t *testing.T) {
	m := newTestModel()
	m = updateModel(m, keyType(tea.KeyDown))

	if m.cursor != 1 {
		t.Errorf("expected cursor=1 after pressing down, got %d", m.cursor)
	}
}

func TestProfileListUpDecrementsCursor(t *testing.T) {
	m := newTestModel()
	m.cursor = 1
	m = updateModel(m, keyType(tea.KeyUp))

	if m.cursor != 0 {
		t.Errorf("expected cursor=0 after pressing up, got %d", m.cursor)
	}
}

func TestProfileListDownWithJKey(t *testing.T) {
	m := newTestModel()
	m = updateModel(m, keyRune('j'))

	if m.cursor != 1 {
		t.Errorf("expected cursor=1 after pressing 'j', got %d", m.cursor)
	}
}

func TestProfileListUpWithKKey(t *testing.T) {
	m := newTestModel()
	m.cursor = 1
	m = updateModel(m, keyRune('k'))

	if m.cursor != 0 {
		t.Errorf("expected cursor=0 after pressing 'k', got %d", m.cursor)
	}
}

func TestProfileListCursorDoesNotGoAboveZero(t *testing.T) {
	m := newTestModel()
	m.cursor = 0
	m = updateModel(m, keyType(tea.KeyUp))

	if m.cursor != 0 {
		t.Errorf("cursor should remain at 0 when pressing up at top, got %d", m.cursor)
	}
}

func TestProfileListCursorDoesNotGoBelowMax(t *testing.T) {
	m := newTestModel()
	// totalItems = 2 profiles + 1 Config row = 3, max index is 2
	m.cursor = m.totalItems() - 1
	m = updateModel(m, keyType(tea.KeyDown))

	if m.cursor != m.totalItems()-1 {
		t.Errorf("cursor should remain at max when pressing down at bottom, got %d", m.cursor)
	}
}

// ── Profile list: enter to select profile ─────────────────────────────────────

func TestProfileListEnterSetsChosenAndQuits(t *testing.T) {
	m := newTestModel()
	m.cursor = 0

	m = updateModel(m, keyType(tea.KeyEnter))

	if m.chosen != "prof1" {
		t.Errorf("expected chosen=\"prof1\", got %q", m.chosen)
	}
	if !m.quitting {
		t.Error("expected quitting=true after selecting a profile")
	}
}

func TestProfileListEnterSecondProfile(t *testing.T) {
	m := newTestModel()
	m.cursor = 1

	m = updateModel(m, keyType(tea.KeyEnter))

	if m.chosen != "prof2" {
		t.Errorf("expected chosen=\"prof2\", got %q", m.chosen)
	}
	if !m.quitting {
		t.Error("expected quitting=true after selecting second profile")
	}
}

// ── Profile list: quit keys ───────────────────────────────────────────────────

func TestProfileListQKeyQuits(t *testing.T) {
	m := newTestModel()
	m = updateModel(m, keyRune('q'))

	if !m.quitting {
		t.Error("expected quitting=true after pressing 'q'")
	}
}

func TestProfileListCtrlCQuits(t *testing.T) {
	m := newTestModel()
	m = updateModel(m, keyType(tea.KeyCtrlC))

	if !m.quitting {
		t.Error("expected quitting=true after pressing ctrl+c")
	}
}

// ── Delete confirm: y confirms deletion ───────────────────────────────────────

func TestDeleteConfirmYCallsDeleteCallback(t *testing.T) {
	deleted := ""
	profiles := newTestProfiles()
	cb := Callbacks{
		DeleteProfile: func(id string) error {
			deleted = id
			return nil
		},
		Reload: func() ([]DisplayProfile, error) {
			return profiles, nil
		},
	}
	m := NewModel(profiles, 0, cb)
	m.mode = ModeDeleteConfirm
	target := profiles[0]
	m.deleteTarget = &target

	m = updateModel(m, keyRune('y'))

	if deleted != "prof1" {
		t.Errorf("DeleteProfile callback should have been called with \"prof1\", got %q", deleted)
	}
	if m.mode != ModeProfileList {
		t.Errorf("expected mode=ModeProfileList after confirm, got %v", m.mode)
	}
	if m.deleteTarget != nil {
		t.Error("deleteTarget should be nil after deletion")
	}
}

func TestDeleteConfirmYSetsDeletedStatusMsg(t *testing.T) {
	profiles := newTestProfiles()
	cb := Callbacks{
		DeleteProfile: func(id string) error { return nil },
		Reload: func() ([]DisplayProfile, error) {
			return profiles, nil
		},
	}
	m := NewModel(profiles, 0, cb)
	m.mode = ModeDeleteConfirm
	target := profiles[0]
	m.deleteTarget = &target

	m = updateModel(m, keyRune('y'))

	if m.statusMsg != "Profile deleted" {
		t.Errorf("expected statusMsg=\"Profile deleted\", got %q", m.statusMsg)
	}
}

func TestDeleteConfirmYWithDeleteError(t *testing.T) {
	profiles := newTestProfiles()
	cb := Callbacks{
		DeleteProfile: func(id string) error {
			return errors.New("disk full")
		},
	}
	m := NewModel(profiles, 0, cb)
	m.mode = ModeDeleteConfirm
	target := profiles[0]
	m.deleteTarget = &target

	m = updateModel(m, keyRune('y'))

	if m.mode != ModeProfileList {
		t.Errorf("expected mode=ModeProfileList even on delete error, got %v", m.mode)
	}
	if m.statusMsg == "" {
		t.Error("expected non-empty statusMsg on delete error")
	}
}

func TestDeleteConfirmNReturnsToListWithoutDeleting(t *testing.T) {
	deleted := false
	profiles := newTestProfiles()
	cb := Callbacks{
		DeleteProfile: func(id string) error {
			deleted = true
			return nil
		},
	}
	m := NewModel(profiles, 0, cb)
	m.mode = ModeDeleteConfirm
	target := profiles[0]
	m.deleteTarget = &target

	m = updateModel(m, keyRune('n'))

	if deleted {
		t.Error("DeleteProfile should not be called when pressing 'n'")
	}
	if m.mode != ModeProfileList {
		t.Errorf("expected mode=ModeProfileList after 'n', got %v", m.mode)
	}
	if m.deleteTarget != nil {
		t.Error("deleteTarget should be nil after cancelling")
	}
}

func TestDeleteConfirmEscReturnsToListWithoutDeleting(t *testing.T) {
	deleted := false
	profiles := newTestProfiles()
	cb := Callbacks{
		DeleteProfile: func(id string) error {
			deleted = true
			return nil
		},
	}
	m := NewModel(profiles, 0, cb)
	m.mode = ModeDeleteConfirm
	target := profiles[0]
	m.deleteTarget = &target

	m = updateModel(m, keyType(tea.KeyEsc))

	if deleted {
		t.Error("DeleteProfile should not be called when pressing Esc")
	}
	if m.mode != ModeProfileList {
		t.Errorf("expected mode=ModeProfileList after Esc, got %v", m.mode)
	}
}

func TestDeleteConfirmCtrlCQuits(t *testing.T) {
	profiles := newTestProfiles()
	m := NewModel(profiles, 0, Callbacks{})
	m.mode = ModeDeleteConfirm
	target := profiles[0]
	m.deleteTarget = &target

	m = updateModel(m, keyType(tea.KeyCtrlC))

	if !m.quitting {
		t.Error("expected quitting=true after ctrl+c in delete confirm")
	}
}

// ── Textarea editor ───────────────────────────────────────────────────────────

// buildTextareaModel sets up a Model in ModeTextareaEdit with a textareaEditor
// that delegates onSave/onCancel to the provided functions.
func buildTextareaModel(onSave func(string) error, onCancel func()) Model {
	profiles := newTestProfiles()
	m := NewModel(profiles, 0, Callbacks{})
	m.mode = ModeTextareaEdit
	m.textEditor = newTextareaEditor(
		"Test Editor",
		"placeholder",
		"initial value",
		onSave,
		onCancel,
	)
	return m
}

func TestTextareaEditorCtrlSWithValidDataCallsOnSave(t *testing.T) {
	saveCalled := false
	m := buildTextareaModel(
		func(text string) error {
			saveCalled = true
			return nil
		},
		func() {},
	)

	m = updateModel(m, keyType(tea.KeyCtrlS))

	if !saveCalled {
		t.Error("onSave should be called when ctrl+s is pressed")
	}
	// After a successful save, textEditor should be cleared.
	if m.textEditor != nil {
		t.Error("textEditor should be nil after successful ctrl+s save")
	}
}

func TestTextareaEditorCtrlSWithSaveErrorSetsStatusMsg(t *testing.T) {
	m := buildTextareaModel(
		func(text string) error {
			return errors.New("parse error: bad input")
		},
		func() {},
	)

	m = updateModel(m, keyType(tea.KeyCtrlS))

	if m.textEditor == nil {
		t.Error("textEditor should remain when save fails")
	}
	if m.textEditor.statusMsg == "" {
		t.Error("textEditor.statusMsg should be set when save returns an error")
	}
}

func TestTextareaEditorEscCallsOnCancel(t *testing.T) {
	cancelCalled := false
	m := buildTextareaModel(
		func(text string) error { return nil },
		func() { cancelCalled = true },
	)

	m = updateModel(m, keyType(tea.KeyEsc))

	if !cancelCalled {
		t.Error("onCancel should be called when Esc is pressed")
	}
	// After cancel, textEditor is cleared (done=true).
	if m.textEditor != nil {
		t.Error("textEditor should be nil after Esc (cancel)")
	}
}

func TestTextareaEditorCtrlCQuitsModel(t *testing.T) {
	m := buildTextareaModel(
		func(text string) error { return nil },
		func() {},
	)

	m = updateModel(m, keyType(tea.KeyCtrlC))

	if !m.quitting {
		t.Error("expected quitting=true after ctrl+c in textarea editor")
	}
}

// ── NewModel / Chosen ─────────────────────────────────────────────────────────

func TestNewModelInitialState(t *testing.T) {
	profiles := newTestProfiles()
	m := NewModel(profiles, 1, Callbacks{})

	if m.cursor != 1 {
		t.Errorf("expected initial cursor=1, got %d", m.cursor)
	}
	if m.mode != ModeProfileList {
		t.Errorf("expected initial mode=ModeProfileList, got %v", m.mode)
	}
	if m.Chosen() != "" {
		t.Errorf("expected Chosen()=\"\" initially, got %q", m.Chosen())
	}
}

func TestChosenReturnsSelectedProfileID(t *testing.T) {
	m := newTestModel()
	m.cursor = 1
	m = updateModel(m, keyType(tea.KeyEnter))

	if m.Chosen() != "prof2" {
		t.Errorf("Chosen() = %q, want \"prof2\"", m.Chosen())
	}
}

// ── Profile list: multi-step navigation ───────────────────────────────────────

func TestProfileListMultipleDownPresses(t *testing.T) {
	m := newTestModel()
	// totalItems = 3 (2 profiles + Config row)
	m = updateModel(m, keyType(tea.KeyDown))
	m = updateModel(m, keyType(tea.KeyDown))

	// Cursor should now be at index 2 (the Config row).
	if m.cursor != 2 {
		t.Errorf("expected cursor=2 after two down presses, got %d", m.cursor)
	}
}

func TestProfileListDownThenUp(t *testing.T) {
	m := newTestModel()
	m = updateModel(m, keyType(tea.KeyDown))
	m = updateModel(m, keyType(tea.KeyUp))

	if m.cursor != 0 {
		t.Errorf("expected cursor=0 after down then up, got %d", m.cursor)
	}
}
