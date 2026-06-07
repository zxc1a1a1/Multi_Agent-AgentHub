// Package dispatcher provides A2A remote agent calling for the Orchestrator.
package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// DispatchInput carries everything needed to call a remote agent.
type DispatchInput struct {
	AgentURL       string
	AgentName      string
	ConversationID string
	RunID          string
	Message        string
	TraceID        string
	// TimeoutMs is the per-call timeout in milliseconds. Zero means use the
	// dispatcher's configured default.
	TimeoutMs int64
}

// DispatchResult carries the response from a remote agent.
type DispatchResult struct {
	Text string
}

// DispatchChunk is one streamed text chunk from DispatchStream. When Err != nil
// the stream is finished with an error and no further chunks follow.
type DispatchChunk struct {
	Text string
	Err  error
}

// A2ADispatcher is a minimal A2A dispatcher that calls one remote agent.
type A2ADispatcher struct {
	client *a2a.Client

	// Resilience policies (all opt-in; nil/zero means disabled).
	retry          *retryPolicy
	defaultTimeout time.Duration
	breaker        *circuitBreaker
}

// Option customizes A2ADispatcher.
type Option func(*A2ADispatcher)

// WithClient injects a custom A2A client.
func WithClient(client *a2a.Client) Option {
	return func(d *A2ADispatcher) {
		if d == nil || client == nil {
			return
		}
		d.client = client
	}
}

// NewA2ADispatcher creates a minimal A2A dispatcher.
func NewA2ADispatcher(opts ...Option) *A2ADispatcher {
	d := &A2ADispatcher{
		client: a2a.NewClient(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(d)
	}
	if d.client == nil {
		d.client = a2a.NewClient()
	}
	return d
}

// Dispatch sends a task to the configured remote agent and returns the aggregated text.
// It applies the configured resilience policies: circuit breaker, per-call
// timeout, and retry on idempotent pre-stream failures.
func (d *A2ADispatcher) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
	if d == nil {
		return nil, errors.New("dispatcher is nil")
	}

	url := strings.TrimSpace(input.AgentURL)
	if url == "" {
		return nil, errors.New("agent url is required")
	}
	sessionID := strings.TrimSpace(input.ConversationID)
	if sessionID == "" {
		return nil, errors.New("conversationID is required")
	}
	msg := strings.TrimSpace(input.Message)
	if msg == "" {
		return nil, errors.New("message is required")
	}

	if !d.breaker.allow(url) {
		return nil, errors.New("agent dispatch failed: circuit open")
	}

	req := a2a.RunRequest{
		SessionID: sessionID,
		Message: a2a.Message{
			Role:    "user",
			Content: msg,
		},
		TraceID: input.TraceID,
	}

	attempts := 1
	if d.retry != nil {
		attempts = d.retry.maxAttempts
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		res, err := d.dispatchOnce(ctx, url, req, input)
		if err == nil {
			d.breaker.recordSuccess(url)
			return res, nil
		}
		lastErr = err
		// Do not retry on non-retryable errors or once attempts are exhausted.
		if !isRetryable(err) || attempt == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			d.breaker.recordFailure(url)
			return nil, ctx.Err()
		case <-time.After(d.retry.backoffFor(attempt)):
		}
	}

	d.breaker.recordFailure(url)
	return nil, lastErr
}

// dispatchOnce performs a single buffered A2A call with the resolved timeout.
func (d *A2ADispatcher) dispatchOnce(ctx context.Context, url string, req a2a.RunRequest, input DispatchInput) (*DispatchResult, error) {
	callCtx := ctx
	if timeout := d.resolveTimeout(input); timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	resp, err := d.client.SendJSONRPC(callCtx, url, req)
	if err != nil {
		return nil, fmt.Errorf("agent dispatch failed: %w", err)
	}
	if resp == nil {
		return nil, errors.New("agent returned nil response")
	}

	var texts []string
	for _, event := range resp.Events {
		for _, part := range event.Parts {
			if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
				texts = append(texts, part.Text)
			}
		}
	}

	return &DispatchResult{
		Text: strings.Join(texts, "\n"),
	}, nil
}

// DispatchStream sends a task to the configured remote agent and yields text
// chunks as they arrive. Validation failures and transport errors surface as a
// terminal DispatchChunk{Err: ...}. It reuses the A2A client's streaming path
// (SSE with buffered-JSON fallback).
func (d *A2ADispatcher) DispatchStream(ctx context.Context, input DispatchInput) func(yield func(DispatchChunk) bool) {
	return func(yield func(DispatchChunk) bool) {
		if d == nil {
			yield(DispatchChunk{Err: errors.New("dispatcher is nil")})
			return
		}

		url := strings.TrimSpace(input.AgentURL)
		if url == "" {
			yield(DispatchChunk{Err: errors.New("agent url is required")})
			return
		}
		sessionID := strings.TrimSpace(input.ConversationID)
		if sessionID == "" {
			yield(DispatchChunk{Err: errors.New("conversationID is required")})
			return
		}
		msg := strings.TrimSpace(input.Message)
		if msg == "" {
			yield(DispatchChunk{Err: errors.New("message is required")})
			return
		}

		req := a2a.RunRequest{
			SessionID: sessionID,
			Message: a2a.Message{
				Role:    "user",
				Content: msg,
			},
			TraceID: input.TraceID,
		}

		// Streaming uses the per-call timeout but does not retry: once chunks
		// begin flowing, a retry would duplicate user-visible output.
		callCtx := ctx
		if timeout := d.resolveTimeout(input); timeout > 0 {
			var cancel context.CancelFunc
			callCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		d.client.SendJSONRPCStream(callCtx, url, req)(func(c a2a.StreamChunk) bool {
			if c.Err != nil {
				return yield(DispatchChunk{Err: fmt.Errorf("agent dispatch failed: %w", c.Err)})
			}
			var sb strings.Builder
			for _, part := range c.Event.Parts {
				if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(part.Text)
				}
			}
			if sb.Len() == 0 {
				return true // non-text event (e.g. tool_call); skip without emitting
			}
			return yield(DispatchChunk{Text: sb.String()})
		})
	}
}
