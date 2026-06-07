package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
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
	summaries := taskSummaries(p)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 task summary, got %d", len(summaries))
	}
	s := summaries[0]
	if s["taskId"] != "task_001" {
		t.Errorf("expected taskId=task_001, got %v", s["taskId"])
	}
	if s["agentName"] != "code-agent" {
		t.Errorf("expected agentName=code-agent, got %v", s["agentName"])
	}
	if s["priority"] != 1 {
		t.Errorf("expected priority=1, got %v", s["priority"])
	}
}

func TestTaskSummariesNil(t *testing.T) {
	if summaries := taskSummaries(nil); summaries != nil {
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
		"tasks":                taskSummaries(p),
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
		"tasks":                taskSummaries(p),
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

func TestHeartbeatEventFormat(t *testing.T) {
	// Verify that heartbeat STATE_UPDATE events sent during awaiting_confirmation
	// contain all required fields and are valid JSON.
	heartbeatEvent := OrchestratorStreamEvent{
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

	var parsed OrchestratorStreamEvent
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
	heartbeatEvent := OrchestratorStreamEvent{
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
		"tasks":                taskSummaries(p),
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
