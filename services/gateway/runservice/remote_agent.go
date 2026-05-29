package runservice

import (
	"context"
	"errors"
	"iter"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

// RemoteAgentRunService is a minimal single-remote-agent adapter for Gateway.
// It is not an orchestrator and does not implement planner/executor behavior.
type RemoteAgentRunService struct {
	agentName string
	agentURL  string
	client    *a2a.Client
}

// Option customizes RemoteAgentRunService.
type Option func(*RemoteAgentRunService)

// WithClient injects a custom A2A client.
func WithClient(client *a2a.Client) Option {
	return func(s *RemoteAgentRunService) {
		if s == nil || client == nil {
			return
		}
		s.client = client
	}
}

// NewRemoteAgentRunService builds a minimal run service backed by one remote A2A agent.
func NewRemoteAgentRunService(agentName, agentURL string, opts ...Option) (*RemoteAgentRunService, error) {
	service := &RemoteAgentRunService{
		agentName: strings.TrimSpace(agentName),
		agentURL:  strings.TrimSpace(agentURL),
	}

	if service.agentName == "" {
		return nil, errors.New("agentName is required")
	}
	if service.agentURL == "" {
		return nil, errors.New("agentURL is required")
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(service)
	}

	if service.client == nil {
		service.client = a2a.NewClient()
	}
	return service, nil
}

// Run executes one user message against the configured remote agent.
func (s *RemoteAgentRunService) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		if s == nil {
			_ = yield(adk.Event{}, errors.New("remote agent run service is nil"))
			return
		}

		sessionID := strings.TrimSpace(conversationID)
		if sessionID == "" {
			_ = yield(adk.Event{}, errors.New("conversationID is required"))
			return
		}
		if userContent == nil {
			_ = yield(adk.Event{}, errors.New("userContent is required"))
			return
		}

		userText, err := extractUserText(userContent)
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}

		if ctx == nil {
			ctx = context.Background()
		}

		remote := a2a.NewRemoteAgent(
			s.agentName,
			s.agentURL,
			a2a.WithClient(s.client),
			a2a.WithSessionID(sessionID),
		)

		resp, err := remote.Generate(ctx, &adk.GenerateRequest{
			Contents: []*adk.Content{
				{
					Role: adk.RoleUser,
					Parts: []adk.Part{
						adk.TextPart{Text: userText},
					},
				},
			},
		})
		if err != nil {
			_ = yield(adk.Event{}, err)
			return
		}
		if resp == nil {
			_ = yield(adk.Event{}, errors.New("remote response is nil"))
			return
		}

		parts := sanitizeAssistantParts(resp.Parts)
		_ = yield(adk.Event{
			Author: s.agentName,
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: parts,
			},
			Final: true,
		}, nil)
	}
}

func extractUserText(content *adk.Content) (string, error) {
	texts := make([]string, 0)
	for _, part := range content.Parts {
		switch value := part.(type) {
		case adk.TextPart:
			trimmed := strings.TrimSpace(value.Text)
			if trimmed == "" {
				continue
			}
			texts = append(texts, trimmed)
		case *adk.TextPart:
			if value == nil {
				continue
			}
			trimmed := strings.TrimSpace(value.Text)
			if trimmed == "" {
				continue
			}
			texts = append(texts, trimmed)
		}
	}
	if len(texts) == 0 {
		return "", errors.New("userContent must include at least one non-empty text part")
	}
	return strings.Join(texts, "\n"), nil
}

func sanitizeAssistantParts(parts []adk.Part) []adk.Part {
	if len(parts) == 0 {
		return nil
	}

	out := make([]adk.Part, 0, len(parts))
	for _, part := range parts {
		switch value := part.(type) {
		case adk.ThinkingPart:
			continue
		case *adk.ThinkingPart:
			continue
		case adk.TextPart:
			if strings.TrimSpace(value.Text) == "" {
				continue
			}
			out = append(out, value)
		case *adk.TextPart:
			if value == nil || strings.TrimSpace(value.Text) == "" {
				continue
			}
			out = append(out, *value)
		default:
			out = append(out, part)
		}
	}
	return out
}
