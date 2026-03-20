package tui

import (
	"testing"

	"github.com/cliswitch/gocc/internal/config"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func makeProfiles(specs ...struct{ id, name string }) []DisplayProfile {
	ps := make([]DisplayProfile, len(specs))
	for i, s := range specs {
		ps[i] = DisplayProfile{ID: s.id, Name: s.name}
	}
	return ps
}

// ── newFallbackEditModel ─────────────────────────────────────────────────────

func TestNewFallbackEditModelExcludesNativeProfile(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{config.NativeProfileID, "Anthropic Official"},
		struct{ id, name string }{"abc123", "My Profile"},
	)
	fe := newFallbackEditModel("someprofile", nil, profiles)

	for _, c := range fe.candidates {
		if c.ID == config.NativeProfileID {
			t.Errorf("native profile should be excluded from candidates, but found it: %+v", c)
		}
	}
}

func TestNewFallbackEditModelExcludesSelf(t *testing.T) {
	selfID := "selfid1"
	profiles := makeProfiles(
		struct{ id, name string }{selfID, "Self"},
		struct{ id, name string }{"other1", "Other"},
	)
	fe := newFallbackEditModel(selfID, nil, profiles)

	for _, c := range fe.candidates {
		if c.ID == selfID {
			t.Errorf("self profile should be excluded from candidates, found: %+v", c)
		}
	}
	if len(fe.candidates) != 1 || fe.candidates[0].ID != "other1" {
		t.Errorf("expected only \"other1\" in candidates, got %+v", fe.candidates)
	}
}

func TestNewFallbackEditModelPreSelectsCurrentChain(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"prof1", "Profile 1"},
		struct{ id, name string }{"prof2", "Profile 2"},
		struct{ id, name string }{"prof3", "Profile 3"},
	)
	chain := []string{"prof1", "prof3"}
	fe := newFallbackEditModel("editme", chain, profiles)

	selectedIDs := map[string]bool{}
	for _, c := range fe.candidates {
		if c.Selected {
			selectedIDs[c.ID] = true
		}
	}

	if !selectedIDs["prof1"] {
		t.Error("expected prof1 to be pre-selected")
	}
	if !selectedIDs["prof3"] {
		t.Error("expected prof3 to be pre-selected")
	}
	if selectedIDs["prof2"] {
		t.Error("prof2 should not be pre-selected")
	}

	// Order should mirror the current chain.
	if len(fe.order) != 2 {
		t.Fatalf("expected 2 items in order, got %d: %v", len(fe.order), fe.order)
	}
	if fe.order[0] != "prof1" || fe.order[1] != "prof3" {
		t.Errorf("order mismatch: got %v, want [prof1 prof3]", fe.order)
	}
}

func TestNewFallbackEditModelEmptyChain(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"prof1", "Profile 1"},
	)
	fe := newFallbackEditModel("editme", nil, profiles)

	if len(fe.order) != 0 {
		t.Errorf("expected empty order for nil chain, got %v", fe.order)
	}
	if fe.candidates[0].Selected {
		t.Error("candidate should not be selected when chain is empty")
	}
}

// ── toggleCandidate ──────────────────────────────────────────────────────────

func TestToggleCandidateSelectAddsToOrder(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"prof1", "Profile 1"},
		struct{ id, name string }{"prof2", "Profile 2"},
	)
	fe := newFallbackEditModel("editme", nil, profiles)
	fe.section = 0
	fe.cursor = 0

	fe.toggleCandidate()

	if !fe.candidates[0].Selected {
		t.Error("candidate should be selected after toggle")
	}
	if len(fe.order) != 1 || fe.order[0] != "prof1" {
		t.Errorf("expected order=[prof1], got %v", fe.order)
	}
}

func TestToggleCandidateDeselectRemovesFromOrder(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"prof1", "Profile 1"},
		struct{ id, name string }{"prof2", "Profile 2"},
	)
	fe := newFallbackEditModel("editme", []string{"prof1", "prof2"}, profiles)
	fe.section = 0
	fe.cursor = 0 // prof1 is selected

	fe.toggleCandidate()

	if fe.candidates[0].Selected {
		t.Error("candidate should be deselected after second toggle")
	}
	for _, id := range fe.order {
		if id == "prof1" {
			t.Error("prof1 should have been removed from order")
		}
	}
}

func TestToggleCandidateNoOpWhenSectionIsNotZero(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"prof1", "Profile 1"},
	)
	fe := newFallbackEditModel("editme", nil, profiles)
	fe.section = 1 // order section — toggle should be no-op
	fe.cursor = 0

	fe.toggleCandidate()

	if fe.candidates[0].Selected {
		t.Error("toggle should be a no-op when section != 0")
	}
	if len(fe.order) != 0 {
		t.Errorf("order should remain empty, got %v", fe.order)
	}
}

