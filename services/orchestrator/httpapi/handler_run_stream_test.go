package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/executionpath"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

func testPlanSingle() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		PlanID:         "plan_001",
		RunID:          "run_001",
		ConversationID: "conv_001",
		Strategy:       plan.StrategySingle,
		IntentSummary:  "generate code",
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_001",
				AgentName:       "code-agent",
				TaskContent:     "write Go code",
				DependsOn:       []string{},
				Priority:        1,
				RiskLevel:       "low",
				TimeoutMs:       120000,
				CapabilityIDs:   []string{"code_generation"},
				ExpectedOutputs: []string{"code"},
			},
		},
		Aggregation: plan.Aggregation{Required: false, Mode: "none"},
	}
}

func testPlanMultiAgent() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		PlanID:         "plan_002",
		RunID:          "run_002",
		ConversationID: "conv_002",
		Strategy:       plan.StrategyOrderedParallel,
		IntentSummary:  "build login page and API",
		PlanningMode:   "auto",
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_web-agent",
				AgentName:       "web-agent",
				TaskContent:     "create login page",
				DependsOn:       []string{},
				Priority:        1,
				RiskLevel:       "low",
				TimeoutMs:       120000,
				CapabilityIDs:   []string{"web_generation"},
				ExpectedOutputs: []string{"webpage"},
			},
			{
				TaskID:          "task_code-agent",
				AgentName:       "code-agent",
				TaskContent:     "create login API",
				DependsOn:       []string{},
				Priority:        2,
				RiskLevel:       "low",
				TimeoutMs:       120000,
				CapabilityIDs:   []string{"code_generation"},
				ExpectedOutputs: []string{"code"},
			},
		},
		Aggregation: plan.Aggregation{Required: true, Mode: "summary"},
	}
}

func testPlanConversational() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		PlanID:         "plan_003",
		RunID:          "run_003",
		ConversationID: "conv_003",
		Strategy:       plan.StrategyConversational,
		IntentSummary:  "greeting",
		Aggregation:    plan.Aggregation{Required: false, Mode: "none"},
	}
}

func testPlanSequential() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		PlanID:         "plan_004",
		RunID:          "run_004",
		ConversationID: "conv_004",
		Strategy:       plan.StrategySequential,
		IntentSummary:  "code generation with security review",
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_code-agent",
				AgentName:       "code-agent",
				TaskContent:     "write code",
				DependsOn:       []string{},
				Priority:        1,
				RiskLevel:       "low",
				TimeoutMs:       120000,
				CapabilityIDs:   []string{"code_generation"},
				ExpectedOutputs: []string{"code"},
			},
			{
				TaskID:          "task_security-agent",
				AgentName:       "security-agent",
				TaskContent:     "review code for security",
				DependsOn:       []string{"task_code-agent"},
				Priority:        2,
				RiskLevel:       "low",
				TimeoutMs:       120000,
			},
		},
		Aggregation: plan.Aggregation{Required: true, Mode: "summary"},
	}
}

func TestBuildPlanState(t *testing.T) {
	p := testPlanMultiAgent()
	state := buildPlanState(p)

	if state["planId"] != "plan_002" {
		t.Errorf("expected planId=plan_002, got %v", state["planId"])
	}
	if state["strategy"] != "ordered_parallel" {
		t.Errorf("expected strategy=ordered_parallel, got %v", state["strategy"])
	}
	if state["taskCount"] != 2 {
		t.Errorf("expected taskCount=2, got %v", state["taskCount"])
	}
	if state["planningMode"] != "auto" {
		t.Errorf("expected planningMode=auto, got %v", state["planningMode"])
	}
}

func TestBuildPlanStateNil(t *testing.T) {
	state := buildPlanState(nil)
	if state == nil {
		t.Fatal("expected non-nil state from nil plan")
	}
}

func TestPlannedAgentNames(t *testing.T) {
	p := testPlanMultiAgent()
	names := plannedAgentNames(p)
	if len(names) != 2 {
		t.Fatalf("expected 2 agent names, got %d", len(names))
	}
	hasWeb := false
	hasCode := false
	for _, n := range names {
		if n == "web-agent" {
			hasWeb = true
		}
		if n == "code-agent" {
			hasCode = true
		}
	}
	if !hasWeb || !hasCode {
		t.Errorf("expected web-agent and code-agent, got %v", names)
	}
}

