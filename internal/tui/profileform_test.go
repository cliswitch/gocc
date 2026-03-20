package tui

import (
	"strings"
	"testing"

	"github.com/cliswitch/gocc/internal/config"
)

// ── isDirty ──────────────────────────────────────────────────────────────────

func TestIsDirtyCleanProfile(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test Profile",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.example.com",
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	if pf.isDirty() {
		t.Error("freshly created form should not be dirty")
	}
}

func TestIsDirtyChangedName(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Original Name",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	// Simulate user typing a new name into the name input (index 0).
	pf.inputs[0].SetValue("Changed Name")

	if !pf.isDirty() {
		t.Error("form should be dirty after changing Name input")
	}
}

func TestIsDirtyChangedCustomEnv(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	// Directly mutate the profile map (as the editor callbacks would do).
	pf.profile.CustomEnv = map[string]string{"MY_VAR": "value"}

	if !pf.isDirty() {
		t.Error("form should be dirty after changing CustomEnv")
	}
}

func TestIsDirtyChangedInheritGlobalArgs(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	// Toggle InheritGlobalArgs (originally nil, now explicitly false).
	pf.profile.InheritGlobalArgs = toggleBool(pf.profile.InheritGlobalArgs)

	if !pf.isDirty() {
		t.Error("form should be dirty after changing InheritGlobalArgs")
	}
}

func TestIsDirtyChangedClaudeArgs(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	pf.profile.ClaudeArgs = []string{"--verbose"}

	if !pf.isDirty() {
		t.Error("form should be dirty after changing ClaudeArgs")
	}
}

