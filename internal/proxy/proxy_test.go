package proxy

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cliswitch/gocc/internal/config"
	"github.com/llmapimux/llmapimux"
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
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-primary",
		ExtraBody: map[string]any{
			"service_tier": "priority",
		},
		FallbackChain: []string{"def"},
	}
	fallback := config.Profile{
		ID:       "def",
		Protocol: config.ProtocolAnthropic,
		BaseURL:  "https://api.anthropic.com",
		APIKey:   "sk-fallback",
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
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-primary",
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
		Protocol: llmapimux.ProtocolAnthropic,
		BaseURL:  "https://api.anthropic.com",
		APIKey:   "sk-fallback",
	}
	reqMod(context.Background(), req2, target2)
	if req2.OutboundExtra != nil {
		t.Error("expected nil OutboundExtra for fallback with no extra_body")
	}
}

func TestBuildRequestModifierNoExtraBody(t *testing.T) {
	primary := config.Profile{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-primary",
	}
	allProfiles := map[string]config.Profile{"abc": primary}

	reqMod, err := buildRequestModifier(resolveProfileChain(primary, allProfiles))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqMod == nil {
		t.Fatal("expected non-nil modifier to strip Claude Code billing header")
	}
	req := &llmapimux.Request{
		SystemPrompt: []llmapimux.ContentPart{textPart("x-anthropic-billing-header: cc_version=2.1; cch=abcd;\nYou are Claude Code.")},
	}
	reqMod(context.Background(), req, llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com/",
		APIKey:   "sk-primary",
	})
	if req.OutboundExtra != nil {
		t.Error("expected nil OutboundExtra when no profiles have extra_body")
	}
	if got := req.SystemPrompt[0].Text.Text; got != "You are Claude Code." {
		t.Fatalf("system text = %q, want billing header stripped", got)
	}
}

func TestRouteEndpointKeyNormalizesBaseURLAndDoesNotLeakAPIKey(t *testing.T) {
	rr := llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://API.OpenAI.com/v1/",
		APIKey:   "sk-secret",
	}
	key := routeEndpointKey(rr)
	if strings.Contains(key, "sk-secret") {
		t.Fatalf("endpoint key leaks API key: %s", key)
	}
	if !strings.Contains(key, "openai_chat|https://api.openai.com/v1|key=") {
		t.Fatalf("endpoint key = %q, want protocol + normalized base URL + key digest", key)
	}

	same := routeEndpointKey(llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-secret",
	})
	if key != same {
		t.Fatalf("normalized endpoint keys differ: %q != %q", key, same)
	}

	otherKey := routeEndpointKey(llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-other",
	})
	if key == otherKey {
		t.Fatal("different API keys should produce different endpoint keys")
	}
}

func TestBuildResilienceControllerNilWhenDisabled(t *testing.T) {
	controller := buildResilienceController([]config.Profile{{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
	}})
	if controller != nil {
		t.Fatalf("controller = %#v, want nil when resilience is unset", controller)
	}
}

func TestResilienceControllerAcquireLimitsByEndpoint(t *testing.T) {
	controller := buildResilienceController([]config.Profile{{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
		Resilience: config.Resilience{
			MaxConcurrentRequests: 1,
			QueueTimeoutMS:        10,
		},
	}})
	rc, ok := controller.(*resilienceController)
	if !ok {
		t.Fatalf("controller type = %T, want *resilienceController", controller)
	}

	target := llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com/",
		APIKey:   "sk-test",
	}
	first, err := rc.Acquire(context.Background(), llmapimux.RouteInfo{}, target, 1, 0)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if first.Permit == nil || first.Limit != 1 || first.Active != 1 {
		t.Fatalf("first admission = %+v, want permit limit=1 active=1", first)
	}

	_, err = rc.Acquire(context.Background(), llmapimux.RouteInfo{}, target, 1, 0)
	if !errors.Is(err, errQueueTimeout) {
		t.Fatalf("second acquire err = %v, want errQueueTimeout", err)
	}

	first.Permit.Release()
	third, err := rc.Acquire(context.Background(), llmapimux.RouteInfo{}, target, 1, 0)
	if err != nil {
		t.Fatalf("third acquire after release: %v", err)
	}
	third.Permit.Release()
}

func TestResilienceControllerRetryPolicy(t *testing.T) {
	retry5xx := true
	retryTransport := true
	retry429 := false
	controller := buildResilienceController([]config.Profile{{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
		Resilience: config.Resilience{
			Retry: config.RetryConfig{
				MaxRetries:     2,
				BaseDelayMS:    100,
				MaxDelayMS:     1000,
				Retry5xx:       &retry5xx,
				RetryTransport: &retryTransport,
				Retry429:       &retry429,
			},
		},
	}})
	rc := controller.(*resilienceController)
	target := llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
	}

	delay, ok := rc.RetryDelay(context.Background(), llmapimux.RouteInfo{}, target, llmapimux.SendError{StatusCode: 500}, 1, 0)
	if !ok || delay != 100*time.Millisecond {
		t.Fatalf("500 retry = %v/%v, want true/100ms", ok, delay)
	}
	delay, ok = rc.RetryDelay(context.Background(), llmapimux.RouteInfo{}, target, llmapimux.SendError{IsConnError: true}, 1, 1)
	if !ok || delay != 200*time.Millisecond {
		t.Fatalf("transport retry = %v/%v, want true/200ms", ok, delay)
	}
	if _, ok := rc.RetryDelay(context.Background(), llmapimux.RouteInfo{}, target, llmapimux.SendError{StatusCode: 400}, 1, 0); ok {
		t.Fatal("400 should not retry")
	}
	if _, ok := rc.RetryDelay(context.Background(), llmapimux.RouteInfo{}, target, llmapimux.SendError{StatusCode: 429}, 1, 0); ok {
		t.Fatal("429 should not retry when retry_429=false")
	}
	if _, ok := rc.RetryDelay(context.Background(), llmapimux.RouteInfo{}, target, llmapimux.SendError{StatusCode: 500}, 1, 2); ok {
		t.Fatal("retry attempt at max_retries should not retry")
	}
}

func TestResilienceControllerRetryAfterWhen429Enabled(t *testing.T) {
	retry429 := true
	controller := buildResilienceController([]config.Profile{{
		ID:       "abc",
		Protocol: config.ProtocolOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
		Resilience: config.Resilience{
			Retry: config.RetryConfig{
				MaxRetries: 1,
				Retry429:   &retry429,
			},
		},
	}})
	rc := controller.(*resilienceController)
	target := llmapimux.RouteResult{
		Protocol: llmapimux.ProtocolOpenAIChat,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
	}

	delay, ok := rc.RetryDelay(context.Background(), llmapimux.RouteInfo{}, target, llmapimux.SendError{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"3"}},
	}, 1, 0)
	if !ok || delay != 3*time.Second {
		t.Fatalf("429 Retry-After retry = %v/%v, want true/3s", ok, delay)
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
