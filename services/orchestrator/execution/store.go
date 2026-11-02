// Package execution defines the Orchestrator run-lifecycle persistence boundary
// used by the HTTP handler and implemented by the MySQL ledger.
package execution

import (
	"errors"
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
	ReleaseLease(ctx context.Context, runID, owner string, fencingVersion int64) error
	AppendState(ctx context.Context, s State) (seq int64, err error)
	SaveCheckpoint(ctx context.Context, c Checkpoint) (version int64, err error)
	SavePlan(ctx context.Context, p Plan) error
	CreatePlanApproval(ctx context.Context, a Approval) (approvalID string, deduplicated bool, err error)
	ResolvePlanApproval(ctx context.Context, approvalID, status string) error
	LoadResume(ctx context.Context, runID string) (*ResumeState, error)
	GetRun(ctx context.Context, runID string) (*RunStatus, error)
	ListEventsAfter(ctx context.Context, runID string, afterSeq int64, limit int) ([]Event, error)
	AppendRunSignal(ctx context.Context, s Signal) (id int64, duplicate bool, err error)
	ClaimRunSignal(ctx context.Context, runID string) (*Signal, error)
	RegisterRunTask(ctx context.Context, runID string, ref TaskRef) error
	ListRunTasks(ctx context.Context, runID string) ([]TaskRef, error)
	RequestRunCancel(ctx context.Context, runID string) error
	RunCancelRequested(ctx context.Context, runID string) (bool, error)
	BeginConfirm(ctx context.Context, runID, key, requestHash string) (*ConfirmRecord, bool, error)
	CompleteConfirm(ctx context.Context, runID, key, status string, httpStatus int, response []byte) error
	SaveRouteDecision(ctx context.Context, decision RouteDecision) error
	ListRouteDecisions(ctx context.Context, runID string) ([]RouteDecision, error)
}

// RejectedAgent records why a candidate was not dispatched.
type RejectedAgent struct {
	AgentName  string `json:"agentName"`
	ReasonCode string `json:"reasonCode"`
}

// RouteDecision is one explainable routing record for a run.
type RouteDecision struct {
	RunID         string          `json:"runId"`
	DecisionSeq   int             `json:"decisionSeq"`
	ExecutionPath string          `json:"executionPath"`
	Strategy      string          `json:"strategy"`
	Candidates    []string        `json:"candidates"`
	Selected      []string        `json:"selected"`
	Rejected      []RejectedAgent `json:"rejected"`
	PlannerSource string          `json:"plannerSource"`
	ReasonSummary string          `json:"reasonSummary"`
	CreatedAt     time.Time       `json:"createdAt"`
}

// TaskRef is one dispatched remote A2A task, shared across instances.
type TaskRef struct {
	TaskID    string
	AgentName string
	AgentURL  string
}

var (
	// ErrConfirmConflict means an idempotency key was reused with a different body.
	ErrConfirmConflict = errors.New("idempotency key reused with a different request")
	// ErrConfirmInFlight means another instance is processing the same key.
	ErrConfirmInFlight = errors.New("confirmation is already in flight")
)

// ConfirmRecord is a stored HITL confirmation result for one idempotency key.
type ConfirmRecord struct {
	Status       string
	HTTPStatus   int
	ResponseJSON []byte
	CreatedAt    time.Time
}

// Signal is a durable HITL decision written by any instance and claimed by the
// instance waiting on the run.
type Signal struct {
	ID                   int64
	RunID                string
	ActionID             string
	PlanID               string
	Revision             int
	Action               string
	Feedback             string
	RejectReason         string
	SelectedParticipants []string
	IdempotencyKey       string
	Status               string
	CreatedAt            time.Time
}
