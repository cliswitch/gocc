package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/cliswitch/gocc/internal/config"
	"github.com/llmapimux/llmapimux"
)

func StartProxy(primary config.Profile, allProfiles map[string]config.Profile, token string, statsReporter llmapimux.StatsReporter) (int, func(), error) {
	profiles := resolveProfileChain(primary, allProfiles)
	candidateFn := buildCandidateFunc(profiles)
	router := llmapimux.NewCircuitBreakerRouter(candidateFn)
	auth := &tokenAuthenticator{token: token}

	opts := []llmapimux.MuxOption{llmapimux.WithAuthenticator(auth)}
	if statsReporter != nil {
		opts = append(opts, llmapimux.WithStatsReporter(statsReporter))
	}
	reqMod, err := buildRequestModifier(profiles)
	if err != nil {
		return 0, nil, fmt.Errorf("build request modifier: %w", err)
	}
	opts = append(opts, llmapimux.WithRequestModifier(reqMod))
	mux := llmapimux.NewMux(router, opts...)

	httpMux := http.NewServeMux()
	httpMux.Handle("/", mux.AnthropicHandler())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, fmt.Errorf("listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	server := &http.Server{Handler: httpMux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "gocc: proxy serve error: %v\n", err)
		}
	}()

	shutdown := func() {
		server.Close()
	}

	return port, shutdown, nil
}

func protocolToLLM(proto string) llmapimux.Protocol {
	switch proto {
	case config.ProtocolOpenAI:
		return llmapimux.ProtocolOpenAIChat
	case config.ProtocolResponses:
		return llmapimux.ProtocolOpenAIResponses
	case config.ProtocolAnthropic:
		return llmapimux.ProtocolAnthropic
	case config.ProtocolGemini:
		return llmapimux.ProtocolGemini
	default:
		return llmapimux.ProtocolAnthropic
	}
}

func resolveProfileChain(primary config.Profile, allProfiles map[string]config.Profile) []config.Profile {
	profiles := []config.Profile{primary}
	for _, fbID := range primary.FallbackChain {
		if p, ok := allProfiles[fbID]; ok {
			profiles = append(profiles, p)
		}
	}
	return profiles
}

func buildCandidateFunc(profiles []config.Profile) llmapimux.CandidateFunc {
	primary := profiles[0]

	return func(info llmapimux.RouteInfo) []llmapimux.RouteResult {
		level, _, _ := config.ParseAnnotatedModel(info.Model)
		if level == "" {
			level = primary.Models.LevelForModel(info.Model)
		}

		var results []llmapimux.RouteResult
		for _, p := range profiles {
			model := resolveModel(p, level, info.Model)
			rr := llmapimux.RouteResult{
				Protocol: protocolToLLM(p.Protocol),
				BaseURL:  p.BaseURL,
				APIKey:   p.APIKey,
				Model:    model,
				ProxyURL: proxyURL(p.Proxy),
			}
			if len(p.CustomHeaders) > 0 {
				rr.Header = make(http.Header)
				for k, v := range p.CustomHeaders {
					rr.Header.Set(k, v)
				}
			}
			results = append(results, rr)
		}
		return results
	}
}

func proxyURL(p config.Proxy) string {
	if p.HTTPSProxy != "" {
		return p.HTTPSProxy
	}
	return p.HTTPProxy
}

func resolveModel(p config.Profile, level, rawModel string) string {
	if level != "" {
		if m := p.Models.ModelForLevel(level); m != "" {
			return m
		}
	}
	_, model, ok := config.ParseAnnotatedModel(rawModel)
	if ok {
		return model
	}
	return rawModel
}

func extraBodyToRaw(m map[string]any) (map[string]json.RawMessage, error) {
	if m == nil {
		return nil, nil
	}
	result := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal extra_body key %q: %w", k, err)
		}
		result[k] = raw
	}
	return result, nil
}

type endpointKey struct {
	baseURL string
	apiKey  string
}

// buildRequestModifier produces the single RequestModifier hooked into the
// mux. It always strips Claude Code's per-request billing header on
// non-Anthropic targets (critical for prompt-cache hits), and additionally
// injects any profile-configured ExtraBody keyed by the outbound endpoint.
func buildRequestModifier(profiles []config.Profile) (llmapimux.RequestModifier, error) {
	lookup := make(map[endpointKey]map[string]json.RawMessage, len(profiles))
	for _, p := range profiles {
		if len(p.ExtraBody) == 0 {
			continue
		}
		raw, err := extraBodyToRaw(p.ExtraBody)
		if err != nil {
			return nil, fmt.Errorf("profile %q: %w", p.Name, err)
		}
		key := endpointKey{p.BaseURL, p.APIKey}
		// First profile with this key wins (matches fallback priority order).
		if _, exists := lookup[key]; !exists {
			lookup[key] = raw
		}
	}

	return func(ctx context.Context, req *llmapimux.Request, target llmapimux.RouteResult) {
		stripClaudeCodeBillingHeader(req, target)
		if extra, ok := lookup[endpointKey{target.BaseURL, target.APIKey}]; ok {
			req.OutboundExtra = extra
		}
	}, nil
}

type tokenAuthenticator struct {
	token string
}

func (a *tokenAuthenticator) Authenticate(_ context.Context, apiKey string) error {
	if apiKey != a.token {
		return fmt.Errorf("unauthorized")
	}
	return nil
}
