package config

import "testing"

func TestAnnotateModel(t *testing.T) {
	tests := []struct {
		level, model, want string
	}{
		{"haiku", "gpt-4o-mini", "h-gpt-4o-mini"},
		{"sonnet", "gpt-4o", "s-gpt-4o"},
		{"opus", "gpt-4.1", "o-gpt-4.1"},
		{"subagent", "gpt-4o-mini", "sa-gpt-4o-mini"},
	}
	for _, tt := range tests {
		got := AnnotateModel(tt.level, tt.model)
		if got != tt.want {
			t.Errorf("AnnotateModel(%q, %q) = %q, want %q", tt.level, tt.model, got, tt.want)
		}
	}
}

func TestParseAnnotatedModel(t *testing.T) {
	tests := []struct {
		input     string
		wantLevel string
		wantModel string
		wantOK    bool
	}{
		{"s-gpt-4o", "sonnet", "gpt-4o", true},
		{"h-gpt-4o-mini", "haiku", "gpt-4o-mini", true},
		{"o-gpt-4.1", "opus", "gpt-4.1", true},
		{"sa-gpt-4o-mini", "subagent", "gpt-4o-mini", true},
		{"gpt-4o", "", "gpt-4o", false},
		{"Unknown(foo)", "", "Unknown(foo)", false},
		{"s-", "sonnet", "", true},
	}
	for _, tt := range tests {
		level, model, ok := ParseAnnotatedModel(tt.input)
		if level != tt.wantLevel || model != tt.wantModel || ok != tt.wantOK {
			t.Errorf("ParseAnnotatedModel(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.input, level, model, ok, tt.wantLevel, tt.wantModel, tt.wantOK)
		}
	}
}

func TestLevelForModel(t *testing.T) {
	m := Models{
		HaikuModel:    "gpt-4o-mini",
		SonnetModel:   "gpt-4o",
		OpusModel:     "gpt-4.1",
		SubagentModel: "gpt-3.5-turbo",
	}
	tests := []struct {
		model, want string
	}{
		{"gpt-4o-mini", "haiku"},
		{"gpt-4o", "sonnet"},
		{"gpt-4.1", "opus"},
		{"gpt-3.5-turbo", "subagent"},
		{"unknown-model", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := m.LevelForModel(tt.model)
		if got != tt.want {
			t.Errorf("LevelForModel(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}

func TestNeedAnnotation(t *testing.T) {
	tests := []struct {
		name string
		m    Models
		want bool
	}{
		{"all unique", Models{HaikuModel: "a", SonnetModel: "b", OpusModel: "c", SubagentModel: "d"}, false},
		{"haiku=subagent", Models{HaikuModel: "a", SonnetModel: "b", OpusModel: "c", SubagentModel: "a"}, true},
		{"sonnet=opus", Models{HaikuModel: "a", SonnetModel: "b", OpusModel: "b"}, true},
		{"only one set", Models{SonnetModel: "b"}, false},
		{"all empty", Models{}, false},
		{"two empty rest unique", Models{HaikuModel: "a", OpusModel: "b"}, false},
	}
	for _, tt := range tests {
		got := tt.m.NeedAnnotation()
		if got != tt.want {
			t.Errorf("NeedAnnotation(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestModelForLevel(t *testing.T) {
	m := Models{
		HaikuModel:    "gpt-4o-mini",
		SonnetModel:   "gpt-4o",
		OpusModel:     "gpt-4.1",
		SubagentModel: "gpt-4o-mini",
	}
	tests := []struct {
		level, want string
	}{
		{"haiku", "gpt-4o-mini"},
		{"sonnet", "gpt-4o"},
		{"opus", "gpt-4.1"},
		{"subagent", "gpt-4o-mini"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := m.ModelForLevel(tt.level)
		if got != tt.want {
			t.Errorf("ModelForLevel(%q) = %q, want %q", tt.level, got, tt.want)
		}
	}
}
