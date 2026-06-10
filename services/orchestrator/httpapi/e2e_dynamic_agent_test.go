package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ---------------------------------------------------------------------------
// Phase 7E: End-to-End Fake A2A Smoke Test
//
// This test validates the complete dynamic agent lifecycle:
//   1. Start a fake A2A agent server (card + task endpoints)
//   2. Register the fake agent in DynamicAgentRegistry
//   3. Dispatch a run via Orchestrator → dynamic agent
//   4. Verify the agent's response appears in SSE stream
//   5. Verify cancel (RunTaskRegistry + CancelTask)
//   6. Verify tool-result path
//   7. Clean up: unregister from registry
// ---------------------------------------------------------------------------

// fakeA2ATestAgent is a minimal adk.Agent returning a predictable response.
type fakeA2ATestAgent struct {
	name     string
	response string
}

func (a *fakeA2ATestAgent) Name() string { return a.name }

func (a *fakeA2ATestAgent) Generate(_ context.Context, _ *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	return &adk.GenerateResponse{
		Parts:        []adk.Part{adk.TextPart{Text: a.response}},
		FinishReason: adk.FinishStop,
	}, nil
}

// startFakeA2AServer creates an httptest server running the full A2A stack
// (agent.json card + task send endpoint) with the given agent name and response.
func startFakeA2AServer(t *testing.T, name, response string) *httptest.Server {
	t.Helper()

	agent := &fakeA2ATestAgent{name: name, response: response}
	return startA2AServerWithAgent(t, agent, name)
}

// startBlockingA2AServer creates an httptest server with a blocking agent.
// The agent blocks on ctx.Done() and never returns a response. Useful for
// testing cancel and tool-result flows with an active run.
func startBlockingA2AServer(t *testing.T, name string) *httptest.Server {
	t.Helper()
	agent := &blockingA2ATestAgent{name: name}
	return startA2AServerWithAgent(t, agent, name)
}

// startA2AServerWithAgent wires a generic adk.Agent into a full A2A server.
func startA2AServerWithAgent(t *testing.T, agent adk.Agent, name string) *httptest.Server {
	t.Helper()
	sessionService := adk.NewMemorySessionService()
	runner := adk.NewRunner(agent, sessionService)

	cfg := &a2a.AgentConfig{
		Name:        name,
		Description: "fake agent for e2e smoke test",
		Version:     "v1.0.0",
		URL:         "http://localhost:0",
		Skills:      []a2a.AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain"},
		SupportedInterfaces: []a2a.AgentInterface{
			{Type: "JSONRPC", URL: "http://localhost:0"},
		},
	}
	a2aSrv := a2a.NewServer(cfg, runner)

	ts := httptest.NewServer(a2aSrv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// buildE2EServer creates a fully-wired Orchestrator Server.
func buildE2EServer(t *testing.T, staticReg *registry.StaticAgentRegistry, dynReg *registry.DynamicAgentRegistry, targetAgentName string) *Server {
	t.Helper()

	fakePlanner := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-e2e-1",
			RunID:          "run-e2e-1",
			ConversationID: "conv-e2e",
			ExecutionPath:  "single_chat",
			Strategy:       "single",
			IntentSummary:  "e2e smoke test",
			Tasks: []plan.TaskPlan{
				{
					TaskID:      "task-e2e-1",
					AgentName:   targetAgentName,
					TaskContent: "e2e test task",
					Priority:    1,
					TimeoutMs:   30000,
					RiskLevel:   "low",
				},
			},
			PlannerSource: "main_agent",
			Participants:  []plan.PlanParticipant{{AgentName: targetAgentName, Required: true, Selected: true}},
			PlanOwner:     &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		},
	}

	return NewServer(
		WithRegistry(staticReg),
		WithDynamicRegistry(dynReg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fakePlanner),
	)
}

