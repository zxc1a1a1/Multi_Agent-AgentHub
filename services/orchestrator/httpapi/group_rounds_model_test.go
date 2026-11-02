package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

type fakeSpeakerSelector struct {
	mu       sync.Mutex
	choices  []planner.SpeakerChoice
	err      error
	requests []planner.SpeakerRequest
}

func (f *fakeSpeakerSelector) SelectSpeaker(_ context.Context, req planner.SpeakerRequest) (planner.SpeakerChoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.err != nil {
		return planner.SpeakerChoice{}, f.err
	}
	if len(f.choices) == 0 {
		return planner.SpeakerChoice{}, errors.New("fake selector has no choice")
	}
	idx := len(f.requests) - 1
	if idx >= len(f.choices) {
		idx = len(f.choices) - 1
	}
	return f.choices[idx], nil
}

func modelGroupPlan(runID string) *plan.OrchestrationPlan {
	p := makeTestPlan(runID, plan.StrategyOrderedParallel, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "seed")})
	p.ExecutionPath = "group_chat"
	p.Participants = []plan.PlanParticipant{
		{AgentName: "code-agent", Role: "executor", Selected: true},
		{AgentName: "review-agent", Role: "reviewer", Selected: true},
	}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	p.AllowedAgents = []string{"code-agent", "review-agent"}
	return p
}

func TestExpandGroupChatWithSelectorUsesReasonsAndStops(t *testing.T) {
	selector := &fakeSpeakerSelector{choices: []planner.SpeakerChoice{
		{Speaker: "review-agent", Reason: "review the design first"},
		{Speaker: "code-agent", Reason: "implement after review"},
		{Done: true, Reason: "request answered"},
	}}
	p := modelGroupPlan("run-selector")
	expanded, records, termination, err := expandGroupChatWithSelector(context.Background(), selector, p, 2, "design a cache", nil)
	if err != nil {
		t.Fatalf("expand failed: %v", err)
	}
	if termination != "request answered" {
		t.Fatalf("termination=%q", termination)
	}
	if len(expanded.Tasks) != 2 || len(records) != 3 {
		t.Fatalf("tasks=%d records=%d, want 2/3", len(expanded.Tasks), len(records))
	}
	if expanded.Tasks[0].AgentName != "review-agent" || expanded.Tasks[1].AgentName != "code-agent" {
		t.Fatalf("unexpected order: %#v", expanded.Tasks)
	}
	if expanded.Tasks[0].Reason != "review the design first" || expanded.Tasks[1].Reason != "implement after review" {
		t.Fatalf("task reasons lost: %#v", expanded.Tasks)
	}
	if expanded.Strategy != plan.StrategySequential || expanded.Tasks[1].DependsOn[0] != expanded.Tasks[0].TaskID {
		t.Fatalf("sequential dependency missing: %#v", expanded)
	}
	if !strings.Contains(expanded.Tasks[1].TaskContent, "{{deps."+expanded.Tasks[0].TaskID+".output}}") {
		t.Fatalf("previous output placeholder missing: %s", expanded.Tasks[1].TaskContent)
	}
	if len(selector.requests) != 3 {
		t.Fatalf("selector calls=%d, want 3", len(selector.requests))
	}
	if len(selector.requests[1].Previous) != 1 || selector.requests[1].Previous[0].AgentName != "review-agent" {
		t.Fatalf("previous turns not propagated: %#v", selector.requests[1].Previous)
	}
	if selector.requests[0].Candidates[1].Role != "reviewer" {
		t.Fatalf("candidate role lost: %#v", selector.requests[0].Candidates)
	}
	if p.Strategy != plan.StrategyOrderedParallel || len(p.Tasks) != 1 {
		t.Fatal("original plan must not be mutated")
	}
}

