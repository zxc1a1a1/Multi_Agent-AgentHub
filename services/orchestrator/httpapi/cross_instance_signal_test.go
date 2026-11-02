package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// TestTwoInstancesShareHITLSignal exercises the cross-instance path with a
// shared in-memory store: instance A owns the run, instance B confirms it and
// A consumes the durable signal.
func TestTwoInstancesShareHITLSignal(t *testing.T) {
	var dispatches int32
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&dispatches, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"task-remote","status":"completed","events":[{"author":"code-agent","role":"assistant","parts":[{"type":"text","text":"done"}],"final":true}]}`))
	}))
	defer child.Close()
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{Name: "code-agent", URL: child.URL}})
	if err != nil {
		t.Fatal(err)
	}
	runID := fmt.Sprintf("run-mem-two-%d", time.Now().UnixNano())
	planID := "plan-" + runID
	p := makeTestPlan(runID, plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.PlanID = planID
	p.Revision = 1
	p.ExecutionPath = "main_agent_orchestration"
	p.Participants = []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "awaiting_approval", FencingVersion: 7, PlanJSON: planJSON,
	}}
	srvA := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: p}),
		WithExecutionStore(store),
		WithInstanceID("instance-a"),
	)
	srvA.hitlPollInterval = 10 * time.Millisecond

	done := make(chan struct{})
	go func() {
		defer close(done)
		body := fmt.Sprintf(`{"runId":%q,"requestId":"req-%s","userId":"user-two","conversationId":"conv-two","messages":[{"role":"user","text":"do it"}],"requestedPath":"main_agent_orchestration"}`, runID, runID)
		srvA.Handler().ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	}()
	waitFor(t, func() bool {
		srvA.hitlMu.RLock()
		defer srvA.hitlMu.RUnlock()
		return srvA.pendingPlans[runID] != nil
	}, 5*time.Second, "instance A pending plan")

	srvB := NewServer(WithExecutionStore(store), WithInstanceID("instance-b"))
	confirm := fmt.Sprintf(`{"runId":%q,"actionId":%q,"planId":%q,"action":"approve","revision":1,"idempotencyKey":"idem-%s","selectedParticipants":["code-agent"]}`, runID, planID, planID, runID)
	rec := httptest.NewRecorder()
	srvB.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(confirm)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("instance B confirm failed: %d %s", rec.Code, rec.Body.String())
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("instance A did not finish after cross-instance approval")
	}
	if got := atomic.LoadInt32(&dispatches); got != 1 {
		t.Fatalf("dispatches=%d, want 1", got)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.signals) == 0 || store.signals[0].Status != "consumed" {
		t.Fatalf("unexpected signals: %#v", store.signals)
	}
}

func TestTwoInstancesSharePendingCancel(t *testing.T) {
	var dispatches int32
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&dispatches, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"task-remote","status":"completed","events":[{"author":"code-agent","role":"assistant","parts":[{"type":"text","text":"done"}],"final":true}]}`))
	}))
	defer child.Close()
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{Name: "code-agent", URL: child.URL}})
	if err != nil {
		t.Fatal(err)
	}
	runID := fmt.Sprintf("run-cancel-two-%d", time.Now().UnixNano())
	planID := "plan-" + runID
	p := makeTestPlan(runID, plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.PlanID = planID
	p.Revision = 1
	p.ExecutionPath = "main_agent_orchestration"
	p.Participants = []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "awaiting_approval", FencingVersion: 7, PlanJSON: planJSON,
	}}
	srvA := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: p}),
		WithExecutionStore(store),
		WithInstanceID("instance-a"),
	)
	srvA.hitlPollInterval = 10 * time.Millisecond

	done := make(chan struct{})
	go func() {
		defer close(done)
		body := fmt.Sprintf(`{"runId":%q,"requestId":"req-%s","userId":"user-two","conversationId":"conv-two","messages":[{"role":"user","text":"do it"}],"requestedPath":"main_agent_orchestration"}`, runID, runID)
		srvA.Handler().ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	}()
	waitFor(t, func() bool {
		srvA.hitlMu.RLock()
		defer srvA.hitlMu.RUnlock()
		return srvA.pendingPlans[runID] != nil
	}, 5*time.Second, "instance A pending plan")

	srvB := NewServer(WithExecutionStore(store), WithInstanceID("instance-b"))
	rec := httptest.NewRecorder()
	srvB.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/cancel",
		strings.NewReader(fmt.Sprintf(`{"runId":%q}`, runID))))
	if rec.Code != http.StatusOK {
		t.Fatalf("instance B cancel failed: %d %s", rec.Code, rec.Body.String())
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("instance A did not stop after cross-instance cancel")
	}
	if got := atomic.LoadInt32(&dispatches); got != 0 {
		t.Fatalf("cancelled run dispatched %d task(s)", got)
	}
}

func waitFor(t *testing.T, condition func() bool, timeout time.Duration, label string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", label)
}