func TestPlannedAgentNamesNil(t *testing.T) {
	if names := plannedAgentNames(nil); names != nil {
		t.Errorf("expected nil from nil plan, got %v", names)
	}
}

func TestTaskSummaries(t *testing.T) {
	p := testPlanSingle()
	summaries := toAGUITaskSummaries(p.Tasks)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 task summary, got %d", len(summaries))
	}
	s := summaries[0]
	if s.TaskID != "task_001" {
		t.Errorf("expected taskId=task_001, got %v", s.TaskID)
	}
	if s.AgentName != "code-agent" {
		t.Errorf("expected agentName=code-agent, got %v", s.AgentName)
	}
	if s.Priority != 1 {
		t.Errorf("expected priority=1, got %v", s.Priority)
	}
}

func TestTaskSummariesNil(t *testing.T) {
	if summaries := toAGUITaskSummaries(nil); summaries != nil {
		t.Errorf("expected nil from nil plan, got %v", summaries)
	}
}

func TestConversationalResponse(t *testing.T) {
	resp := conversationalResponse()
	if !strings.Contains(resp, "AgentHub") {
		t.Error("conversational response must mention AgentHub")
	}
	if !strings.Contains(resp, "自动编排") {
		t.Error("conversational response must mention 自动编排")
	}
	if strings.Contains(resp, "code-agent") {
		t.Error("conversational response must not expose internal agent names")
	}
}

func TestConfirmArgsJSON(t *testing.T) {
	p := testPlanMultiAgent()
	args := map[string]any{
		"runId":                p.RunID,
		"planId":               p.PlanID,
		"strategy":             p.Strategy,
		"plannedAgents":        plannedAgentNames(p),
		"tasks":                toAGUITaskSummaries(p.Tasks),
		"intentSummary":        p.IntentSummary,
		"requiresConfirmation": true,
	}
	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal confirm args: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal confirm args: %v", err)
	}
	if parsed["requiresConfirmation"] != true {
		t.Error("requiresConfirmation must be true")
	}
	if parsed["strategy"] != "ordered_parallel" {
		t.Errorf("expected strategy=ordered_parallel, got %v", parsed["strategy"])
	}
}

func TestConfirmArgsSequentialPlan(t *testing.T) {
	p := testPlanSequential()
	args := map[string]any{
		"runId":                p.RunID,
		"planId":               p.PlanID,
		"strategy":             p.Strategy,
		"plannedAgents":        plannedAgentNames(p),
		"tasks":                toAGUITaskSummaries(p.Tasks),
		"intentSummary":        p.IntentSummary,
		"requiresConfirmation": true,
	}
	data, _ := json.Marshal(args)
	var parsed map[string]any
	json.Unmarshal(data, &parsed)
	if parsed["strategy"] != "sequential" {
		t.Errorf("expected strategy=sequential, got %v", parsed["strategy"])
	}
	tasksArr, ok := parsed["tasks"].([]any)
	if !ok || len(tasksArr) != 2 {
		t.Fatalf("expected 2 tasks, got %v", parsed["tasks"])
	}
}

func TestBuildRevisionPlanOnlyMessage(t *testing.T) {
	msg := buildRevisionPlanOnlyMessage(
		"帮我写一个登录页面",
		"简化步骤，不要包含测试",
		2,
		"使用 React 和 Tailwind 创建登录页面",
	)

	if !strings.Contains(msg, "原始用户任务") {
		t.Error("must contain '原始用户任务' header")
	}
	if !strings.Contains(msg, "帮我写一个登录页面") {
		t.Error("must contain original userText")
	}
	if !strings.Contains(msg, "用户对上一版方案的修改意见") {
		t.Error("must contain feedback header")
	}
	if !strings.Contains(msg, "简化步骤，不要包含测试") {
		t.Error("must contain feedback text")
	}
	if !strings.Contains(msg, "上一版方案摘要") {
		t.Error("must contain previous plan summary header")
	}
	if !strings.Contains(msg, "使用 React 和 Tailwind 创建登录页面") {
		t.Error("must contain previous plan summary text")
	}
	if !strings.Contains(msg, "revision 2") {
		t.Error("must contain revision number 2")
	}
	if !strings.Contains(msg, "只返回 JSON plan") {
		t.Error("must contain plan_only instruction")
	}
}

