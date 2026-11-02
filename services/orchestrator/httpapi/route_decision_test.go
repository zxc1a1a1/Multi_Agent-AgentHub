package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestBuildRouteDecisionExplainsRejections(t *testing.T) {
	p := &plan.OrchestrationPlan{
		RunID: "run-route", ExecutionPath: "main_agent_orchestration", Strategy: plan.StrategyOrderedParallel,
		IntentSummary: "route me",
		CandidateParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Selected: true},
			{AgentName: "review-agent", Selected: false},
			{AgentName: "test-agent", Selected: true},
		},
		Tasks: []plan.TaskPlan{{TaskID: "t1", AgentName: "code-agent"}},
	}
	decision := buildRouteDecision(p, 0)
	if decision.ExecutionPath != "main_agent_orchestration" || decision.Strategy != plan.StrategyOrderedParallel {
		t.Fatalf("unexpected path/strategy: %#v", decision)
	}
	if len(decision.Candidates) != 3 || decision.Candidates[1] != "review-agent" {
		t.Fatalf("candidates lost: %#v", decision.Candidates)
	}
	if len(decision.Selected) != 1 || decision.Selected[0] != "code-agent" {
		t.Fatalf("selected lost: %#v", decision.Selected)
	}
	reasons := map[string]string{}
	for _, rejected := range decision.Rejected {
		reasons[rejected.AgentName] = rejected.ReasonCode
	}
	if reasons["review-agent"] != "not_selected" || reasons["test-agent"] != "no_task_assigned" {
		t.Fatalf("unexpected rejection reasons: %#v", decision.Rejected)
	}
	if decision.PlannerSource != "main_agent" || decision.ReasonSummary != "route me" {
		t.Fatalf("planner metadata lost: %#v", decision)
	}
}

func TestRunRecordsRouteDecision(t *testing.T) {
	store := &fakeExecutionStore{}
	p := makeTestPlan("run-route-live", plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.ExecutionPath = "single_chat"
	p.AllowedAgents = []string{"code-agent"}
	p.Participants = []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	srv := newLedgerTestServer(t, store, &FakeMainAgent{PlanToReturn: p})
	body := `{"runId":"run-route-live","conversationId":"conv","messages":[{"role":"user","text":"do it"}],"requestedPath":"single_chat","agentName":"code-agent"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))

	store.mu.Lock()
	decisions := append([]execution.RouteDecision(nil), store.routeDecisions["run-route-live"]...)
	store.mu.Unlock()
	if len(decisions) != 1 {
		t.Fatalf("route decisions=%d, want 1", len(decisions))
	}
	if decisions[0].ExecutionPath != "single_chat" || len(decisions[0].Selected) != 1 || decisions[0].Selected[0] != "code-agent" {
		t.Fatalf("unexpected decision: %#v", decisions[0])
	}
}

func TestRouteDecisionsEndpoint(t *testing.T) {
	store := &fakeExecutionStore{routeDecisions: map[string][]execution.RouteDecision{
		"run-route": {{RunID: "run-route", DecisionSeq: 0, ExecutionPath: "single_chat", Strategy: "single", Candidates: []string{"code-agent"}, Selected: []string{"code-agent"}}},
	}}
	srv := NewServer(WithExecutionStore(store))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/orchestrator/runs/run-route/route-decisions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("endpoint failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Decisions []execution.RouteDecision `json:"decisions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Decisions) != 1 || payload.Decisions[0].Selected[0] != "code-agent" {
		t.Fatalf("unexpected payload: %s", rec.Body.String())
	}
}
