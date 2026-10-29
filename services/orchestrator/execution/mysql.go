package execution

import (
	"context"
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