func TestBuildRevisionPlanOnlyMessage_NoPreviousSummary(t *testing.T) {
	msg := buildRevisionPlanOnlyMessage(
		"帮我写一个登录页面",
		"简化步骤",
		1,
		"",
	)

	if !strings.Contains(msg, "帮我写一个登录页面") {
		t.Error("must contain original userText")
	}
	if !strings.Contains(msg, "简化步骤") {
		t.Error("must contain feedback")
	}
	if !strings.Contains(msg, "revision 1") {
		t.Error("must contain revision number 1")
	}
	if strings.Contains(msg, "上一版方案摘要") {
		t.Error("must NOT contain previous plan summary when empty")
	}
}

func TestHeartbeatEventFormat(t *testing.T) {
	// Verify that heartbeat STATE_UPDATE events sent during awaiting_confirmation
	// contain all required fields and are valid JSON.
	heartbeatEvent := agui.InternalStreamEvent{
		Type:  "state_update",
		RunID: "run_hb_001",
		State: map[string]any{
			"phase":                "awaiting_confirmation",
			"requiresConfirmation": true,
			"confirmationActionId": "plan_hb_001",
			"heartbeat":            true,
		},
	}

	data, err := json.Marshal(heartbeatEvent)
	if err != nil {
		t.Fatalf("heartbeat event must marshal to valid JSON: %v", err)
	}

	var parsed agui.InternalStreamEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("heartbeat event must round-trip: %v", err)
	}
	if parsed.Type != "state_update" {
		t.Errorf("expected type=state_update, got %q", parsed.Type)
	}
	if parsed.RunID != "run_hb_001" {
		t.Errorf("expected runId=run_hb_001, got %q", parsed.RunID)
	}
	if parsed.State == nil {
		t.Fatal("expected non-nil state in heartbeat event")
	}
	if phase, ok := parsed.State["phase"].(string); !ok || phase != "awaiting_confirmation" {
		t.Errorf("expected phase=awaiting_confirmation, got %v", parsed.State["phase"])
	}
	if heartbeat, ok := parsed.State["heartbeat"].(bool); !ok || !heartbeat {
		t.Errorf("expected heartbeat=true, got %v", parsed.State["heartbeat"])
	}
	if parsed.State["requiresConfirmation"] != true {
		t.Error("requiresConfirmation must be true in heartbeat")
	}
	if parsed.State["confirmationActionId"] != "plan_hb_001" {
		t.Errorf("expected confirmationActionId in heartbeat, got %v", parsed.State["confirmationActionId"])
	}
}

func TestHeartbeatEventDoesNotChangeRunPhase(t *testing.T) {
	// Heartbeat events must carry heartbeat=true so the frontend can
	// distinguish them from phase transitions (e.g. awaiting_confirmation → executing).
	heartbeatEvent := agui.InternalStreamEvent{
		Type:  "state_update",
		RunID: "run_hb_002",
		State: map[string]any{
			"phase":     "awaiting_confirmation",
			"heartbeat": true,
		},
	}

	data, _ := json.Marshal(heartbeatEvent)

	var parsed map[string]any
	json.Unmarshal(data, &parsed)

	state, ok := parsed["state"].(map[string]any)
	if !ok {
		t.Fatal("state must be an object")
	}
	// A heartbeat must not be mistaken for a phase transition.
	if state["heartbeat"] != true {
		t.Error("heartbeat=true required for frontend filtering")
	}
	if state["phase"] != "awaiting_confirmation" {
		t.Error("phase must remain awaiting_confirmation in heartbeat")
	}
}

func TestConfirmPlanEventCompleteness(t *testing.T) {
	// Verify that the confirm_plan TOOL_CALL payload contains all fields
	// needed by the frontend to render HITLConfirm dialog.
	p := testPlanSequential()
	args := map[string]any{
		"runId":                p.RunID,
		"planId":               p.PlanID,
		"strategy":             p.Strategy,
		"plannedAgents":        plannedAgentNames(p),
		"tasks":                toAGUITaskSummaries(p.Tasks),
		"intentSummary":        p.IntentSummary,
		"requiresConfirmation": true,
	}

	data, _ := json.Marshal(args)
	var parsed map[string]any
	json.Unmarshal(data, &parsed)

	required := []string{"runId", "planId", "strategy", "plannedAgents", "tasks", "requiresConfirmation"}
	for _, key := range required {
		if _, ok := parsed[key]; !ok {
			t.Errorf("confirm_plan args missing required field: %q", key)
		}
	}

	agents, ok := parsed["plannedAgents"].([]any)
	if !ok || len(agents) == 0 {
		t.Error("plannedAgents must be non-empty in confirm_plan")
	}
	tasks, ok := parsed["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Error("tasks must be non-empty in confirm_plan")
	}
}

