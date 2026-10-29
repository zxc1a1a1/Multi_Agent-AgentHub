package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// TestRunEntryWithoutPlannerDoesNotDispatch proves the real handler entry fails
// before any child agent is called when no model-backed planner is wired.
func TestRunEntryWithoutPlannerDoesNotDispatch(t *testing.T) {
	var childCalls int32
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&childCalls, 1)
		http.Error(w, "unexpected dispatch", http.StatusInternalServerError)
	}))
	defer child.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	body := `{"runId":"run-no-planner","conversationId":"conv-1","messages":[{"role":"user","text":"write Go code"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if got := atomic.LoadInt32(&childCalls); got != 0 {
		t.Fatalf("child agent dispatched %d time(s) without a planner", got)
	}
	if !strings.Contains(rec.Body.String(), "ORCHESTRATOR_PLANNER_UNAVAILABLE") {
		t.Fatalf("expected ORCHESTRATOR_PLANNER_UNAVAILABLE, got: %s", rec.Body.String())
	}
}

// TestRunEntryPlannerFailuresDoNotDispatch covers model error and nil-plan cases.
func TestRunEntryPlannerFailuresDoNotDispatch(t *testing.T) {
	cases := []struct {
		name string
		fake *FakeMainAgent
	}{
		{name: "model error", fake: &FakeMainAgent{PlanErr: errors.New("provider unavailable")}},
		{name: "nil plan nil error", fake: &FakeMainAgent{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var childCalls int32
			child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&childCalls, 1)
				http.Error(w, "unexpected dispatch", http.StatusInternalServerError)
			}))
			defer child.Close()

			reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
				{Name: "code-agent", URL: child.URL},
			})
			if err != nil {
				t.Fatalf("create registry: %v", err)
			}

			srv := NewServer(
				WithRegistry(reg),
				WithDispatcher(dispatcher.NewA2ADispatcher()),
				WithMainAgentPlanner(tc.fake),
			)
			body := `{"runId":"run-planner-fail","conversationId":"conv-1","messages":[{"role":"user","text":"write Go code"}],"requestedPath":"main_agent_orchestration"}`
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
			srv.Handler().ServeHTTP(rec, req)

			if got := atomic.LoadInt32(&childCalls); got != 0 {
				t.Fatalf("child agent dispatched %d time(s) after planner failure", got)
			}
			if !strings.Contains(rec.Body.String(), "ORCHESTRATOR_PLANNER_FAILED") {
				t.Fatalf("expected ORCHESTRATOR_PLANNER_FAILED, got: %s", rec.Body.String())
			}
		})
	}
}

// TestRunEntryRejectsUnknownAgentPlan proves an invalid plan is rejected, not
// repaired with a default agent or dispatched.
func TestRunEntryRejectsUnknownAgentPlan(t *testing.T) {
	var childCalls int32
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&childCalls, 1)
		http.Error(w, "unexpected dispatch", http.StatusInternalServerError)
	}))
	defer child.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	badPlan := makeTestPlan("run-ghost", plan.StrategySingle, []plan.TaskPlan{
		makeTaskPlan("task-1", "ghost-agent", "do it"),
	})
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: badPlan}),
	)
	body := `{"runId":"run-ghost","conversationId":"conv-1","messages":[{"role":"user","text":"do it"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if got := atomic.LoadInt32(&childCalls); got != 0 {
		t.Fatalf("child agent dispatched %d time(s) for an invalid plan", got)
	}
	bodyText := rec.Body.String()
	if !strings.Contains(bodyText, "ORCHESTRATOR_PLAN_INVALID") && !strings.Contains(bodyText, "AGENT_BOUNDARY_VIOLATION") {
		t.Fatalf("expected plan rejection, got: %s", bodyText)
	}
}

// TestRunEntryConversationalUsesModelAnswer proves answer mode streams the
// planner-provided text and never falls back to a canned response.
func TestRunEntryConversationalUsesModelAnswer(t *testing.T) {
	var childCalls int32
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&childCalls, 1)
		http.Error(w, "unexpected dispatch", http.StatusInternalServerError)
	}))
	defer child.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	answerPlan := makeTestPlan("run-answer", plan.StrategyConversational, nil)
	answerPlan.Answer = "model-generated greeting"
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: answerPlan}),
	)
	body := `{"runId":"run-answer","conversationId":"conv-1","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if got := atomic.LoadInt32(&childCalls); got != 0 {
		t.Fatalf("conversational plan dispatched a child agent %d time(s)", got)
	}
	if !strings.Contains(rec.Body.String(), "model-generated greeting") {
		t.Fatalf("expected model answer in stream, got: %s", rec.Body.String())
	}
}
