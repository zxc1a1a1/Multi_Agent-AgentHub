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

type selectedAgentNamesContextKey struct{}

// WithSelectedAgentNames stores optional selectedAgentNames into context.
func WithSelectedAgentNames(ctx context.Context, names []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if names == nil {
		names = []string{}
	}
	return context.WithValue(ctx, selectedAgentNamesContextKey{}, names)
}

// SelectedAgentNamesFromContext loads optional selectedAgentNames from context.
func SelectedAgentNamesFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	raw := ctx.Value(selectedAgentNamesContextKey{})
	names, ok := raw.([]string)
	if !ok {
		return nil
	}
	return names
}

type mentionsContextKey struct{}

// WithMentions stores optional mentions into context.
func WithMentions(ctx context.Context, mentions []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if mentions == nil {
		mentions = []string{}
	}
	return context.WithValue(ctx, mentionsContextKey{}, mentions)
}

// MentionsFromContext loads optional mentions from context.
func MentionsFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	raw := ctx.Value(mentionsContextKey{})
	mentions, ok := raw.([]string)
	if !ok {
		return nil
	}
	return mentions
}

type planningModeContextKey struct{}

// PlanningMode is the user-requested orchestration routing mode.
// Valid values: "auto", "direct", "manual", "mention".
// This is distinct from the server-side PlannerMode (rule/llm/llm_with_rule_fallback).
type PlanningMode string

const (
	PlanningModeAuto    PlanningMode = "auto"
	PlanningModeDirect  PlanningMode = "direct"
	PlanningModeManual  PlanningMode = "manual"
	PlanningModeMention PlanningMode = "mention"
)

// WithPlanningMode stores the user-requested planning mode into context.
func WithPlanningMode(ctx context.Context, mode PlanningMode) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, planningModeContextKey{}, mode)
}

// PlanningModeFromContext loads the user-requested planning mode from context.
// Returns empty string if none was set.
func PlanningModeFromContext(ctx context.Context) PlanningMode {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value(planningModeContextKey{})
	mode, ok := raw.(PlanningMode)
	if !ok {
		return ""
	}
	return mode
}

type requestedPathContextKey struct{}

// WithRequestedPath stores the frontend-requested path into context.
func WithRequestedPath(ctx context.Context, path string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestedPathContextKey{}, path)
}

// RequestedPathFromContext loads the frontend-requested path from context.
func RequestedPathFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value(requestedPathContextKey{})
	path, ok := raw.(string)
	if !ok {
		return ""
	}
	return path
}

type executionPathContextKey struct{}

// WithExecutionPath stores the derived executionPath into context.
func WithExecutionPath(ctx context.Context, path string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, executionPathContextKey{}, path)
}

// ExecutionPathFromContext loads the derived executionPath from context.
func ExecutionPathFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value(executionPathContextKey{})
	path, ok := raw.(string)
	if !ok {
		return ""
	}
	return path
}

type replyToContextKey struct{}

// WithReplyTo stores reply-to metadata into context.
func WithReplyTo(ctx context.Context, replyTo map[string]any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, replyToContextKey{}, replyTo)
}

// ReplyToFromContext loads reply-to metadata from context.
func ReplyToFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	raw := ctx.Value(replyToContextKey{})
	val, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return val
}

type quoteContextKey struct{}

// WithQuote stores text quote metadata into context.
func WithQuote(ctx context.Context, quote map[string]any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, quoteContextKey{}, quote)
}

// QuoteFromContext loads text quote metadata from context.
func QuoteFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	raw := ctx.Value(quoteContextKey{})
	val, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return val
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

type pinnedMessageIDsContextKey struct{}

// WithPinnedMessageIDs stores pinned message IDs so Gateway can forward them to Orchestrator.
func WithPinnedMessageIDs(ctx context.Context, ids []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	cleaned := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		cleaned = append(cleaned, id)
	}
	return context.WithValue(ctx, pinnedMessageIDsContextKey{}, cleaned)
}

// PinnedMessageIDsFromContext loads pinned message IDs.
func PinnedMessageIDsFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	ids, ok := ctx.Value(pinnedMessageIDsContextKey{}).([]string)
	if !ok {
		return nil
	}
	return append([]string(nil), ids...)
}

// ContextMessage is a lightweight message snapshot forwarded from the frontend
// through Gateway to Orchestrator so pinned message IDs can be resolved.
type ContextMessage struct {
	ID   string `json:"id,omitempty"`
	Role string `json:"role"`
	Text string `json:"text"`
}

type contextMessagesContextKey struct{}

// WithContextMessages stores the conversation context snapshot so Gateway can
// forward it to Orchestrator for pinned message resolution.
func WithContextMessages(ctx context.Context, msgs []ContextMessage) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextMessagesContextKey{}, msgs)
}

// ContextMessagesFromContext loads the conversation context snapshot.
func ContextMessagesFromContext(ctx context.Context) []ContextMessage {
	if ctx == nil {
		return nil
	}
	msgs, ok := ctx.Value(contextMessagesContextKey{}).([]ContextMessage)
	if !ok {
		return nil
	}
	return msgs
}