func TestIsDirtyNativeProfileUnchanged(t *testing.T) {
	p := config.Profile{
		ID:       config.NativeProfileID,
		Name:     config.NativeProfileName,
		Protocol: config.ProtocolNative,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	if pf.isDirty() {
		t.Error("native profile with no changes should not be dirty")
	}
}

// ── validate ─────────────────────────────────────────────────────────────────

func TestValidateValidInputs(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
		Reasoning: config.Reasoning{
			MaxOutputTokens:   4096,
			MaxThinkingTokens: 1024,
		},
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	if err := pf.validate(); err != nil {
		t.Errorf("expected no error for valid inputs, got: %v", err)
	}
}

func TestValidateNonNumericMaxOutputTokens(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	// Input index 8 is max_output_tokens.
	pf.inputs[8].SetValue("not-a-number")

	err := pf.validate()
	if err == nil {
		t.Error("expected validation error for non-numeric max_output_tokens")
	}
	if !strings.Contains(err.Error(), "max_output_tokens") {
		t.Errorf("error message should mention \"max_output_tokens\", got: %v", err)
	}
}

func TestValidateNonNumericMaxThinkingTokens(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	// Input index 9 is max_thinking_tokens.
	pf.inputs[9].SetValue("abc")

	err := pf.validate()
	if err == nil {
		t.Error("expected validation error for non-numeric max_thinking_tokens")
	}
	if !strings.Contains(err.Error(), "max_thinking_tokens") {
		t.Errorf("error message should mention \"max_thinking_tokens\", got: %v", err)
	}
}

func TestValidateEmptyNumericFieldsAreValid(t *testing.T) {
	p := config.Profile{
		ID:       "abc123",
		Name:     "Test",
		Protocol: config.ProtocolOpenAI,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	// Leave inputs[8] and inputs[9] empty — empty is allowed.
	pf.inputs[8].SetValue("")
	pf.inputs[9].SetValue("")

	if err := pf.validate(); err != nil {
		t.Errorf("empty token fields should be valid, got: %v", err)
	}
}

func TestValidateNativeProfileAlwaysNil(t *testing.T) {
	p := config.Profile{
		ID:       config.NativeProfileID,
		Name:     config.NativeProfileName,
		Protocol: config.ProtocolNative,
	}
	pf := newProfileFormModel(p, false, Callbacks{})

	if err := pf.validate(); err != nil {
		t.Errorf("native profile validate() should always return nil, got: %v", err)
	}
}

// ── maskAPIKey ───────────────────────────────────────────────────────────────

func TestMaskAPIKeyEmpty(t *testing.T) {
	result := maskAPIKey("")
	// dimStyle.Render("(empty)") — in a no-TTY test context lipgloss strips
	// ANSI codes, so we check that the plain label text is present.
	if !strings.Contains(result, "empty") {
		t.Errorf("expected \"(empty)\" indicator, got %q", result)
	}
}

func TestMaskAPIKeyShortKeyAllBullets(t *testing.T) {
	// Keys of 8 chars or fewer → all bullets.
	key := "sk-12345" // 8 chars
	result := maskAPIKey(key)
	expected := strings.Repeat("•", len(key))
	if result != expected {
		t.Errorf("short key: expected all bullets %q, got %q", expected, result)
	}
}

func TestMaskAPIKeyVeryShortKey(t *testing.T) {
	key := "abc" // 3 chars — all bullets
	result := maskAPIKey(key)
	if result != "•••" {
		t.Errorf("expected \"•••\", got %q", result)
	}
}

func TestMaskAPIKeyNormalKey(t *testing.T) {
	// For a key longer than 8 chars: first 3 + bullets + last 5.
	key := "sk-abcdefghijklmnop" // 19 chars
	result := maskAPIKey(key)

	if !strings.HasPrefix(result, "sk-") {
		t.Errorf("expected prefix \"sk-\", got %q", result)
	}
	// Last 5 chars are "lmnop".
	if !strings.HasSuffix(result, "lmnop") {
		t.Errorf("expected suffix \"lmnop\", got %q", result)
	}
	// Middle should be bullets.
	middle := result[3 : len(result)-5]
	for _, r := range middle {
		if r != '•' {
			t.Errorf("middle section should be all bullets, got %q", string(r))
		}
	}
	// Total bullet count = len(key) - 8.
	bulletCount := len(key) - 8
	if len([]rune(middle)) != bulletCount {
		t.Errorf("expected %d bullets in middle, got %d", bulletCount, len([]rune(middle)))
	}
}

func TestMaskAPIKeyExactly9Chars(t *testing.T) {
	// 9-char key: first 3 + 1 bullet + last 5.
	key := "123456789"
	result := maskAPIKey(key)
	if !strings.HasPrefix(result, "123") {
		t.Errorf("expected prefix \"123\", got %q", result)
	}
	if !strings.HasSuffix(result, "56789") {
		t.Errorf("expected suffix \"56789\", got %q", result)
	}
}

// ── cycleValue ───────────────────────────────────────────────────────────────

func TestCycleValueForward(t *testing.T) {
	values := []string{"a", "b", "c"}
	got := cycleValue("a", values, 1)
	if got != "b" {
		t.Errorf("cycleValue forward: expected \"b\", got %q", got)
	}
}

func TestCycleValueBackward(t *testing.T) {
	values := []string{"a", "b", "c"}
	got := cycleValue("c", values, -1)
	if got != "b" {
		t.Errorf("cycleValue backward: expected \"b\", got %q", got)
	}
}

func TestCycleValueWrapAroundAtEnd(t *testing.T) {
	values := []string{"a", "b", "c"}
	got := cycleValue("c", values, 1)
	if got != "a" {
		t.Errorf("cycleValue wrap at end: expected \"a\", got %q", got)
	}
}

func TestCycleValueWrapAroundAtStart(t *testing.T) {
	values := []string{"a", "b", "c"}
	got := cycleValue("a", values, -1)
	if got != "c" {
		t.Errorf("cycleValue wrap at start: expected \"c\", got %q", got)
	}
}

func TestCycleValueUnknownCurrentUsesFirstElement(t *testing.T) {
	// When current is not in values, idx defaults to 0.
	values := []string{"x", "y", "z"}
	got := cycleValue("unknown", values, 1)
	if got != "y" {
		t.Errorf("cycleValue unknown current: expected \"y\" (0+1), got %q", got)
	}
}

// ── toggleBool ───────────────────────────────────────────────────────────────

func TestToggleBoolNilReturnsFalse(t *testing.T) {
	result := toggleBool(nil)
	if result == nil {
		t.Fatal("toggleBool(nil) returned nil, expected *bool")
	}
	if *result != false {
		t.Errorf("toggleBool(nil): expected false, got %v", *result)
	}
}

func TestToggleBoolTrueReturnsFalse(t *testing.T) {
	v := true
	result := toggleBool(&v)
	if result == nil {
		t.Fatal("toggleBool(true) returned nil")
	}
	if *result != false {
		t.Errorf("toggleBool(true): expected false, got %v", *result)
	}
}

func TestToggleBoolFalseReturnsTrue(t *testing.T) {
	v := false
	result := toggleBool(&v)
	if result == nil {
		t.Fatal("toggleBool(false) returned nil")
	}
	if *result != true {
		t.Errorf("toggleBool(false): expected true, got %v", *result)
	}
}

func TestToggleBoolDoesNotMutateOriginal(t *testing.T) {
	v := true
	ptr := &v
	_ = toggleBool(ptr)
	if v != true {
		t.Error("toggleBool should not mutate the original bool")
	}
}
