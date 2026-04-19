package launcher

import (
	"strings"
	"testing"
)

func TestFindClaude(t *testing.T) {
	path, err := FindClaude()
	if err != nil {
		t.Skipf("claude not in PATH: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestMergeEnv(t *testing.T) {
	base := []string{"A=1", "B=2", "C=3"}
	extra := []string{"B=override", "D=4"}
	merged := mergeEnv(base, extra)

	m := make(map[string]string)
	for _, e := range merged {
		parts := splitEnv(e)
		m[parts[0]] = parts[1]
	}

	if m["A"] != "1" {
		t.Error("A should be preserved")
	}
	if m["B"] != "override" {
		t.Error("B should be overridden")
	}
	if m["C"] != "3" {
		t.Error("C should be preserved")
	}
	if m["D"] != "4" {
		t.Error("D should be added")
	}
}

func TestMergeEnv_PreservesOrder(t *testing.T) {
	base := []string{"A=1", "B=2", "C=3"}
	extra := []string{"B=override", "D=4"}
	result := mergeEnv(base, extra)
	expected := []string{"A=1", "B=override", "C=3", "D=4"}

	if len(result) != len(expected) {
		t.Fatalf("len(result) = %d, want %d; result = %v", len(result), len(expected), result)
	}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestExtractGoccFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantProfile string
		wantDebug   bool
		wantRest    []string
	}{
		{
			name:        "no gocc flags",
			args:        []string{"--dangerously-skip-permissions", "-p", "hello"},
			wantProfile: "",
			wantRest:    []string{"--dangerously-skip-permissions", "-p", "hello"},
		},
		{
			name:        "goccprofile with space",
			args:        []string{"--goccprofile", "myprofile", "--help"},
			wantProfile: "myprofile",
			wantRest:    []string{"--help"},
		},
		{
			name:        "goccprofile with equals",
			args:        []string{"--goccprofile=myprofile", "--help"},
			wantProfile: "myprofile",
			wantRest:    []string{"--help"},
		},
		{
			name:        "goccprofile only",
			args:        []string{"--goccprofile", "test"},
			wantProfile: "test",
			wantRest:    nil,
		},
		{
			name:        "goccdebug alone",
			args:        []string{"--goccdebug", "--help"},
			wantProfile: "",
			wantDebug:   true,
			wantRest:    []string{"--help"},
		},
		{
			name:        "goccprofile and goccdebug",
			args:        []string{"--goccprofile", "p", "--goccdebug", "--help"},
			wantProfile: "p",
			wantDebug:   true,
			wantRest:    []string{"--help"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, debug, rest := ExtractGoccFlags(tt.args)
			if profile != tt.wantProfile {
				t.Errorf("profile = %q, want %q", profile, tt.wantProfile)
			}
			if debug != tt.wantDebug {
				t.Errorf("debug = %v, want %v", debug, tt.wantDebug)
			}
			if len(rest) != len(tt.wantRest) {
				t.Errorf("rest = %v, want %v", rest, tt.wantRest)
			}
		})
	}
}

func splitEnv(s string) []string {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return []string{s, ""}
	}
	return []string{s[:idx], s[idx+1:]}
}
