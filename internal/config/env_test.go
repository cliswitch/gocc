package config

import (
	"strings"
	"testing"
)

func TestProfileEnvVars(t *testing.T) {
	p := Profile{
		Protocol: ProtocolOpenAI,
		Models: Models{
			MainModel:  "sonnet",
			HaikuModel:    "gpt-4o-mini",
			SonnetModel:   "gpt-4o",
			OpusModel:     "gpt-4.1",
			SubagentModel: "gpt-4o-mini",
		},
		Reasoning: Reasoning{
			MaxOutputTokens:   16000,
			MaxThinkingTokens: 8000,
			EffortLevel:       "high",
		},
		Proxy: Proxy{
			HTTPSProxy: "http://proxy:8080",
		},
	}
	envs := p.EnvVars()
	m := envMap(envs)

	if m["ANTHROPIC_MODEL"] != "sonnet" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", m["ANTHROPIC_MODEL"], "sonnet")
	}
	if m["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "s-gpt-4o" {
		t.Errorf("SONNET = %q", m["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
	if m["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "h-gpt-4o-mini" {
		t.Errorf("HAIKU = %q", m["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if m["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "o-gpt-4.1" {
		t.Errorf("OPUS = %q", m["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if m["CLAUDE_CODE_SUBAGENT_MODEL"] != "sa-gpt-4o-mini" {
		t.Errorf("SUBAGENT = %q", m["CLAUDE_CODE_SUBAGENT_MODEL"])
	}
	if m["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] != "16000" {
		t.Errorf("MAX_OUTPUT = %q", m["CLAUDE_CODE_MAX_OUTPUT_TOKENS"])
	}
	if m["MAX_THINKING_TOKENS"] != "8000" {
		t.Errorf("MAX_THINKING = %q", m["MAX_THINKING_TOKENS"])
	}
	if m["CLAUDE_CODE_EFFORT_LEVEL"] != "high" {
		t.Errorf("EFFORT = %q", m["CLAUDE_CODE_EFFORT_LEVEL"])
	}
	if m["HTTPS_PROXY"] != "http://proxy:8080" {
		t.Errorf("HTTPS_PROXY = %q", m["HTTPS_PROXY"])
	}
}

func TestProfileEnvVarsNoAnnotation(t *testing.T) {
	p := Profile{
		Protocol: ProtocolOpenAI,
		Models: Models{
			MainModel:   "sonnet",
			HaikuModel:  "gpt-4o-mini",
			SonnetModel: "gpt-4o",
			OpusModel:   "gpt-4.1",
		},
	}
	envs := p.EnvVars()
	m := envMap(envs)

	// All model names are unique — no annotation prefix expected.
	if m["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "gpt-4o-mini" {
		t.Errorf("HAIKU = %q, want gpt-4o-mini", m["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if m["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "gpt-4o" {
		t.Errorf("SONNET = %q, want gpt-4o", m["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
	if m["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "gpt-4.1" {
		t.Errorf("OPUS = %q, want gpt-4.1", m["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
}

func TestProfileEnvVarsSkipsEmpty(t *testing.T) {
	p := Profile{
		Protocol: ProtocolOpenAI,
		Models:   Models{MainModel: "sonnet", SonnetModel: "gpt-4o"},
	}
	envs := p.EnvVars()
	m := envMap(envs)

	if _, ok := m["ANTHROPIC_DEFAULT_HAIKU_MODEL"]; ok {
		t.Error("empty haiku_model should not be set")
	}
	if _, ok := m["CLAUDE_CODE_MAX_OUTPUT_TOKENS"]; ok {
		t.Error("zero max_output_tokens should not be set")
	}
	if _, ok := m["HTTPS_PROXY"]; ok {
		t.Error("empty proxy should not be set")
	}
}

func TestNativeProfileEnvVars(t *testing.T) {
	p := Profile{
		Protocol: ProtocolNative,
		Proxy:    Proxy{HTTPSProxy: "http://proxy:8080"},
	}
	envs := p.EnvVars()
	m := envMap(envs)

	if _, ok := m["ANTHROPIC_MODEL"]; ok {
		t.Error("native should not set ANTHROPIC_MODEL")
	}
	if m["HTTPS_PROXY"] != "http://proxy:8080" {
		t.Errorf("HTTPS_PROXY = %q", m["HTTPS_PROXY"])
	}
}

func envMap(envs []string) map[string]string {
	m := make(map[string]string)
	for _, e := range envs {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}
