package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/executor"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/validator"
)

// handleResumeRun resumes a paused durable run from its latest checkpoint and
// plan revision. Completed task results are not yet reused; they are re-run.
func (s *Server) handleResumeRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.executionStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "execution store unavailable"})
		return
	}
	runID := strings.Trim(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/internal/orchestrator/runs/"), "/resume"), "/")
	if runID == "" || strings.Contains(runID, "/") {
		http.NotFound(w, r)
		return
	}
	state, err := s.executionStore.LoadResume(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run or checkpoint not found"})
		return
	}
	if state.Status != "paused" {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "run is not paused", "status": state.Status})
		return
	}
	var p plan.OrchestrationPlan
	if err := json.Unmarshal(state.PlanJSON, &p); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid checkpoint plan"})
		return
	}
	// Resume must never reset a run's budget: an unreadable or exhausted
	// checkpoint means the durable state is not trustworthy, so fail closed
	// before taking the lease or streaming anything.
	var usage harness.Usage
	if len(state.BudgetJSON) == 0 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "checkpoint budget unavailable", "code": "BUDGET_RESTORE_FAILED",
		})
		return
	}
	if err := json.Unmarshal(state.BudgetJSON, &usage); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "checkpoint budget invalid", "code": "BUDGET_RESTORE_FAILED",
		})
		return
	}
	budget, err := harness.RestoreBudget(harness.DefaultLimits(), usage)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "checkpoint budget exceeds limits", "code": "BUDGET_RESTORE_FAILED",
		})
		return
	}
	leaseTTL := s.leaseTTL
	if leaseTTL <= 0 {
		leaseTTL = 15 * time.Minute
	}
	renewInterval := s.leaseRenewInterval
	if renewInterval <= 0 {
		renewInterval = leaseTTL / 3
	}
	fencing, err := s.executionStore.AcquireLease(r.Context(), runID, s.instanceID, leaseTTL)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "run lease unavailable"})
		return
	}
	ledger := &runLedger{
		store: s.executionStore, runID: runID, fencing: fencing, owner: s.instanceID,
		leaseTTL: leaseTTL, renewInterval: renewInterval,
	}
	defer ledger.releaseLease()
	budgetJSON, _ := json.Marshal(budget.Usage())

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	msgID := "msg_resume_" + runID
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type: agui.InternalTypeRunStarted, RunID: runID,
		State: map[string]any{"phase": "resuming"},
	})

	fail := func(reason string) {
		_, _ = s.executionStore.AppendState(context.Background(), execution.State{
			RunID: runID, FencingVersion: fencing, Status: "failed", TerminalReason: reason,
			EventType: "run_finished", Payload: []byte(`{"status":"failed"}`), BudgetJSON: budgetJSON,
		})
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_RESUME_FAILED", "run resume failed")
	}

	if result := validator.New(&validatorRegistryAdapter{resolver: s.dispatchResolver()}).Validate(&p); !result.Valid {
		fail("checkpoint_plan_invalid")
		return
	}
	if result := validator.NewPathAwareValidator(p.ExecutionPath, p.AllowedAgents).Validate(&p); !result.Valid {
		fail("checkpoint_boundary_invalid")
		return
	}
	if _, err := s.executionStore.AppendState(r.Context(), execution.State{
		RunID: runID, FencingVersion: fencing, Status: "executing", EventType: "state_update",
		Payload: []byte(`{"status":"executing","resumed":true}`), BudgetJSON: budgetJSON,
	}); err != nil {
		log.Printf("orchestrator: resume %s state failed: %v", runID, err)
		s.emitErrorEvent(w, flusher, runID, "ORCHESTRATOR_STORE_UNAVAILABLE", "run persistence unavailable")
		return
	}
	_, _ = s.executionStore.SaveCheckpoint(r.Context(), execution.Checkpoint{
		RunID: runID, FencingVersion: fencing,
		StateJSON: []byte(`{"status":"executing","resumed":true}`), BudgetJSON: budgetJSON,
	})
	s.emitEvent(w, flusher, agui.InternalStreamEvent{
		Type: agui.InternalTypeStateUpdate, RunID: runID,
		State: map[string]any{"phase": "executing", "resumed": true},
	})

	execCtx, cancel := context.WithCancel(r.Context())
	defer cancel()
	execCtx = adk.WithRunIdentity(execCtx, adk.RunIdentity{
		RunID: runID, WorkspaceID: "default", PrincipalID: "resume",
		FencingVersion: ledger.fencing,
	})
	ledger.startLeaseRenewal(execCtx, cancel)
	defer ledger.stopLeaseRenewal()
	go s.watchRunCancel(execCtx, runID, cancel)
	if s.runTaskRegistry != nil {
		s.runTaskRegistry.RegisterRun(runID, cancel)
		defer s.runTaskRegistry.RemoveRun(runID)
	}
	req := r.WithContext(execCtx)
	outcome := "completed"
	switch p.Strategy {
	case plan.StrategyConversational:
		if !s.handleConversational(w, flusher, runID, msgID, p.Answer) {
			outcome = "failed"
		}
	case plan.StrategySingle:
		outcome = s.executeViaStreamingExecutor(w, flusher, req, &p, msgID,
			executor.NewSingleExecutor(s.dispatchResolver(), s.dispatcher, executor.WithSingleSynthesizer(s.synthesizer)))
	case plan.StrategySequential:
		outcome = s.executeViaStreamingExecutor(w, flusher, req, &p, msgID,
			executor.NewDAGExecutor(s.dispatchResolver(), s.dispatcher, executor.WithDAGSynthesizer(s.synthesizer), executor.WithDAGMaxParallel(s.budgetLimits.ParallelTasks)))
	case plan.StrategyOrderedParallel:
		outcome = s.executeViaStreamingExecutor(w, flusher, req, &p, msgID,
			executor.NewSerialExecutor(s.dispatchResolver(), s.dispatcher, executor.WithSerialSynthesizer(s.synthesizer)))
	default:
		outcome = "failed"
	}
	status, reason := "completed", "resumed_completed"
	if outcome == "failed" {
		status, reason = "failed", "resumed_execution_failed"
	} else if outcome == "cancelled" {
		status, reason = "cancelled", "resumed_cancelled"
	}
	_, _ = s.executionStore.AppendState(context.Background(), execution.State{
		RunID: runID, FencingVersion: fencing, Status: status, TerminalReason: reason,
		EventType: "run_finished", Payload: []byte(`{"status":"` + status + `"}`), BudgetJSON: budgetJSON,
	})
}
