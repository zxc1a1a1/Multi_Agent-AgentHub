package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// runLedger persists a run's durable lifecycle. When no execution store is
// configured it still owns the in-process budget so production code paths never
// silently skip accounting.
type runLedger struct {
	store   execution.Store
	runID   string
	fencing int64
	budget  *harness.Budget
}

func (s *Server) newRunLedger(ctx context.Context, req OrchestratorRequest, runID, executionPath string) (*runLedger, error) {
	limits := s.budgetLimits
	if limits.Validate() != nil {
		limits = harness.DefaultLimits()
	}
	budget, err := harness.NewBudget(limits)
	if err != nil {
		return nil, err
	}
	ledger := &runLedger{store: s.executionStore, runID: runID, budget: budget}
	if s.executionStore == nil {
		return ledger, nil
	}

	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = runID
	}
	principal := strings.TrimSpace(req.UserID)
	if principal == "" {
		principal = "anonymous"
	}
	dedup, err := s.executionStore.AcceptRun(ctx, execution.Accept{
		RunID: runID, RequestID: requestID, ConversationID: req.ConversationID,
		WorkspaceID: "default", PrincipalID: principal, ExecutionPath: executionPath,
		BudgetJSON: ledger.budgetJSON(),
	})
	if err != nil {
		return nil, fmt.Errorf("accept run: %w", err)
	}
	if dedup {
		log.Printf("orchestrator: run=%s request=%s already accepted; continuing with same run", runID, requestID)
	}
	fencing, err := s.executionStore.AcquireLease(ctx, runID, s.instanceID, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("acquire lease: %w", err)
	}
	ledger.fencing = fencing
	return ledger, nil
}

func (l *runLedger) enrolled() bool {
	return l != nil && l.store != nil
}

func (l *runLedger) budgetJSON() []byte {
	if l == nil || l.budget == nil {
		return []byte(`{}`)
	}
	data, err := json.Marshal(l.budget.Usage())
	if err != nil {
		return []byte(`{}`)
	}
	return data
}

func (l *runLedger) state(ctx context.Context, status, reason, eventType string, payload map[string]any) error {
	if !l.enrolled() {
		return nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["status"] = status
	if reason != "" {
		payload["reason"] = reason
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = l.store.AppendState(ctx, execution.State{
		RunID: l.runID, FencingVersion: l.fencing, Status: status,
		TerminalReason: reason, EventType: eventType, Payload: data, BudgetJSON: l.budgetJSON(),
	})
	return err
}

func (l *runLedger) savePlan(ctx context.Context, p *plan.OrchestrationPlan) error {
	if !l.enrolled() || p == nil {
		return nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	revision := p.Revision
	if revision < 1 {
		revision = 1
	}
	sum := sha256.Sum256(data)
	return l.store.SavePlan(ctx, execution.Plan{
		RunID: l.runID, PlanID: p.PlanID, Revision: revision,
		PlanJSON: data, PlanHash: hex.EncodeToString(sum[:]),
	})
}

func (l *runLedger) checkpoint(ctx context.Context, state map[string]any, status string) error {
	if !l.enrolled() {
		return nil
	}
	if state == nil {
		state = map[string]any{}
	}
	state["status"] = status
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = l.store.SaveCheckpoint(ctx, execution.Checkpoint{
		RunID: l.runID, FencingVersion: l.fencing, StateJSON: data, BudgetJSON: l.budgetJSON(),
	})
	return err
}

func (l *runLedger) terminal(ctx context.Context, status, reason string) {
	if !l.enrolled() {
		return
	}
	if err := l.state(ctx, status, reason, "run_finished", nil); err != nil {
		log.Printf("orchestrator: run=%s persist terminal %s failed: %v", l.runID, status, err)
	}
}

func planLedgerState(p *plan.OrchestrationPlan) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	return map[string]any{
		"planId": p.PlanID, "revision": p.Revision, "executionPath": p.ExecutionPath,
		"allowedAgents": p.AllowedAgents, "taskCount": len(p.Tasks),
	}
}

func (s *Server) abortRunWithLedger(w http.ResponseWriter, flusher http.Flusher, runID string, ledger *runLedger, message string, cause error) {
	log.Printf("orchestrator: run=%s %s: %v", runID, message, cause)
	ledger.terminal(context.Background(), "failed", "store_unavailable")
	s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_STORE_UNAVAILABLE", "run persistence unavailable")
}
