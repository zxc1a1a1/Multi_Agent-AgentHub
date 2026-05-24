package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"sync"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
)

type clientFactory func(context.Context, []*a2a.AgentInterface) (*a2aclient.Client, error)

// StructuredMessage represents a role-aware conversational turn.
type StructuredMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type structuredMessagesPayload struct {
	Messages []StructuredMessage `json:"messages"`
}

// Client wraps the official a2a-go/v2 client for communicating with child agents.
// Per a2a-agent-contract: uses official A2A protocol for agent communication.
type Client struct {
	mu      sync.RWMutex
	clients map[string]*a2aclient.Client
	create  clientFactory
}

// NewClient creates a new A2A client.
func NewClient() *Client {
	return &Client{
		clients: make(map[string]*a2aclient.Client),
		create:  defaultClientFactory,
	}
}

// SendStreamingMessage sends a message to the agent and returns a streaming iterator of events.
// Uses the official a2a-go/v2 client library for protocol compliance.
func (c *Client) SendStreamingMessage(
	ctx context.Context,
	agentURL string,
	userMessage string,
) (iter.Seq2[a2a.Event, error], error) {
	client, err := c.getOrCreateClient(ctx, agentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent at %s: %w", agentURL, err)
	}

	// Build the A2A message
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(userMessage))
	req := &a2a.SendMessageRequest{
		Message: msg,
	}

	// Send streaming message
	return client.SendStreamingMessage(ctx, req), nil
}

// SendStreamingMessages sends role-aware structured messages by converting them
// into a single A2A text payload that preserves per-message role metadata.
func (c *Client) SendStreamingMessages(
	ctx context.Context,
	agentURL string,
	messages []StructuredMessage,
) (iter.Seq2[a2a.Event, error], error) {
	payload, err := buildStructuredPayload(messages)
	if err != nil {
		return nil, err
	}

	return c.SendStreamingMessage(ctx, agentURL, payload)
}

func buildStructuredPayload(messages []StructuredMessage) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages to send")
	}

	raw, err := json.Marshal(structuredMessagesPayload{Messages: messages})
	if err != nil {
		return "", fmt.Errorf("failed to marshal structured messages: %w", err)
	}

	return string(raw), nil
}

func (c *Client) getOrCreateClient(ctx context.Context, agentURL string) (*a2aclient.Client, error) {
	c.mu.RLock()
	if client, ok := c.clients[agentURL]; ok {
		c.mu.RUnlock()
		return client, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if client, ok := c.clients[agentURL]; ok {
		return client, nil
	}

	endpoints := []*a2a.AgentInterface{
		a2a.NewAgentInterface(agentURL, a2a.TransportProtocolJSONRPC),
	}
	client, err := c.create(ctx, endpoints)
	if err != nil {
		return nil, err
	}

	c.clients[agentURL] = client
	return client, nil
}

func defaultClientFactory(ctx context.Context, endpoints []*a2a.AgentInterface) (*a2aclient.Client, error) {
	return a2aclient.NewFromEndpoints(ctx, endpoints)
}
