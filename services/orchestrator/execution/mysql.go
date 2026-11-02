package execution

import (
	"errors"
	"context"
	"fmt"
	"time"

	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

// MySQLStore adapts the MySQL execution ledger to the handler-facing Store.
type MySQLStore struct {
	repo *ledger.Repository
}

func NewMySQLStore(repo *ledger.Repository) *MySQLStore {
	return &MySQLStore{repo: repo}
}

func (s *MySQLStore) AcceptRun(ctx context.Context, p Accept) (bool, error) {
	_, dedup, err := s.repo.AcceptRun(ctx, ledger.AcceptRunParams{
		RunID: p.RunID, RequestID: p.RequestID, ConversationID: p.ConversationID,
		WorkspaceID: p.WorkspaceID, PrincipalID: p.PrincipalID, ExecutionPath: p.ExecutionPath,
		BudgetJSON: p.BudgetJSON,
	})
	return dedup, err
}

func (s *MySQLStore) AcquireLease(ctx context.Context, runID, owner string, ttl time.Duration) (int64, error) {
	return s.repo.AcquireLease(ctx, runID, owner, ttl)
}

func (s *MySQLStore) RenewLease(ctx context.Context, runID, owner string, fencingVersion int64, ttl time.Duration) error {
	return s.repo.RenewLease(ctx, runID, owner, fencingVersion, ttl)
}

func (s *MySQLStore) ReleaseLease(ctx context.Context, runID, owner string, fencingVersion int64) error {
	return s.repo.ReleaseLease(ctx, runID, owner, fencingVersion)
}

func (s *MySQLStore) AppendState(ctx context.Context, st State) (int64, error) {
	return s.repo.AppendStateChange(ctx, ledger.StateChange{
		RunID: st.RunID, FencingVersion: st.FencingVersion, Status: st.Status,
		TerminalReason: st.TerminalReason, EventType: st.EventType,
		Payload: st.Payload, BudgetJSON: st.BudgetJSON,
	})
}

func (s *MySQLStore) SaveCheckpoint(ctx context.Context, c Checkpoint) (int64, error) {
	return s.repo.SaveCheckpoint(ctx, ledger.CheckpointParams{
		RunID: c.RunID, FencingVersion: c.FencingVersion,
		StateJSON: c.StateJSON, BudgetJSON: c.BudgetJSON,
	})
}

func (s *MySQLStore) SavePlan(ctx context.Context, p Plan) error {
	return s.repo.SavePlanRevision(ctx, p.RunID, p.PlanID, p.Revision, p.PlanJSON, p.PlanHash)
}

func (s *MySQLStore) CreatePlanApproval(ctx context.Context, a Approval) (string, bool, error) {
	approval, dedup, err := s.repo.CreateApproval(ctx, ledger.Approval{
		ApprovalID: a.ApprovalID, RunID: a.RunID, PlanID: a.PlanID, PlanRevision: a.PlanRevision,
		WorkspaceID: a.WorkspaceID, PrincipalID: a.PrincipalID,
		ExpiresAt: a.ExpiresAt, IdempotencyKey: a.IdempotencyKey,
	})
	if err != nil {
		return "", false, err
	}
	return approval.ApprovalID, dedup, nil
}

func (s *MySQLStore) ResolvePlanApproval(ctx context.Context, approvalID, status string) error {
	return s.repo.ResolveApproval(ctx, approvalID, status)
}

func (s *MySQLStore) LoadResume(ctx context.Context, runID string) (*ResumeState, error) {
	exec, err := s.repo.GetExecution(ctx, runID)
	if err != nil {
		return nil, err
	}
	plan, err := s.repo.LatestPlanRevision(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load plan: %w", err)
	}
	checkpoint, err := s.repo.LatestCheckpoint(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load checkpoint: %w", err)
	}
	return &ResumeState{
		RunID: runID, Status: exec.Status, FencingVersion: exec.FencingVersion,
		PlanJSON: plan.PlanJSON, BudgetJSON: checkpoint.BudgetJSON, CheckpointJSON: checkpoint.StateJSON,
	}, nil
}

func (s *MySQLStore) GetRun(ctx context.Context, runID string) (*RunStatus, error) {
	exec, err := s.repo.GetExecution(ctx, runID)
	if err != nil {
		return nil, err
	}
	return &RunStatus{
		RunID: exec.RunID, Status: exec.Status, CurrentSeq: exec.CurrentSeq,
		FencingVersion: exec.FencingVersion, TerminalReason: exec.TerminalReason,
	}, nil
}

func (s *MySQLStore) AppendRunSignal(ctx context.Context, sig Signal) (int64, bool, error) {
	return s.repo.AppendRunSignal(ctx, ledger.SignalParams{
		RunID: sig.RunID, ActionID: sig.ActionID, PlanID: sig.PlanID,
		PlanRevision: sig.Revision, Action: sig.Action, Feedback: sig.Feedback,
		RejectReason: sig.RejectReason, SelectedParticipants: sig.SelectedParticipants,
		IdempotencyKey: sig.IdempotencyKey,
	})
}

func (s *MySQLStore) ClaimRunSignal(ctx context.Context, runID string) (*Signal, error) {
	row, err := s.repo.ClaimRunSignal(ctx, runID)
	if err != nil || row == nil {
		return nil, err
	}
	return &Signal{
		ID: row.ID, RunID: row.RunID, ActionID: row.ActionID, PlanID: row.PlanID,
		Revision: row.PlanRevision, Action: row.Action, Feedback: row.Feedback,
		RejectReason: row.RejectReason, SelectedParticipants: row.SelectedParticipants,
		IdempotencyKey: row.IdempotencyKey, Status: row.Status, CreatedAt: row.CreatedAt,
	}, nil
}

func (s *MySQLStore) RegisterRunTask(ctx context.Context, runID string, ref TaskRef) error {
	return s.repo.RegisterRunTask(ctx, runID, ledger.RunTaskRef{
		TaskID: ref.TaskID, AgentName: ref.AgentName, AgentURL: ref.AgentURL,
	})
}

func (s *MySQLStore) ListRunTasks(ctx context.Context, runID string) ([]TaskRef, error) {
	refs, err := s.repo.ListRunTasks(ctx, runID)
	if err != nil {
		return nil, err
	}
	out := make([]TaskRef, len(refs))
	for i, ref := range refs {
		out[i] = TaskRef{TaskID: ref.TaskID, AgentName: ref.AgentName, AgentURL: ref.AgentURL}
	}
	return out, nil
}

func (s *MySQLStore) RequestRunCancel(ctx context.Context, runID string) error {
	return s.repo.RequestRunCancel(ctx, runID)
}

func (s *MySQLStore) RunCancelRequested(ctx context.Context, runID string) (bool, error) {
	return s.repo.RunCancelRequested(ctx, runID)
}

func (s *MySQLStore) BeginConfirm(ctx context.Context, runID, key, requestHash string) (*ConfirmRecord, bool, error) {
	record, claimed, err := s.repo.BeginConfirm(ctx, runID, key, requestHash)
	if err != nil {
		if errors.Is(err, ledger.ErrConfirmIdempotencyConflict) {
			return nil, false, ErrConfirmConflict
		}
		if errors.Is(err, ledger.ErrConfirmInFlight) {
			return nil, false, ErrConfirmInFlight
		}
		return nil, false, err
	}
	return &ConfirmRecord{
		Status: record.Status, HTTPStatus: record.HTTPStatus,
		ResponseJSON: record.ResponseJSON, CreatedAt: record.CreatedAt,
	}, claimed, nil
}

func (s *MySQLStore) CompleteConfirm(ctx context.Context, runID, key, status string, httpStatus int, response []byte) error {
	return s.repo.CompleteConfirm(ctx, runID, key, status, httpStatus, response)
}

func (s *MySQLStore) SaveRouteDecision(ctx context.Context, decision RouteDecision) error {
	rejected := make([]ledger.RejectedAgent, len(decision.Rejected))
	for i, agent := range decision.Rejected {
		rejected[i] = ledger.RejectedAgent{AgentName: agent.AgentName, ReasonCode: agent.ReasonCode}
	}
	return s.repo.SaveRouteDecision(ctx, ledger.RouteDecision{
		RunID: decision.RunID, DecisionSeq: decision.DecisionSeq,
		ExecutionPath: decision.ExecutionPath, Strategy: decision.Strategy,
		Candidates: decision.Candidates, Selected: decision.Selected,
		Rejected: rejected, PlannerSource: decision.PlannerSource,
		ReasonSummary: decision.ReasonSummary,
	})
}

func (s *MySQLStore) ListRouteDecisions(ctx context.Context, runID string) ([]RouteDecision, error) {
	rows, err := s.repo.ListRouteDecisions(ctx, runID)
	if err != nil {
		return nil, err
	}
	out := make([]RouteDecision, len(rows))
	for i, row := range rows {
		rejected := make([]RejectedAgent, len(row.Rejected))
		for j, agent := range row.Rejected {
			rejected[j] = RejectedAgent{AgentName: agent.AgentName, ReasonCode: agent.ReasonCode}
		}
		out[i] = RouteDecision{
			RunID: row.RunID, DecisionSeq: row.DecisionSeq,
			ExecutionPath: row.ExecutionPath, Strategy: row.Strategy,
			Candidates: row.Candidates, Selected: row.Selected,
			Rejected: rejected, PlannerSource: row.PlannerSource,
			ReasonSummary: row.ReasonSummary, CreatedAt: row.CreatedAt,
		}
	}
	return out, nil
}

func (s *MySQLStore) RoutingMetrics(ctx context.Context, since time.Time) (*RouteMetrics, error) {
	metrics, err := s.repo.RouteDecisionMetrics(ctx, since)
	if err != nil {
		return nil, err
	}
	return &RouteMetrics{
		TotalRuns: metrics.TotalRuns, TotalDecisions: metrics.TotalDecisions,
		SelectedCounts: metrics.SelectedCounts, RejectedCounts: metrics.RejectedCounts,
		MultiDecisionRuns: metrics.MultiDecisionRuns,
		ModelCalls: metrics.ModelCalls, ToolCalls: metrics.ToolCalls,
		Repairs: metrics.Repairs, Replans: metrics.Replans,
	}, nil
}

func (s *MySQLStore) OperationalMetrics(ctx context.Context, since time.Time) (*OperationalMetrics, error) {
	metrics, err := s.repo.QueryOperationalMetrics(ctx, since)
	if err != nil {
		return nil, err
	}
	return &OperationalMetrics{
		Executions: metrics.Executions, Approvals: metrics.Approvals, ToolCalls: metrics.ToolCalls,
	}, nil
}

func (s *MySQLStore) ListEventsAfter(ctx context.Context, runID string, afterSeq int64, limit int) ([]Event, error) {
	rows, err := s.repo.ListEventsAfter(ctx, runID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Event, len(rows))
	for i, row := range rows {
		out[i] = Event{Seq: row.Seq, Type: row.EventType, Payload: row.Payload, CreatedAt: row.CreatedAt}
	}
	return out, nil
}
