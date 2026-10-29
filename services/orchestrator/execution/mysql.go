package execution

import (
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
