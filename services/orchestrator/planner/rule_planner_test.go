package planner

import (
	"context"
	"strings"
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
		UserMessage:     "do something technical",
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

func TestRulePlannerConversationalIntentChinese(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	tests := []string{
		"你好",
		"你是谁",
		"你能做什么",
		"你能干嘛",
		"帮我介绍一下",
		"介绍一下你自己",
		"有什么功能",
		"帮助",
	}
	for _, msg := range tests {
		plan, err := rp.Plan(context.Background(), PlannerInput{
			RunID:           "run_conv",
			ConversationID:  "conv_conv",
			UserMessage:     msg,
			AvailableAgents: testAgents(),
		})
		if err != nil {
			t.Fatalf("msg=%q unexpected error: %v", msg, err)
		}
		if plan.Strategy != "conversational" {
			t.Errorf("msg=%q expected strategy conversational, got %q", msg, plan.Strategy)
		}
		if len(plan.Tasks) != 0 {
			t.Errorf("msg=%q expected 0 tasks for conversational, got %d", msg, len(plan.Tasks))
		}
		if plan.Validation.Validated {
			t.Error("expected validated=false")
		}
	}
}

func TestRulePlannerConversationalIntentEnglish(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	tests := []string{
		"hello",
		"hi there",
		"who are you",
		"what can you do",
		"what do you do",
		"help me",
	}
	for _, msg := range tests {
		plan, err := rp.Plan(context.Background(), PlannerInput{
			RunID:           "run_conv_en",
			ConversationID:  "conv_conv_en",
			UserMessage:     msg,
			AvailableAgents: testAgents(),
		})
		if err != nil {
			t.Fatalf("msg=%q unexpected error: %v", err, msg)
		}
		if plan.Strategy != "conversational" {
			t.Errorf("msg=%q expected strategy conversational, got %q", msg, plan.Strategy)
		}
	}
}

func TestRulePlannerConversationalHasNoFallback(t *testing.T) {
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_conv_nf",
		ConversationID:  "conv_conv_nf",
		UserMessage:     "你好，你是谁",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Fallback.Enabled {
		t.Error("conversational plan should not have fallback enabled")
	}
}

func TestRulePlannerMultiAgentWebCodeDocument(t *testing.T) {
	rp := NewRulePlanner([]string{"code-agent", "web-agent", "document-agent"})
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_multi3",
		ConversationID:  "conv_multi3",
		UserMessage:     "帮我做一个登录页面，并同时生成 Go 后端登录 API，还要写接口说明文档",
		AvailableAgents: []string{"code-agent", "web-agent", "document-agent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "ordered_parallel" {
		t.Errorf("expected strategy ordered_parallel, got %q", plan.Strategy)
	}
	if len(plan.Tasks) < 3 {
		t.Fatalf("expected at least 3 tasks, got %d", len(plan.Tasks))
	}
	hasWeb := false
	hasCode := false
	hasDoc := false
	for _, task := range plan.Tasks {
		switch task.AgentName {
		case "web-agent":
			hasWeb = true
		case "code-agent":
			hasCode = true
		case "document-agent":
			hasDoc = true
		}
	}
	if !hasWeb || !hasCode || !hasDoc {
		t.Errorf("expected web-agent, code-agent, and document-agent tasks, got %+v", plan.Tasks)
	}
	if !plan.Aggregation.Required {
		t.Error("expected aggregation.required=true for multi-agent plan")
	}
}

func TestRulePlannerMultiAgentCodeDocument(t *testing.T) {
	rp := NewRulePlanner([]string{"code-agent", "document-agent"})
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_code_doc",
		ConversationID:  "conv_code_doc",
		UserMessage:     "写一个 Go API 并生成对应的接口文档",
		AvailableAgents: []string{"code-agent", "document-agent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "ordered_parallel" {
		t.Errorf("expected strategy ordered_parallel, got %q", plan.Strategy)
	}
	if len(plan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(plan.Tasks))
	}
}

func TestRulePlannerExplicitAgentNameBypassesConversational(t *testing.T) {
	// When an explicit agentName is provided, conversational detection is skipped.
	// This allows direct agent selection (e.g., "Code Agent") to work even for
	// messages like "你好" that would otherwise be conversational.
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_bypass",
		ConversationID:  "conv_bypass",
		UserMessage:     "你好",
		AgentName:       "code-agent",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "single" {
		t.Errorf("expected single strategy when explicit agentName bypasses conversational, got %q", plan.Strategy)
	}
	if len(plan.Tasks) != 1 || plan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent task, got %+v", plan.Tasks)
	}
}

func TestRulePlannerConversationalDoesNotDispatch(t *testing.T) {
	// Conversational plans must have zero tasks — no child agent dispatch.
	rp := NewRulePlanner(testAgents())
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_no_dispatch",
		ConversationID:  "conv_no_dispatch",
		UserMessage:     "你能做什么",
		AvailableAgents: testAgents(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Tasks) != 0 {
		t.Errorf("conversational plan must have 0 tasks (no dispatch), got %d", len(plan.Tasks))
	}
	if plan.Strategy != "conversational" {
		t.Errorf("expected strategy conversational, got %q", plan.Strategy)
	}
}

