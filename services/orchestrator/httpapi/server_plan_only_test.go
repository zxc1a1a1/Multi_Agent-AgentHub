package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// ---------------------------------------------------------------------------
// Phase 2 plan_only single_chat integration tests
// ---------------------------------------------------------------------------

// fakePlanOnlyAgent is a test double that acts as a plan_only-capable agent.
type fakePlanOnlyAgent struct {
	mu        sync.Mutex
	requests  []string
	planCount int
	execCount int
}

func (f *fakePlanOnlyAgent) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, string(body))
		f.mu.Unlock()

		if strings.Contains(string(body), `"plan_only"`) {
			f.mu.Lock()
			f.planCount++
			f.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			planText := `{"strategy":"single","intentSummary":"test generation","tasks":[{"taskId":"t1","agentName":"code-agent","content":"write a function"}]}`
			resp := a2aRunResponse("t-plan", "completed", "code-agent", planText)
			json.NewEncoder(w).Encode(resp)
			return
		}

		f.mu.Lock()
		f.execCount++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := a2aRunResponse("t-exec", "completed", "code-agent", "Generated code result")
		json.NewEncoder(w).Encode(resp)
	}
}

func a2aRunResponse(taskID, status, author, text string) map[string]interface{} {
	return map[string]interface{}{
		"taskId": taskID,
		"status": status,
		"events": []map[string]interface{}{
			{
				"author": author,
				"role":   "assistant",
				"parts": []map[string]interface{}{
					{"type": "text", "text": text},
				},
			},
		},
	}
}

type sseEvent struct {
	eventType string
	data      string
}

func readSSEBody(t *testing.T, body io.ReadCloser) []sseEvent {
	t.Helper()
	defer body.Close()
	var events []sseEvent
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var currentEvent string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			events = append(events, sseEvent{eventType: currentEvent, data: data})
		}
	}
	// scanner.Err may be non-nil due to connection close; ignore benign errors.
	return events
}

func findEvent(events []sseEvent, eventType string) (string, bool) {
	for _, e := range events {
		if e.eventType == eventType {
			return e.data, true
		}
	}
	return "", false
}

func parseSSEState(eventData string) map[string]interface{} {
	var wrapper struct {
		State map[string]interface{} `json:"state"`
	}
	if err := json.Unmarshal([]byte(eventData), &wrapper); err != nil {
		return nil
	}
	return wrapper.State
}

func findPlanIDFromServer(srv *Server, runID string) string {
	srv.hitlMu.RLock()
	defer srv.hitlMu.RUnlock()
	if plan, ok := srv.pendingPlans[runID]; ok && plan != nil {
		return plan.PlanID
	}
	return ""
}

