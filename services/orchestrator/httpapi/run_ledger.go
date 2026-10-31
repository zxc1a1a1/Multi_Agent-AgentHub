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
	"sync"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// runLedger persists a run's durable lifecycle. When no execution store is
// configured it still owns the in-process budget so production code paths never
// silently skip accounting.
type runLedger struct {
	store      execution.Store
	runID      string
	fencing    int64
	budget     *harness.Budget
	workspace  string
	principal  string
	approvalID string

	owner         string
	leaseTTL      time.Duration
	renewInterval time.Duration
	renewMu       sync.Mutex
	renewStop     chan struct{}
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
	leaseTTL := s.leaseTTL
	if leaseTTL <= 0 {
		leaseTTL = 15 * time.Minute
	}
	renewInterval := s.leaseRenewInterval
	if renewInterval <= 0 {
		renewInterval = leaseTTL / 3
	}
	ledger := &runLedger{
		store: s.executionStore, runID: runID, budget: budget,
		workspace: "default", principal: "anonymous",
		owner: s.instanceID, leaseTTL: leaseTTL, renewInterval: renewInterval,
	}
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
	ledger.principal = principal
	dedup, err := s.executionStore.AcceptRun(ctx, execution.Accept{
		RunID: runID, RequestID: requestID, ConversationID: req.ConversationID,
		WorkspaceID: ledger.workspace, PrincipalID: principal, ExecutionPath: executionPath,
		BudgetJSON: ledger.budgetJSON(),
	})
	if err != nil {
		return nil, fmt.Errorf("accept run: %w", err)
	}
	if dedup {
		log.Printf("orchestrator: run=%s request=%s already accepted; continuing with same run", runID, requestID)
	}
	fencing, err := s.executionStore.AcquireLease(ctx, runID, s.instanceID, ledger.leaseTTL)
	if err != nil {
		return nil, fmt.Errorf("acquire lease: %w", err)
	}
	ledger.fencing = fencing
	return ledger, nil
}

// startLeaseRenewal keeps the run lease alive for long executions. onLost is
// called once when the lease can no longer be renewed (for example because
// another instance took it over); callers should cancel the run context.
func (l *runLedger) startLeaseRenewal(ctx context.Context, onLost func()) {
	if l == nil || !l.enrolled() || l.owner == "" {
		return
	}
	l.renewMu.Lock()
	defer l.renewMu.Unlock()
	if l.renewStop != nil {
		return
	}
	stop := make(chan struct{})
	l.renewStop = stop
	interval := l.renewInterval
	if interval <= 0 {
		interval = l.leaseTTL / 3
	}
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				renewCtx, cancel := context.WithTimeout(context.Background(), interval)
				err := l.store.RenewLease(renewCtx, l.runID, l.owner, l.fencing, l.leaseTTL)
				cancel()
				if err != nil {
					log.Printf("orchestrator: run=%s lease renewal failed: %v", l.runID, err)
					if onLost != nil {
						onLost()
					}
					return
				}
			}
		}
	}()
}

func (l *runLedger) stopLeaseRenewal() {
	if l == nil {
		return
	}
	l.renewMu.Lock()
	defer l.renewMu.Unlock()
	if l.renewStop == nil {
		return
	}
	close(l.renewStop)
	l.renewStop = nil
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

func (l *runLedger) requestApproval(ctx context.Context, p *plan.OrchestrationPlan) error {
	if !l.enrolled() || p == nil {
		return nil
	}
	revision := p.Revision
	if revision < 1 {
		revision = 1
	}
	key := fmt.Sprintf("plan:%s:%s:%d", l.runID, p.PlanID, revision)
	sum := sha256.Sum256([]byte(key))
	approvalID := "appr_" + hex.EncodeToString(sum[:12])
	id, _, err := l.store.CreatePlanApproval(ctx, execution.Approval{
		ApprovalID: approvalID, RunID: l.runID, PlanID: p.PlanID, PlanRevision: revision,
		WorkspaceID: l.workspace, PrincipalID: l.principal,
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute), IdempotencyKey: key,
	})
	if err != nil {
		return err
	}
	l.approvalID = id
	return nil
}

func (l *runLedger) resolveApproval(ctx context.Context, status string) {
	if !l.enrolled() || l.approvalID == "" {
		return
	}
	if err := l.store.ResolvePlanApproval(ctx, l.approvalID, status); err != nil {
		log.Printf("orchestrator: run=%s resolve approval %s=%s failed: %v", l.runID, l.approvalID, status, err)
	}
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
