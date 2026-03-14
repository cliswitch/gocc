package config

import (
	"slices"
	"strings"
	"testing"
)

// resolvedEnvMap converts the EnvVars slice into a map for easy lookup.
func resolvedEnvMap(envs []string) map[string]string {
	m := make(map[string]string)
	for _, e := range envs {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

// ---- Args tests ----

func TestResolveArgs_InheritTrue_MergesGlobalThenProfile(t *testing.T) {
	global := GlobalConfig{ClaudeArgs: []string{"--verbose", "--dangerously-skip-permissions"}}
	profile := Profile{
		ClaudeArgs: []string{"--max-turns", "10"},
	}
	// InheritGlobalArgs nil → defaults to true
	got := Resolve(global, profile)
	want := []string{"--verbose", "--dangerously-skip-permissions", "--max-turns", "10"}
	if !slices.Equal(got.ClaudeArgs, want) {
		t.Errorf("ClaudeArgs = %v, want %v", got.ClaudeArgs, want)
	}
}

func TestResolveArgs_InheritFalse_ProfileOnly(t *testing.T) {
	global := GlobalConfig{ClaudeArgs: []string{"--verbose"}}
	f := false
	profile := Profile{
		ClaudeArgs:        []string{"--max-turns", "5"},
		InheritGlobalArgs: &f,
	}
	got := Resolve(global, profile)
	want := []string{"--max-turns", "5"}
	if !slices.Equal(got.ClaudeArgs, want) {
		t.Errorf("ClaudeArgs = %v, want %v", got.ClaudeArgs, want)
	}
}

func TestResolveArgs_BothEmpty(t *testing.T) {
	global := GlobalConfig{}
	profile := Profile{}
	got := Resolve(global, profile)
	if len(got.ClaudeArgs) != 0 {
		t.Errorf("ClaudeArgs = %v, want empty", got.ClaudeArgs)
	}
}

func TestResolveArgs_OnlyGlobal(t *testing.T) {
	global := GlobalConfig{ClaudeArgs: []string{"--verbose"}}
	profile := Profile{} // no profile args
	got := Resolve(global, profile)
	want := []string{"--verbose"}
	if !slices.Equal(got.ClaudeArgs, want) {
		t.Errorf("ClaudeArgs = %v, want %v", got.ClaudeArgs, want)
	}
}

func TestResolveArgs_OnlyProfile(t *testing.T) {
	global := GlobalConfig{} // no global args
	f := false
	profile := Profile{
		ClaudeArgs:        []string{"--max-turns", "3"},
		InheritGlobalArgs: &f,
	}
	got := Resolve(global, profile)
	want := []string{"--max-turns", "3"}
	if !slices.Equal(got.ClaudeArgs, want) {
		t.Errorf("ClaudeArgs = %v, want %v", got.ClaudeArgs, want)
	}
}

// ---- Env tests ----

func TestResolveEnv_InheritTrue_ProfileWinsConflicts(t *testing.T) {
	global := GlobalConfig{
		CustomEnv: map[string]string{
			"MY_KEY":     "global-value",
			"GLOBAL_KEY": "only-in-global",
		},
	}
	profile := Profile{
		Protocol: ProtocolNative, // no structured env vars
		CustomEnv: map[string]string{
			"MY_KEY":      "profile-value", // overrides global
			"PROFILE_KEY": "only-in-profile",
		},
	}
	got := Resolve(global, profile)
	m := resolvedEnvMap(got.EnvVars)

	if m["MY_KEY"] != "profile-value" {
		t.Errorf("MY_KEY = %q, want %q", m["MY_KEY"], "profile-value")
	}
	if m["GLOBAL_KEY"] != "only-in-global" {
		t.Errorf("GLOBAL_KEY = %q, want %q", m["GLOBAL_KEY"], "only-in-global")
	}
	if m["PROFILE_KEY"] != "only-in-profile" {
		t.Errorf("PROFILE_KEY = %q, want %q", m["PROFILE_KEY"], "only-in-profile")
	}
}

func TestResolveEnv_InheritFalse_OnlyProfileCustomEnv(t *testing.T) {
	global := GlobalConfig{
		CustomEnv: map[string]string{
			"GLOBAL_KEY": "should-not-appear",
		},
	}
	f := false
	profile := Profile{
		Protocol: ProtocolNative,
		CustomEnv: map[string]string{
			"PROFILE_KEY": "profile-value",
		},
		InheritGlobalEnv: &f,
	}
	got := Resolve(global, profile)
	m := resolvedEnvMap(got.EnvVars)

	if _, ok := m["GLOBAL_KEY"]; ok {
		t.Errorf("GLOBAL_KEY should not appear when InheritGlobalEnv=false")
	}
	if m["PROFILE_KEY"] != "profile-value" {
		t.Errorf("PROFILE_KEY = %q, want %q", m["PROFILE_KEY"], "profile-value")
	}
}

func TestResolveEnv_CustomEnvOverridesStructuredEnv(t *testing.T) {
	profile := Profile{
		Protocol: ProtocolOpenAI,
		Models: Models{
			MainModel: "sonnet",
			SonnetModel:  "gpt-4o",
		},
		CustomEnv: map[string]string{
			"ANTHROPIC_MODEL": "opus", // overrides structured env
		},
	}
	got := Resolve(GlobalConfig{}, profile)
	m := resolvedEnvMap(got.EnvVars)

	if m["ANTHROPIC_MODEL"] != "opus" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q (custom env should override structured env)", m["ANTHROPIC_MODEL"], "opus")
	}
}

func TestResolveEnv_EmptyCustomEnv_OnlyStructuredEnv(t *testing.T) {
	profile := Profile{
		Protocol: ProtocolOpenAI,
		Models: Models{
			MainModel: "sonnet",
			SonnetModel:  "gpt-4o",
		},
		// no CustomEnv
	}
	got := Resolve(GlobalConfig{}, profile)
	m := resolvedEnvMap(got.EnvVars)

	if m["ANTHROPIC_MODEL"] != "sonnet" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", m["ANTHROPIC_MODEL"], "sonnet")
	}
	if m["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "gpt-4o" {
		t.Errorf("ANTHROPIC_DEFAULT_SONNET_MODEL = %q, want %q", m["ANTHROPIC_DEFAULT_SONNET_MODEL"], "gpt-4o")
	}
}

