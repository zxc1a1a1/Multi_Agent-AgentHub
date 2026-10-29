package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
)

// ContextSource adapts SQL sessions/snapshots to the harness context assembler.
type ContextSource struct {
	service   *SQLSessionService
	snapshots *SnapshotStore
}

func NewContextSource(service *SQLSessionService, snapshots *SnapshotStore) (*ContextSource, error) {
	if service == nil {
		return nil, fmt.Errorf("session service is required")
	}
	return &ContextSource{service: service, snapshots: snapshots}, nil
}

func (s *ContextSource) Transcript(ctx context.Context, sessionID string) ([]harness.ContextItem, error) {
	sess, err := s.service.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var items []harness.ContextItem
	for _, event := range sess.Events {
		if event.Content == nil {
			continue
		}
		role := string(event.Content.Role)
		for _, part := range event.Content.Parts {
			switch p := part.(type) {
			case adk.TextPart:
				if p.Text == "" {
					continue
				}
				items = append(items, harness.ContextItem{Role: role, Text: p.Text})
			case adk.ToolCallPart:
				items = append(items, harness.ContextItem{
					Role: role, Text: "[tool call] " + p.Name, ToolCallID: p.ID, ToolName: p.Name,
				})
			case adk.ToolResultPart:
				items = append(items, harness.ContextItem{
					Role: "tool", Text: p.Content, ToolCallID: p.CallID, ToolName: p.Name, ToolResult: true,
				})
			case adk.ThinkingPart:
				// Hidden reasoning is never re-injected.
				continue
			}
		}
	}
	return items, nil
}

func (s *ContextSource) LatestSnapshot(ctx context.Context, sessionID, agentName string) (string, int64, error) {
	if s.snapshots == nil {
		return "", 0, nil
	}
	snap, err := s.snapshots.Latest(ctx, sessionID, agentName)
	if err != nil {
		if errors.Is(err, ErrSnapshotNotFound) {
			return "", 0, nil
		}
		return "", 0, err
	}
	return snap.Summary, snap.SourceEventTo, nil
}
