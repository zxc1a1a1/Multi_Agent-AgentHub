package dispatcher

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// retryPolicy configures pre-stream retry behavior.
type retryPolicy struct {
	maxAttempts int           // total attempts; 1 means no retry
	baseBackoff time.Duration // base backoff before exponential growth + jitter
}

// WithRetry enables retrying idempotent, pre-stream failures (connection
// errors, timeouts, HTTP 5xx). maxAttempts is the total number of attempts
// (1 = no retry). 4xx and remote validation errors are never retried.
func WithRetry(maxAttempts int, baseBackoff time.Duration) Option {
	return func(d *A2ADispatcher) {
		if d == nil || maxAttempts < 1 {
			return
		}
		d.retry = &retryPolicy{maxAttempts: maxAttempts, baseBackoff: baseBackoff}
	}
}

// WithPerCallTimeout sets a default per-call timeout. DispatchInput.TimeoutMs
// (when > 0) overrides this per call.
func WithPerCallTimeout(d time.Duration) Option {
	return func(disp *A2ADispatcher) {
		if disp == nil || d <= 0 {
			return
		}
		disp.defaultTimeout = d
	}
}

// WithCircuitBreaker enables a per-agent-URL circuit breaker.
func WithCircuitBreaker(failThreshold int, cooldown time.Duration) Option {
	return func(d *A2ADispatcher) {
		if d == nil || failThreshold < 1 || cooldown <= 0 {
			return
		}
		d.breaker = newCircuitBreaker(failThreshold, cooldown)
	}
}

// isRetryable reports whether err represents an idempotent, pre-stream failure
// that is safe to retry. 4xx and remote validation errors are not retryable.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var se *a2a.StatusError
	if errors.As(err, &se) {
		return se.StatusCode >= http.StatusInternalServerError
	}
	// Transport-level errors (connection refused, reset) surface as opaque
	// errors from the client; treat them as retryable.
	return true
}

// resolveTimeout picks the effective per-call timeout.
func (d *A2ADispatcher) resolveTimeout(input DispatchInput) time.Duration {
	if input.TimeoutMs > 0 {
		return time.Duration(input.TimeoutMs) * time.Millisecond
	}
	return d.defaultTimeout
}

// backoffFor returns the backoff duration for a given zero-based attempt index.
func (p *retryPolicy) backoffFor(attempt int) time.Duration {
	if p == nil || p.baseBackoff <= 0 {
		return 0
	}
	d := p.baseBackoff << attempt // exponential
	// Add up to 25% jitter.
	jitter := time.Duration(rand.Int63n(int64(d)/4 + 1))
	return d + jitter
}

// circuitBreaker is a minimal per-key consecutive-failure breaker.
type circuitBreaker struct {
	threshold int
	cooldown  time.Duration

	mu     sync.Mutex
	states map[string]*breakerState
}

type breakerState struct {
	failures int
	openedAt time.Time
}

func newCircuitBreaker(threshold int, cooldown time.Duration) *circuitBreaker {
	return &circuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
		states:    make(map[string]*breakerState),
	}
}

// allow reports whether a call to key may proceed (breaker closed/half-open).
func (b *circuitBreaker) allow(key string) bool {
	if b == nil {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.states[key]
	if st == nil || st.failures < b.threshold {
		return true
	}
	// Breaker is open; allow a half-open probe after cooldown.
	if time.Since(st.openedAt) >= b.cooldown {
		return true
	}
	return false
}

func (b *circuitBreaker) recordSuccess(key string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.states, key)
}

func (b *circuitBreaker) recordFailure(key string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.states[key]
	if st == nil {
		st = &breakerState{}
		b.states[key] = st
	}
	st.failures++
	if st.failures >= b.threshold {
		st.openedAt = time.Now()
	}
}