func TestConversationalPlanHasNoPlannedAgents(t *testing.T) {
	p := testPlanConversational()
	names := plannedAgentNames(p)
	if len(names) != 0 {
		t.Errorf("conversational plan should have 0 planned agents, got %d", len(names))
	}
}

// TestPrePlannerValidationAGENT_SELECTION_CONFLICT proves that when
// requestedPath="group_chat" with non-overlapping selectedAgentNames and
// mentions, the validation chain (DeriveExecutionPath → ValidateAgentSelection)
// returns AGENT_SELECTION_CONFLICT before the Planner is ever invoked.
//
// The handler calls these exact functions in this order. If ValidateAgentSelection
// returns an error, the handler returns early — no PlannerInput is built, no
// Planner.Plan() is called, and no agent is dispatched.
func TestPrePlannerValidationAgentSelectionConflict(t *testing.T) {
	// Simulate the exact handler flow before Planner is called.
	// Step 1: Derive execution path from request fields.
	derivedPath, deriveErr := executionpath.DeriveExecutionPath(
		"group_chat",       // requestedPath
		"",                 // agentName
		[]string{"code-agent"}, // selectedAgentNames
		[]string{"web-agent"},  // mentions
	)
	if deriveErr != nil {
		t.Fatalf("DeriveExecutionPath should succeed for valid input: %v", deriveErr)
	}
	if derivedPath != executionpath.PathGroupChat {
		t.Fatalf("expected PathGroupChat, got %q", derivedPath)
	}

	// Step 2: Validate agent selection (same call the handler makes).
	result := executionpath.ValidateAgentSelection(
		derivedPath,
		"",
		[]string{"code-agent"},
		[]string{"web-agent"},
		[]string{"code-agent", "web-agent", "review-agent"},
	)
	if result.Error == nil {
		t.Fatal("expected AGENT_SELECTION_CONFLICT error, but got nil")
	}
	if result.Error.Code != executionpath.ErrCodeAgentSelectionConflict {
		t.Fatalf("expected error code %q, got %q", executionpath.ErrCodeAgentSelectionConflict, result.Error.Code)
	}

	// At this point in the handler, an error would be emitted via SSE and
	// the handler returns — no PlannerInput, no Planner.Plan(), no plan, no
	// agent dispatch. This test proves the validation catches the conflict
	// before any orchestration work begins.
}

// ---------------------------------------------------------------------------
// Pin context tests (Phase 8 — pinned messages reach downstream)
// ---------------------------------------------------------------------------

func TestSafeRoleLabelEmpty(t *testing.T) {
	if got := safeRoleLabel(""); got != "Unknown" {
		t.Errorf("empty role: expected Unknown, got %q", got)
	}
}

func TestSafeRoleLabelNormal(t *testing.T) {
	if got := safeRoleLabel("user"); got != "User" {
		t.Errorf("user role: expected User, got %q", got)
	}
	if got := safeRoleLabel("assistant"); got != "Assistant" {
		t.Errorf("assistant role: expected Assistant, got %q", got)
	}
}

func TestSafeRoleLabelSingleChar(t *testing.T) {
	if got := safeRoleLabel("u"); got != "U" {
		t.Errorf("single char: expected U, got %q", got)
	}
}

func TestSafeRoleLabelWhitespace(t *testing.T) {
	if got := safeRoleLabel("  "); got != "Unknown" {
		t.Errorf("whitespace: expected Unknown, got %q", got)
	}
}

func TestBuildPinnedUserText_EmptyRoleDoesNotPanic(t *testing.T) {
	messages := []MessageInput{
		{ID: "m1", Role: "", Text: "message with empty role"},
		{ID: "m2", Role: "user", Text: "hello"},
	}
	pinnedIDs := []string{"m1"}
	// Must not panic — safeRoleLabel handles empty role.
	result := buildPinnedUserText(messages, pinnedIDs, "current message")
	if !strings.Contains(result, "Unknown:") {
		t.Errorf("expected 'Unknown:' label for empty role, got: %q", result)
	}
}

