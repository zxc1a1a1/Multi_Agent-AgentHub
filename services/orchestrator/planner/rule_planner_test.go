package planner

import (
	"context"
	"testing"
)

func testAgents() []string {
	return []string{"code-agent", "web-agent"}
}

func TestRulePlannerExplicitAgentName(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_001",
		ConversationID:  "conv_001",
		UserMessage:     "do something",
		AgentName:       "web-agent",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "single" {
		t.Errorf("expected strategy single, got %q", plan.Strategy)
	}
	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected web-agent, got %q", plan.Tasks[0].AgentName)
	}
	if plan.Validation.Validated {
		t.Error("expected validated=false")
	}
	if plan.PlanningMode != "" {
		// PlanningMode from input is passed through, empty is fine for this test.
	}
}

func TestRulePlannerSelectedAgentNames(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:              "run_002",
		ConversationID:     "conv_002",
		UserMessage:        "build something",
		SelectedAgentNames: []string{"code-agent"},
		AvailableAgents:    testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent from selectedAgentNames, got %q", plan.Tasks[0].AgentName)
	}
}

func TestRulePlannerAgentNameOverridesSelected(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:              "run_003",
		ConversationID:     "conv_003",
		UserMessage:        "build something",
		AgentName:          "web-agent",
		SelectedAgentNames: []string{"code-agent"},
		AvailableAgents:    testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected agentName (web-agent) to take priority over selectedAgentNames, got %q", plan.Tasks[0].AgentName)
	}
}

func TestRulePlannerWebKeywords(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	tests := []string{
		"帮我写一个登录页面",
		"create a React UI component",
		"设计一个 HTML 前端页面",
		"make a webpage layout with CSS",
	}
	for _, msg := range tests {
		plan, err := rp.Plan(context.Background(), PlannerInput{
			RunID:           "run_web",
			ConversationID:  "conv_web",
			UserMessage:     msg,
			AvailableAgents: testAgents(),
		})
		if err != nil {
			t.Fatalf("msg=%q unexpected error: %v", msg, err)
		}
		if plan.Tasks[0].AgentName != "web-agent" {
			t.Errorf("msg=%q expected web-agent, got %q", msg, plan.Tasks[0].AgentName)
		}
		if len(plan.Tasks[0].CapabilityIDs) == 0 {
			t.Errorf("msg=%q expected non-empty capabilityIds", msg)
		}
		if len(plan.Tasks[0].ExpectedOutputs) == 0 {
			t.Errorf("msg=%q expected non-empty expectedOutputs", msg)
		}
	}
}

func TestRulePlannerCodeKeywords(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	tests := []string{
		"帮我写一个 Go API server",
		"implement a backend endpoint",
		"写一个 SQL database handler",
		"create a Golang service",
	}
	for _, msg := range tests {
		plan, err := rp.Plan(context.Background(), PlannerInput{
			RunID:           "run_code",
			ConversationID:  "conv_code",
			UserMessage:     msg,
			AvailableAgents: testAgents(),
		})
		if err != nil {
			t.Fatalf("msg=%q unexpected error: %v", msg, err)
		}
		if plan.Tasks[0].AgentName != "code-agent" {
			t.Errorf("msg=%q expected code-agent, got %q", msg, plan.Tasks[0].AgentName)
		}
	}
}

func TestRulePlannerMixedKeywordsCreatesOrderedParallelPlan(t *testing.T) {
	// Mixed web+code keywords must produce an ordered_parallel plan with 2 tasks.
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_mixed",
		ConversationID:  "conv_mixed",
		UserMessage:     "帮我做一个登录页面和 Go 登录接口",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "ordered_parallel" {
		t.Errorf("expected strategy ordered_parallel, got %q", plan.Strategy)
	}
	if len(plan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks (ordered_parallel), got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected tasks[0] web-agent, got %q", plan.Tasks[0].AgentName)
	}
	if plan.Tasks[1].AgentName != "code-agent" {
		t.Errorf("expected tasks[1] code-agent, got %q", plan.Tasks[1].AgentName)
	}
	if plan.Validation.Validated {
		t.Error("expected validated=false")
	}
	if !plan.Aggregation.Required {
		t.Error("expected aggregation.required=true for ordered_parallel")
	}
	if plan.Aggregation.Mode != "summary" {
		t.Errorf("expected aggregation mode=summary, got %q", plan.Aggregation.Mode)
	}
}

func TestRulePlannerUnknownDefaultsToCodeAgent(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_unknown",
		ConversationID:  "conv_unknown",
		UserMessage:     "hello, how are you?",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected default code-agent, got %q", plan.Tasks[0].AgentName)
	}
}

func TestRulePlannerUnknownAgentNameFallsThrough(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_unknown_agent",
		ConversationID:  "conv_unknown_agent",
		UserMessage:     "写一个 HTML 页面",
		AgentName:       "nonexistent-agent",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown agentName is ignored; keywords route to web-agent.
	if plan.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected web-agent (keyword match), got %q", plan.Tasks[0].AgentName)
	}
}

func TestRulePlannerTaskFields(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_task_fields",
		ConversationID:  "conv_task_fields",
		UserMessage:     "make a Go API",
		PlanningMode:    "auto",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := plan.Tasks[0]
	if task.TaskID != "task_001" {
		t.Errorf("expected task_001, got %q", task.TaskID)
	}
	if task.TaskContent != "make a Go API" {
		t.Errorf("expected taskContent to match userMessage, got %q", task.TaskContent)
	}
	if task.TimeoutMs != 120000 {
		t.Errorf("expected timeoutMs=120000, got %d", task.TimeoutMs)
	}
	if task.RiskLevel != "low" {
		t.Errorf("expected riskLevel=low, got %q", task.RiskLevel)
	}
	if task.Priority != 1 {
		t.Errorf("expected priority=1, got %d", task.Priority)
	}
	if len(task.DependsOn) != 0 {
		t.Errorf("expected empty dependsOn, got %v", task.DependsOn)
	}
	if plan.PlanningMode != "auto" {
		t.Errorf("expected planningMode=auto, got %q", plan.PlanningMode)
	}
}

func TestRulePlannerImplementsPlanner(t *testing.T) {
	// Compile-time check via var _ in rule_planner.go; runtime sanity.
	var p Planner = NewRulePlanner(testAgents())
	if p == nil {
		t.Fatal("expected non-nil Planner")
	}
}

func TestRulePlannerIntentSummary(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	longMsg := "this is a very long message that exceeds the maximum summary length limit of one hundred and twenty characters to test truncation behavior in the summarizer"
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_summary",
		ConversationID:  "conv_summary",
		UserMessage:     longMsg,
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.IntentSummary) > 120 {
		t.Errorf("expected summary <= 120 chars, got %d: %q", len(plan.IntentSummary), plan.IntentSummary)
	}
}

func TestRulePlannerFallbackConfig(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_fallback",
		ConversationID:  "conv_fallback",
		UserMessage:     "do something",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !plan.Fallback.Enabled {
		t.Error("expected fallback enabled")
	}
	if plan.Fallback.Reason != "rule_default" {
		t.Errorf("expected fallback reason=rule_default, got %q", plan.Fallback.Reason)
	}
}

func TestRulePlannerEmptyAgents(t *testing.T) {
	rp := NewRulePlanner(nil)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:       "run_empty",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls through to "code-agent" even if not in registry.
	if plan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected fallback to code-agent, got %q", plan.Tasks[0].AgentName)
	}
}
