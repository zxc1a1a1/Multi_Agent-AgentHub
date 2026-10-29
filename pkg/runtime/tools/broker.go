package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var (
	ErrApprovalRequired = errors.New("tool approval required")
	ErrInvalidEnvelope  = errors.New("invalid tool call envelope")
)

// Envelope binds a model proposal to platform-generated execution identity.
type Envelope struct {
	RunID          string
	TaskID         string
	ToolCallID     string
	ToolName       string
	ToolVersion    string
	ArgsHash       string
	WorkspaceID    string
	ApprovalID     string
	FencingVersion int64
	PrincipalID    string
	IdempotencyKey string
}

// Broker authorizes calls. Side effects must not run before Authorize succeeds.
type Broker interface {
	Authorize(ctx context.Context, e Envelope, d Descriptor) error
}

// Guard is the P3a reference Broker: it enforces identity fields and approval
// requirements. It does not replace the SQL ledger in P3b.
type Guard struct{}

func (Guard) Authorize(_ context.Context, e Envelope, d Descriptor) error {
	if e.RunID == "" || e.ToolCallID == "" || e.PrincipalID == "" || e.ArgsHash == "" || e.FencingVersion <= 0 {
		return fmt.Errorf("%w: missing identity", ErrInvalidEnvelope)
	}
	if e.ToolName != d.Name || e.ToolVersion != d.Version {
		return fmt.Errorf("%w: tool name/version mismatch", ErrInvalidEnvelope)
	}
	if !d.ReadOnly && e.ApprovalID == "" {
		return fmt.Errorf("%w: %s", ErrApprovalRequired, d.Name)
	}
	return nil
}

// Execute authorizes and then runs a tool. Write tools cannot execute without
// a successfully authorized envelope (including approval), so an unapproved
// call never reaches the tool implementation.
func Execute(ctx context.Context, broker Broker, e Envelope, d Descriptor, tool adk.Tool, args json.RawMessage) (*adk.ToolResult, error) {
	if tool == nil {
		return nil, fmt.Errorf("tool is nil")
	}
	if broker == nil {
		broker = Guard{}
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if err := broker.Authorize(ctx, e, d); err != nil {
		return nil, err
	}
	return tool.Execute(ctx, args)
}
