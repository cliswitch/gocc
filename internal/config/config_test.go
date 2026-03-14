package config

import (
	"path/filepath"
	"testing"
)

func TestGenerateProfileID(t *testing.T) {
	id := GenerateProfileID()
	if len(id) != 10 {
		t.Errorf("expected 10 chars, got %d: %q", len(id), id)
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex char in id: %c", c)
		}
	}
	id2 := GenerateProfileID()
	if id == id2 {
		t.Error("two generated IDs should differ")
	}
}

func TestEnsureNative(t *testing.T) {
	cfg := &Config{}
	changed := EnsureNative(cfg)
	if !changed {
		t.Error("expected changed=true for empty config")
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[0]
	if p.ID != NativeProfileID {
		t.Errorf("expected id %q, got %q", NativeProfileID, p.ID)
	}
	if p.Name != NativeProfileName {
		t.Errorf("expected name %q, got %q", NativeProfileName, p.Name)
	}
	if p.Protocol != ProtocolNative {
		t.Errorf("expected protocol %q, got %q", ProtocolNative, p.Protocol)
	}
	changed2 := EnsureNative(cfg)
	if changed2 {
		t.Error("expected changed=false when native already exists")
	}
}

func TestFindProfile(t *testing.T) {
	cfg := &Config{
		Profiles: []Profile{
			{ID: "__native__", Name: "Anthropic Official", Protocol: "native"},
			{ID: "abc1234567", Name: "my-openai", Protocol: "openai_chat"},
		},
	}
	p, ok := FindProfile(cfg, "abc1234567")
	if !ok || p.Name != "my-openai" {
		t.Error("expected to find by ID")
	}
	p, ok = FindProfile(cfg, "my-openai")
	if !ok || p.ID != "abc1234567" {
		t.Error("expected to find by name")
	}
	_, ok = FindProfile(cfg, "nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestValidateConfig(t *testing.T) {
	cfg := &Config{
		Profiles: []Profile{
			{ID: "__native__", Name: "Anthropic Official", Protocol: "native"},
			{ID: "abc1234567", Name: "test", Protocol: "openai_chat",
				FallbackChain: []string{"__native__"}},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error: native in fallback chain")
	}
}

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Global: GlobalConfig{
			ClaudeArgs:  []string{"--dangerously-skip-permissions"},
			LastProfile: "__native__",
		},
		Profiles: []Profile{
			{ID: "__native__", Name: "Anthropic Official", Protocol: "native"},
			{
				ID: "abc1234567", Name: "test", Protocol: "openai_chat",
				BaseURL: "https://api.openai.com", APIKey: "sk-test",
				Models: Models{MainModel: "sonnet", SonnetModel: "gpt-4o"},
			},
		},
	}

	if err := SaveConfigTo(cfg, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[1].Models.SonnetModel != "gpt-4o" {
		t.Error("sonnet model not preserved")
	}
	if len(loaded.Global.ClaudeArgs) != 1 || loaded.Global.ClaudeArgs[0] != "--dangerously-skip-permissions" {
		t.Error("claude args not preserved")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	cfg, err := LoadConfigFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Profiles) != 0 {
		t.Error("expected empty profiles for missing file")
	}
}

func TestShouldInheritGlobalArgs(t *testing.T) {
	// nil pointer → default true
	p := Profile{}
	if !p.ShouldInheritGlobalArgs() {
		t.Error("nil InheritGlobalArgs should return true")
	}

	// explicit true → true
	trueVal := true
	p.InheritGlobalArgs = &trueVal
	if !p.ShouldInheritGlobalArgs() {
		t.Error("InheritGlobalArgs=true should return true")
	}

	// explicit false → false
	falseVal := false
	p.InheritGlobalArgs = &falseVal
	if p.ShouldInheritGlobalArgs() {
		t.Error("InheritGlobalArgs=false should return false")
	}
}

func TestShouldInheritGlobalEnv(t *testing.T) {
	// nil pointer → default true
	p := Profile{}
	if !p.ShouldInheritGlobalEnv() {
		t.Error("nil InheritGlobalEnv should return true")
	}

	// explicit true → true
	trueVal := true
	p.InheritGlobalEnv = &trueVal
	if !p.ShouldInheritGlobalEnv() {
		t.Error("InheritGlobalEnv=true should return true")
	}

	// explicit false → false
	falseVal := false
	p.InheritGlobalEnv = &falseVal
	if p.ShouldInheritGlobalEnv() {
		t.Error("InheritGlobalEnv=false should return false")
	}
}

func TestCloneProfileDeepCopiesNewFields(t *testing.T) {
	falseVal := false
	orig := Profile{
		ID:        "abc123",
		Name:      "test",
		ClaudeArgs: []string{"--flag1", "--flag2"},
		CustomEnv: map[string]string{"KEY": "value"},
		InheritGlobalArgs: &falseVal,
		InheritGlobalEnv:  &falseVal,
	}

	clone := CloneProfile(orig)

	// Verify values are equal
	if len(clone.ClaudeArgs) != 2 || clone.ClaudeArgs[0] != "--flag1" {
		t.Error("ClaudeArgs not correctly cloned")
	}
	if clone.CustomEnv["KEY"] != "value" {
		t.Error("CustomEnv not correctly cloned")
	}
	if clone.InheritGlobalArgs == nil || *clone.InheritGlobalArgs != false {
		t.Error("InheritGlobalArgs not correctly cloned")
	}
	if clone.InheritGlobalEnv == nil || *clone.InheritGlobalEnv != false {
		t.Error("InheritGlobalEnv not correctly cloned")
	}

	// Verify deep copy: modifying clone does not affect original
	clone.ClaudeArgs[0] = "--modified"
	if orig.ClaudeArgs[0] != "--flag1" {
		t.Error("ClaudeArgs clone is not independent (shares underlying array)")
	}

	clone.CustomEnv["KEY"] = "modified"
	if orig.CustomEnv["KEY"] != "value" {
		t.Error("CustomEnv clone is not independent (shares underlying map)")
	}

	*clone.InheritGlobalArgs = true
	if *orig.InheritGlobalArgs != false {
		t.Error("InheritGlobalArgs clone is not independent (shares pointer)")
	}

	*clone.InheritGlobalEnv = true
	if *orig.InheritGlobalEnv != false {
		t.Error("InheritGlobalEnv clone is not independent (shares pointer)")
	}
}

func TestCloneProfileNilNewFields(t *testing.T) {
	orig := Profile{
		ID:   "abc123",
		Name: "test",
		// ClaudeArgs, CustomEnv, InheritGlobalArgs, InheritGlobalEnv all nil
	}
	clone := CloneProfile(orig)
	if clone.ClaudeArgs != nil {
		t.Error("expected nil ClaudeArgs in clone")
	}
	if clone.CustomEnv != nil {
		t.Error("expected nil CustomEnv in clone")
	}
	if clone.InheritGlobalArgs != nil {
		t.Error("expected nil InheritGlobalArgs in clone")
	}
	if clone.InheritGlobalEnv != nil {
		t.Error("expected nil InheritGlobalEnv in clone")
	}
}

func TestCloneProfileExtraBody(t *testing.T) {
	orig := Profile{
		ID:   "abc123",
		Name: "test",
		ExtraBody: map[string]any{
			"service_tier": "priority",
			"count":        42,
			"nested":       map[string]any{"a": 1},
		},
	}

	clone := CloneProfile(orig)

	// Verify values match
	if clone.ExtraBody["service_tier"] != "priority" {
		t.Error("ExtraBody service_tier not cloned")
	}

	// Verify deep copy: modify clone's nested map
	nested := clone.ExtraBody["nested"].(map[string]any)
	nested["a"] = 999
	origNested := orig.ExtraBody["nested"].(map[string]any)
	if origNested["a"] != 1 {
		t.Error("ExtraBody clone is not independent (nested map shared)")
	}
}

func TestCloneProfileExtraBodyNil(t *testing.T) {
	orig := Profile{ID: "abc123", Name: "test"}
	clone := CloneProfile(orig)
	if clone.ExtraBody != nil {
		t.Error("expected nil ExtraBody in clone")
	}
}

func TestYAMLRoundTripExtraBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Profiles: []Profile{
			{
				ID:       "abc1234567",
				Name:     "test",
				Protocol: "openai_chat",
				BaseURL:  "https://api.openai.com",
				APIKey:   "sk-test",
				ExtraBody: map[string]any{
					"service_tier": "priority",
					"user":         "my-team",
				},
			},
		},
	}

	if err := SaveConfigTo(cfg, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	eb := loaded.Profiles[0].ExtraBody
	if eb["service_tier"] != "priority" {
		t.Errorf("service_tier = %v, want priority", eb["service_tier"])
	}
	if eb["user"] != "my-team" {
		t.Errorf("user = %v, want my-team", eb["user"])
	}
}

func TestYAMLRoundTripNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	falseVal := false
	cfg := &Config{
		Global: GlobalConfig{
			ClaudeArgs:  []string{"--global-flag"},
			CustomEnv:   map[string]string{"GLOBAL_KEY": "global_val"},
			LastProfile: "__native__",
		},
		Profiles: []Profile{
			{
				ID:        "__native__",
				Name:      "Anthropic Official",
				Protocol:  "native",
				ClaudeArgs: []string{"--profile-flag"},
				CustomEnv: map[string]string{"PROFILE_KEY": "profile_val"},
				InheritGlobalArgs: &falseVal,
				InheritGlobalEnv:  &falseVal,
			},
		},
	}

	if err := SaveConfigTo(cfg, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Verify GlobalConfig.CustomEnv
	if loaded.Global.CustomEnv["GLOBAL_KEY"] != "global_val" {
		t.Error("GlobalConfig.CustomEnv not preserved in YAML round-trip")
	}

	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	p := loaded.Profiles[0]

	// Verify Profile.ClaudeArgs
	if len(p.ClaudeArgs) != 1 || p.ClaudeArgs[0] != "--profile-flag" {
		t.Error("Profile.ClaudeArgs not preserved in YAML round-trip")
	}

	// Verify Profile.CustomEnv
	if p.CustomEnv["PROFILE_KEY"] != "profile_val" {
		t.Error("Profile.CustomEnv not preserved in YAML round-trip")
	}

	// Verify Profile.InheritGlobalArgs
	if p.InheritGlobalArgs == nil || *p.InheritGlobalArgs != false {
		t.Error("Profile.InheritGlobalArgs not preserved in YAML round-trip")
	}

	// Verify Profile.InheritGlobalEnv
	if p.InheritGlobalEnv == nil || *p.InheritGlobalEnv != false {
		t.Error("Profile.InheritGlobalEnv not preserved in YAML round-trip")
	}
}
