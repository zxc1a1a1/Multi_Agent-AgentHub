package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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
	approvals    map[string]string
	resumeState  *execution.ResumeState
	failOnStatus string
	renewCalls   int32
	renewErr     error
	releaseCalls    int32
	releaseErr      error
	signals         []execution.Signal
	taskRefs        map[string][]execution.TaskRef
	cancelRequested map[string]bool
	confirmRecords  map[string]*execution.ConfirmRecord
	confirmHashes   map[string]string
	routeDecisions  map[string][]execution.RouteDecision
}

func (f *fakeExecutionStore) SaveRouteDecision(_ context.Context, decision execution.RouteDecision) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.routeDecisions == nil {
		f.routeDecisions = map[string][]execution.RouteDecision{}
	}
	for _, existing := range f.routeDecisions[decision.RunID] {
		if existing.DecisionSeq == decision.DecisionSeq {
			return nil
		}
	}
	f.routeDecisions[decision.RunID] = append(f.routeDecisions[decision.RunID], decision)
	return nil
}

func (f *fakeExecutionStore) ListRouteDecisions(_ context.Context, runID string) ([]execution.RouteDecision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]execution.RouteDecision(nil), f.routeDecisions[runID]...), nil
}

func (f *fakeExecutionStore) BeginConfirm(_ context.Context, runID, key, requestHash string) (*execution.ConfirmRecord, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	mapKey := runID + "|" + key
	if f.confirmRecords == nil {
		f.confirmRecords = map[string]*execution.ConfirmRecord{}
		f.confirmHashes = map[string]string{}
	}
	record, ok := f.confirmRecords[mapKey]
	if !ok {
		record = &execution.ConfirmRecord{Status: "processing", CreatedAt: time.Now().UTC()}
		f.confirmRecords[mapKey] = record
		f.confirmHashes[mapKey] = requestHash
		return record, true, nil
	}
	if f.confirmHashes[mapKey] != requestHash {
		return nil, false, execution.ErrConfirmConflict
	}
	if record.Status == "completed" || record.Status == "failed" {
		return record, false, nil
	}
	return record, false, execution.ErrConfirmInFlight
}

func (f *fakeExecutionStore) CompleteConfirm(_ context.Context, runID, key, status string, httpStatus int, response []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	mapKey := runID + "|" + key
	record := f.confirmRecords[mapKey]
	if record == nil {
		record = &execution.ConfirmRecord{}
		f.confirmRecords[mapKey] = record
	}
	record.Status = status
	record.HTTPStatus = httpStatus
	record.ResponseJSON = append([]byte(nil), response...)
	return nil
}

func (f *fakeExecutionStore) RegisterRunTask(_ context.Context, runID string, ref execution.TaskRef) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.taskRefs == nil {
		f.taskRefs = map[string][]execution.TaskRef{}
	}
	for _, existing := range f.taskRefs[runID] {
		if existing.TaskID == ref.TaskID && existing.AgentName == ref.AgentName {
			return nil
		}
	}
	f.taskRefs[runID] = append(f.taskRefs[runID], ref)
	return nil
}

func (f *fakeExecutionStore) ListRunTasks(_ context.Context, runID string) ([]execution.TaskRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]execution.TaskRef(nil), f.taskRefs[runID]...), nil
}

func (f *fakeExecutionStore) RequestRunCancel(_ context.Context, runID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancelRequested == nil {
		f.cancelRequested = map[string]bool{}
	}
	f.cancelRequested[runID] = true
	return nil
}

func (f *fakeExecutionStore) RunCancelRequested(_ context.Context, runID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cancelRequested[runID], nil
}

func (f *fakeExecutionStore) AppendRunSignal(_ context.Context, sig execution.Signal) (int64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sig.IdempotencyKey != "" {
		for _, existing := range f.signals {
			if existing.IdempotencyKey == sig.IdempotencyKey {
				return existing.ID, true, nil
			}
		}
	}
	sig.ID = int64(len(f.signals) + 1)
	sig.Status = "pending"
	f.signals = append(f.signals, sig)
	return sig.ID, false, nil
}