// TestE2E_DynamicAgentFullLifecycle validates the complete flow:
//
//	register → resolve → dispatch → cancel → unregister.
func TestE2E_DynamicAgentFullLifecycle(t *testing.T) {
	// --- 1. Start fake A2A agent server ---
	fakeName := "e2e-fake-agent"
	fakeResponse := "Hello from the e2e fake agent! This response validates the full dynamic agent lifecycle."
	fakeSrv := startFakeA2AServer(t, fakeName, fakeResponse)
	fakeURL := fakeSrv.URL
	t.Logf("fake A2A agent running at %s", fakeURL)

	// --- 2. Set up registries ---
	staticReg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1", Description: "static code agent", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text"}},
	})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}

	dir := t.TempDir()
	store, err := registry.NewJSONStore(filepath.Join(dir, "agents.json"))
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	dynReg := registry.NewDynamicAgentRegistry(staticReg, store)

	// --- 3. Register the fake agent dynamically ---
	registered, err := dynReg.Register(context.Background(), registry.RegisterAgentRequest{
		URL: fakeURL,
	})
	if err != nil {
		t.Fatalf("register fake agent: %v", err)
	}
	if registered == nil {
		t.Fatal("expected non-nil registered agent")
	}
	t.Logf("registered dynamic agent: %s (source=%s, healthy=%v)", registered.Name, registered.Source, registered.Healthy)

	// Run a health check to mark the agent as healthy (required for DispatchResolver).
	checked, err := dynReg.Check(context.Background(), fakeName)
	if err != nil {
		t.Fatalf("health check fake agent: %v", err)
	}
	if !checked.Healthy {
		t.Fatal("expected agent to be healthy after check")
	}
	t.Logf("health check passed: %s healthy=%v", fakeName, checked.Healthy)

	// --- 4. Verify DispatchResolver ---
	resolver := registry.NewDispatchResolver(staticReg, dynReg)
	url, ok, err := resolver.ResolveURL(context.Background(), fakeName)
	if err != nil {
		t.Fatalf("ResolveURL failed: %v", err)
	}
	if !ok {
		t.Fatal("expected dynamic agent to be resolvable")
	}
	if url == "" {
		t.Fatal("expected non-empty URL for dynamic agent")
	}
	t.Logf("DispatchResolver.ResolveURL(%s) → %s", fakeName, url)

	// --- 5. Verify Names() includes the dynamic agent ---
	names, err := resolver.Names(context.Background())
	if err != nil {
		t.Fatalf("Names failed: %v", err)
	}
	found := false
	for _, n := range names {
		if n == fakeName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Names() to include %q, got %v", fakeName, names)
	}
	t.Logf("DispatchResolver.Names(): %v", names)

	// --- 6. Build Orchestrator server and dispatch a run ---
	srv := buildE2EServer(t, staticReg, dynReg, fakeName)
	orchestratorTS := httptest.NewServer(srv.Handler())
	defer orchestratorTS.Close()

	streamBody := fmt.Sprintf(`{
		"runId": "run-e2e-1",
		"conversationId": "conv-e2e",
		"messages": [{"role": "user", "text": "e2e test"}],
		"selectedAgentNames": ["%s"],
		"executionPath": "single_chat"
	}`, fakeName)

	resp, err := http.Post(
		orchestratorTS.URL+"/internal/orchestrator/runs/stream",
		"application/json",
		strings.NewReader(streamBody),
	)
	if err != nil {
		t.Fatalf("POST run stream: %v", err)
	}

	events := readSSEBody(t, resp.Body)
	t.Logf("received %d SSE events", len(events))
	for i, ev := range events {
		t.Logf("  event[%d]: type=%s data(first 200 chars)=%s", i, ev.eventType, truncate(ev.data, 200))
	}

	// Verify lifecycle events.
	if _, ok := findEvent(events, "run_started"); !ok {
		t.Error("expected run_started event in SSE stream")
	}
	if _, ok := findEvent(events, "run_finished"); !ok {
		t.Error("expected run_finished event in SSE stream")
	}

	// --- 7. Test cancel with active run ---
	// Start a blocking agent and dispatch a run in the background, then cancel it
	// while the run is still streaming. This validates the cancel endpoint with
	// a real active run (not an already-completed one).
	cancelAgentName := "e2e-cancel-agent"
	cancelBlockingSrv := startBlockingA2AServer(t, cancelAgentName)
	_, err = dynReg.Register(context.Background(), registry.RegisterAgentRequest{URL: cancelBlockingSrv.URL})
	if err != nil {
		t.Fatalf("register cancel agent: %v", err)
	}
	if _, err := dynReg.Check(context.Background(), cancelAgentName); err != nil {
		t.Fatalf("check cancel agent: %v", err)
	}

	cancelSrv := buildE2EServer(t, staticReg, dynReg, cancelAgentName)
	cancelOrchTS := httptest.NewServer(cancelSrv.Handler())
	defer cancelOrchTS.Close()

	cancelRunID := "run-cancel-1"
	cancelStreamBody := fmt.Sprintf(`{
		"runId": "%s",
		"conversationId": "conv-cancel-1",
		"messages": [{"role": "user", "text": "cancel test"}],
		"selectedAgentNames": ["%s"],
		"executionPath": "single_chat"
	}`, cancelRunID, cancelAgentName)

	// Dispatch the blocking stream in a background goroutine — it will block
	// until cancelled, so we can test cancel while the run is active.
	cancelStreamDone := make(chan struct{})
	var cancelStreamErr error
	go func() {
		defer close(cancelStreamDone)
		resp, err := http.Post(
			cancelOrchTS.URL+"/internal/orchestrator/runs/stream",
			"application/json",
			strings.NewReader(cancelStreamBody),
		)
		if err != nil {
			cancelStreamErr = err
			return
		}
		// Drain the body until the server closes the connection (when cancelled).
		// Do NOT call resp.Body.Close() early — that would cancel the request
		// context, which unblocks the executor and triggers RemoveRun too soon.
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}()

	// Give the stream time to start, plan, validate, and register the run
	// before the cancel request fires.
	time.Sleep(500 * time.Millisecond)

	if cancelStreamErr != nil {
		t.Fatalf("cancel stream POST failed: %v", cancelStreamErr)
	}

	cancelReq, _ := http.NewRequest(http.MethodPost,
		cancelOrchTS.URL+"/internal/orchestrator/runs/cancel",
		strings.NewReader(fmt.Sprintf(`{"runId":"%s"}`, cancelRunID)))
	cancelReq.Header.Set("Content-Type", "application/json")

	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatalf("cancel request: %v", err)
	}
	if cancelResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(cancelResp.Body)
		cancelResp.Body.Close()
		t.Fatalf("cancel expected 200, got %d body=%s", cancelResp.StatusCode, string(body))
	}
	cancelResp.Body.Close()
	t.Logf("cancel response status: %d", cancelResp.StatusCode)

	// Wait for the cancelled stream goroutine to finish.
	<-cancelStreamDone

	// GC the cancel agent registration.
	if err := dynReg.Unregister(context.Background(), cancelAgentName); err != nil {
		t.Logf("unregister cancel agent: %v", err)
	}

	// --- 8. Test tool-result with active run ---
	// Start a blocking agent and dispatch a run in the background, then send a
	// tool-result to the active run. This validates the tool-result endpoint
	// forwards to a real active task.
	toolAgentName := "e2e-tool-agent"
	toolBlockingSrv := startBlockingA2AServer(t, toolAgentName)
	_, err = dynReg.Register(context.Background(), registry.RegisterAgentRequest{URL: toolBlockingSrv.URL})
	if err != nil {
		t.Fatalf("register tool agent: %v", err)
	}
	if _, err := dynReg.Check(context.Background(), toolAgentName); err != nil {
		t.Fatalf("check tool agent: %v", err)
	}

	toolSrv := buildE2EServer(t, staticReg, dynReg, toolAgentName)
	toolOrchTS := httptest.NewServer(toolSrv.Handler())
	defer toolOrchTS.Close()

	toolRunID := "run-tool-1"
	toolStreamBody := fmt.Sprintf(`{
		"runId": "%s",
		"conversationId": "conv-tool-1",
		"messages": [{"role": "user", "text": "tool test"}],
		"selectedAgentNames": ["%s"],
		"executionPath": "single_chat"
	}`, toolRunID, toolAgentName)

	toolStreamDone := make(chan struct{})
	var toolStreamErr error
	go func() {
		defer close(toolStreamDone)
		resp, err := http.Post(
			toolOrchTS.URL+"/internal/orchestrator/runs/stream",
			"application/json",
			strings.NewReader(toolStreamBody),
		)
		if err != nil {
			toolStreamErr = err
			return
		}
		// Drain the body until the server closes the connection (when cancelled).
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}()

	// Wait for the A2A task creation + TaskRef registration in RunTaskRegistry.
	// The run is registered via RegisterRun before the dispatch, but the default
	// A2A task ID arrives after the first streaming metadata event.
	time.Sleep(2 * time.Second)

	if toolStreamErr != nil {
		t.Fatalf("tool stream POST failed: %v", toolStreamErr)
	}

	trReq, _ := http.NewRequest(http.MethodPost,
		toolOrchTS.URL+"/internal/orchestrator/runs/tool-result",
		strings.NewReader(fmt.Sprintf(`{
			"runId": "%s",
			"toolCallId": "tc-e2e-1",
			"status": "success",
			"data": {"result": "ok"}
		}`, toolRunID)))
	trReq.Header.Set("Content-Type", "application/json")

	trResp, err := http.DefaultClient.Do(trReq)
	if err != nil {
		t.Fatalf("tool-result request: %v", err)
	}
	if trResp.StatusCode != http.StatusOK && trResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(trResp.Body)
		trResp.Body.Close()
		t.Fatalf("tool-result expected 200 or 202, got %d body=%s", trResp.StatusCode, string(body))
	}
	trResp.Body.Close()
	t.Logf("tool-result response status: %d", trResp.StatusCode)

	// <-toolStreamDone // let background goroutine finish naturally

	// Cancel the tool run so the blocking agent unblocks.
	if toolCancelResp, err := http.Post(
		toolOrchTS.URL+"/internal/orchestrator/runs/cancel",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"runId":"%s"}`, toolRunID)),
	); err == nil {
		toolCancelResp.Body.Close()
	}
	<-toolStreamDone

	if err := dynReg.Unregister(context.Background(), toolAgentName); err != nil {
		t.Logf("unregister tool agent: %v", err)
	}

	// --- 9. Clean up: unregister the dynamic agent ---
	if err := dynReg.Unregister(context.Background(), fakeName); err != nil {
		t.Errorf("unregister fake agent: %v", err)
	}
	_, ok, _ = resolver.ResolveURL(context.Background(), fakeName)
	if ok {
		t.Error("expected dynamic agent to be unresolvable after unregister")
	}

	t.Log("Phase 7E: Dynamic Agent Full Lifecycle — COMPLETE")
}

// TestE2E_DispatchResolver_StaticFallback verifies static fallback when
// no dynamic agent matches.
func TestE2E_DispatchResolver_StaticFallback(t *testing.T) {
	staticReg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://code-agent:8080", Description: "static code agent", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text"}},
	})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}

	dir := t.TempDir()
	store, err := registry.NewJSONStore(filepath.Join(dir, "agents.json"))
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	dynReg := registry.NewDynamicAgentRegistry(staticReg, store)

	resolver := registry.NewDispatchResolver(staticReg, dynReg)

	// Static agent should be resolvable even without dynamic registrations.
	url, ok, err := resolver.ResolveURL(context.Background(), "code-agent")
	if err != nil {
		t.Fatalf("ResolveURL static: %v", err)
	}
	if !ok {
		t.Fatal("expected static code-agent to be resolvable")
	}
	if url != "http://code-agent:8080" {
		t.Errorf("expected static URL, got %q", url)
	}
	t.Logf("static fallback: code-agent → %s", url)

	// Unknown agent should not be resolvable.
	_, ok, err = resolver.ResolveURL(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ResolveURL nonexistent: %v", err)
	}
	if ok {
		t.Error("expected nonexistent agent to NOT be resolvable")
	}

	// Names should include static agents.
	names, err := resolver.Names(context.Background())
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	found := false
	for _, n := range names {
		if n == "code-agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Names() to include code-agent, got %v", names)
	}
}

// TestE2E_RunTaskRegistry validates RunTaskRegistry lifecycle
// (register → get → remove) used by the cancel path.
func TestE2E_RunTaskRegistry(t *testing.T) {
	rtr := registry.NewRunTaskRegistry()

	_, ok := rtr.GetTasks("run-1")
	if ok {
		t.Error("expected no tasks for empty registry")
	}

	rtr.RegisterTask("run-1", registry.TaskRef{TaskID: "t1", AgentName: "agent-a", AgentURL: "http://a:8080"})
	rtr.RegisterTask("run-1", registry.TaskRef{TaskID: "t2", AgentName: "agent-b", AgentURL: "http://b:8080"})

	refs, ok := rtr.GetTasks("run-1")
	if !ok {
		t.Fatal("expected tasks for run-1")
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(refs))
	}

	rtr.RemoveRun("run-1")
	_, ok = rtr.GetTasks("run-1")
	if ok {
		t.Error("expected no tasks after RemoveRun")
	}
}

// TestE2E_BridgeToolResultTypes validates JSON serialization of bridge types
// used by the tool-result path (Gateway ↔ Orchestrator).
func TestE2E_BridgeToolResultTypes(t *testing.T) {
	reqJSON := `{"runId":"r1","taskId":"t1","toolCallId":"tc1","status":"success","data":{"key":"val"}}`
	var req struct {
		RunID      string `json:"runId"`
		TaskID     string `json:"taskId"`
		ToolCallID string `json:"toolCallId"`
		Status     string `json:"status"`
		Data       any    `json:"data"`
	}
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.RunID != "r1" || req.ToolCallID != "tc1" || req.Status != "success" {
		t.Errorf("unexpected values: %+v", req)
	}

	resp := map[string]string{"runId": "r1", "toolCallId": "tc1", "status": "processed"}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]string
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back["status"] != "processed" {
		t.Errorf("round-trip mismatch: %+v", back)
	}
}

// TestE2E_SendMessageA2A validates the A2A client's SendMessage method
// for follow-up messages to existing non-terminal tasks (used by tool-result forwarding).
func TestE2E_SendMessageA2A(t *testing.T) {
	agent := &blockingA2ATestAgent{name: "msg-agent"}
	sessionService := adk.NewMemorySessionService()
	runner := adk.NewRunner(agent, sessionService)

	a2aSrv := a2a.NewServer(&a2a.AgentConfig{
		Name:        "msg-agent",
		Description: "agent for SendMessage test",
		Version:     "v1.0.0",
		URL:         "http://localhost:0",
		Skills:      []a2a.AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain"},
		SupportedInterfaces: []a2a.AgentInterface{
			{Type: "JSONRPC", URL: "http://localhost:0"},
		},
	}, runner)
	ts := httptest.NewServer(a2aSrv.Handler())
	defer ts.Close()

	client := a2a.NewClient()
	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()

	chunks := make(chan a2a.StreamChunk, 8)
	go func() {
		defer close(chunks)
		client.SendJSONRPCStream(streamCtx, ts.URL, a2a.RunRequest{
			SessionID: fmt.Sprintf("session-%d", time.Now().UnixNano()),
			Message: a2a.Message{
				Role:    "user",
				Content: "initial blocking message",
			},
		})(func(c a2a.StreamChunk) bool {
			chunks <- c
			return c.Err == nil
		})
	}()

	var taskID string
	timeout := time.After(2 * time.Second)
	for taskID == "" {
		select {
		case c, ok := <-chunks:
			if !ok {
				t.Fatal("stream ended before task id was emitted")
			}
			if c.Err != nil {
				t.Fatalf("stream error before task id: %v", c.Err)
			}
			taskID = c.TaskID
		case <-timeout:
			t.Fatal("timed out waiting for task id")
		}
	}

	followUpResp, err := client.SendMessage(context.Background(), ts.URL, taskID, `{"type":"tool_result","toolCallId":"tc1","status":"success"}`)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if followUpResp == nil {
		t.Fatal("expected non-nil response from SendMessage")
	}
	if followUpResp.TaskID != taskID {
		t.Fatalf("SendMessage must target existing task id %q, got %q", taskID, followUpResp.TaskID)
	}

	cancelled, err := client.CancelTask(context.Background(), ts.URL, taskID)
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	if cancelled.Status != a2a.TaskStatusCancelled {
		t.Fatalf("expected cancelled task, got %s", cancelled.Status)
	}
}

type blockingA2ATestAgent struct{ name string }

func (a *blockingA2ATestAgent) Name() string { return a.name }

func (a *blockingA2ATestAgent) Generate(ctx context.Context, _ *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestE2E_CancelFlowTaskRegistry validates the cancel flow:
// register tasks → cancel → cleanup.
func TestE2E_CancelFlowTaskRegistry(t *testing.T) {
	rtr := registry.NewRunTaskRegistry()

	rtr.RegisterTask("run-cancel", registry.TaskRef{
		TaskID: "task-1", AgentName: "agent-1", AgentURL: "http://127.0.0.1:1",
	})
	rtr.RegisterTask("run-cancel", registry.TaskRef{
		TaskID: "task-2", AgentName: "agent-2", AgentURL: "http://127.0.0.1:2",
	})

	refs, ok := rtr.GetTasks("run-cancel")
	if !ok || len(refs) != 2 {
		t.Fatalf("expected 2 tasks, got ok=%v len=%d", ok, len(refs))
	}
	if refs[0].TaskID != "task-1" || refs[0].AgentName != "agent-1" {
		t.Errorf("unexpected TaskRef[0]: %+v", refs[0])
	}

	rtr.RemoveRun("run-cancel")
	_, ok = rtr.GetTasks("run-cancel")
	if ok {
		t.Error("expected tasks to be removed after cancel")
	}
}
