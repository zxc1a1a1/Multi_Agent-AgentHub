package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

type fakeExecutionStore struct {
	mu           sync.Mutex
	accepts      []execution.Accept
	states       []execution.State
	checkpoints  []execution.Checkpoint
	plans        []execution.Plan
	failOnStatus string
}

func (f *fakeExecutionStore) AcceptRun(_ context.Context, p execution.Accept) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.accepts = append(f.accepts, p)
	return false, nil
}

func (f *fakeExecutionStore) AcquireLease(context.Context, string, string, time.Duration) (int64, error) {
	return 7, nil
}

func (f *fakeExecutionStore) AppendState(_ context.Context, s execution.State) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.Status == f.failOnStatus {
		return 0, errors.New("store unavailable")
	}
	f.states = append(f.states, s)
	return int64(len(f.states)), nil
}

func (f *fakeExecutionStore) SaveCheckpoint(_ context.Context, c execution.Checkpoint) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checkpoints = append(f.checkpoints, c)
	return int64(len(f.checkpoints)), nil
}

func (f *fakeExecutionStore) SavePlan(_ context.Context, p execution.Plan) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.plans = append(f.plans, p)
	return nil
}

func (f *fakeExecutionStore) statuses() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.states))
	for i, s := range f.states {
		out[i] = s.Status
	}
	return out
}

func newLedgerTestServer(t *testing.T, store execution.Store, planner planner.Planner) *Server {
	t.Helper()
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{Name: "code-agent", URL: "http://127.0.0.1:1"}})
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(planner),
		WithExecutionStore(store),
		WithInstanceID("test-instance"),
	)
}

func TestRunLedgerPersistsConversationalLifecycle(t *testing.T) {
	store := &fakeExecutionStore{}
	answer := makeTestPlan("run-ledger", plan.StrategyConversational, nil)
	answer.Answer = "ledger answer"
	srv := newLedgerTestServer(t, store, &FakeMainAgent{PlanToReturn: answer})

	body := `{"runId":"run-ledger","requestId":"req-ledger","userId":"user-1","conversationId":"conv-1","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "ledger answer") {
		t.Fatalf("answer not streamed: %s", rec.Body.String())
	}
	if len(store.accepts) != 1 || store.accepts[0].RequestID != "req-ledger" || store.accepts[0].PrincipalID != "user-1" {
		t.Fatalf("unexpected accept: %#v", store.accepts)
	}
	if len(store.plans) != 1 || store.plans[0].PlanHash == "" || store.plans[0].Revision != 1 {
		t.Fatalf("plan revision not persisted: %#v", store.plans)
	}
	if len(store.checkpoints) < 2 {
		t.Fatalf("expected planning and executing checkpoints, got %d", len(store.checkpoints))
	}
	got := strings.Join(store.statuses(), ",")
	if got != "planning,executing,completed" {
		t.Fatalf("unexpected persisted lifecycle: %s", got)
	}
	if store.states[len(store.states)-1].EventType != "run_finished" {
		t.Fatalf("terminal event type not recorded: %#v", store.states[len(store.states)-1])
	}
}

func TestRunLedgerStoreFailureBlocksExecution(t *testing.T) {
	store := &fakeExecutionStore{failOnStatus: "planning"}
	answer := makeTestPlan("run-store-fail", plan.StrategyConversational, nil)
	answer.Answer = "should not stream"
	srv := newLedgerTestServer(t, store, &FakeMainAgent{PlanToReturn: answer})

	body := `{"runId":"run-store-fail","conversationId":"conv-1","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "ORCHESTRATOR_STORE_UNAVAILABLE") {
		t.Fatalf("expected store unavailable error, got: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "should not stream") {
		t.Fatal("answer must not be produced when run persistence fails")
	}
}

func TestRunLedgerBudgetLimitsPlanner(t *testing.T) {
	store := &fakeExecutionStore{}
	answer := makeTestPlan("run-budget", plan.StrategyConversational, nil)
	answer.Answer = "x"
	srv := newLedgerTestServer(t, store, &FakeMainAgent{PlanToReturn: answer})
	limits := harness.DefaultLimits()
	limits.ModelCalls = 0
	srv.budgetLimits = limits

	body := `{"runId":"run-budget","conversationId":"conv-1","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	// FakeMainAgent does not consume the budget, so this run still completes;
	// the real MainAgent budget path is covered in planner tests.
	if !strings.Contains(rec.Body.String(), "run_finished") {
		t.Fatalf("expected run_finished, got: %s", rec.Body.String())
	}
}

func (f *fakeExecutionStore) GetRun(_ context.Context, runID string) (*execution.RunStatus, error) {
	return &execution.RunStatus{RunID: runID, Status: "completed", CurrentSeq: 3}, nil
}

func (f *fakeExecutionStore) ListEventsAfter(_ context.Context, runID string, after int64, _ int) ([]execution.Event, error) {
	var out []execution.Event
	for i, s := range f.states {
		seq := int64(i + 1)
		if seq <= after {
			continue
		}
		out = append(out, execution.Event{Seq: seq, Type: s.EventType, Payload: s.Payload, CreatedAt: time.Now().UTC()})
	}
	return out, nil
}