func (f *fakeExecutionStore) ClaimRunSignal(_ context.Context, runID string) (*execution.Signal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.signals {
		if f.signals[i].RunID != runID || f.signals[i].Status != "pending" {
			continue
		}
		f.signals[i].Status = "consumed"
		sig := f.signals[i]
		return &sig, nil
	}
	return nil, nil
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

func (f *fakeExecutionStore) RenewLease(context.Context, string, string, int64, time.Duration) error {
	atomic.AddInt32(&f.renewCalls, 1)
	return f.renewErr
}

func (f *fakeExecutionStore) ReleaseLease(context.Context, string, string, int64) error {
	atomic.AddInt32(&f.releaseCalls, 1)
	return f.releaseErr
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

func (f *fakeExecutionStore) CreatePlanApproval(_ context.Context, a execution.Approval) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.approvals == nil {
		f.approvals = map[string]string{}
	}
	if status, ok := f.approvals[a.ApprovalID]; ok {
		return a.ApprovalID, status != "", nil
	}
	f.approvals[a.ApprovalID] = "pending"
	return a.ApprovalID, false, nil
}

func (f *fakeExecutionStore) ResolvePlanApproval(_ context.Context, approvalID, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.approvals == nil {
		f.approvals = map[string]string{}
	}
	f.approvals[approvalID] = status
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

func (f *fakeExecutionStore) LoadResume(_ context.Context, runID string) (*execution.ResumeState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resumeState == nil {
		return nil, errors.New("not found")
	}
	copy := *f.resumeState
	copy.RunID = runID
	return &copy, nil
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

func TestRunLedgerPersistsPlanApproval(t *testing.T) {
	store := &fakeExecutionStore{}
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{Name: "code-agent", URL: "http://127.0.0.1:1"}})
	if err != nil {
		t.Fatal(err)
	}
	p := makeTestPlan("run-approval", plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.Revision = 1
	p.Participants = []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: p}),
		WithExecutionStore(store),
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		body := `{"runId":"run-approval","conversationId":"conv","messages":[{"role":"user","text":"do it"}],"requestedPath":"main_agent_orchestration"}`
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	}()

	var planID string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		srv.hitlMu.RLock()
		pending := srv.pendingPlans["run-approval"]
		srv.hitlMu.RUnlock()
		if pending != nil {
			planID = pending.PlanID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if planID == "" {
		t.Fatal("plan was not registered for approval")
	}
	confirmBody := fmt.Sprintf(`{"runId":"run-approval","actionId":%q,"planId":%q,"action":"approve","revision":1,"idempotencyKey":"idem-approval"}`, planID, planID)
	confirmRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(confirmRec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(confirmBody)))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not finish after approval")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.approvals) != 1 {
		t.Fatalf("expected one persisted plan approval, got %#v", store.approvals)
	}
	for id, status := range store.approvals {
		if status != "approved" {
			t.Fatalf("approval %s status=%s, want approved", id, status)
		}
	}
}

