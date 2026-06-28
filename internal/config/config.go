package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	NativeProfileID   = "__native__"
	NativeProfileName = "Anthropic Official"
	ProtocolNative    = "native"
	ProtocolOpenAI    = "openai_chat"
	ProtocolResponses = "openai_responses"
	ProtocolAnthropic = "anthropic"
	ProtocolGemini    = "gemini"
)

var ValidProtocols = []string{ProtocolOpenAI, ProtocolResponses, ProtocolAnthropic, ProtocolGemini}

type Config struct {
	Global   GlobalConfig `yaml:"global,omitempty"`
	Profiles []Profile    `yaml:"profiles"`
}

type GlobalConfig struct {
	ClaudeArgs  []string          `yaml:"claude_args,omitempty"`
	CustomEnv   map[string]string `yaml:"custom_env,omitempty"`
	LastProfile string            `yaml:"last_profile,omitempty"`
}

type Profile struct {
	ID                 string            `yaml:"id,omitempty"`
	Name               string            `yaml:"name"`
	Protocol           string            `yaml:"protocol,omitempty"`
	BaseURL            string            `yaml:"base_url,omitempty"`
	APIKey             string            `yaml:"api_key,omitempty"`
	Models             Models            `yaml:"models,omitempty"`
	Reasoning          Reasoning         `yaml:"reasoning,omitempty"`
	CustomHeaders      map[string]string `yaml:"custom_headers,omitempty"`
	ExtraBody          map[string]any    `yaml:"extra_body,omitempty"`
	Proxy              Proxy             `yaml:"proxy,omitempty"`
	Resilience         Resilience        `yaml:"resilience,omitempty"`
	FallbackChain      []string          `yaml:"fallback_chain,omitempty"`
	ClaudeArgs         []string          `yaml:"claude_args,omitempty"`
	CustomEnv          map[string]string `yaml:"custom_env,omitempty"`
	InheritGlobalArgs  *bool             `yaml:"inherit_global_args,omitempty"`
	InheritGlobalEnv   *bool             `yaml:"inherit_global_env,omitempty"`
}

func (p Profile) ShouldInheritGlobalArgs() bool {
	return p.InheritGlobalArgs == nil || *p.InheritGlobalArgs
}

func (p Profile) ShouldInheritGlobalEnv() bool {
	return p.InheritGlobalEnv == nil || *p.InheritGlobalEnv
}

type Models struct {
	MainModel      string `yaml:"main_model,omitempty"`
	SmallFastModel string `yaml:"small_fast_model,omitempty"`
	HaikuModel     string `yaml:"haiku_model,omitempty"`
	SonnetModel    string `yaml:"sonnet_model,omitempty"`
	OpusModel      string `yaml:"opus_model,omitempty"`
	SubagentModel  string `yaml:"subagent_model,omitempty"`
}

type Reasoning struct {
	MaxOutputTokens   int    `yaml:"max_output_tokens,omitempty"`
	MaxThinkingTokens int    `yaml:"max_thinking_tokens,omitempty"`
	EffortLevel       string `yaml:"effort_level,omitempty"`
}

type Proxy struct {
	HTTPProxy  string `yaml:"http_proxy,omitempty"`
	HTTPSProxy string `yaml:"https_proxy,omitempty"`
	NoProxy    string `yaml:"no_proxy,omitempty"`
}

type Resilience struct {
	MaxConcurrentRequests int         `yaml:"max_concurrent_requests,omitempty"`
	QueueTimeoutMS        int         `yaml:"queue_timeout_ms,omitempty"`
	Retry                 RetryConfig `yaml:"retry,omitempty"`
}

type RetryConfig struct {
	MaxRetries     int   `yaml:"max_retries,omitempty"`
	BaseDelayMS    int   `yaml:"base_delay_ms,omitempty"`
	MaxDelayMS     int   `yaml:"max_delay_ms,omitempty"`
	Retry5xx       *bool `yaml:"retry_5xx,omitempty"`
	RetryTransport *bool `yaml:"retry_transport,omitempty"`
	Retry429       *bool `yaml:"retry_429,omitempty"`
}

func (r RetryConfig) ShouldRetry5xx() bool {
	return r.Retry5xx == nil || *r.Retry5xx
}

func (r RetryConfig) ShouldRetryTransport() bool {
	return r.RetryTransport == nil || *r.RetryTransport
}

func (r RetryConfig) ShouldRetry429() bool {
	return r.Retry429 != nil && *r.Retry429
}

const profileIDBytes = 5

// GenerateProfileID returns a random 10-char hex string.
// crypto/rand.Read is guaranteed to succeed on all supported platforms;
// the panic is a theoretical safety net that should never trigger.
func GenerateProfileID() string {
	b := make([]byte, profileIDBytes)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random id: %v", err))
	}
	return hex.EncodeToString(b)
}

