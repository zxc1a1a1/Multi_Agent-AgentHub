package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestRunCancelCancelsPendingPlan(t *testing.T) {
	srv := NewServer()
	runID := "run-pending-cancel"
	p := &plan.OrchestrationPlan{
		Version: "1.0", PlanID: "plan-pending-1", RunID: runID,
		ExecutionPath: "single_chat", Strategy: "single",
	}
	ch := srv.registerPendingPlan(runID, p, []string{"code-agent"})

	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/cancel", strings.NewReader(`{"runId":"`+runID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for pending plan cancel, got %d body=%s", rec.Code, rec.Body.String())
	}
	select {
	case result := <-ch:
		if result.Action != "cancel" || result.Confirmed {
			t.Fatalf("unexpected cancel signal: %#v", result)
		}
	default:
		t.Fatal("pending plan channel received no cancel signal")
	}

	srv.hitlMu.Lock()
	state := srv.hitlStates[runID]
	planStatus := srv.pendingPlanStates[runID].Status
	srv.hitlMu.Unlock()
	if state != HITLCancelled || planStatus != PlanStatusCancelled {
		t.Fatalf("pending plan not cancelled: state=%s plan=%s", state, planStatus)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/cancel", strings.NewReader(`{"runId":"`+runID+`"}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second cancel, got %d body=%s", rec.Code, rec.Body.String())
	}
}
