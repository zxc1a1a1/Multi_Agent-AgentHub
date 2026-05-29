package runservice

import (
	"context"
	"errors"
	"iter"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type agentNameContextKey struct{}

// WithAgentName stores an optional explicit agentName into context.
func WithAgentName(ctx context.Context, agentName string) context.Context {
	name := strings.TrimSpace(agentName)
	if name == "" {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, agentNameContextKey{}, name)
}

// AgentNameFromContext loads an optional explicit agentName from context.
func AgentNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value(agentNameContextKey{})
	name, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

// AgentNameSelector decides a target agentName for one run.
type AgentNameSelector func(ctx context.Context, conversationID string, userContent *adk.Content) string

// RoutingOption customizes RoutingRunService behavior.
type RoutingOption func(*RoutingRunService)

// WithAgentNameSelector sets a selector for explicit routing.
func WithAgentNameSelector(selector AgentNameSelector) RoutingOption {
	return func(s *RoutingRunService) {
		if s == nil || selector == nil {
			return
		}
		s.selector = selector
	}
}

// RoutingRunService routes one run to one static remote agent.
type RoutingRunService struct {
	registry         *StaticAgentRegistry
	defaultAgentName string
	selector         AgentNameSelector
	services         map[string]*RemoteAgentRunService
}

// NewRoutingRunService creates a minimal static-name router.
func NewRoutingRunService(registry *StaticAgentRegistry, defaultAgentName string, opts ...RoutingOption) (*RoutingRunService, error) {
	if registry == nil {
		return nil, errors.New("registry is required")
	}

	defaultName := strings.TrimSpace(defaultAgentName)
	if defaultName == "" {
		return nil, errors.New("defaultAgentName is required")
	}

	if _, ok := registry.Get(defaultName); !ok {
		return nil, errors.New("default agent is not registered: " + defaultName)
	}

	service := &RoutingRunService{
		registry:         registry,
		defaultAgentName: defaultName,
		services:         make(map[string]*RemoteAgentRunService),
	}

	for _, endpoint := range registry.List() {
		remote, err := NewRemoteAgentRunService(endpoint.Name, endpoint.URL)
		if err != nil {
			return nil, err
		}
		service.services[endpoint.Name] = remote
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(service)
	}

	if service.selector == nil {
		service.selector = defaultSelector
	}

	return service, nil
}

// Run executes one user message by selected remote agent.
func (s *RoutingRunService) Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error] {
	return func(yield func(adk.Event, error) bool) {
		if s == nil {
			_ = yield(adk.Event{}, errors.New("routing run service is nil"))
			return
		}

		targetName := s.defaultAgentName
		if s.selector != nil {
			candidate := strings.TrimSpace(s.selector(ctx, conversationID, userContent))
			if candidate != "" {
				targetName = candidate
			}
		}

		remote, ok := s.services[targetName]
		if !ok || remote == nil {
			_ = yield(adk.Event{}, errors.New("requested agent is not available"))
			return
		}

		seq := remote.Run(ctx, conversationID, userContent)
		if seq == nil {
			return
		}
		seq(func(event adk.Event, err error) bool {
			if err != nil {
				return yield(adk.Event{}, err)
			}
			event.Author = targetName
			return yield(event, nil)
		})
	}
}

func defaultSelector(ctx context.Context, _ string, _ *adk.Content) string {
	return AgentNameFromContext(ctx)
}