func TestResumePausedRunFromCheckpoint(t *testing.T) {
	p := makeTestPlan("run-resume", plan.StrategyConversational, nil)
	p.Answer = "resumed answer"
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "paused", FencingVersion: 3, PlanJSON: planJSON,
		BudgetJSON: []byte(`{"modelCalls":3}`), CheckpointJSON: []byte(`{"status":"paused"}`),
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("resumer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/run-resume/resume", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("resume failed: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "resumed answer") {
		t.Fatalf("resumed answer missing: %s", rec.Body.String())
	}
	got := strings.Join(store.statuses(), ",")
	if !strings.Contains(got, "executing") || !strings.Contains(got, "completed") {
		t.Fatalf("resume lifecycle not persisted: %s", got)
	}
	store.mu.Lock()
	checkpoints := len(store.checkpoints)
	store.mu.Unlock()
	if checkpoints == 0 {
		t.Fatal("resume must save a checkpoint")
	}
}

func TestResumeRejectsUnreadableBudgetCheckpoint(t *testing.T) {
	p := makeTestPlan("run-resume-bad-budget", plan.StrategyConversational, nil)
	p.Answer = "must not resume"
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "paused", FencingVersion: 3, PlanJSON: planJSON,
		BudgetJSON: []byte(`{"modelCalls":`), CheckpointJSON: []byte(`{"status":"paused"}`),
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("resumer"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/run-resume-bad-budget/resume", nil))

	if rec.Code == http.StatusOK {
		t.Fatalf("resume with unreadable budget must fail closed, got 200: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "must not resume") {
		t.Fatal("resume must not execute when the budget cannot be restored")
	}
	got := strings.Join(store.statuses(), ",")
	if strings.Contains(got, "executing") || strings.Contains(got, "completed") {
		t.Fatalf("unreadable budget must not reset and execute: %s", got)
	}
}

func TestResumeRejectsMissingBudgetCheckpoint(t *testing.T) {
	p := makeTestPlan("run-resume-no-budget", plan.StrategyConversational, nil)
	p.Answer = "must not resume"
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "paused", FencingVersion: 3, PlanJSON: planJSON,
		CheckpointJSON: []byte(`{"status":"paused"}`),
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("resumer"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/run-resume-no-budget/resume", nil))

	if rec.Code == http.StatusOK {
		t.Fatalf("resume without a budget checkpoint must fail closed, got 200: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "must not resume") {
		t.Fatal("resume must not execute without a budget checkpoint")
	}
}

func TestResumeRejectsExhaustedBudgetCheckpoint(t *testing.T) {
	p := makeTestPlan("run-resume-exhausted", plan.StrategyConversational, nil)
	p.Answer = "must not resume"
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	limits := harness.DefaultLimits()
	usage, err := json.Marshal(harness.Usage{ModelCalls: limits.ModelCalls + 1})
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "paused", FencingVersion: 3, PlanJSON: planJSON,
		BudgetJSON: usage, CheckpointJSON: []byte(`{"status":"paused"}`),
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("resumer"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/run-resume-exhausted/resume", nil))

	if rec.Code == http.StatusOK {
		t.Fatalf("resume with an over-limit budget must fail closed, got 200: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "must not resume") {
		t.Fatal("resume must not execute when checkpoint usage exceeds limits")
	}
}

func TestResumeRejectsNonPausedRun(t *testing.T) {
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{Status: "completed"}}
	srv := NewServer(WithExecutionStore(store))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/run-done/resume", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRunLedgerRenewsLease(t *testing.T) {
	store := &fakeExecutionStore{}
	ledger := &runLedger{
		store: store, runID: "run-renew", fencing: 7, owner: "owner",
		leaseTTL: time.Second, renewInterval: 5 * time.Millisecond,
	}
	lost := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ledger.startLeaseRenewal(ctx, func() { lost <- struct{}{} })
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&store.renewCalls) >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	ledger.stopLeaseRenewal()
	if got := atomic.LoadInt32(&store.renewCalls); got < 3 {
		t.Fatalf("lease was renewed %d times, want at least 3", got)
	}
	select {
	case <-lost:
		t.Fatal("healthy lease must not report loss")
	default:
	}
}

func TestRunLedgerLeaseLossCancelsRun(t *testing.T) {
	store := &fakeExecutionStore{renewErr: errors.New("run lease is held by another owner")}
	ledger := &runLedger{
		store: store, runID: "run-lost", fencing: 7, owner: "owner",
		leaseTTL: time.Second, renewInterval: 5 * time.Millisecond,
	}
	lost := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ledger.startLeaseRenewal(ctx, func() { lost <- struct{}{} })
	select {
	case <-lost:
	case <-time.After(2 * time.Second):
		t.Fatal("lease renewal failure must cancel the run")
	}
	ledger.stopLeaseRenewal()
}

func TestRunHandlerRenewsLeaseDuringExecution(t *testing.T) {
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		http.Error(w, "agent unavailable", http.StatusInternalServerError)
	}))
	defer child.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{Name: "code-agent", URL: child.URL}})
	if err != nil {
		t.Fatal(err)
	}
	p := makeTestPlan("run-renew-live", plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.ExecutionPath = "single_chat"
	p.AllowedAgents = []string{"code-agent"}
	p.Participants = []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	store := &fakeExecutionStore{}
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: p}),
		WithExecutionStore(store),
		WithInstanceID("lease-owner"),
	)
	srv.leaseRenewInterval = 10 * time.Millisecond

	body := `{"runId":"run-renew-live","conversationId":"conv","messages":[{"role":"user","text":"do it"}],"requestedPath":"single_chat","agentName":"code-agent"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))

	if got := atomic.LoadInt32(&store.renewCalls); got == 0 {
		t.Fatalf("run handler did not renew the lease during execution; body=%s", rec.Body.String())
	}
}

func TestDurableHITLConfirmFromSecondInstance(t *testing.T) {
	p := makeTestPlan("run-cross", plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.Revision = 1
	p.Participants = []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "awaiting_approval", PlanJSON: planJSON,
		BudgetJSON: []byte(`{}`), CheckpointJSON: []byte(`{"status":"awaiting_approval"}`),
	}}
	srvA := newLedgerTestServer(t, store, &FakeMainAgent{PlanToReturn: p})
	srvA.hitlPollInterval = 5 * time.Millisecond

	done := make(chan struct{})
	go func() {
		defer close(done)
		body := `{"runId":"run-cross","conversationId":"conv","messages":[{"role":"user","text":"do it"}],"requestedPath":"main_agent_orchestration"}`
		srvA.Handler().ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	}()

	var planID string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		srvA.hitlMu.RLock()
		pending := srvA.pendingPlans["run-cross"]
		srvA.hitlMu.RUnlock()
		if pending != nil {
			planID = pending.PlanID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if planID == "" {
		t.Fatal("plan was not registered for approval")
	}

	srvB := NewServer(WithExecutionStore(store), WithInstanceID("instance-b"))
	confirmBody := fmt.Sprintf(`{"runId":"run-cross","actionId":%q,"planId":%q,"action":"approve","revision":1,"idempotencyKey":"idem-cross","selectedParticipants":["code-agent"]}`, planID, planID)
	rec := httptest.NewRecorder()
	srvB.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(confirmBody)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("second instance confirm failed: %d %s", rec.Code, rec.Body.String())
	}

	firstBody := rec.Body.String()
	rec = httptest.NewRecorder()
	srvB.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(confirmBody)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("idempotent confirm must replay the stored 202 response, got %d %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != firstBody {
		t.Fatalf("idempotent confirm must replay the stored body: first=%s second=%s", firstBody, rec.Body.String())
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("owning instance did not consume the durable signal")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.signals) != 1 {
		t.Fatalf("expected one durable signal, got %d", len(store.signals))
	}
	if store.signals[0].Status != "consumed" {
		t.Fatalf("signal status=%q, want consumed", store.signals[0].Status)
	}
	for id, status := range store.approvals {
		if status != "approved" {
			t.Fatalf("approval %s status=%s, want approved", id, status)
		}
	}
}

func TestDurableHITLConfirmRejectsRevisionMismatch(t *testing.T) {
	p := makeTestPlan("run-cross-bad", plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.Revision = 1
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "awaiting_approval", PlanJSON: planJSON,
		BudgetJSON: []byte(`{}`), CheckpointJSON: []byte(`{}`),
	}}
	srvB := NewServer(WithExecutionStore(store), WithInstanceID("instance-b"))
	body := fmt.Sprintf(`{"runId":"run-cross-bad","actionId":%q,"planId":%q,"action":"approve","revision":2,"idempotencyKey":"idem-bad"}`, p.PlanID, p.PlanID)
	rec := httptest.NewRecorder()
	srvB.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(body)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("revision mismatch must be rejected, got %d %s", rec.Code, rec.Body.String())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.signals) != 0 {
		t.Fatal("rejected confirmation must not append a signal")
	}
}

func TestRunHandlerReleasesLeaseOnCompletion(t *testing.T) {
	store := &fakeExecutionStore{}
	answer := makeTestPlan("run-release", plan.StrategyConversational, nil)
	answer.Answer = "done"
	srv := newLedgerTestServer(t, store, &FakeMainAgent{PlanToReturn: answer})
	body := `{"runId":"run-release","conversationId":"conv","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	if got := atomic.LoadInt32(&store.releaseCalls); got != 1 {
		t.Fatalf("completed run released the lease %d times, want 1", got)
	}
}

