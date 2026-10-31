// Package execution defines the Orchestrator run-lifecycle persistence boundary
// used by the HTTP handler and implemented by the MySQL ledger.
package execution

import (
	"context"
	"encoding/json"
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

type Approval struct {
	ApprovalID     string
	RunID          string
	PlanID         string
	PlanRevision   int
	WorkspaceID    string
	PrincipalID    string
	ExpiresAt      time.Time
	IdempotencyKey string
}

type ResumeState struct {
	RunID          string
	Status         string
	FencingVersion int64
	PlanJSON       []byte
	BudgetJSON     []byte
	CheckpointJSON []byte
}

type Plan struct {
	RunID    string
	PlanID   string
	Revision int
	PlanJSON []byte
	PlanHash string
}

type RunStatus struct {
	RunID          string
	Status         string
	CurrentSeq     int64
	FencingVersion int64
	TerminalReason string
}

type Event struct {
	Seq       int64           `json:"seq"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
}

type Store interface {
	AcceptRun(ctx context.Context, p Accept) (deduplicated bool, err error)
	AcquireLease(ctx context.Context, runID, owner string, ttl time.Duration) (fencingVersion int64, err error)
	RenewLease(ctx context.Context, runID, owner string, fencingVersion int64, ttl time.Duration) error
	AppendState(ctx context.Context, s State) (seq int64, err error)
	SaveCheckpoint(ctx context.Context, c Checkpoint) (version int64, err error)
	SavePlan(ctx context.Context, p Plan) error
	CreatePlanApproval(ctx context.Context, a Approval) (approvalID string, deduplicated bool, err error)
	ResolvePlanApproval(ctx context.Context, approvalID, status string) error
	LoadResume(ctx context.Context, runID string) (*ResumeState, error)
	GetRun(ctx context.Context, runID string) (*RunStatus, error)
	ListEventsAfter(ctx context.Context, runID string, afterSeq int64, limit int) ([]Event, error)
}
