package tui

import (
	"strings"
	"testing"
)

func TestParseEnvVars(t *testing.T) {
	input := "FOO=bar\nBAZ=qux\n\n"
	env, err := parseEnvVars(input)
	if err != nil {
		t.Fatal(err)
	}
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Errorf("unexpected: %v", env)
	}
}

func TestParseEnvVarsEmptyValue(t *testing.T) {
	// KEY= is valid — value may be empty
	input := "MY_VAR="
	env, err := parseEnvVars(input)
	if err != nil {
		t.Fatalf("expected no error for empty value, got: %v", err)
	}
	v, ok := env["MY_VAR"]
	if !ok {
		t.Error("expected key MY_VAR to be present")
	}
	if v != "" {
		t.Errorf("expected empty value, got: %q", v)
	}
}

func TestParseEnvVarsEmptyLinesSkipped(t *testing.T) {
	input := "\n\nFOO=bar\n\n"
	env, err := parseEnvVars(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 1 || env["FOO"] != "bar" {
		t.Errorf("unexpected: %v", env)
	}
}

func TestParseEnvVarsMissingEquals(t *testing.T) {
	_, err := parseEnvVars("INVALID_NO_EQUALS")
	if err == nil {
		t.Error("expected error for missing '='")
	}
}

func TestParseEnvVarsEmptyKey(t *testing.T) {
	_, err := parseEnvVars("=value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestParseEnvVarsEmpty(t *testing.T) {
	env, err := parseEnvVars("")
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 0 {
		t.Errorf("expected empty map, got: %v", env)
	}
}

func TestParseEnvVarsValueWithEquals(t *testing.T) {
	// Value containing '=' should be preserved
	input := "TOKEN=abc=def=ghi"
	env, err := parseEnvVars(input)
	if err != nil {
		t.Fatal(err)
	}
	if env["TOKEN"] != "abc=def=ghi" {
		t.Errorf("unexpected value: %q", env["TOKEN"])
	}
}

func TestFormatEnvVars(t *testing.T) {
	env := map[string]string{"FOO": "bar", "BAZ": "qux"}
	text := formatEnvVars(env)
	if text == "" {
		t.Error("expected non-empty output")
	}
	// Should be sorted by key: BAZ before FOO
	lines := strings.Split(text, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), text)
	}
	if lines[0] != "BAZ=qux" {
		t.Errorf("expected first line 'BAZ=qux', got: %q", lines[0])
	}
	if lines[1] != "FOO=bar" {
		t.Errorf("expected second line 'FOO=bar', got: %q", lines[1])
	}
}

func TestFormatEnvVarsEmpty(t *testing.T) {
	text := formatEnvVars(nil)
	if text != "" {
		t.Errorf("expected empty string, got: %q", text)
	}
}

func TestFormatEnvVarsEmptyMap(t *testing.T) {
	text := formatEnvVars(map[string]string{})
	if text != "" {
		t.Errorf("expected empty string, got: %q", text)
	}
}
