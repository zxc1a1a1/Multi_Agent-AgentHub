// Package execution defines the Orchestrator run-lifecycle persistence boundary
// used by the HTTP handler and implemented by the MySQL ledger.
package execution

import (
	"context"
	"time"
)

type Accept struct {
	RunID          string
	RequestID      string
	ConversationID string
	WorkspaceID    string
	PrincipalID    string
	ExecutionPath  string
	BudgetJSON     []byte
}

type State struct {
	RunID          string
	FencingVersion int64
	Status         string
	TerminalReason string
	EventType      string
	Payload        []byte
	BudgetJSON     []byte
}

type Checkpoint struct {
	RunID          string
	FencingVersion int64
	StateJSON      []byte
	BudgetJSON     []byte
}

type Plan struct {
	RunID    string
	PlanID   string
	Revision int
	PlanJSON []byte
	PlanHash string
}

type Store interface {
	AcceptRun(ctx context.Context, p Accept) (deduplicated bool, err error)
	AcquireLease(ctx context.Context, runID, owner string, ttl time.Duration) (fencingVersion int64, err error)
	AppendState(ctx context.Context, s State) (seq int64, err error)
	SaveCheckpoint(ctx context.Context, c Checkpoint) (version int64, err error)
	SavePlan(ctx context.Context, p Plan) error
}
