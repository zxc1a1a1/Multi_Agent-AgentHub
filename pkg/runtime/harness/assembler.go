package harness

import (
	"context"
	"fmt"
)

// ContextSource supplies durable short-term transcript and snapshot data.
// Implementations live in the runtime session storage adapter.
type ContextSource interface {
	Transcript(ctx context.Context, sessionID string) ([]ContextItem, error)
	LatestSnapshot(ctx context.Context, sessionID, agentName string) (summary string, sourceEventTo int64, err error)
}

// ContextAssembler builds the bounded model context from a session snapshot and
// transcript. It never reads another session's data; the source enforces scope.
type ContextAssembler struct {
	Source ContextSource
	Limits ContextLimits
}

func NewContextAssembler(source ContextSource, limits ContextLimits) (*ContextAssembler, error) {
	if source == nil {
		return nil, fmt.Errorf("context source is required")
	}
	if limits.MaxChars <= 0 {
		limits = DefaultContextLimits()
	}
	return &ContextAssembler{Source: source, Limits: limits}, nil
}

func (a *ContextAssembler) Assemble(ctx context.Context, sessionID, agentName string) ([]ContextItem, error) {
	if sessionID == "" {
		return nil, nil
	}
	items, err := a.Source.Transcript(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	summary, sourceTo, err := a.Source.LatestSnapshot(ctx, sessionID, agentName)
	if err != nil {
		return nil, err
	}
	if summary != "" {
		items = append([]ContextItem{{
			Role:   "system",
			Text:   fmt.Sprintf("Summary of earlier conversation (source events through %d): %s", sourceTo, summary),
			Pinned: true,
		}}, items...)
	}
	return BuildContext(items, a.Limits)
}
