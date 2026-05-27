package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const defaultRemoteSessionID = "remote-session"

// RemoteAgent wraps a remote A2A endpoint as a local ADK agent.
type RemoteAgent struct {
	name             string
	url              string
	client           *Client
	defaultSessionID string
}

// RemoteAgentOption customizes RemoteAgent behavior.
type RemoteAgentOption func(*RemoteAgent)

// NewRemoteAgent creates a remote-backed ADK agent wrapper.
func NewRemoteAgent(name, url string, opts ...RemoteAgentOption) *RemoteAgent {
	agent := &RemoteAgent{
		name:             strings.TrimSpace(name),
		url:              strings.TrimSpace(url),
		defaultSessionID: defaultRemoteSessionID,
	}
	if agent.name == "" {
		agent.name = "remote-agent"
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(agent)
	}

	if agent.client == nil {
		agent.client = NewClient()
	}
	if strings.TrimSpace(agent.defaultSessionID) == "" {
		agent.defaultSessionID = defaultRemoteSessionID
	}
	return agent
}

// WithClient injects a custom A2A client.
func WithClient(client *Client) RemoteAgentOption {
	return func(agent *RemoteAgent) {
		if agent == nil || client == nil {
			return
		}
		agent.client = client
	}
}

// WithSessionID sets default session id used for remote calls.
func WithSessionID(sessionID string) RemoteAgentOption {
	return func(agent *RemoteAgent) {
		if agent == nil {
			return
		}
		trimmed := strings.TrimSpace(sessionID)
		if trimmed == "" {
			return
		}
		agent.defaultSessionID = trimmed
	}
}

// Name returns the local-facing agent name.
func (r *RemoteAgent) Name() string {
	if r == nil || strings.TrimSpace(r.name) == "" {
		return "remote-agent"
	}
	return r.name
}

// Generate delegates generation to remote A2A server and maps events into ADK response.
func (r *RemoteAgent) Generate(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	if r == nil {
		return nil, errors.New("remote agent is nil")
	}
	if strings.TrimSpace(r.url) == "" {
		return nil, errors.New("remote agent url is required")
	}
	if req == nil {
		return nil, errors.New("generate request is required")
	}

	userText, err := extractUserText(req.Contents)
	if err != nil {
		return nil, err
	}

	client := r.client
	if client == nil {
		client = NewClient()
	}

	runResp, err := client.Send(ctx, r.url, RunRequest{
		SessionID: r.sessionID(),
		Message: Message{
			Role:    "user",
			Content: userText,
		},
	})
	if err != nil {
		return nil, err
	}

	parts := mapRunEventsToADKParts(runResp.Events)
	return &adk.GenerateResponse{
		Parts:        parts,
		FinishReason: inferFinishReason(parts),
	}, nil
}

func (r *RemoteAgent) sessionID() string {
	sessionID := strings.TrimSpace(r.defaultSessionID)
	if sessionID == "" {
		return defaultRemoteSessionID
	}
	return sessionID
}

func extractUserText(contents []*adk.Content) (string, error) {
	for i := len(contents) - 1; i >= 0; i-- {
		content := contents[i]
		if content == nil || content.Role != adk.RoleUser {
			continue
		}
		for _, part := range content.Parts {
			textPart, ok := part.(adk.TextPart)
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(textPart.Text)
			if trimmed != "" {
				return trimmed, nil
			}
		}
	}

	allTexts := make([]string, 0)
	for _, content := range contents {
		if content == nil {
			continue
		}
		for _, part := range content.Parts {
			textPart, ok := part.(adk.TextPart)
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(textPart.Text)
			if trimmed == "" {
				continue
			}
			allTexts = append(allTexts, trimmed)
		}
	}
	if len(allTexts) == 0 {
		return "", errors.New("no text content found in generate request")
	}
	return strings.Join(allTexts, "\n"), nil
}

func mapRunEventsToADKParts(events []EventDTO) []adk.Part {
	selected := selectAssistantParts(events)
	parts := make([]adk.Part, 0, len(selected))

	for _, part := range selected {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "text":
			text := strings.TrimSpace(part.Text)
			if text == "" {
				continue
			}
			parts = append(parts, adk.TextPart{Text: text})
		case "tool_call":
			parts = append(parts, adk.ToolCallPart{
				ID:        strings.TrimSpace(part.ID),
				Name:      strings.TrimSpace(part.Name),
				Arguments: encodeArguments(part.Arguments),
			})
		case "tool_result":
			parts = append(parts, adk.ToolResultPart{
				CallID:  strings.TrimSpace(part.CallID),
				Name:    strings.TrimSpace(part.Name),
				Content: part.Content,
				IsError: part.IsError,
			})
		case "thinking":
			// Never expose remote thinking content.
			continue
		default:
			continue
		}
	}
	return parts
}

func selectAssistantParts(events []EventDTO) []PartDTO {
	for i := len(events) - 1; i >= 0; i-- {
		if !strings.EqualFold(strings.TrimSpace(events[i].Role), string(adk.RoleAssistant)) {
			continue
		}
		if events[i].Final {
			return append([]PartDTO(nil), events[i].Parts...)
		}
	}

	parts := make([]PartDTO, 0)
	for _, event := range events {
		if !strings.EqualFold(strings.TrimSpace(event.Role), string(adk.RoleAssistant)) {
			continue
		}
		parts = append(parts, event.Parts...)
	}
	return parts
}

func encodeArguments(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return nil
	}
	return raw
}

func inferFinishReason(parts []adk.Part) adk.FinishReason {
	hasToolCall := false
	hasText := false

	for _, part := range parts {
		switch value := part.(type) {
		case adk.TextPart:
			if strings.TrimSpace(value.Text) != "" {
				hasText = true
			}
		case adk.ToolCallPart:
			hasToolCall = true
		}
	}

	if hasToolCall && !hasText {
		return adk.FinishToolUse
	}
	return adk.FinishStop
}