func TestExpandGroupChatWithSelectorRejectsOutsideAllowlist(t *testing.T) {
	selector := &fakeSpeakerSelector{choices: []planner.SpeakerChoice{{Speaker: "test-agent", Reason: "run tests"}}}
	if _, _, _, err := expandGroupChatWithSelector(context.Background(), selector, modelGroupPlan("run-selector-bad"), 1, "x", nil); err == nil ||
		!strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestExpandGroupChatWithSelectorRejectsStopBeforeFirstSpeaker(t *testing.T) {
	selector := &fakeSpeakerSelector{choices: []planner.SpeakerChoice{{Done: true, Reason: "nothing to do"}}}
	if _, _, _, err := expandGroupChatWithSelector(context.Background(), selector, modelGroupPlan("run-selector-stop"), 1, "x", nil); err == nil ||
		!strings.Contains(err.Error(), "before choosing any speaker") {
		t.Fatalf("expected early-stop error, got %v", err)
	}
}

func TestGroupChatModelSelectionDispatchesChosenSpeakerAndRecordsReason(t *testing.T) {
	t.Setenv("GROUP_CHAT_SPEAKER_MODE", "model")
	t.Setenv("GROUP_CHAT_MAX_ROUNDS", "1")
	t.Setenv("ORCHESTRATOR_MAX_TASKS", "10")
	var messages []string
	var mu sync.Mutex
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		messages = append(messages, body.Params.Message.Content)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"task-1","status":"completed","events":[{"author":"agent","role":"assistant","parts":[{"type":"text","text":"ok"}],"final":true}]}`))
	}))
	defer child.Close()
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
		{Name: "review-agent", URL: child.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	selector := &fakeSpeakerSelector{choices: []planner.SpeakerChoice{
		{Speaker: "review-agent", Reason: "reviewer should inspect the cache design"},
		{Done: true, Reason: "review complete"},
	}}
	store := &fakeExecutionStore{}
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: modelGroupPlan("run-model-select")}),
		WithSpeakerSelector(selector),
		WithExecutionStore(store),
		WithInstanceID("test-instance"),
	)
	body := `{"runId":"run-model-select","conversationId":"conv","messages":[{"role":"user","text":"design cache"}],"requestedPath":"group_chat","selectedAgentNames":["code-agent","review-agent"]}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))

	mu.Lock()
	dispatched := append([]string(nil), messages...)
	mu.Unlock()
	if len(dispatched) != 1 || !strings.Contains(dispatched[0], "speaker review-agent") {
		t.Fatalf("unexpected dispatches: %#v body=%s", dispatched, rec.Body.String())
	}

	store.mu.Lock()
	decisions := append([]execution.RouteDecision(nil), store.routeDecisions["run-model-select"]...)
	store.mu.Unlock()
	if len(decisions) != 1 {
		t.Fatalf("route decisions=%d, want 1: %#v", len(decisions), decisions)
	}
	decision := decisions[0]
	if decision.DecisionSeq != 0 || decision.PlannerSource != "main_agent" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	if len(decision.Selected) != 1 || decision.Selected[0] != "review-agent" {
		t.Fatalf("selected speakers lost: %#v", decision.Selected)
	}
	if !strings.Contains(decision.ReasonSummary, "turn 1 review-agent: reviewer should inspect") ||
		!strings.Contains(decision.ReasonSummary, "stop: review complete") {
		t.Fatalf("reasons missing: %q", decision.ReasonSummary)
	}
	if len(decision.Rejected) != 1 || decision.Rejected[0].AgentName != "code-agent" ||
		decision.Rejected[0].ReasonCode != "no_task_assigned" {
		t.Fatalf("unexpected rejected: %#v", decision.Rejected)
	}
}

func TestGroupChatRoundRobinRemainsDefault(t *testing.T) {
	t.Setenv("GROUP_CHAT_SPEAKER_MODE", "round_robin")
	t.Setenv("GROUP_CHAT_MAX_ROUNDS", "2")
	t.Setenv("ORCHESTRATOR_MAX_TASKS", "10")
	var messages []string
	var mu sync.Mutex
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		messages = append(messages, body.Params.Message.Content)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"task-1","status":"completed","events":[{"author":"agent","role":"assistant","parts":[{"type":"text","text":"ok"}],"final":true}]}`))
	}))
	defer child.Close()
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
		{Name: "review-agent", URL: child.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	selector := &fakeSpeakerSelector{choices: []planner.SpeakerChoice{{Speaker: "review-agent", Reason: "should be ignored"}}}
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: modelGroupPlan("run-round-default")}),
		WithSpeakerSelector(selector),
	)
	body := `{"runId":"run-round-default","conversationId":"conv","messages":[{"role":"user","text":"design cache"}],"requestedPath":"group_chat","selectedAgentNames":["code-agent","review-agent"]}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))

	mu.Lock()
	defer mu.Unlock()
	want := []string{"code-agent", "review-agent", "code-agent", "review-agent"}
	if len(messages) != len(want) {
		t.Fatalf("round-robin default changed: %#v", messages)
	}
	for i, agent := range want {
		if !strings.Contains(messages[i], "speaker "+agent) {
			t.Fatalf("dispatch %d speaker wrong: %s", i, messages[i])
		}
	}
	selector.mu.Lock()
	defer selector.mu.Unlock()
	if len(selector.requests) != 0 {
		t.Fatalf("selector must not be called in round-robin mode: %d", len(selector.requests))
	}
}

func TestGroupChatModelSelectionErrorFailsBeforeDispatch(t *testing.T) {
	t.Setenv("GROUP_CHAT_SPEAKER_MODE", "model")
	t.Setenv("GROUP_CHAT_MAX_ROUNDS", "1")
	selector := &fakeSpeakerSelector{err: errors.New("model unavailable")}
	store := &fakeExecutionStore{}
	srv := NewServer(
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: modelGroupPlan("run-model-error")}),
		WithSpeakerSelector(selector),
		WithExecutionStore(store),
		WithInstanceID("test-instance"),
	)
	body := `{"runId":"run-model-error","conversationId":"conv","messages":[{"role":"user","text":"design cache"}],"requestedPath":"group_chat","selectedAgentNames":["code-agent","review-agent"]}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	if !strings.Contains(rec.Body.String(), "ORCHESTRATOR_SPEAKER_SELECTION_FAILED") {
		t.Fatalf("expected speaker selection error: %s", rec.Body.String())
	}
}