func TestResolveEnv_BothEmpty(t *testing.T) {
	got := Resolve(GlobalConfig{}, Profile{Protocol: ProtocolNative})
	if len(got.EnvVars) != 0 {
		t.Errorf("EnvVars = %v, want empty", got.EnvVars)
	}
}

// ---- Dedup tests ----

func TestResolveEnv_DedupLastWins(t *testing.T) {
	// Structured env will have ANTHROPIC_MODEL=sonnet
	// Custom env will override with ANTHROPIC_MODEL=haiku
	// The result should have ANTHROPIC_MODEL=haiku (custom/last wins)
	profile := Profile{
		Protocol: ProtocolOpenAI,
		Models: Models{
			MainModel: "sonnet",
			SonnetModel:  "gpt-4o",
		},
		CustomEnv: map[string]string{
			"ANTHROPIC_MODEL": "haiku",
		},
	}
	got := Resolve(GlobalConfig{}, profile)
	m := resolvedEnvMap(got.EnvVars)

	// Verify exactly one occurrence of ANTHROPIC_MODEL
	count := 0
	for _, e := range got.EnvVars {
		if strings.HasPrefix(e, "ANTHROPIC_MODEL=") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("ANTHROPIC_MODEL appears %d times in EnvVars, want 1", count)
	}
	if m["ANTHROPIC_MODEL"] != "haiku" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q (last-wins dedup)", m["ANTHROPIC_MODEL"], "haiku")
	}
}

func TestResolveEnv_DedupNoSpuriousDuplicates(t *testing.T) {
	profile := Profile{
		Protocol: ProtocolOpenAI,
		Models: Models{
			MainModel: "sonnet",
			SonnetModel:  "gpt-4o",
			HaikuModel:   "gpt-4o-mini",
		},
		CustomEnv: map[string]string{
			"FOO": "bar",
		},
	}
	got := Resolve(GlobalConfig{}, profile)

	// Verify no key appears more than once
	seen := make(map[string]int)
	for _, e := range got.EnvVars {
		key, _, _ := strings.Cut(e, "=")
		seen[key]++
	}
	for key, cnt := range seen {
		if cnt > 1 {
			t.Errorf("key %q appears %d times, want 1", key, cnt)
		}
	}
}
