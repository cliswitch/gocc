package tui

import "testing"

func TestParseExtraBody(t *testing.T) {
	input := "service_tier: \"priority\"\ncount: 42\nflag: true"
	m, err := parseExtraBody(input)
	if err != nil {
		t.Fatal(err)
	}
	if m["service_tier"] != "priority" {
		t.Errorf("service_tier = %v", m["service_tier"])
	}
	// JSON unmarshal produces float64 for numbers
	if m["count"] != float64(42) {
		t.Errorf("count = %v (%T)", m["count"], m["count"])
	}
	if m["flag"] != true {
		t.Errorf("flag = %v", m["flag"])
	}
}

func TestParseExtraBodyBareString(t *testing.T) {
	input := "service_tier: priority"
	m, err := parseExtraBody(input)
	if err != nil {
		t.Fatal(err)
	}
	if m["service_tier"] != "priority" {
		t.Errorf("service_tier = %v", m["service_tier"])
	}
}

func TestParseExtraBodyNestedJSON(t *testing.T) {
	input := `nested: {"a": 1, "b": "two"}`
	m, err := parseExtraBody(input)
	if err != nil {
		t.Fatal(err)
	}
	nested, ok := m["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested is %T, want map[string]any", m["nested"])
	}
	if nested["b"] != "two" {
		t.Errorf("nested.b = %v", nested["b"])
	}
}

func TestParseExtraBodyInvalidJSONStructure(t *testing.T) {
	// Values starting with { must parse as valid JSON
	_, err := parseExtraBody(`key: {invalid`)
	if err == nil {
		t.Error("expected error for invalid JSON object")
	}

	// Values starting with [ must parse as valid JSON
	_, err = parseExtraBody(`key: [invalid`)
	if err == nil {
		t.Error("expected error for invalid JSON array")
	}

	// Values starting with " must parse as valid JSON
	_, err = parseExtraBody(`key: "unterminated`)
	if err == nil {
		t.Error("expected error for invalid JSON string")
	}
}

func TestParseExtraBodyMissingSeparator(t *testing.T) {
	_, err := parseExtraBody("no-separator")
	if err == nil {
		t.Error("expected error for missing ': ' separator")
	}
}

func TestParseExtraBodyEmptyKey(t *testing.T) {
	_, err := parseExtraBody(": value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestParseExtraBodyEmpty(t *testing.T) {
	m, err := parseExtraBody("")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestFormatExtraBody(t *testing.T) {
	m := map[string]any{
		"service_tier": "priority",
		"count":        42,
	}
	text := formatExtraBody(m)
	// Keys sorted: count before service_tier
	expected := "count: 42\nservice_tier: \"priority\""
	if text != expected {
		t.Errorf("got:\n%s\nwant:\n%s", text, expected)
	}
}

func TestFormatExtraBodyEmpty(t *testing.T) {
	if text := formatExtraBody(nil); text != "" {
		t.Errorf("expected empty, got %q", text)
	}
}

func TestExtraBodyRoundTrip(t *testing.T) {
	m := map[string]any{
		"service_tier": "priority",
		"count":        float64(42),
		"flag":         true,
	}
	text := formatExtraBody(m)
	parsed, err := parseExtraBody(text)
	if err != nil {
		t.Fatalf("round-trip parse error: %v", err)
	}
	if parsed["service_tier"] != "priority" {
		t.Error("round-trip: service_tier mismatch")
	}
	if parsed["count"] != float64(42) {
		t.Error("round-trip: count mismatch")
	}
	if parsed["flag"] != true {
		t.Error("round-trip: flag mismatch")
	}
}
