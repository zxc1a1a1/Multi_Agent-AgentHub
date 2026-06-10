// Package dispatcher provides A2A remote agent calling for the Orchestrator.
package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// ArtifactMeta is metadata-only artifact information extracted from A2A
// responses. No binary/file content is stored — only name, kind, mime type,
// size, and provenance.
type ArtifactMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Kind        string `json:"kind,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Size        int64  `json:"size,omitempty"`
	SourceAgent string `json:"sourceAgent,omitempty"`
}

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
	// Mode is the run mode: "" (default=full), "plan_only", or "execute".
	Mode string
}

// DispatchResult carries the response from a remote agent.
type DispatchResult struct {
	Text      string
	TaskID    string
	Artifacts []ArtifactMeta
}

// DispatchChunk is one streamed text chunk from DispatchStream. When Err != nil
// the stream is finished with an error and no further chunks follow.
type DispatchChunk struct {
	// TaskID carries the real remote A2A task id returned by the child agent.
	// It may appear before any text chunk and is metadata-only.
	TaskID   string
	Text     string
	Artifact *ArtifactMeta // non-nil when this chunk carries artifact metadata
	Err      error
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
		Mode:    input.Mode,
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
	var artifacts []ArtifactMeta
	for _, event := range resp.Events {
		for _, part := range event.Parts {
			if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
				texts = append(texts, part.Text)
			}
			if art := extractArtifactFromPart(part, input.AgentName); art != nil {
				artifacts = append(artifacts, *art)
			}
		}
	}

	return &DispatchResult{
		Text:      strings.Join(texts, "\n"),
		TaskID:    strings.TrimSpace(resp.TaskID),
		Artifacts: artifacts,
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
			Mode:    input.Mode,
		}

		// Streaming uses the per-call timeout but does not retry: once chunks
		// begin flowing, a retry would duplicate user-visible output.
		callCtx := ctx
		if timeout := d.resolveTimeout(input); timeout > 0 {
			var cancel context.CancelFunc
			callCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		var sentText strings.Builder
		d.client.SendJSONRPCStream(callCtx, url, req)(func(c a2a.StreamChunk) bool {
			if c.Err != nil {
				return yield(DispatchChunk{Err: fmt.Errorf("agent dispatch failed: %w", c.Err)})
			}
			if taskID := strings.TrimSpace(c.TaskID); taskID != "" {
				if !yield(DispatchChunk{TaskID: taskID}) {
					return false
				}
			}
			var sb strings.Builder
			for _, part := range c.Event.Parts {
				if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(part.Text)
				}
				if art := extractArtifactFromPart(part, input.AgentName); art != nil {
					if !yield(DispatchChunk{Artifact: art}) {
						return false
					}
				}
			}
			if sb.Len() == 0 {
				return true // non-text event (e.g. tool_call); skip without emitting
			}
			// Deduplicate: real agents may send a final consolidated event
			// after progressive streaming deltas. Only yield the suffix not
			// already forwarded so the frontend never sees duplicated text.
			chunkText := sb.String()
			already := sentText.String()
			if already != "" && strings.HasPrefix(chunkText, already) {
				chunkText = strings.TrimPrefix(chunkText, already)
			}
			if chunkText == "" {
				return true
			}
			sentText.WriteString(chunkText)
			return yield(DispatchChunk{Text: chunkText})
		})
	}
}

// extractArtifactFromPart extracts artifact metadata from an A2A part.
// Strict convention: part.Type=="tool_result" AND part.Name=="artifact_metadata"
// AND part.Content is valid JSON with required fields id/artifactId + name.
// Ordinary tool_results (e.g. file_search, code_execution) are never classified
// as artifacts. Returns nil for anything that doesn't match the convention.
func extractArtifactFromPart(part a2a.PartDTO, sourceAgent string) *ArtifactMeta {
	if part.Type != "tool_result" {
		return nil
	}
	if strings.TrimSpace(part.Name) != "artifact_metadata" {
		return nil
	}
	content := strings.TrimSpace(part.Content)
	if content == "" {
		return nil
	}

	var meta struct {
		ID          string `json:"id"`
		ArtifactID  string `json:"artifactId"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		MimeType    string `json:"mimeType"`
		Size        int64  `json:"size"`
		SourceAgent string `json:"sourceAgent"`
	}
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return nil
	}

	id := strings.TrimSpace(meta.ID)
	if id == "" {
		id = strings.TrimSpace(meta.ArtifactID)
	}
	name := strings.TrimSpace(meta.Name)
	if id == "" || name == "" {
		return nil
	}

	kind := strings.TrimSpace(meta.Kind)
	if kind == "" {
		kind = "file"
	}
	mimeType := strings.TrimSpace(meta.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	sa := strings.TrimSpace(meta.SourceAgent)
	if sa == "" {
		sa = sourceAgent
	}

	return &ArtifactMeta{
		ID:          id,
		Name:        name,
		Kind:        kind,
		MimeType:    mimeType,
		Size:        meta.Size,
		SourceAgent: sa,
	}
}