func EnsureNative(cfg *Config) bool {
	for _, p := range cfg.Profiles {
		if p.ID == NativeProfileID {
			return false
		}
	}
	native := Profile{
		ID:       NativeProfileID,
		Name:     NativeProfileName,
		Protocol: ProtocolNative,
	}
	cfg.Profiles = append([]Profile{native}, cfg.Profiles...)
	return true
}

func FindProfile(cfg *Config, idOrName string) (Profile, bool) {
	for _, p := range cfg.Profiles {
		if p.ID == idOrName || p.Name == idOrName {
			return p, true
		}
	}
	return Profile{}, false
}

func CloneProfile(p Profile) Profile {
	dup := p
	if p.CustomHeaders != nil {
		dup.CustomHeaders = make(map[string]string, len(p.CustomHeaders))
		maps.Copy(dup.CustomHeaders, p.CustomHeaders)
	}
	if p.ExtraBody != nil {
		dup.ExtraBody = nil // clear shallow copy before attempting deep copy
		data, err := json.Marshal(p.ExtraBody)
		if err == nil {
			var m map[string]any
			if err := json.Unmarshal(data, &m); err == nil {
				dup.ExtraBody = m
			}
		}
	}
	if p.FallbackChain != nil {
		dup.FallbackChain = make([]string, len(p.FallbackChain))
		copy(dup.FallbackChain, p.FallbackChain)
	}
	if p.ClaudeArgs != nil {
		dup.ClaudeArgs = make([]string, len(p.ClaudeArgs))
		copy(dup.ClaudeArgs, p.ClaudeArgs)
	}
	if p.CustomEnv != nil {
		dup.CustomEnv = make(map[string]string, len(p.CustomEnv))
		maps.Copy(dup.CustomEnv, p.CustomEnv)
	}
	if p.InheritGlobalArgs != nil {
		v := *p.InheritGlobalArgs
		dup.InheritGlobalArgs = &v
	}
	if p.InheritGlobalEnv != nil {
		v := *p.InheritGlobalEnv
		dup.InheritGlobalEnv = &v
	}
	dup.Resilience.Retry.Retry5xx = cloneBoolPtr(p.Resilience.Retry.Retry5xx)
	dup.Resilience.Retry.RetryTransport = cloneBoolPtr(p.Resilience.Retry.RetryTransport)
	dup.Resilience.Retry.Retry429 = cloneBoolPtr(p.Resilience.Retry.Retry429)
	return dup
}

func cloneBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	dup := *v
	return &dup
}

func ValidateConfig(cfg *Config) []error {
	profileIDs := make(map[string]bool, len(cfg.Profiles))
	for _, p := range cfg.Profiles {
		profileIDs[p.ID] = true
	}

	var errs []error
	for _, p := range cfg.Profiles {
		if p.Protocol != ProtocolNative {
			if p.Protocol == "" {
				errs = append(errs, fmt.Errorf("profile %q: protocol is required", p.Name))
			}
			if p.BaseURL == "" {
				errs = append(errs, fmt.Errorf("profile %q: base_url is required", p.Name))
			}
			if p.APIKey == "" {
				errs = append(errs, fmt.Errorf("profile %q: api_key is required", p.Name))
			}
		}
		for _, fbID := range p.FallbackChain {
			if fbID == NativeProfileID {
				errs = append(errs, fmt.Errorf("profile %q: native profile cannot be in fallback chain", p.Name))
			}
			if !profileIDs[fbID] {
				errs = append(errs, fmt.Errorf("profile %q: fallback references unknown profile %q", p.Name, fbID))
			}
		}
		if p.Resilience.MaxConcurrentRequests < 0 {
			errs = append(errs, fmt.Errorf("profile %q: resilience.max_concurrent_requests must be >= 0", p.Name))
		}
		if p.Resilience.QueueTimeoutMS < 0 {
			errs = append(errs, fmt.Errorf("profile %q: resilience.queue_timeout_ms must be >= 0", p.Name))
		}
		if p.Resilience.Retry.MaxRetries < 0 {
			errs = append(errs, fmt.Errorf("profile %q: resilience.retry.max_retries must be >= 0", p.Name))
		}
		if p.Resilience.Retry.BaseDelayMS < 0 {
			errs = append(errs, fmt.Errorf("profile %q: resilience.retry.base_delay_ms must be >= 0", p.Name))
		}
		if p.Resilience.Retry.MaxDelayMS < 0 {
			errs = append(errs, fmt.Errorf("profile %q: resilience.retry.max_delay_ms must be >= 0", p.Name))
		}
	}
	return errs
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".gocc"), nil
}

func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadConfigFrom(path)
}

func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveConfigTo(cfg, path)
}

func SaveConfigTo(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
