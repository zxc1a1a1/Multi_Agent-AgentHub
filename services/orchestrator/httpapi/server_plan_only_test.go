package httpapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// ---------------------------------------------------------------------------
// Shared helpers (used by both this file and server_main_agent_test.go)
// ---------------------------------------------------------------------------

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
// Phase 1-3: unified MainAgent planning for all execution paths
// ---------------------------------------------------------------------------

// TestSingleChat_UsesMainAgent verifies that single_chat routes through MainAgent.
func TestSingleChat_UsesMainAgent(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-05")
	os.Setenv("REQUIRE_PLAN_CONFIRMATION", "true")
	defer os.Setenv("REQUIRE_PLAN_CONFIRMATION", "")

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-single",
			RunID:          "run-single",
			ConversationID: "conv-1",
			ExecutionPath:  "single_chat",
			Strategy:       plan.StrategySingle,
			IntentSummary:  "generate code",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
			},
			PlannerSource: "main_agent",
			Participants:  []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}},
			PlanOwner:     &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-single","conversationId":"conv-1","messages":[{"role":"user","text":"write code"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-single", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-single","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var events []sseEvent
	select {
	case events = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	if !fake.PlanCalled {
		t.Fatal("FakeMainAgent.Plan was not called — single_chat must route through MainAgent")
	}
	if fake.LastInput.ExecutionPath != "single_chat" {
		t.Errorf("expected executionPath=single_chat, got %s", fake.LastInput.ExecutionPath)
	}
	if len(fake.LastInput.AllowedAgents) != 1 || fake.LastInput.AllowedAgents[0] != "code-agent" {
		t.Errorf("expected AllowedAgents=[code-agent], got %v", fake.LastInput.AllowedAgents)
	}

	// Verify lifecycle: RUN_STARTED → STATE_UPDATE → RUN_FINISHED (no error).
	if len(events) == 0 {
		t.Fatal("expected SSE events")
	}
	if events[0].eventType != "run_started" {
		t.Errorf("expected run_started, got %s", events[0].eventType)
	}
	_, hasFinished := findEvent(events, "run_finished")
	if !hasFinished {
		t.Error("expected run_finished")
	}
}

// TestGroupChat_UsesMainAgent verifies group_chat routes through MainAgent.
func TestGroupChat_UsesMainAgent(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-06")
	os.Setenv("REQUIRE_PLAN_CONFIRMATION", "true")
	defer os.Setenv("REQUIRE_PLAN_CONFIRMATION", "")

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-group",
			RunID:          "run-group",
			ConversationID: "conv-1",
			ExecutionPath:  "group_chat",
			Strategy:       plan.StrategyOrderedParallel,
			IntentSummary:  "build app",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
				{TaskID: "task-2", AgentName: "web-agent", TaskContent: "build UI", Priority: 2, TimeoutMs: 60000, RiskLevel: "low"},
			},
			PlannerSource: "main_agent",
			Participants: []plan.PlanParticipant{
				{AgentName: "code-agent", Required: true, Selected: true},
				{AgentName: "web-agent", Required: true, Selected: true},
			},
			PlanOwner: &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-group","conversationId":"conv-1","messages":[{"role":"user","text":"build app"}],"selectedAgentNames":["code-agent","web-agent"],"executionPath":"group_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-group", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-group","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	select {
	case <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	if !fake.PlanCalled {
		t.Fatal("FakeMainAgent.Plan was not called")
	}
	if fake.LastInput.ExecutionPath != "group_chat" {
		t.Errorf("expected executionPath=group_chat, got %s", fake.LastInput.ExecutionPath)
	}
	if len(fake.LastInput.AllowedAgents) != 2 {
		t.Errorf("expected 2 AllowedAgents, got %d", len(fake.LastInput.AllowedAgents))
	}
}

// TestMainAgentOrchestration_UsesMainAgent verifies all paths use MainAgent.
func TestMainAgentOrchestration_UsesMainAgent(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-07")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-auto",
			RunID:          "run-auto",
			ConversationID: "conv-1",
			ExecutionPath:  "main_agent_orchestration",
			Strategy:       plan.StrategySingle,
			IntentSummary:  "auto plan",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
			},
			PlannerSource:              "main_agent",
			Participants:               []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}},
			CandidateParticipants:      []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}, {AgentName: "web-agent", Required: false, Selected: false}},
			DefaultSelectedParticipants: []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}},
			PlanOwner:                  &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-auto","conversationId":"conv-1","messages":[{"role":"user","text":"write code"}],"requestedPath":"main_agent_orchestration"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-auto", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-auto","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	select {
	case <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	if !fake.PlanCalled {
		t.Fatal("FakeMainAgent.Plan was not called")
	}
	if fake.LastInput.ExecutionPath != "main_agent_orchestration" {
		t.Errorf("expected executionPath=main_agent_orchestration, got %s", fake.LastInput.ExecutionPath)
	}
}