// ── moveOrder ────────────────────────────────────────────────────────────────

func TestMoveOrderUp(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"a", "A"},
		struct{ id, name string }{"b", "B"},
		struct{ id, name string }{"c", "C"},
	)
	fe := newFallbackEditModel("editme", []string{"a", "b", "c"}, profiles)
	fe.section = 1
	fe.cursor = 1 // pointing at "b"

	fe.moveOrder(-1) // move b up

	if fe.order[0] != "b" || fe.order[1] != "a" {
		t.Errorf("expected order [b a c], got %v", fe.order)
	}
	if fe.cursor != 0 {
		t.Errorf("cursor should follow moved item to index 0, got %d", fe.cursor)
	}
}

func TestMoveOrderDown(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"a", "A"},
		struct{ id, name string }{"b", "B"},
		struct{ id, name string }{"c", "C"},
	)
	fe := newFallbackEditModel("editme", []string{"a", "b", "c"}, profiles)
	fe.section = 1
	fe.cursor = 1 // pointing at "b"

	fe.moveOrder(1) // move b down

	if fe.order[1] != "c" || fe.order[2] != "b" {
		t.Errorf("expected order [a c b], got %v", fe.order)
	}
	if fe.cursor != 2 {
		t.Errorf("cursor should follow moved item to index 2, got %d", fe.cursor)
	}
}

func TestMoveOrderBoundaryTopNoMove(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"a", "A"},
		struct{ id, name string }{"b", "B"},
	)
	fe := newFallbackEditModel("editme", []string{"a", "b"}, profiles)
	fe.section = 1
	fe.cursor = 0 // already at top

	fe.moveOrder(-1)

	if fe.order[0] != "a" || fe.order[1] != "b" {
		t.Errorf("order should be unchanged at top boundary, got %v", fe.order)
	}
	if fe.cursor != 0 {
		t.Errorf("cursor should remain at 0, got %d", fe.cursor)
	}
}

func TestMoveOrderBoundaryBottomNoMove(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"a", "A"},
		struct{ id, name string }{"b", "B"},
	)
	fe := newFallbackEditModel("editme", []string{"a", "b"}, profiles)
	fe.section = 1
	fe.cursor = 1 // already at bottom

	fe.moveOrder(1)

	if fe.order[0] != "a" || fe.order[1] != "b" {
		t.Errorf("order should be unchanged at bottom boundary, got %v", fe.order)
	}
	if fe.cursor != 1 {
		t.Errorf("cursor should remain at 1, got %d", fe.cursor)
	}
}

func TestMoveOrderNoOpWhenSectionIsNotOne(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"a", "A"},
		struct{ id, name string }{"b", "B"},
	)
	fe := newFallbackEditModel("editme", []string{"a", "b"}, profiles)
	fe.section = 0 // candidates section
	fe.cursor = 0

	fe.moveOrder(1) // should be no-op

	if fe.order[0] != "a" {
		t.Errorf("order should be unchanged when section != 1, got %v", fe.order)
	}
}

// ── totalItems ───────────────────────────────────────────────────────────────

func TestTotalItemsSection0ReturnsCandidateCount(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"p1", "P1"},
		struct{ id, name string }{"p2", "P2"},
		struct{ id, name string }{"p3", "P3"},
	)
	fe := newFallbackEditModel("editme", nil, profiles)
	fe.section = 0

	if fe.totalItems() != 3 {
		t.Errorf("expected totalItems()=3 for section 0, got %d", fe.totalItems())
	}
}

func TestTotalItemsSection1ReturnsOrderCount(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"p1", "P1"},
		struct{ id, name string }{"p2", "P2"},
		struct{ id, name string }{"p3", "P3"},
	)
	fe := newFallbackEditModel("editme", []string{"p1", "p2"}, profiles)
	fe.section = 1

	if fe.totalItems() != 2 {
		t.Errorf("expected totalItems()=2 for section 1, got %d", fe.totalItems())
	}
}

// ── candidateName ────────────────────────────────────────────────────────────

func TestCandidateNameKnownID(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"abc123", "My Profile"},
	)
	fe := newFallbackEditModel("editme", nil, profiles)

	got := fe.candidateName("abc123")
	if got != "My Profile" {
		t.Errorf("expected \"My Profile\", got %q", got)
	}
}

func TestCandidateNameUnknownIDReturnsSelf(t *testing.T) {
	profiles := makeProfiles(
		struct{ id, name string }{"abc123", "My Profile"},
	)
	fe := newFallbackEditModel("editme", nil, profiles)

	unknownID := "does-not-exist"
	got := fe.candidateName(unknownID)
	if got != unknownID {
		t.Errorf("expected the ID itself (%q) for unknown ID, got %q", unknownID, got)
	}
}
