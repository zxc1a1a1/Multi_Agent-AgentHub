// Package dispatcher provides A2A remote agent calling for the Orchestrator.
package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// DispatchInput carries everything needed to call a remote agent.
type DispatchInput struct {
	AgentURL       string
	AgentName      string
	ConversationID string
	RunID          string
	Message        string
}

// DispatchResult carries the response from a remote agent.
type DispatchResult struct {
	Text string
}

// A2ADispatcher is a minimal A2A dispatcher that calls one remote agent.
type A2ADispatcher struct {
	client *a2a.Client
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

	req := a2a.RunRequest{
		SessionID: sessionID,
		Message: a2a.Message{
			Role:    "user",
			Content: msg,
		},
	}

	resp, err := d.client.SendJSONRPC(ctx, url, req)
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