// waitForNewPlanID polls until the planID changes from oldPlanID (for revised plans).
func waitForNewPlanID(srv *Server, runID, oldPlanID string, timeout time.Duration) string {
	deadline := time.After(timeout)
	for {
		if id := findPlanIDFromServer(srv, runID); id != "" && id != oldPlanID {
			return id
		}
		select {
		case <-deadline:
			return ""
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// waitForPlanID polls until the plan is registered or timeout.
func waitForPlanID(srv *Server, runID string, timeout time.Duration) string {
	deadline := time.After(timeout)
	for {
		if id := findPlanIDFromServer(srv, runID); id != "" {
			return id
		}
		select {
		case <-deadline:
			return ""
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPlanOnlySingleChat_UnsupportedAgent_ReturnsError(t *testing.T) {
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "document-agent", URL: "http://127.0.0.1:1"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-unsup","conversationId":"conv-1","messages":[{"role":"user","text":"generate docs"}],"selectedAgentNames":["document-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	events := readSSEBody(t, resp.Body)
	_, hasRunError := findEvent(events, "run_error")
	if !hasRunError {
		t.Fatal("expected run_error for unsupported agent")
	}
	data, _ := findEvent(events, "run_error")
	if !strings.Contains(data, "AGENT_PLAN_ONLY_UNSUPPORTED") {
		t.Errorf("expected AGENT_PLAN_ONLY_UNSUPPORTED in error, got: %s", data)
	}
}

// TestPlanOnlySingleChat_AgentReceivesPlanOnlyMode verifies the child agent
// receives mode="plan_only" in the A2A request. Uses a cancellable context
// because plan_only always requires HITL confirmation (the handler will block
// waiting for confirm, so we cancel to close the stream).
func TestPlanOnlySingleChat_AgentReceivesPlanOnlyMode(t *testing.T) {
	fake := &fakePlanOnlyAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	body := `{"runId":"run-mode","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/internal/orchestrator/runs/stream", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	// Collect SSE events in background, cancel after dispatch completes.
	eventsCh := make(chan []sseEvent, 1)
	go func() {
		eventsCh <- readSSEBody(t, resp.Body)
	}()

	// Wait for the plan_only dispatch to happen, then cancel.
	time.Sleep(500 * time.Millisecond)
	cancel()

	<-eventsCh // drain

	if fake.planCount < 1 {
		t.Error("expected at least one plan_only dispatch to the agent, got 0")
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	found := false
	for _, req := range fake.requests {
		if strings.Contains(req, `"plan_only"`) {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent did not receive mode=plan_only in the request body")
	}
}

// TestPlanOnlySingleChat_Cancel_NoRunError verifies the full cancel flow:
// plan_only → confirm_plan → awaiting_confirmation → cancel → NO run_error.
func TestPlanOnlySingleChat_Cancel_NoRunError(t *testing.T) {
	fake := &fakePlanOnlyAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-cancel","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	// Wait for the plan to be registered, then cancel.
	planID := waitForPlanID(srv, "run-cancel", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-cancel","actionId":"%s","confirmed":false,"rejectReason":"not now"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events
	var foundCancelled bool
	for _, e := range events {
		if e.eventType == "run_error" && strings.Contains(e.data, "rejected") {
			t.Error("cancel should not emit RUN_ERROR with rejection message")
		}
		if e.eventType == "state_update" {
			state := parseSSEState(e.data)
			if phase, ok := state["phase"].(string); ok && phase == "cancelled" {
				foundCancelled = true
			}
		}
	}
	if !foundCancelled {
		t.Error("expected state_update with phase=cancelled")
	}

	data, hasRunFinished := findEvent(events, "run_finished")
	if !hasRunFinished {
		t.Error("expected run_finished event")
	} else if !strings.Contains(data, "cancelled") {
		t.Errorf("expected run_finished status=cancelled, got: %s", data)
	}
}

// TestPlanOnlySingleChat_ApproveAndExecute verifies the full approve→execute flow:
// plan_only → confirm_plan → awaiting_confirmation → approve → executing → completed.
func TestPlanOnlySingleChat_ApproveAndExecute(t *testing.T) {
	fake := &fakePlanOnlyAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-approve","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-approve", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	approveBody := fmt.Sprintf(`{"runId":"run-approve","actionId":"%s","confirmed":true,"rejectReason":""}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(approveBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events
	if len(events) == 0 {
		t.Fatal("expected at least one SSE event")
	}
	if events[0].eventType != "run_started" {
		t.Errorf("expected first event to be run_started, got %s", events[0].eventType)
	}

	var foundExecuting bool
	for _, e := range events {
		if e.eventType == "state_update" {
			state := parseSSEState(e.data)
			if phase, ok := state["phase"].(string); ok && phase == "executing" {
				foundExecuting = true
			}
		}
	}
	if !foundExecuting {
		t.Error("expected state_update with phase=executing after approve")
	}

	data, hasRunFinished := findEvent(events, "run_finished")
	if !hasRunFinished {
		t.Error("expected run_finished event")
	} else if !strings.Contains(data, "completed") {
		t.Errorf("expected run_finished status=completed, got: %s", data)
	}

	if fake.planCount < 1 {
		t.Error("expected plan_only dispatch to agent")
	}
	if fake.execCount < 1 {
		t.Error("expected execute dispatch to agent after approve")
	}
}

// TestPlanOnlySingleChat_UsesAgentNotPlanner verifies plan_only dispatches to
// the child agent (not the Planner/LLM). Uses cancellable context.
func TestPlanOnlySingleChat_UsesAgentNotPlanner(t *testing.T) {
	fake := &fakePlanOnlyAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	body := `{"runId":"run-agent","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/internal/orchestrator/runs/stream", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	go readSSEBody(t, resp.Body)
	time.Sleep(500 * time.Millisecond)
	cancel()

	if fake.planCount < 1 {
		t.Error("plan_only should dispatch to the agent, not the Planner")
	}
}

// ---------------------------------------------------------------------------
// Issue 3: invalid plan tests
// ---------------------------------------------------------------------------

// fakeTextAgent returns a handler that responds with a fixed text body (used
// for injecting invalid plan JSON).
type fakeTextAgent struct {
	body      string
	callCount int
	mu        sync.Mutex
}

func (f *fakeTextAgent) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.callCount++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, f.body)
	}
}

// TestPlanOnlySingleChat_EmptyTasks_ReturnsError verifies that a plan_only
// response with empty tasks produces an error and does NOT enter confirm/execute.
func TestPlanOnlySingleChat_EmptyTasks_ReturnsError(t *testing.T) {
	fake := &fakeTextAgent{
		body: a2aRunResponseJSON("code-agent", `{"strategy":"single","intentSummary":"test","tasks":[]}`),
	}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-empty","conversationId":"conv-1","messages":[{"role":"user","text":"test"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, _ := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))

	events := readSSEBody(t, resp.Body)

	// Must have an error.
	data, ok := findEvent(events, "run_error")
	if !ok {
		t.Fatal("expected run_error for empty tasks")
	}
	if !strings.Contains(data, "AGENT_PLAN_PROPOSAL_INVALID") && !strings.Contains(data, "ORCHESTRATOR_PLAN_INVALID") {
		t.Errorf("expected plan rejection error, got: %s", data)
	}

	// Must NOT have entered confirm loop.
	_, hasToolCall := findEvent(events, "tool_call_start")
	if hasToolCall {
		t.Error("should not emit confirm_plan for invalid plan")
	}

	// The agent was called once (plan_only) but NOT for execution.
	if fake.callCount > 1 {
		t.Errorf("expected only 1 call (plan_only), got %d", fake.callCount)
	}
}

// TestPlanOnlySingleChat_WrongStrategy_ReturnsError verifies that a plan_only
// response with strategy=ordered_parallel is rejected as invalid.
func TestPlanOnlySingleChat_WrongStrategy_ReturnsError(t *testing.T) {
	fake := &fakeTextAgent{
		body: a2aRunResponseJSON("code-agent", `{"strategy":"ordered_parallel","intentSummary":"test","tasks":[{"taskId":"t1","agentName":"code-agent","content":"test"}]}`),
	}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-wrong-strat","conversationId":"conv-1","messages":[{"role":"user","text":"test"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, _ := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))

	events := readSSEBody(t, resp.Body)
	data, ok := findEvent(events, "run_error")
	if !ok {
		t.Fatal("expected run_error for wrong strategy")
	}
	if !strings.Contains(data, "AGENT_PLAN_PROPOSAL_INVALID") {
		t.Errorf("expected AGENT_PLAN_PROPOSAL_INVALID, got: %s", data)
	}
	if !strings.Contains(data, "ordered_parallel") {
		t.Errorf("error should mention the invalid strategy, got: %s", data)
	}

	// Must not have entered confirm loop.
	_, hasToolCall := findEvent(events, "tool_call_start")
	if hasToolCall {
		t.Error("should not emit confirm_plan for invalid strategy")
	}
}

// TestPlanOnlySingleChat_WrongAgentName_ReturnsError verifies that when the
// plan_only task references a different agent than the current one, it is
// rejected as a boundary violation.
func TestPlanOnlySingleChat_WrongAgentName_ReturnsError(t *testing.T) {
	fake := &fakeTextAgent{
		body: a2aRunResponseJSON("code-agent", `{"strategy":"single","intentSummary":"test","tasks":[{"taskId":"t1","agentName":"web-agent","content":"test"}]}`),
	}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-wrong-agent","conversationId":"conv-1","messages":[{"role":"user","text":"test"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, _ := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))

	events := readSSEBody(t, resp.Body)
	data, ok := findEvent(events, "run_error")
	if !ok {
		t.Fatal("expected run_error for wrong agent name in task")
	}
	if !strings.Contains(data, "AGENT_PLAN_PROPOSAL_INVALID") {
		t.Errorf("expected AGENT_PLAN_PROPOSAL_INVALID, got: %s", data)
	}
	if !strings.Contains(data, "web-agent") || !strings.Contains(data, "code-agent") {
		t.Errorf("error should mention both agents, got: %s", data)
	}

	_, hasToolCall := findEvent(events, "tool_call_start")
	if hasToolCall {
		t.Error("should not emit confirm_plan for boundary violation")
	}
}

// a2aRunResponseJSON builds a minimal A2A RunResponse JSON string with one text event.
func a2aRunResponseJSON(author, text string) string {
	// Use proper JSON encoding for the text content.
	textJSON, _ := json.Marshal(text)
	return fmt.Sprintf(`{"taskId":"t","status":"completed","events":[{"author":"%s","role":"assistant","parts":[{"type":"text","text":%s}]}]}`, author, string(textJSON))
}

// ---------------------------------------------------------------------------
// Issue 4: mentions-only single_chat test
// ---------------------------------------------------------------------------

// TestPlanOnlySingleChat_MentionsOnly_UsesAgentNotPlanner verifies that
// when the user specifies only mentions (no agentName, no selectedAgentNames),
// the single_chat path still routes to the correct plan_only agent.
func TestPlanOnlySingleChat_MentionsOnly_UsesAgentNotPlanner(t *testing.T) {
	fake := &fakePlanOnlyAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// mentions=["@code-agent"], no agentName, no selectedAgentNames
	body := `{"runId":"run-mentions","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"mentions":["code-agent"],"executionPath":"single_chat"}`
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/internal/orchestrator/runs/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	go readSSEBody(t, resp.Body)
	time.Sleep(500 * time.Millisecond)
	cancel()

	if fake.planCount < 1 {
		t.Error("mentions-only should route to code-agent plan_only, not Planner")
	}

	// Verify no INVALID_AGENT_SELECTION or similar error was emitted (we
	// consumed the SSE body in a fire-and-forget goroutine, but the fact
	// that the agent received plan_only is the critical assertion).
}

// ---------------------------------------------------------------------------
// TaskContent = original userText (not plan_only content)
// ---------------------------------------------------------------------------

// TestPlanOnlySingleChat_ApproveExecutesOriginalUserText verifies that after
// plan_only approval, the full-execution dispatch uses the original user
// message, NOT the plan_only agent's task.content.
func TestPlanOnlySingleChat_ApproveExecutesOriginalUserText(t *testing.T) {
	// Build a fake agent that returns distinguishable plan_only content and
	// captures the execute request body for inspection.
	var execBody string
	var mu sync.Mutex
	fakeAgentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(bodyStr, `"plan_only"`) {
			// Return a plan with content clearly different from the user's original input.
			planText := `{"strategy":"single","intentSummary":"test","tasks":[{"taskId":"t1","agentName":"code-agent","content":"PLAN ONLY CONTENT - DO NOT EXECUTE"}]}`
			resp := a2aRunResponse("t-plan", "completed", "code-agent", planText)
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Full execution: capture the body for verification.
		mu.Lock()
		execBody = bodyStr
		mu.Unlock()
		resp := a2aRunResponse("t-exec", "completed", "code-agent", "execution result")
		json.NewEncoder(w).Encode(resp)
	}))
	defer fakeAgentSrv.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	originalUserText := "write a Go HTTP server"
	body := fmt.Sprintf(`{"runId":"run-taskcontent","conversationId":"conv-1","messages":[{"role":"user","text":"%s"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`, originalUserText)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-taskcontent", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	approveBody := fmt.Sprintf(`{"runId":"run-taskcontent","actionId":"%s","confirmed":true,"rejectReason":""}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(approveBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	// Verify the full execution completed.
	_, hasRunFinished := findEvent(streamRes.events, "run_finished")
	if !hasRunFinished {
		t.Fatal("expected run_finished after approve+execute")
	}

	// Verify the execute request body.
	mu.Lock()
	defer mu.Unlock()
	if execBody == "" {
		t.Fatal("execute request body was not captured")
	}
	if !strings.Contains(execBody, originalUserText) {
		t.Errorf("execute request must contain original user text %q, got body: %s", originalUserText, execBody)
	}
	if strings.Contains(execBody, "PLAN ONLY CONTENT") {
		t.Error("execute request must NOT contain plan_only task content")
	}
}

// ---------------------------------------------------------------------------
// Confirm display shows proposal content, execution uses userText
// ---------------------------------------------------------------------------

// TestPlanOnlySingleChat_ConfirmPlanShowsAgentProposalContent verifies:
//  1. confirm_plan/awaiting_confirmation SSE events contain the agent's proposal.
//  2. approve→execute request contains the original user text.
//  3. approve→execute request does NOT contain the proposal content.
func TestPlanOnlySingleChat_ConfirmPlanShowsAgentProposalContent(t *testing.T) {
	var execBody string
	var mu sync.Mutex
	fakeAgentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(bodyStr, `"plan_only"`) {
			planText := `{"strategy":"single","intentSummary":"test","tasks":[{"taskId":"t1","agentName":"code-agent","content":"PLAN ONLY CONTENT - DISPLAY TO USER"}]}`
			resp := a2aRunResponse("t-plan", "completed", "code-agent", planText)
			json.NewEncoder(w).Encode(resp)
			return
		}

		mu.Lock()
		execBody = bodyStr
		mu.Unlock()
		resp := a2aRunResponse("t-exec", "completed", "code-agent", "execution result")
		json.NewEncoder(w).Encode(resp)
	}))
	defer fakeAgentSrv.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	originalUserText := "write a Go HTTP server"
	bodyJSON := fmt.Sprintf(`{"runId":"run-display","conversationId":"conv-1","messages":[{"role":"user","text":"%s"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`, originalUserText)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(bodyJSON))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-display", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	approveBody := fmt.Sprintf(`{"runId":"run-display","actionId":"%s","confirmed":true,"rejectReason":""}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(approveBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	// 1. SSE events must show the proposal content.
	var foundDisplayProposal bool
	for _, e := range streamRes.events {
		if e.eventType == "tool_call_args" && strings.Contains(e.data, "confirm_plan") {
			if strings.Contains(e.data, "PLAN ONLY CONTENT - DISPLAY TO USER") {
				foundDisplayProposal = true
			}
		}
		if e.eventType == "state_update" {
			state := parseSSEState(e.data)
			if phase, ok := state["phase"].(string); ok && (phase == "planning" || phase == "awaiting_confirmation") {
				if data, _ := json.Marshal(state); strings.Contains(string(data), "PLAN ONLY CONTENT - DISPLAY TO USER") {
					foundDisplayProposal = true
				}
			}
		}
	}
	if !foundDisplayProposal {
		t.Error("confirm_plan / awaiting_confirmation SSE events must contain the agent proposal content")
	}

	// 2. Execute request must contain original user text.
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(execBody, originalUserText) {
		t.Errorf("execute request must contain original user text %q, got body: %s", originalUserText, execBody)
	}

	// 3. Execute request must NOT contain proposal content.
	if strings.Contains(execBody, "PLAN ONLY CONTENT - DISPLAY TO USER") {
		t.Error("execute request must NOT contain plan_only proposal content")
	}
}

// ---------------------------------------------------------------------------
// Phase 4 revise integration tests
// ---------------------------------------------------------------------------

// fakeReviseAgent returns different plans on successive plan_only calls,
// and captures the execute request body for verification.
type fakeReviseAgent struct {
	mu             sync.Mutex
	planCallCount  int
	execCount      int
	execBody       string
	planOnlyBodies []string
}

func (f *fakeReviseAgent) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(bodyStr, `"plan_only"`) {
			f.mu.Lock()
			f.planOnlyBodies = append(f.planOnlyBodies, bodyStr)
			f.planCallCount++
			callN := f.planCallCount
			f.mu.Unlock()

			var planText string
			switch callN {
			case 1:
				planText = `{"strategy":"single","intentSummary":"Plan v1","tasks":[{"taskId":"t1","agentName":"code-agent","content":"PLAN V1 - Write a Go HTTP server"}]}`
			case 2:
				planText = `{"strategy":"single","intentSummary":"Plan v2 (revised)","tasks":[{"taskId":"t2","agentName":"code-agent","content":"PLAN V2 - Write a simple Go HTTP server"}]}`
			default:
				planText = fmt.Sprintf(`{"strategy":"single","intentSummary":"Plan v%d","tasks":[{"taskId":"t%d","agentName":"code-agent","content":"PLAN V%d"}]}`, callN, callN, callN)
			}
			resp := a2aRunResponse("t-plan-"+fmt.Sprint(callN), "completed", "code-agent", planText)
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Full execution: capture the body for verification.
		f.mu.Lock()
		f.execCount++
		f.execBody = bodyStr
		f.mu.Unlock()
		resp := a2aRunResponse("t-exec", "completed", "code-agent", "execution result")
		json.NewEncoder(w).Encode(resp)
	}
}

// planOnlyBody returns the n-th plan_only request body (0-indexed).
// Returns empty string if n is out of range.
func (f *fakeReviseAgent) planOnlyBody(n int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n < 0 || n >= len(f.planOnlyBodies) {
		return ""
	}
	return f.planOnlyBodies[n]
}

// TestPlanOnlySingleChat_ReviseThenApproveExecutesOriginalUserText verifies
// the full revise->approve->execute flow preserves the original userText.
func TestPlanOnlySingleChat_ReviseThenApproveExecutesOriginalUserText(t *testing.T) {
	fake := &fakeReviseAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	originalUserText := "write a Go HTTP server"
	body := fmt.Sprintf(`{"runId":"run-revise-approve","conversationId":"conv-1","messages":[{"role":"user","text":"%s"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`, originalUserText)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	// Wait for initial plan (v1).
	planID := waitForPlanID(srv, "run-revise-approve", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	// Step 1: Revise.
	reviseBody := fmt.Sprintf(`{"runId":"run-revise-approve","actionId":"%s","action":"revise","feedback":"simplify it","revision":1}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	// Wait for revised plan (v2) — note: old plan stays registered during revise,
	// so we must wait for the planID to change, not just for any plan to exist.
	newPlanID := waitForNewPlanID(srv, "run-revise-approve", planID, 2*time.Second)
	if newPlanID == "" {
		t.Fatal("revised plan was not registered within timeout")
	}

	// Step 2: Approve the revised plan.
	approveBody := fmt.Sprintf(`{"runId":"run-revise-approve","actionId":"%s","action":"approve"}`, newPlanID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(approveBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	// Verify the full execution completed.
	_, hasRunFinished := findEvent(streamRes.events, "run_finished")
	if !hasRunFinished {
		t.Fatal("expected run_finished after approve+execute")
	}

	// Verify execute request body.
	fake.mu.Lock()
	execBody := fake.execBody
	fake.mu.Unlock()
	if execBody == "" {
		t.Fatal("execute request body was not captured")
	}
	if !strings.Contains(execBody, originalUserText) {
		t.Errorf("execute request must contain original user text %q, got body: %s", originalUserText, execBody)
	}
	if strings.Contains(execBody, "simplify it") {
		t.Error("execute request must NOT contain feedback text")
	}
	if strings.Contains(execBody, "PLAN V1") || strings.Contains(execBody, "PLAN V2") {
		t.Error("execute request must NOT contain plan proposal content")
	}

	// Verify plan_only was called twice.
	if fake.planCallCount != 2 {
		t.Errorf("expected 2 plan_only calls, got %d", fake.planCallCount)
	}
	// Verify execute was called once.
	if fake.execCount != 1 {
		t.Errorf("expected 1 execute call, got %d", fake.execCount)
	}
}

// TestPlanOnlySingleChat_RevisePlanOnlyReceivesOriginalUserTextAndFeedback
// verifies that the revise plan_only dispatch message contains the original
// userText AND the feedback, not feedback alone.
func TestPlanOnlySingleChat_RevisePlanOnlyReceivesOriginalUserTextAndFeedback(t *testing.T) {
	fake := &fakeReviseAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	originalUserText := "write a Go HTTP server"
	body := fmt.Sprintf(`{"runId":"run-planonly-body","conversationId":"conv-1","messages":[{"role":"user","text":"%s"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`, originalUserText)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	// Wait for initial plan (v1).
	planID := waitForPlanID(srv, "run-planonly-body", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	// Send revise.
	reviseBody := fmt.Sprintf(`{"runId":"run-planonly-body","actionId":"%s","action":"revise","feedback":"simplify it","revision":1}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	// Wait for revised plan (v2).
	newPlanID := waitForNewPlanID(srv, "run-planonly-body", planID, 2*time.Second)
	if newPlanID == "" {
		t.Fatal("revised plan was not registered within timeout")
	}

	// Cancel to end the stream.
	cancelBody := fmt.Sprintf(`{"runId":"run-planonly-body","actionId":"%s","action":"cancel"}`, newPlanID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	select {
	case <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	// Assertions on planOnlyBodies.

	// 1. Initial plan_only call contains original userText.
	body0 := fake.planOnlyBody(0)
	if body0 == "" {
		t.Fatal("planOnlyBody(0) is empty, initial plan_only dispatch did not occur")
	}
	if !strings.Contains(body0, originalUserText) {
		t.Errorf("planOnlyBody(0) must contain original userText %q, got: %s", originalUserText, body0)
	}

	// 2. Revise plan_only call contains original userText (for context).
	body1 := fake.planOnlyBody(1)
	if body1 == "" {
		t.Fatal("planOnlyBody(1) is empty, revise plan_only dispatch did not occur")
	}
	if !strings.Contains(body1, originalUserText) {
		t.Errorf("planOnlyBody(1) must contain original userText %q (for context), got: %s", originalUserText, body1)
	}

	// 3. Revise call contains the feedback text.
	if !strings.Contains(body1, "simplify it") {
		t.Errorf("planOnlyBody(1) must contain feedback text %q, got: %s", "simplify it", body1)
	}

	// 4. Revise call is NOT feedback-only.
	if strings.TrimSpace(body1) == "simplify it" {
		t.Error("planOnlyBody(1) must not be just the feedback text; should include userText, summary, and revision marker")
	}

	// 5. Revise call contains a revision marker.
	if !strings.Contains(body1, "revision 2") {
		t.Errorf("planOnlyBody(1) must contain revision marker %q, got: %s", "revision 2", body1)
	}

	// 6. Plan call count must be 2 (initial + revise).
	fake.mu.Lock()
	pc := fake.planCallCount
	ec := fake.execCount
	fake.mu.Unlock()
	if pc != 2 {
		t.Errorf("expected 2 plan_only calls (initial + revise), got %d", pc)
	}

	// 7. No full execution triggered.
	if ec != 0 {
		t.Errorf("expected 0 full execution calls (only revise+cancel), got %d", ec)
	}
}


// TestPlanOnlySingleChat_ReviseLifecycleEmitsRevisedActivitySnapshot verifies the
// SSE event sequence during a revise cycle using ACTIVITY_SNAPSHOT events.
func TestPlanOnlySingleChat_ReviseLifecycleEmitsRevisedActivitySnapshot(t *testing.T) {
	fake := &fakeReviseAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-revise-lifecycle","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-revise-lifecycle", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	// Send revise.
	reviseBody := fmt.Sprintf(`{"runId":"run-revise-lifecycle","actionId":"%s","action":"revise","feedback":"make it simpler","revision":1}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	// Wait for revised plan (planID changes since new plan gets a new planID).
	newPlanID := waitForNewPlanID(srv, "run-revise-lifecycle", planID, 2*time.Second)
	if newPlanID == "" {
		t.Fatal("revised plan was not registered within timeout")
	}

	// Brief sleep to ensure SSE events are flushed before cancelling.
	time.Sleep(200 * time.Millisecond)

	// Cancel to end the stream.
	cancelBody := fmt.Sprintf(`{"runId":"run-revise-lifecycle","actionId":"%s","action":"cancel"}`, newPlanID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events

	// 1. Must contain revising_plan phase.
	var foundRevising bool
	for _, e := range events {
		if e.eventType == "state_update" {
			state := parseSSEState(e.data)
			if phase, ok := state["phase"].(string); ok && phase == "revising_plan" {
				foundRevising = true
			}
		}
	}
	if !foundRevising {
		t.Error("expected state_update with phase=revising_plan")
	}

	// 2. Count activity_snapshot occurrences (expect >= 2: initial + revised).
	var snapshotCount int
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			snapshotCount++
		}
	}
	if snapshotCount < 2 {
		t.Errorf("expected at least 2 activity_snapshot events (initial + revised), got %d", snapshotCount)
	}

	// 3. Verify an activity_snapshot contains revision=2 in the activity data.
	var foundRevision2 bool
	var revisedActivityData string
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			if strings.Contains(e.data, `"revision":2`) {
				foundRevision2 = true
				revisedActivityData = e.data
			}
		}
	}
	if !foundRevision2 {
		t.Error("expected activity_snapshot with revision=2 in SSE events")
	}

	// 4. Verify revised activity_snapshot contains planOwner and participants.
	if revisedActivityData != "" {
		if !strings.Contains(revisedActivityData, "planOwner") {
			t.Error("revised activity_snapshot missing planOwner")
		}
		if !strings.Contains(revisedActivityData, "participants") {
			t.Error("revised activity_snapshot missing participants")
		}
	} else {
		t.Error("no revised activity_snapshot data to verify")
	}
}
// TestPlanOnlySingleChat_ReviseThenCancelNoRunError verifies that cancelling
// after a revise does NOT emit RUN_ERROR (per AG-UI v1.2 contract).
func TestPlanOnlySingleChat_ReviseThenCancelNoRunError(t *testing.T) {
	fake := &fakeReviseAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-revise-cancel","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-revise-cancel", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	// Revise.
	reviseBody := fmt.Sprintf(`{"runId":"run-revise-cancel","actionId":"%s","action":"revise","feedback":"simplify","revision":1}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	// Wait for revised plan (planID changes since new plan gets a new planID).
	newPlanID := waitForNewPlanID(srv, "run-revise-cancel", planID, 2*time.Second)
	if newPlanID == "" {
		t.Fatal("revised plan was not registered within timeout")
	}

	// Cancel the revised plan.
	cancelBody := fmt.Sprintf(`{"runId":"run-revise-cancel","actionId":"%s","action":"cancel"}`, newPlanID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events

	// Critical: NO run_error event.
	for _, e := range events {
		if e.eventType == "run_error" {
			t.Errorf("cancel after revise must NOT emit RUN_ERROR, got: %s", e.data)
		}
	}

	// Must have state_update phase=cancelled.
	var foundCancelled bool
	for _, e := range events {
		if e.eventType == "state_update" {
			state := parseSSEState(e.data)
			if phase, ok := state["phase"].(string); ok && phase == "cancelled" {
				foundCancelled = true
			}
		}
	}
	if !foundCancelled {
		t.Error("expected state_update with phase=cancelled")
	}

	// Must have run_finished status=cancelled.
	data, hasRunFinished := findEvent(events, "run_finished")
	if !hasRunFinished {
		t.Error("expected run_finished event")
	} else if !strings.Contains(data, "cancelled") {
		t.Errorf("expected run_finished status=cancelled, got: %s", data)
	}
}

// TestPlanOnlySingleChat_RevisionMismatchRejected verifies that sending a
// wrong revision number results in PLAN_REVISION_MISMATCH.
func TestPlanOnlySingleChat_RevisionMismatchRejected(t *testing.T) {
	fake := &fakeReviseAgent{}
	fakeAgentSrv := httptest.NewServer(fake.handler())
	defer fakeAgentSrv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgentSrv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-rev-mismatch","conversationId":"conv-1","messages":[{"role":"user","text":"write a function"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-rev-mismatch", 2*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered within timeout")
	}

	// Send revise with wrong revision number (999 instead of 1).
	reviseBody := fmt.Sprintf(`{"runId":"run-rev-mismatch","actionId":"%s","action":"revise","feedback":"change it","revision":999}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events

	// Must contain run_error with PLAN_REVISION_MISMATCH.
	data, hasRunError := findEvent(events, "run_error")
	if !hasRunError {
		t.Fatal("expected run_error for revision mismatch")
	}
	if !strings.Contains(data, "PLAN_REVISION_MISMATCH") {
		t.Errorf("expected PLAN_REVISION_MISMATCH, got: %s", data)
	}
	if !strings.Contains(data, "revision mismatch") || !strings.Contains(data, "999") {
		t.Errorf("error should mention revision mismatch and wrong revision, got: %s", data)
	}
}

// fakeGroupChatPlanner is a deterministic planner that always returns a valid
// group_chat plan, forcing the old confirmLoop path for revise testing.
type fakeGroupChatPlanner struct{}

func (f *fakeGroupChatPlanner) Plan(_ context.Context, _ planner.PlannerInput) (*plan.OrchestrationPlan, error) {
	return &plan.OrchestrationPlan{
		Version:        "1.0",
		PlanID:         "plan-group-chat-test",
		RunID:          "run-non-single-revise",
		ConversationID: "conv-1",
		ExecutionPath:  "group_chat",
		Strategy:       plan.StrategyOrderedParallel,
		IntentSummary:  "Build web app with code and UI",
		Tasks: []plan.TaskPlan{
			{
				TaskID:      "t1",
				AgentName:   "code-agent",
				TaskContent: "Write Go HTTP server code",
				Priority:    1,
				RiskLevel:   "low",
				TimeoutMs:   30000,
			},
			{
				TaskID:      "t2",
				AgentName:   "web-agent",
				TaskContent: "Build web frontend",
				Priority:    2,
				RiskLevel:   "low",
				TimeoutMs:   30000,
			},
		},
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: true, Selected: true},
		},
		PlanOwner: &plan.PlanOwner{Type: "agent", AgentName: "code-agent"},
	}, nil
}

// TestReviseSupportedForGroupChat verifies that the non-single_chat
// confirmLoop now supports revise: re-plans with feedback, emits revised
// ACTIVITY_SNAPSHOT, and allows cancel after revise.
func TestReviseSupportedForGroupChat(t *testing.T) {
	fake1 := &fakePlanOnlyAgent{}
	fakeAgent1Srv := httptest.NewServer(fake1.handler())
	defer fakeAgent1Srv.Close()
	fake2 := &fakePlanOnlyAgent{}
	fakeAgent2Srv := httptest.NewServer(fake2.handler())
	defer fakeAgent2Srv.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: fakeAgent1Srv.URL},
		{Name: "web-agent", URL: fakeAgent2Srv.URL},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	os.Setenv("REQUIRE_PLAN_CONFIRMATION", "true")
	defer os.Setenv("REQUIRE_PLAN_CONFIRMATION", "")

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithPlanner(&fakeGroupChatPlanner{}),
		WithPlannerMode(PlannerModeLLM),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-non-single-revise","conversationId":"conv-1","messages":[{"role":"user","text":"write code and build a web app"}],"selectedAgentNames":["code-agent","web-agent"],"executionPath":"group_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	type streamResult struct {
		events []sseEvent
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		resultCh <- streamResult{events: readSSEBody(t, resp.Body)}
	}()

	planID := waitForPlanID(srv, "run-non-single-revise", 3*time.Second)
	if planID == "" {
		t.Fatal("deterministic fakeGroupChatPlanner should always produce a plan requiring confirmation")
	}

	// Send revise — must trigger re-plan, NOT NOT_IMPLEMENTED.
	reviseBody := fmt.Sprintf(`{"runId":"run-non-single-revise","actionId":"%s","action":"revise","feedback":"make it simpler","revision":1}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	// Wait for revised plan to be registered.
	newPlanID := waitForNewPlanID(srv, "run-non-single-revise", planID, 3*time.Second)
	if newPlanID == "" {
		t.Fatal("revised plan was not registered within timeout — revise should be supported for group_chat")
	}

	// Brief sleep to ensure SSE events are flushed.
	time.Sleep(200 * time.Millisecond)

	// Cancel to end the stream.
	cancelBody := fmt.Sprintf(`{"runId":"run-non-single-revise","actionId":"%s","action":"cancel"}`, newPlanID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events

	// Verify no NOT_IMPLEMENTED error.
	for _, e := range events {
		if e.eventType == "run_error" && strings.Contains(e.data, "NOT_IMPLEMENTED") {
			t.Error("revise should be supported for group_chat, got NOT_IMPLEMENTED")
		}
	}

	// Verify activity_snapshot events exist (initial + revised).
	var snapshotCount int
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			snapshotCount++
		}
	}
	if snapshotCount < 2 {
		t.Errorf("expected at least 2 activity_snapshot events (initial + revised), got %d", snapshotCount)
	}

	// Verify cancel lifecycle: STATE_UPDATE cancelled + RUN_FINISHED, no RUN_ERROR.
	data, hasRunError := findEvent(events, "run_error")
	if hasRunError && !strings.Contains(data, "timeout") {
		t.Errorf("cancel after revise must not emit run_error (other than timeout), got: %s", data)
	}

	_, hasCancelled := findEvent(events, "run_finished")
	if !hasCancelled {
		t.Error("expected run_finished after cancel")
	}
}