func TestRunHandlerReleasesLeaseOnPlannerFailure(t *testing.T) {
	store := &fakeExecutionStore{}
	srv := newLedgerTestServer(t, store, &FakeMainAgent{PlanErr: errors.New("planner failed")})
	body := `{"runId":"run-release-fail","conversationId":"conv","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))
	if got := atomic.LoadInt32(&store.releaseCalls); got != 1 {
		t.Fatalf("failed run released the lease %d times, want 1", got)
	}
}

func TestResumeReleasesLease(t *testing.T) {
	p := makeTestPlan("run-resume-release", plan.StrategyConversational, nil)
	p.Answer = "resumed"
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "paused", FencingVersion: 3, PlanJSON: planJSON,
		BudgetJSON: []byte(`{"modelCalls":1}`), CheckpointJSON: []byte(`{"status":"paused"}`),
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("resumer"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/run-resume-release/resume", nil))
	if got := atomic.LoadInt32(&store.releaseCalls); got != 1 {
		t.Fatalf("resume released the lease %d times, want 1", got)
	}
}

func TestCancelOnSecondInstanceUsesPersistedTaskRefs(t *testing.T) {
	var cancelCalls int32
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cancelCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"task-1","status":"cancelled"}`))
	}))
	defer child.Close()

	store := &fakeExecutionStore{taskRefs: map[string][]execution.TaskRef{
		"run-cancel": {{TaskID: "task-1", AgentName: "code-agent", AgentURL: child.URL}},
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("instance-b"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/cancel",
		strings.NewReader(`{"runId":"run-cancel"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel failed: %d %s", rec.Code, rec.Body.String())
	}
	if got := atomic.LoadInt32(&cancelCalls); got != 1 {
		t.Fatalf("remote task cancel calls=%d, want 1", got)
	}
	if !store.cancelRequested["run-cancel"] {
		t.Fatal("cancel request was not persisted for the owning instance")
	}
}

func TestCancelWatcherStopsLocalRun(t *testing.T) {
	store := &fakeExecutionStore{cancelRequested: map[string]bool{"run-watch": true}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("instance-a"))
	srv.hitlPollInterval = 5 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan struct{})
	go srv.watchRunCancel(ctx, "run-watch", func() {
		close(stopped)
		cancel()
	})
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("persisted cancel request did not stop the local run")
	}
}

func TestConfirmIdempotencyRejectsDifferentPayload(t *testing.T) {
	p := makeTestPlan("run-idem-conflict", plan.StrategySingle, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "do it")})
	p.Revision = 1
	planJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{
		Status: "awaiting_approval", PlanJSON: planJSON,
		BudgetJSON: []byte(`{}`), CheckpointJSON: []byte(`{}`),
	}}
	srv := NewServer(WithExecutionStore(store), WithInstanceID("instance-b"))
	approve := fmt.Sprintf(`{"runId":"run-idem-conflict","actionId":%q,"planId":%q,"action":"approve","revision":1,"idempotencyKey":"same-key"}`, p.PlanID, p.PlanID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(approve)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first confirm failed: %d %s", rec.Code, rec.Body.String())
	}

	cancel := fmt.Sprintf(`{"runId":"run-idem-conflict","actionId":%q,"planId":%q,"action":"cancel","revision":1,"idempotencyKey":"same-key"}`, p.PlanID, p.PlanID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/hitl/confirm", strings.NewReader(cancel)))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "IDEMPOTENCY_KEY_CONFLICT") {
		t.Fatalf("key reuse with different payload must be rejected: %d %s", rec.Code, rec.Body.String())
	}
}