func TestBuildPinnedUserText_PinnedOutsideWindowRetained(t *testing.T) {
	// Build 30 messages so head=20 tail=4 window doesn't include msgs 5-10.
	// Pin msg at index 5 — it must appear in output despite being outside window.
	messages := make([]MessageInput, 30)
	for i := range messages {
		messages[i] = MessageInput{
			ID:   "msg-" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
			Role: "user",
			Text: "message " + string(rune('0'+i)),
		}
	}
	// Insert a clearly identifiable pinned message in the middle.
	messages[5] = MessageInput{ID: "pinned-1", Role: "user", Text: "PINNED OLD MESSAGE"}
	messages[15] = MessageInput{ID: "unpinned-middle", Role: "assistant", Text: "UNPINNED MIDDLE"}

	pinnedIDs := []string{"pinned-1"}
	result := buildPinnedUserText(messages, pinnedIDs, "current user message")
	if !strings.Contains(result, "PINNED OLD MESSAGE") {
		t.Errorf("pinned old message must appear in output, got: %q", result)
	}
}

func TestBuildPinnedUserText_UnpinnedOutsideWindowPruned(t *testing.T) {
	// Build 30 messages. head=20, tail=4 means indices 0-19 (head) and 26-29
	// (tail) are kept. An unpinned message at index 22 falls between windows
	// and must be pruned.
	messages := make([]MessageInput, 30)
	for i := range messages {
		messages[i] = MessageInput{
			ID:   fmt.Sprintf("msg-%02d", i),
			Role: "user",
			Text: fmt.Sprintf("message %d", i),
		}
	}
	messages[22] = MessageInput{ID: "unpinned-middle", Role: "assistant", Text: "UNPINNED MIDDLE"}

	result := buildPinnedUserText(messages, nil, "current user message")
	if strings.Contains(result, "UNPINNED MIDDLE") {
		t.Errorf("unpinned old message outside window must be pruned, but appears in: %q", result)
	}
}

func TestBuildPinnedUserText_UnknownPinnedIDIgnored(t *testing.T) {
	messages := []MessageInput{
		{ID: "m1", Role: "user", Text: "hello"},
		{ID: "m2", Role: "assistant", Text: "world"},
	}
	pinnedIDs := []string{"nonexistent-1", "nonexistent-2"}
	result := buildPinnedUserText(messages, pinnedIDs, "current message")
	// Unknown IDs are silently ignored — output should still have [Pinned conversation context].
	if !strings.Contains(result, "[Pinned conversation context]") {
		t.Errorf("expected pinned context block even with unknown IDs, got: %q", result)
	}
}

func TestBuildPinnedUserText_NoMessages(t *testing.T) {
	result := buildPinnedUserText(nil, []string{"id1"}, "current message")
	if result != "current message" {
		t.Errorf("no messages should return current user text unchanged, got: %q", result)
	}
}

func TestBuildPinnedUserText_DeduplicatesCurrentMessage(t *testing.T) {
	// When the last message in filtered window has the same text as the current
	// user message, [Current user message] block must not be added.
	messages := []MessageInput{
		{ID: "m1", Role: "user", Text: "hello"},
		{ID: "m2", Role: "assistant", Text: "hi there"},
	}
	result := buildPinnedUserText(messages, nil, "hi there")
	if strings.Contains(result, "[Current user message]") {
		t.Errorf("duplicate current message must not add [Current user message] block, got: %q", result)
	}
	if !strings.Contains(result, "hi there") {
		t.Errorf("output must contain the message text, got: %q", result)
	}
}

func TestBuildPinnedUserText_FormatIsStable(t *testing.T) {
	messages := []MessageInput{
		{ID: "m1", Role: "user", Text: "first question"},
	}
	result := buildPinnedUserText(messages, []string{"m1"}, "second question")
	if !strings.Contains(result, "[Pinned conversation context]") {
		t.Errorf("must have [Pinned conversation context] block")
	}
	if !strings.Contains(result, "[/Pinned conversation context]") {
		t.Errorf("must have [/Pinned conversation context] closing tag")
	}
	if !strings.Contains(result, "[Current user message]") {
		t.Errorf("must have [Current user message] block for new message")
	}
	if !strings.Contains(result, "second question") {
		t.Errorf("must contain current user text")
	}
}

// ---------------------------------------------------------------------------
// Artifact SSE E2E test
// ---------------------------------------------------------------------------