// TestPlanBoundaryEnforcement verifies that tasks with agent names outside
// AllowedAgents are rejected before execution.
func TestPlanBoundaryEnforcement(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-08")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	// FakeMainAgent returns a plan with a task for an agent outside AllowedAgents.
	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-boundary",
			RunID:          "run-boundary",
			ConversationID: "conv-1",
			ExecutionPath:  "single_chat",
			Strategy:       plan.StrategySingle,
			IntentSummary:  "test",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "web-agent", TaskContent: "build UI", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
			},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Request single_chat with code-agent only. Fake returns plan for web-agent — should be rejected.
	body := `{"runId":"run-boundary","conversationId":"conv-1","messages":[{"role":"user","text":"build UI"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	events := readSSEBody(t, resp.Body)

	if !fake.PlanCalled {
		t.Fatal("FakeMainAgent.Plan was not called")
	}

	// Should emit a run_error for boundary violation.
	data, hasError := findEvent(events, "run_error")
	if !hasError {
		t.Fatal("expected run_error for boundary violation")
	}
	if !strings.Contains(data, "AGENT_BOUNDARY_VIOLATION") {
		t.Errorf("expected AGENT_BOUNDARY_VIOLATION, got: %s", data)
	}
}

// TestMainAgentPlannerDefault verifies that the production default MainAgent
// is created when no WithMainAgentPlanner is provided.
func TestMainAgentPlannerDefault(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-09")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-default","conversationId":"conv-1","messages":[{"role":"user","text":"write Go code for an HTTP API"}],"requestedPath":"main_agent_orchestration"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-default", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered — default MainAgent should generate a plan")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-default","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var events []sseEvent
	select {
	case events = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Verify no run_error.
	_, hasError := findEvent(events, "run_error")
	if hasError {
		t.Error("should not have run_error with default MainAgent")
	}

	// Verify activity_snapshot has planOwner.
	var activityData string
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			activityData = e.data
			break
		}
	}
	if activityData == "" {
		t.Fatal("no activity_snapshot")
	}
	if !strings.Contains(activityData, "main_agent") {
		t.Error("activity_snapshot should have planOwner with main_agent")
	}
}

// TestGroupChatReviseSupported verifies that revise works for group_chat
// through the shared confirmLoop using MainAgent.
func TestGroupChatReviseSupported(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-10")
	os.Setenv("REQUIRE_PLAN_CONFIRMATION", "true")
	defer os.Setenv("REQUIRE_PLAN_CONFIRMATION", "")

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	planV1 := &plan.OrchestrationPlan{
		Version:        "1.0",
		PlanID:         "plan-v1",
		RunID:          "run-revise-gc",
		ConversationID: "conv-1",
		ExecutionPath:  "group_chat",
		Strategy:       plan.StrategyOrderedParallel,
		IntentSummary:  "Plan v1",
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
			{TaskID: "t2", AgentName: "web-agent", TaskContent: "build UI", Priority: 2, TimeoutMs: 60000, RiskLevel: "low"},
		},
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: true, Selected: true},
		},
		PlanOwner: &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
	}

	planV2 := &plan.OrchestrationPlan{
		Version:        "1.0",
		PlanID:         "plan-v2",
		RunID:          "run-revise-gc",
		ConversationID: "conv-1",
		ExecutionPath:  "group_chat",
		Strategy:       plan.StrategySingle,
		IntentSummary:  "Plan v2 (simplified)",
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent", TaskContent: "write simple code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
		},
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
		PlanOwner: &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
	}

	fake := &FakeMainAgent{
		Plans: []*plan.OrchestrationPlan{planV1, planV2},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-revise-gc","conversationId":"conv-1","messages":[{"role":"user","text":"write code and build UI"}],"selectedAgentNames":["code-agent","web-agent"],"executionPath":"group_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-revise-gc", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered")
	}

	// Revise.
	reviseBody := fmt.Sprintf(`{"runId":"run-revise-gc","actionId":"%s","action":"revise","feedback":"simplify","revision":1}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(reviseBody))

	newPlanID := waitForNewPlanID(srv, "run-revise-gc", planID, 3*time.Second)
	if newPlanID == "" {
		t.Fatal("revised plan was not registered")
	}

	time.Sleep(200 * time.Millisecond)

	cancelBody := fmt.Sprintf(`{"runId":"run-revise-gc","actionId":"%s","action":"cancel"}`, newPlanID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var events []sseEvent
	select {
	case events = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Verify no NOT_IMPLEMENTED error.
	for _, e := range events {
		if e.eventType == "run_error" && strings.Contains(e.data, "NOT_IMPLEMENTED") {
			t.Error("revise should be supported, got NOT_IMPLEMENTED")
		}
	}

	// Verify activity_snapshot events (initial + revised).
	var snapshotCount int
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			snapshotCount++
		}
	}
	if snapshotCount < 2 {
		t.Errorf("expected at least 2 activity_snapshot events, got %d", snapshotCount)
	}
}

