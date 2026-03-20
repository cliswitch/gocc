package tui

import (
	"strings"
	"testing"
)

// ── parseArgs ────────────────────────────────────────────────────────────────

func TestParseArgsEmpty(t *testing.T) {
	result := parseArgs("")
	if len(result) != 0 {
		t.Errorf("expected nil/empty slice for empty string, got %v", result)
	}
}

func TestParseArgsSingleLine(t *testing.T) {
	result := parseArgs("--verbose")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d: %v", len(result), result)
	}
	if result[0] != "--verbose" {
		t.Errorf("expected \"--verbose\", got %q", result[0])
	}
}

func TestParseArgsMultipleLines(t *testing.T) {
	result := parseArgs("--verbose\n--no-color\n--model sonnet")
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d: %v", len(result), result)
	}
	if result[0] != "--verbose" {
		t.Errorf("result[0] = %q, want \"--verbose\"", result[0])
	}
	if result[1] != "--no-color" {
		t.Errorf("result[1] = %q, want \"--no-color\"", result[1])
	}
	if result[2] != "--model sonnet" {
		t.Errorf("result[2] = %q, want \"--model sonnet\"", result[2])
	}
}

func TestParseArgsBlankLinesSkipped(t *testing.T) {
	result := parseArgs("--verbose\n\n--no-color\n\n")
	if len(result) != 2 {
		t.Fatalf("expected 2 elements (blank lines skipped), got %d: %v", len(result), result)
	}
	if result[0] != "--verbose" {
		t.Errorf("result[0] = %q, want \"--verbose\"", result[0])
	}
	if result[1] != "--no-color" {
		t.Errorf("result[1] = %q, want \"--no-color\"", result[1])
	}
}

func TestParseArgsLeadingTrailingWhitespaceTrimmed(t *testing.T) {
	result := parseArgs("  --verbose  \n\t--no-color\t")
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d: %v", len(result), result)
	}
	if result[0] != "--verbose" {
		t.Errorf("result[0] = %q, want \"--verbose\" (whitespace trimmed)", result[0])
	}
	if result[1] != "--no-color" {
		t.Errorf("result[1] = %q, want \"--no-color\" (whitespace trimmed)", result[1])
	}
}

func TestParseArgsWhitespaceOnlyLinesSkipped(t *testing.T) {
	// A line consisting only of spaces becomes empty after TrimSpace and should be skipped.
	result := parseArgs("--foo\n   \n--bar")
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d: %v", len(result), result)
	}
}

// ── collectionSummary ────────────────────────────────────────────────────────

func TestCollectionSummaryZeroWithEmptyLabel(t *testing.T) {
	s := collectionSummary(0, "var", "empty")
	// The dimStyle.Render wraps the text with ANSI codes in a real terminal;
	// in tests (no TTY) lipgloss renders without escape codes, so the plain
	// string "(empty)" should be contained.
	if !strings.Contains(s, "empty") {
		t.Errorf("expected string containing \"empty\", got %q", s)
	}
}

func TestCollectionSummaryZeroWithNoneLabel(t *testing.T) {
	s := collectionSummary(0, "profile", "none")
	if !strings.Contains(s, "none") {
		t.Errorf("expected string containing \"none\", got %q", s)
	}
}

func TestCollectionSummaryNonZeroWithVarUnit(t *testing.T) {
	s := collectionSummary(3, "var", "empty")
	if s != "3 var(s)" {
		t.Errorf("expected \"3 var(s)\", got %q", s)
	}
}

func TestCollectionSummaryNonZeroWithArgUnit(t *testing.T) {
	s := collectionSummary(1, "arg", "none")
	if s != "1 arg(s)" {
		t.Errorf("expected \"1 arg(s)\", got %q", s)
	}
}

func TestCollectionSummaryNonZeroDoesNotContainLabel(t *testing.T) {
	// When count > 0 the empty label should NOT appear in the output.
	s := collectionSummary(5, "header", "empty")
	if strings.Contains(s, "empty") {
		t.Errorf("did not expect \"empty\" in output when count > 0, got %q", s)
	}
}
