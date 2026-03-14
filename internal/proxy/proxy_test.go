package proxy

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/llmapimux/llmapimux"
	"github.com/cliswitch/gocc/internal/config"
)

func TestBuildCandidates(t *testing.T) {
	primary := config.Profile{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-primary",
		Models: config.Models{
			HaikuModel:  "gpt-4o-mini",
			SonnetModel: "gpt-4o",
			OpusModel:   "gpt-4.1",
		},
		CustomHeaders: map[string]string{"X-Foo": "bar"},
		FallbackChain: []string{"def"},
	}
	fallback := config.Profile{
		ID:       "def",
		Protocol: config.ProtocolGemini,
		BaseURL:  "https://generativelanguage.googleapis.com",
		APIKey:   "AIza-fb",
		Models: config.Models{
			HaikuModel:  "gemini-2.5-flash",
			SonnetModel: "gemini-2.5-pro",
			OpusModel:   "gemini-2.5-ultra",
		},
	}
	allProfiles := map[string]config.Profile{
		"abc": primary,
		"def": fallback,
	}

	fn := buildCandidateFunc(resolveProfileChain(primary, allProfiles))
	info := llmapimux.RouteInfo{Model: "s-gpt-4o"}
	results := fn(info)

	if len(results) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(results))
	}

	r0 := results[0]
	if string(r0.Protocol) != "openai_chat" {
		t.Errorf("r0 protocol = %q", r0.Protocol)
	}
	if r0.Model != "gpt-4o" {
		t.Errorf("r0 model = %q, want gpt-4o", r0.Model)
	}
	if r0.Header.Get("X-Foo") != "bar" {
		t.Errorf("r0 missing custom header")
	}

	r1 := results[1]
	if string(r1.Protocol) != "gemini" {
		t.Errorf("r1 protocol = %q", r1.Protocol)
	}
	if r1.Model != "gemini-2.5-pro" {
		t.Errorf("r1 model = %q, want gemini-2.5-pro", r1.Model)
	}
}

func TestBuildCandidatesNoAnnotation(t *testing.T) {
	primary := config.Profile{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-primary",
		Models: config.Models{
			HaikuModel:  "gpt-4o-mini",
			SonnetModel: "gpt-4o",
			OpusModel:   "gpt-4.1",
		},
		FallbackChain: []string{"def"},
	}
	fallback := config.Profile{
		ID:       "def",
		Protocol: config.ProtocolGemini,
		BaseURL:  "https://generativelanguage.googleapis.com",
		APIKey:   "AIza-fb",
		Models: config.Models{
			HaikuModel:  "gemini-2.5-flash",
			SonnetModel: "gemini-2.5-pro",
			OpusModel:   "gemini-2.5-ultra",
		},
	}
	allProfiles := map[string]config.Profile{
		"abc": primary,
		"def": fallback,
	}

	fn := buildCandidateFunc(resolveProfileChain(primary, allProfiles))
	// Non-annotated model name — proxy should reverse-lookup level from primary.
	info := llmapimux.RouteInfo{Model: "gpt-4o"}
	results := fn(info)

	if len(results) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(results))
	}

	r0 := results[0]
	if r0.Model != "gpt-4o" {
		t.Errorf("r0 model = %q, want gpt-4o", r0.Model)
	}

	r1 := results[1]
	if r1.Model != "gemini-2.5-pro" {
		t.Errorf("r1 model = %q, want gemini-2.5-pro (fallback sonnet)", r1.Model)
	}
}

func TestExtraBodyToRaw(t *testing.T) {
	m := map[string]any{
		"service_tier": "priority",
		"count":        42,
		"flag":         true,
	}
	raw, err := extraBodyToRaw(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw["service_tier"]) != `"priority"` {
		t.Errorf("service_tier = %s, want %q", raw["service_tier"], `"priority"`)
	}
	if string(raw["count"]) != `42` {
		t.Errorf("count = %s, want 42", raw["count"])
	}
	if string(raw["flag"]) != `true` {
		t.Errorf("flag = %s, want true", raw["flag"])
	}
}

func TestExtraBodyToRawNil(t *testing.T) {
	raw, err := extraBodyToRaw(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw != nil {
		t.Errorf("expected nil, got %v", raw)
	}
}

func TestBuildRequestModifier(t *testing.T) {
	primary := config.Profile{
		ID:      "abc",
		BaseURL: "https://api.openai.com",
		APIKey:  "sk-primary",
		ExtraBody: map[string]any{
			"service_tier": "priority",
		},
		FallbackChain: []string{"def"},
	}
	fallback := config.Profile{
		ID:      "def",
		BaseURL: "https://api.anthropic.com",
		APIKey:  "sk-fallback",
		// no extra_body
	}
	allProfiles := map[string]config.Profile{
		"abc": primary,
		"def": fallback,
	}

	reqMod, err := buildRequestModifier(resolveProfileChain(primary, allProfiles))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqMod == nil {
		t.Fatal("expected non-nil modifier")
	}

	// Simulate primary target: should set OutboundExtra
	req := &llmapimux.Request{}
	target := llmapimux.RouteResult{
		BaseURL: "https://api.openai.com",
		APIKey:  "sk-primary",
	}
	reqMod(context.Background(), req, target)
	if req.OutboundExtra == nil {
		t.Fatal("expected OutboundExtra to be set for primary")
	}
	if string(req.OutboundExtra["service_tier"]) != `"priority"` {
		t.Errorf("service_tier = %s", req.OutboundExtra["service_tier"])
	}

	// Simulate fallback target: should NOT set OutboundExtra
	req2 := &llmapimux.Request{}
	target2 := llmapimux.RouteResult{
		BaseURL: "https://api.anthropic.com",
		APIKey:  "sk-fallback",
	}
	reqMod(context.Background(), req2, target2)
	if req2.OutboundExtra != nil {
		t.Error("expected nil OutboundExtra for fallback with no extra_body")
	}
}

func TestBuildRequestModifierNoExtraBody(t *testing.T) {
	primary := config.Profile{
		ID:      "abc",
		BaseURL: "https://api.openai.com",
		APIKey:  "sk-primary",
	}
	allProfiles := map[string]config.Profile{"abc": primary}

	reqMod, err := buildRequestModifier(resolveProfileChain(primary, allProfiles))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqMod != nil {
		t.Error("expected nil modifier when no profiles have extra_body")
	}
}

func TestStartProxyListens(t *testing.T) {
	primary := config.Profile{
		ID:       "test",
		Protocol: config.ProtocolAnthropic,
		BaseURL:  "https://api.anthropic.com",
		APIKey:   "sk-test",
		Models:   config.Models{SonnetModel: "claude-sonnet"},
	}
	allProfiles := map[string]config.Profile{"test": primary}
	token := "test-token-123"

	port, shutdown, err := StartProxy(primary, allProfiles, token, nil)
	if err != nil {
		t.Fatalf("StartProxy: %v", err)
	}
	defer shutdown()

	if port <= 0 {
		t.Errorf("expected positive port, got %d", port)
	}

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/health-not-exist")
	if err != nil {
		t.Fatalf("connect to proxy: %v", err)
	}
	resp.Body.Close()
}
