package a2a

import (
	"context"
	"fmt"
	"iter"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
)

// Client wraps the official a2a-go/v2 client for communicating with child agents.
// Per a2a-agent-contract: uses official A2A protocol for agent communication.
type Client struct{}

// NewClient creates a new A2A client.
func NewClient() *Client {
	return &Client{}
}

// SendStreamingMessage sends a message to the agent and returns a streaming iterator of events.
// Uses the official a2a-go/v2 client library for protocol compliance.
func (c *Client) SendStreamingMessage(
	ctx context.Context,
	agentURL string,
	userMessage string,
) (iter.Seq2[a2a.Event, error], error) {
	// Create endpoint definition for the agent
	endpoints := []*a2a.AgentInterface{
		a2a.NewAgentInterface(agentURL, a2a.TransportProtocolJSONRPC),
	}

	// Create a2a client from endpoints
	client, err := a2aclient.NewFromEndpoints(ctx, endpoints)
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