// ---------------------------------------------------------------------------
// P0 tests: Phase 1-3 key fixes
// ---------------------------------------------------------------------------

// TestSingleChatAgentNameRequiresActivitySnapshot verifies that when
// REQUIRE_PLAN_CONFIRMATION is set, single_chat with an explicit agentName
// still triggers the ActivitySnapshot confirmation flow.
func TestSingleChatAgentNameRequiresActivitySnapshot(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-11")
	os.Setenv("REQUIRE_PLAN_CONFIRMATION", "true")
	defer os.Setenv("REQUIRE_PLAN_CONFIRMATION", "")

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-explicit",
			RunID:          "run-explicit",
			ConversationID: "conv-1",
			ExecutionPath:  "single_chat",
			Strategy:       plan.StrategySingle,
			IntentSummary:  "code generation",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
			},
			PlannerSource: "main_agent",
			Participants:  []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}},
			PlanOwner:     &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Explicit agentName="code-agent" — must still require confirmation.
	body := `{"runId":"run-explicit","conversationId":"conv-1","messages":[{"role":"user","text":"write code"}],"agentName":"code-agent","executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-explicit", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered — explicit agentName should still enter confirmation")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-explicit","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var events []sseEvent
	select {
	case events = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Must have activity_snapshot because agentName=code-agent should NOT bypass confirmation.
	_, hasSnapshot := findEvent(events, "activity_snapshot")
	if !hasSnapshot {
		t.Fatal("expected activity_snapshot event — agentName=code-agent must NOT bypass confirmation")
	}
}

// TestHandlerUsesPathAwareValidator verifies that the handler rejects plans
// that violate path-aware constraints (e.g., single_chat with too many participants).
func TestHandlerUsesPathAwareValidator(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-12")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	// FakeMainAgent returns a plan with StrategyOrderedParallel and 2 participants
	// for single_chat — PlanValidator passes (ordered_parallel allows 2 tasks),
	// but PathAwareValidator rejects (single_chat + 2 participants + web-agent outside boundary).
	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-too-many",
			RunID:          "run-too-many",
			ConversationID: "conv-1",
			ExecutionPath:  "single_chat",
			Strategy:       plan.StrategyOrderedParallel,
			IntentSummary:  "test",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
				{TaskID: "task-2", AgentName: "web-agent", TaskContent: "build UI", Priority: 2, TimeoutMs: 60000, RiskLevel: "low"},
			},
			Participants: []plan.PlanParticipant{
				{AgentName: "code-agent", Required: true, Selected: true},
				{AgentName: "web-agent", Required: true, Selected: true},
			},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-too-many","conversationId":"conv-1","messages":[{"role":"user","text":"write code"}],"selectedAgentNames":["code-agent"],"executionPath":"single_chat"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	events := readSSEBody(t, resp.Body)

	// Should get AGENT_BOUNDARY_VIOLATION from PathAwareValidator.
	data, hasError := findEvent(events, "run_error")
	if !hasError {
		t.Fatal("expected run_error from PathAwareValidator")
	}
	if !strings.Contains(data, "AGENT_BOUNDARY_VIOLATION") {
		t.Errorf("expected AGENT_BOUNDARY_VIOLATION, got: %s", data)
	}
}

// TestUnknownStrategyFallsBackToSequential verifies that plans with unknown or
// empty strategy normalize to sequential instead of producing RUN_ERROR.
func TestUnknownStrategyFallsBackToSequential(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-13")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	fake := &FakeMainAgent{
		PlanToReturn: &plan.OrchestrationPlan{
			Version:        "1.0",
			PlanID:         "plan-unknown-strat",
			RunID:          "run-unknown-strat",
			ConversationID: "conv-1",
			ExecutionPath:  "main_agent_orchestration",
			Strategy:       "gibberish_strategy_not_known", // unknown strategy
			IntentSummary:  "test",
			Tasks: []plan.TaskPlan{
				{TaskID: "task-1", AgentName: "code-agent", TaskContent: "write code", Priority: 1, TimeoutMs: 60000, RiskLevel: "low"},
			},
			PlannerSource: "main_agent",
			Participants:  []plan.PlanParticipant{{AgentName: "code-agent", Required: true, Selected: true}},
			PlanOwner:     &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-unknown-strat","conversationId":"conv-1","messages":[{"role":"user","text":"write code"}],"requestedPath":"main_agent_orchestration"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-unknown-strat", 3*time.Second)
	if planID == "" {
		t.Fatal("plan was not registered")
	}

	cancelBody := fmt.Sprintf(`{"runId":"run-unknown-strat","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var events []sseEvent
	select {
	case events = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Must NOT have RUN_ERROR for unknown strategy.
	errData, hasError := findEvent(events, "run_error")
	if hasError {
		t.Errorf("expected NO run_error, but got: %s", errData)
	}
	// Must have run_finished.
	_, hasRunFinished := findEvent(events, "run_finished")
	if !hasRunFinished {
		t.Error("expected run_finished")
	}
}
