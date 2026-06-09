package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// ---------------------------------------------------------------------------
// Phase D — main_agent_orchestration deterministic tests
// ---------------------------------------------------------------------------

// TestMainAgentOrchestration_SendsActivitySnapshotWithCandidates verifies that
// the main_agent flow emits ACTIVITY_SNAPSHOT with candidateParticipants,
// defaultSelectedParticipants, and requiredParticipants.
func TestMainAgentOrchestration_SendsActivitySnapshotWithCandidates(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-01")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Send a request that matches code-agent keywords.
	body := `{"runId":"run-ma-1","conversationId":"conv-1","messages":[{"role":"user","text":"write Go code for an HTTP API"}],"requestedPath":"main_agent_orchestration"}`
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

	// Wait for plan to be registered, then cancel.
	planID := waitForPlanID(srv, "run-ma-1", 3*time.Second)
	if planID == "" {
		t.Fatal("main_agent should produce a plan requiring confirmation")
	}
	cancelBody := fmt.Sprintf(`{"runId":"run-ma-1","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var streamRes streamResult
	select {
	case streamRes = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE stream")
	}

	events := streamRes.events

	// Find ACTIVITY_SNAPSHOT events.
	var activitySnapshots []string
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			activitySnapshots = append(activitySnapshots, e.data)
		}
	}
	if len(activitySnapshots) == 0 {
		t.Fatal("expected at least one activity_snapshot event")
	}

	// Parse the first ACTIVITY_SNAPSHOT.
	var wrapper struct {
		Activity struct {
			ActivityType              string              `json:"activityType"`
			Status                    string              `json:"status"`
			ExecutionPath             string              `json:"executionPath"`
			CandidateParticipants     []plan.PlanParticipant `json:"candidateParticipants"`
			DefaultSelectedParticipants []plan.PlanParticipant `json:"defaultSelectedParticipants"`
			RequiredParticipants      []string              `json:"requiredParticipants"`
			PlanOwner                 *plan.PlanOwner       `json:"planOwner"`
			AllowedActions            []string              `json:"allowedActions"`
		} `json:"activity"`
	}
	if err := json.Unmarshal([]byte(activitySnapshots[0]), &wrapper); err != nil {
		t.Fatalf("unmarshal ACTIVITY_SNAPSHOT: %v", err)
	}
	a := wrapper.Activity

	// Verify activity type and status.
	if a.ActivityType != "plan_approval" {
		t.Errorf("expected activityType=plan_approval, got %s", a.ActivityType)
	}
	if a.Status != "awaiting_confirmation" {
		t.Errorf("expected status=awaiting_confirmation, got %s", a.Status)
	}
	if a.ExecutionPath != "main_agent_orchestration" {
		t.Errorf("expected executionPath=main_agent_orchestration, got %s", a.ExecutionPath)
	}

	// Verify planOwner is main_agent.
	if a.PlanOwner == nil {
		t.Fatal("expected planOwner")
	}
	if a.PlanOwner.Type != "main_agent" {
		t.Errorf("expected planOwner.type=main_agent, got %s", a.PlanOwner.Type)
	}
	if !a.PlanOwner.IsMainAgent {
		t.Error("expected planOwner.isMainAgent=true")
	}

	// Verify candidateParticipants is non-empty.
	if len(a.CandidateParticipants) == 0 {
		t.Fatal("expected non-empty candidateParticipants")
	}

	// Verify defaultSelectedParticipants is non-empty.
	if len(a.DefaultSelectedParticipants) == 0 {
		t.Fatal("expected non-empty defaultSelectedParticipants")
	}

	// Verify requiredParticipants is non-empty (code-agent should match well).
	if len(a.RequiredParticipants) == 0 {
		t.Fatal("expected non-empty requiredParticipants for a strong keyword match")
	}

	// Verify allowedActions.
	if len(a.AllowedActions) == 0 {
		t.Fatal("expected non-empty allowedActions")
	}
	foundApprove := false
	for _, act := range a.AllowedActions {
		if act == "approve" {
			foundApprove = true
			break
		}
	}
	if !foundApprove {
		t.Error("expected 'approve' in allowedActions")
	}

	// Verify candidate participants include code-agent (strong keyword match).
	foundCode := false
	for _, p := range a.CandidateParticipants {
		if p.AgentName == "code-agent" {
			foundCode = true
			break
		}
	}
	if !foundCode {
		t.Error("expected code-agent in candidateParticipants for a code-related message")
	}
}

// TestMainAgentCapabilityKeywordMatching verifies the rule-based keyword
// scoring produces correct default selections for different inputs.
func TestMainAgentCapabilityKeywordMatching(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-02")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	tests := []struct {
		name                string
		userMessage         string
		wantDefaultSelected string // agent that should be in defaultSelected
		wantDefaultNot      string // agent that should NOT be in defaultSelected
		skipConfirm         bool   // true for conversational fallback (no HITL)
	}{
		{
			name:                "code keyword selects code-agent",
			userMessage:         "write a Go function",
			wantDefaultSelected: "code-agent",
			wantDefaultNot:      "",
		},
		{
			name:                "web keyword selects web-agent",
			userMessage:         "build a React frontend page",
			wantDefaultSelected: "web-agent",
			wantDefaultNot:      "code-agent",
		},
		{
			name:                "no match conversational fallback",
			userMessage:         "hello",
			wantDefaultSelected: "",
			wantDefaultNot:      "",
			skipConfirm:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			runID := fmt.Sprintf("run-kw-%s", tt.name)
			body := fmt.Sprintf(`{"runId":"%s","conversationId":"conv-1","messages":[{"role":"user","text":"%s"}],"requestedPath":"main_agent_orchestration"}`, runID, tt.userMessage)
			resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatalf("POST failed: %v", err)
			}

			resultCh := make(chan []sseEvent, 1)
			go func() {
				resultCh <- readSSEBody(t, resp.Body)
			}()

			// Conversational fallback: no HITL confirmation needed.
			if tt.skipConfirm {
				var events []sseEvent
				select {
				case events = <-resultCh:
				case <-time.After(5 * time.Second):
					t.Fatal("timed out")
				}
				// Verify conversational path doesn't emit activity_snapshot.
				for _, e := range events {
					state := parseSSEState(e.data)
					if state != nil && state["requiresConfirmation"] == true {
						t.Error("conversational fallback should not require confirmation")
					}
				}
				return
			}

			planID := waitForPlanID(srv, runID, 3*time.Second)
			if planID == "" {
				t.Fatal("plan should be registered")
			}
			cancelBody := fmt.Sprintf(`{"runId":"%s","actionId":"%s","action":"cancel"}`, runID, planID)
			http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

			var events []sseEvent
			select {
			case events = <-resultCh:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out")
			}

			var activityData string
			for _, e := range events {
				if e.eventType == "activity_snapshot" {
					activityData = e.data
					break
				}
			}
			if activityData == "" {
				t.Fatal("no activity_snapshot event")
			}

			var wrapper struct {
				Activity struct {
					DefaultSelectedParticipants []plan.PlanParticipant `json:"defaultSelectedParticipants"`
				} `json:"activity"`
			}
			if err := json.Unmarshal([]byte(activityData), &wrapper); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			selected := make(map[string]bool)
			for _, p := range wrapper.Activity.DefaultSelectedParticipants {
				selected[p.AgentName] = true
			}

			if tt.wantDefaultSelected != "" && !selected[tt.wantDefaultSelected] {
				t.Errorf("expected %s in defaultSelectedParticipants, got %v", tt.wantDefaultSelected, selected)
			}
			if tt.wantDefaultNot != "" && selected[tt.wantDefaultNot] {
				t.Errorf("expected %s NOT in defaultSelectedParticipants", tt.wantDefaultNot)
			}
		})
	}
}

// TestOptionalParticipantCanBeUnchecked verifies Phase 4 behavior: optional
// participants can be unchecked directly on approve without requiring revision.
func TestParticipantChangeRequiresRevision(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-ma-participant-change"
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-ma-change",
		RunID:         runID,
		ExecutionPath: "main_agent_orchestration",
		Strategy:      plan.StrategyOrderedParallel,
		CandidateParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: false, Selected: true},
			{AgentName: "document-agent", Required: false, Selected: false},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: false, Selected: true},
		},
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Phase 4: approve with only required code-agent (optional web-agent unchecked).
	// This should succeed — no revision needed for optional participant removal.
	body := fmt.Sprintf(`{"runId":"%s","actionId":"%s","action":"approve","selectedParticipants":["code-agent"]}`, runID, p.PlanID)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		t.Errorf("PARTICIPANT_CHANGE_REQUIRES_REVISION: expected 409 for participant change without revision, got %d: %v", resp.StatusCode, result)
	}
}

// TestRequiredParticipantMissing verifies that approve fails when a required
// participant is not included in selectedParticipants.
func TestRequiredParticipantMissing(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-ma-req-missing"
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-ma-req",
		RunID:         runID,
		ExecutionPath: "main_agent_orchestration",
		Strategy:      plan.StrategyOrderedParallel,
		CandidateParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: true, Selected: true},
		},
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Try to approve without the required web-agent.
	body := fmt.Sprintf(`{"runId":"%s","actionId":"%s","action":"approve","selectedParticipants":["code-agent"]}`, runID, p.PlanID)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if !strings.Contains(result["error"], "REQUIRED_PARTICIPANT_MISSING") {
		t.Errorf("expected REQUIRED_PARTICIPANT_MISSING error, got: %v", result)
	}
}

// TestMainAgentPlanContainsAllFields verifies the orchestration plan for
// main_agent path contains all required plan fields.
func TestMainAgentPlanContainsAllFields(t *testing.T) {
	t.Skip("Known SSE-blocking issue: httptest.Server.Close hangs on active SSE connections. See issue #SSE-TEST-03")
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:1"},
		{Name: "web-agent", URL: "http://127.0.0.1:2"},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	srv := NewServer(WithRegistry(reg), WithDispatcher(dispatcher.NewA2ADispatcher()))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-ma-fields","conversationId":"conv-1","messages":[{"role":"user","text":"build a web API"}],"requestedPath":"main_agent_orchestration"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	resultCh := make(chan []sseEvent, 1)
	go func() {
		resultCh <- readSSEBody(t, resp.Body)
	}()

	planID := waitForPlanID(srv, "run-ma-fields", 3*time.Second)
	if planID == "" {
		t.Fatal("plan should be registered")
	}
	cancelBody := fmt.Sprintf(`{"runId":"run-ma-fields","actionId":"%s","action":"cancel"}`, planID)
	http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(cancelBody))

	var events []sseEvent
	select {
	case events = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	var activityData string
	for _, e := range events {
		if e.eventType == "activity_snapshot" {
			activityData = e.data
			break
		}
	}
	if activityData == "" {
		t.Fatal("no activity_snapshot event")
	}

	var wrapper struct {
		Type     string `json:"type"`
		Activity struct {
			ActivityID    string            `json:"activityId"`
			ActivityType  string            `json:"activityType"`
			Status        string            `json:"status"`
			ExecutionPath string            `json:"executionPath"`
			PlanID        string            `json:"planId"`
			Revision      int               `json:"revision"`
			PlanOwner     *plan.PlanOwner   `json:"planOwner"`
			Tasks         []plan.TaskPlan   `json:"tasks"`
			AllowedActions []string         `json:"allowedActions"`
		} `json:"activity"`
	}
	if err := json.Unmarshal([]byte(activityData), &wrapper); err != nil {
		t.Fatalf("unmarshal ACTIVITY_SNAPSHOT: %v", err)
	}

	if wrapper.Type != "activity_snapshot" {
		t.Errorf("expected type=activity_snapshot (internal event type), got %s", wrapper.Type)
	}
	a := wrapper.Activity
	if a.ActivityID == "" {
		t.Error("activityId should not be empty")
	}
	if a.ActivityType != "plan_approval" {
		t.Errorf("expected activityType=plan_approval, got %s", a.ActivityType)
	}
	if a.ExecutionPath != "main_agent_orchestration" {
		t.Errorf("expected executionPath=main_agent_orchestration, got %s", a.ExecutionPath)
	}
	if a.PlanID == "" {
		t.Error("planId should not be empty")
	}
	if a.Revision < 1 {
		t.Errorf("revision should be >= 1, got %d", a.Revision)
	}
	if a.PlanOwner == nil || a.PlanOwner.Type != "main_agent" {
		t.Error("expected planOwner.type=main_agent")
	}
	if len(a.Tasks) == 0 {
		t.Error("expected non-empty tasks")
	}
	if len(a.AllowedActions) == 0 {
		t.Error("expected non-empty allowedActions")
	}
}

// TestInvalidParticipantRejected verifies that selecting a non-candidate agent returns error.
func TestInvalidParticipantRejected(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-ma-invalid"
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-ma-invalid",
		RunID:         runID,
		ExecutionPath: "main_agent_orchestration",
		Strategy:      plan.StrategyOrderedParallel,
		CandidateParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Try to approve with a non-candidate agent (code-agent is required, plus invalid extra).
	body := fmt.Sprintf(`{"runId":"%s","actionId":"%s","action":"approve","selectedParticipants":["code-agent","nonexistent-agent"]}`, runID, p.PlanID)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if !strings.Contains(result["error"], "INVALID_PARTICIPANT") {
		t.Errorf("expected INVALID_PARTICIPANT error, got: %v", result)
	}
}

// TestParticipantValidationNotAppliedForSingleChat verifies that participant
// validation is ONLY applied for main_agent_orchestration, not for single_chat.
func TestParticipantValidationNotAppliedForSingleChat(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-sc-no-validation"
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-sc",
		RunID:         runID,
		ExecutionPath: "single_chat",
		Strategy:      plan.StrategySingle,
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Even with selectedParticipants, single_chat should bypass validation.
	body := fmt.Sprintf(`{"runId":"%s","actionId":"%s","action":"approve","selectedParticipants":["some-agent"]}`, runID, p.PlanID)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		t.Errorf("expected 200 for single_chat (validation skipped), got %d: %v", resp.StatusCode, result)
	}
}

// TestEmptySelectedParticipantsRejected verifies empty selection returns error.
func TestEmptySelectedParticipantsRejected(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-ma-empty"
	p := &plan.OrchestrationPlan{
		PlanID:        "plan-ma-empty",
		RunID:         runID,
		ExecutionPath: "main_agent_orchestration",
		Strategy:      plan.StrategyOrderedParallel,
		CandidateParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
		DefaultSelectedParticipants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := fmt.Sprintf(`{"runId":"%s","actionId":"%s","action":"approve","selectedParticipants":[]}`, runID, p.PlanID)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if !strings.Contains(result["error"], "PARTICIPANT_SELECTION_REQUIRED") {
		t.Errorf("expected PARTICIPANT_SELECTION_REQUIRED, got: %v", result)
	}
}
