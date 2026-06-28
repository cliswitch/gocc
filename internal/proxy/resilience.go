package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cliswitch/gocc/internal/config"
	"github.com/llmapimux/llmapimux"
)

const (
	defaultRetryBaseDelay = 250 * time.Millisecond
	defaultRetryMaxDelay  = 5 * time.Second
)

var errQueueTimeout = errors.New("resilience queue timeout")

type endpointPolicy struct {
	key           string
	maxConcurrent int
	queueTimeout  time.Duration
	retry         retryPolicy
}

type retryPolicy struct {
	maxRetries     int
	baseDelay      time.Duration
	maxDelay       time.Duration
	retry5xx       bool
	retryTransport bool
	retry429       bool
}

type endpointLimiter struct {
	tokens chan struct{}
}

type resilienceController struct {
	mu       sync.Mutex
	policies map[string]endpointPolicy
	limiters map[string]*endpointLimiter
	timeNow  func() time.Time
}

type limiterPermit struct {
	limiter *endpointLimiter
	once    sync.Once
}

func (p *limiterPermit) Release() {
	p.once.Do(func() {
		<-p.limiter.tokens
	})
}

func buildResilienceController(profiles []config.Profile) llmapimux.AttemptController {
	policies := make(map[string]endpointPolicy)
	for _, p := range profiles {
		policy, ok := profileEndpointPolicy(p)
		if !ok {
			continue
		}
		key := profileEndpointKey(p)
		if _, exists := policies[key]; !exists {
			policy.key = key
			policies[key] = policy
		}
	}
	if len(policies) == 0 {
		return nil
	}
	return &resilienceController{
		policies: policies,
		limiters: make(map[string]*endpointLimiter),
		timeNow:  time.Now,
	}
}

func profileEndpointPolicy(p config.Profile) (endpointPolicy, bool) {
	r := p.Resilience
	retry := retryPolicyFromConfig(r.Retry)
	policy := endpointPolicy{
		maxConcurrent: r.MaxConcurrentRequests,
		queueTimeout:  time.Duration(r.QueueTimeoutMS) * time.Millisecond,
		retry:         retry,
	}
	return policy, r.MaxConcurrentRequests > 0 || retry.maxRetries > 0
}

func retryPolicyFromConfig(r config.RetryConfig) retryPolicy {
	if r.MaxRetries <= 0 {
		return retryPolicy{}
	}
	baseDelay := time.Duration(r.BaseDelayMS) * time.Millisecond
	if baseDelay <= 0 {
		baseDelay = defaultRetryBaseDelay
	}
	maxDelay := time.Duration(r.MaxDelayMS) * time.Millisecond
	if maxDelay <= 0 {
		maxDelay = defaultRetryMaxDelay
	}
	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}
	return retryPolicy{
		maxRetries:     r.MaxRetries,
		baseDelay:      baseDelay,
		maxDelay:       maxDelay,
		retry5xx:       r.ShouldRetry5xx(),
		retryTransport: r.ShouldRetryTransport(),
		retry429:       r.ShouldRetry429(),
	}
}

func (c *resilienceController) Acquire(ctx context.Context, info llmapimux.RouteInfo, target llmapimux.RouteResult, routeAttempt int, retryAttempt int) (llmapimux.AttemptAdmission, error) {
	policy, ok := c.policyForTarget(target)
	if !ok || policy.maxConcurrent <= 0 {
		return llmapimux.AttemptAdmission{}, nil
	}

	limiter := c.limiterForPolicy(policy)
	waitCtx := ctx
	var cancel context.CancelFunc
	if policy.queueTimeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, policy.queueTimeout)
		defer cancel()
	}

	start := c.now()
	select {
	case limiter.tokens <- struct{}{}:
		return llmapimux.AttemptAdmission{
			Permit:       &limiterPermit{limiter: limiter},
			WaitDuration: c.now().Sub(start),
			LimitKey:     policy.key,
			Limit:        policy.maxConcurrent,
			Active:       len(limiter.tokens),
		}, nil
	case <-waitCtx.Done():
		err := waitCtx.Err()
		if policy.queueTimeout > 0 && errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			err = errQueueTimeout
		}
		return llmapimux.AttemptAdmission{
			WaitDuration: c.now().Sub(start),
			LimitKey:     policy.key,
			Limit:        policy.maxConcurrent,
			Active:       len(limiter.tokens),
		}, err
	}
}

func (c *resilienceController) RetryDelay(ctx context.Context, info llmapimux.RouteInfo, target llmapimux.RouteResult, sendErr llmapimux.SendError, routeAttempt int, retryAttempt int) (time.Duration, bool) {
	if ctx.Err() != nil {
		return 0, false
	}
	policy, ok := c.policyForTarget(target)
	if !ok || policy.retry.maxRetries <= 0 || retryAttempt >= policy.retry.maxRetries {
		return 0, false
	}
	if !policy.retry.shouldRetry(sendErr) {
		return 0, false
	}
	if sendErr.StatusCode == http.StatusTooManyRequests {
		if delay, ok := retryAfterDelay(sendErr.Header, c.now()); ok {
			return delay, true
		}
	}
	return policy.retry.backoffDelay(retryAttempt), true
}

func (c *resilienceController) policyForTarget(target llmapimux.RouteResult) (endpointPolicy, bool) {
	policy, ok := c.policies[routeEndpointKey(target)]
	return policy, ok
}

func (c *resilienceController) limiterForPolicy(policy endpointPolicy) *endpointLimiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	limiter, ok := c.limiters[policy.key]
	if ok {
		return limiter
	}
	limiter = &endpointLimiter{tokens: make(chan struct{}, policy.maxConcurrent)}
	c.limiters[policy.key] = limiter
	return limiter
}

func (c *resilienceController) now() time.Time {
	if c.timeNow != nil {
		return c.timeNow()
	}
	return time.Now()
}

func (p retryPolicy) shouldRetry(sendErr llmapimux.SendError) bool {
	if sendErr.StatusCode == http.StatusTooManyRequests {
		return p.retry429
	}
	if sendErr.StatusCode >= 500 && p.retry5xx {
		return true
	}
	if p.retryTransport && (sendErr.IsTimeout || sendErr.IsConnError || sendErr.StatusCode == 0) {
		return true
	}
	return false
}

func (p retryPolicy) backoffDelay(retryAttempt int) time.Duration {
	delay := p.baseDelay
	for i := 0; i < retryAttempt; i++ {
		delay *= 2
		if delay >= p.maxDelay {
			return p.maxDelay
		}
	}
	return delay
}

func retryAfterDelay(header http.Header, now time.Time) (time.Duration, bool) {
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	t, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	delay := t.Sub(now)
	if delay < 0 {
		delay = 0
	}
	return delay, true
}

func profileEndpointKey(p config.Profile) string {
	return endpointKeyFor(protocolToLLM(p.Protocol), p.BaseURL, p.APIKey)
}

func routeEndpointKey(rr llmapimux.RouteResult) string {
	return endpointKeyFor(rr.Protocol, rr.BaseURL, rr.APIKey)
}

func endpointKeyFor(protocol llmapimux.Protocol, baseURL, apiKey string) string {
	return strings.Join([]string{
		string(protocol),
		normalizeBaseURL(baseURL),
		"key=" + apiKeyDigest(apiKey),
	}, "|")
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return strings.TrimRight(u.String(), "/")
}

func apiKeyDigest(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])[:12]
}