func TestRulePlannerCodeAnalysisDependencyChain(t *testing.T) {
	// code + test + review + security must produce a sequential plan
	// with code-agent first and analysis agents depending on its output.
	agents := []string{"code-agent", "test-agent", "review-agent", "security-agent"}
	rp := NewRulePlanner(agents)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_dep",
		ConversationID:  "conv_dep",
		UserMessage:     "写一个牛顿法解方程的代码，测试并审查这段代码的安全性",
		AvailableAgents: agents,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "sequential" {
		t.Errorf("expected strategy sequential for code+analysis pattern, got %q", plan.Strategy)
	}
	if len(plan.Tasks) < 2 {
		t.Fatalf("expected at least 2 tasks, got %d", len(plan.Tasks))
	}

	// First task must be code-agent with no dependencies.
	if plan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected tasks[0] to be code-agent, got %q", plan.Tasks[0].AgentName)
	}
	if len(plan.Tasks[0].DependsOn) != 0 {
		t.Errorf("code-agent task must have no dependsOn, got %v", plan.Tasks[0].DependsOn)
	}

	// Analysis tasks must depend on code-agent.
	for i := 1; i < len(plan.Tasks); i++ {
		task := plan.Tasks[i]
		if task.AgentName == "code-agent" {
			t.Errorf("only one code-agent task expected")
		}
		if len(task.DependsOn) != 1 || task.DependsOn[0] != "task_code-agent" {
			t.Errorf("task %q should depend on task_code-agent, got dependsOn=%v", task.AgentName, task.DependsOn)
		}
		// Task content must include the dependency placeholder.
		if !strings.Contains(task.TaskContent, "{{deps.task_code-agent.output}}") {
			t.Errorf("task %q content must contain {{deps.task_code-agent.output}} placeholder", task.AgentName)
		}
	}

	if !plan.Aggregation.Required {
		t.Error("expected aggregation.required=true for sequential plan")
	}
}

func TestRulePlannerCodeTestOnly(t *testing.T) {
	agents := []string{"code-agent", "test-agent"}
	rp := NewRulePlanner(agents)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_ct",
		ConversationID:  "conv_ct",
		UserMessage:     "写代码并编写单元测试",
		AvailableAgents: agents,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "sequential" {
		t.Errorf("expected strategy sequential, got %q", plan.Strategy)
	}
	if len(plan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].AgentName != "code-agent" || len(plan.Tasks[0].DependsOn) != 0 {
		t.Error("first task must be code-agent with no deps")
	}
	if plan.Tasks[1].AgentName != "test-agent" || len(plan.Tasks[1].DependsOn) != 1 {
		t.Error("second task must be test-agent depending on code-agent")
	}
}

func TestRulePlannerCodeSecurityOnly(t *testing.T) {
	agents := []string{"code-agent", "security-agent"}
	rp := NewRulePlanner(agents)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_cs",
		ConversationID:  "conv_cs",
		UserMessage:     "写一段代码并检查安全漏洞",
		AvailableAgents: agents,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "sequential" {
		t.Errorf("expected strategy sequential, got %q", plan.Strategy)
	}
	if plan.Tasks[1].AgentName != "security-agent" {
		t.Errorf("expected security-agent, got %q", plan.Tasks[1].AgentName)
	}
	if len(plan.Tasks[1].DependsOn) != 1 || plan.Tasks[1].DependsOn[0] != "task_code-agent" {
		t.Errorf("security-agent must depend on task_code-agent")
	}
}

func TestRulePlannerCodeReviewNoSecurity(t *testing.T) {
	agents := []string{"code-agent", "review-agent"}
	rp := NewRulePlanner(agents)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_cr",
		ConversationID:  "conv_cr",
		UserMessage:     "生成一段代码并审查",
		AvailableAgents: agents,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "sequential" {
		t.Errorf("expected strategy sequential, got %q", plan.Strategy)
	}
	if plan.Tasks[1].AgentName != "review-agent" {
		t.Errorf("expected review-agent, got %q", plan.Tasks[1].AgentName)
	}
}

func TestRulePlannerWebCodeNoDependency(t *testing.T) {
	// web-agent + code-agent without analysis agents should still be ordered_parallel.
	agents := []string{"code-agent", "web-agent"}
	rp := NewRulePlanner(agents)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_wc",
		ConversationID:  "conv_wc",
		UserMessage:     "帮我做一个登录页面和 Go 登录接口",
		AvailableAgents: agents,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "ordered_parallel" {
		t.Errorf("web+code without analysis should be ordered_parallel, got %q", plan.Strategy)
	}
}

func TestRulePlannerCodeAnalysisWithDocument(t *testing.T) {
	// code + review + document: document should be in wave 0 alongside code.
	agents := []string{"code-agent", "review-agent", "document-agent"}
	rp := NewRulePlanner(agents)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:           "run_crd",
		ConversationID:  "conv_crd",
		UserMessage:     "帮我写代码、审查并生成文档",
		AvailableAgents: agents,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Strategy != "sequential" {
		t.Errorf("expected strategy sequential, got %q", plan.Strategy)
	}
	// document-agent should have no deps (wave 0 alongside code-agent).
	hasDocNoDep := false
	hasReviewDep := false
	for _, task := range plan.Tasks {
		if task.AgentName == "document-agent" && len(task.DependsOn) == 0 {
			hasDocNoDep = true
		}
		if task.AgentName == "review-agent" && len(task.DependsOn) == 1 {
			hasReviewDep = true
		}
	}
	if !hasDocNoDep {
		t.Error("document-agent should have no dependsOn (wave 0)")
	}
	if !hasReviewDep {
		t.Error("review-agent should depend on code-agent")
	}
}

func TestRulePlannerEmptyAgents(t *testing.T) {
	rp := NewRulePlanner(nil)
	plan, err := rp.Plan(context.Background(), PlannerInput{
		RunID:       "run_empty",
		UserMessage: "do something",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls through to "code-agent" even if not in registry.
	if plan.Strategy == "conversational" {
		// Empty agents with non-conversational message still falls through.
		t.Log("message triggered conversational despite empty agents")
	}
	if plan.Strategy != "conversational" {
		if plan.Tasks[0].AgentName != "code-agent" {
			t.Errorf("expected fallback to code-agent, got %q", plan.Tasks[0].AgentName)
		}
	}
}