// mockA2AArtifactServer returns an httptest server that produces a buffered
// JSON A2A RunResponse containing an artifact_metadata tool_result part.
func mockA2AArtifactServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		artifactMeta := map[string]any{
			"id":       "art-001",
			"name":     "report.md",
			"kind":     "text",
			"mimeType": "text/markdown",
			"size":     1024,
		}
		artifactJSON, _ := json.Marshal(artifactMeta)
		resp := map[string]any{
			"taskId": "task-art-001",
			"status": "completed",
			"events": []map[string]any{
				{
					"author":  "code-agent",
					"role":    "assistant",
					"final":   true,
					"partial": false,
					"parts": []map[string]any{
						{"type": "text", "text": "I created a report for you."},
						{
							"type":    "tool_result",
							"name":    "artifact_metadata",
							"content": string(artifactJSON),
							"callId":  "call-art-1",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRunStreamEmitsArtifactDeltaSSE(t *testing.T) {
	os.Setenv("REQUIRE_PLAN_CONFIRMATION", "")
	defer os.Unsetenv("REQUIRE_PLAN_CONFIRMATION")

	mockAgent := mockA2AArtifactServer(t)

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: mockAgent.URL, Description: "code agent", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text"}},
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	artStore, err := artifacts.NewJSONStore(filepath.Join(t.TempDir(), "artifacts.json"))
	if err != nil {
		t.Fatalf("create artifact store: %v", err)
	}

	fake := &FakeMainAgent{
		PlanToReturn: makeTestPlan("run-art-test", plan.StrategySingle, []plan.TaskPlan{
			makeTaskPlan("task-1", "code-agent", "write code"),
		}),
	}
	fake.PlanToReturn.ExecutionPath = "single_chat"
	fake.PlanToReturn.Participants = []plan.PlanParticipant{
		{AgentName: "code-agent", Required: true, Selected: true},
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(fake),
		WithArtifactStore(artStore),
	)
	orchTS := httptest.NewServer(srv.Handler())
	defer orchTS.Close()

	body := fmt.Sprintf(`{"runId":"run-art-test","conversationId":"conv-art","messages":[{"role":"user","text":"write code"}],"selectedAgentNames":["code-agent"]}`)
	resp, err := http.Post(orchTS.URL+"/internal/orchestrator/runs/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	events := readSSEBody(t, resp.Body)

	if _, ok := findEvent(events, "run_started"); !ok {
		t.Error("expected run_started event")
	}
	if _, ok := findEvent(events, "run_finished"); !ok {
		t.Error("expected run_finished event")
	}

	var artifactEvents []sseEvent
	for _, ev := range events {
		if ev.eventType == "artifact.delta" {
			artifactEvents = append(artifactEvents, ev)
		}
	}
	if len(artifactEvents) == 0 {
		t.Fatal("expected at least one artifact.delta SSE event")
	}

	for _, ae := range artifactEvents {
		var parsed struct {
			Type     string `json:"type"`
			Artifact *struct {
				Type     string            `json:"type"`
				Title    string            `json:"title"`
				Metadata map[string]string `json:"metadata"`
			} `json:"artifact"`
		}
		if err := json.Unmarshal([]byte(ae.data), &parsed); err != nil {
			t.Errorf("failed to parse artifact event data: %v", err)
			continue
		}
		if parsed.Artifact == nil {
			t.Error("artifact.delta event missing 'artifact' field")
			continue
		}
		if parsed.Artifact.Type != "text" {
			t.Errorf("expected artifact type=text, got %q", parsed.Artifact.Type)
		}
		if parsed.Artifact.Title != "report.md" {
			t.Errorf("expected artifact title=report.md, got %q", parsed.Artifact.Title)
		}
		if parsed.Artifact.Metadata["id"] != "art-001" {
			t.Errorf("expected artifact metadata id=art-001, got %q", parsed.Artifact.Metadata["id"])
		}
		if parsed.Artifact.Metadata["mimeType"] != "text/markdown" {
			t.Errorf("expected artifact metadata mimeType=text/markdown, got %q", parsed.Artifact.Metadata["mimeType"])
		}
		if parsed.Artifact.Metadata["size"] != "1024" {
			t.Errorf("expected artifact metadata size=1024, got %q", parsed.Artifact.Metadata["size"])
		}
		if parsed.Artifact.Metadata["sourceAgent"] != "code-agent" {
			t.Errorf("expected artifact metadata sourceAgent=code-agent, got %q", parsed.Artifact.Metadata["sourceAgent"])
		}
	}
}
