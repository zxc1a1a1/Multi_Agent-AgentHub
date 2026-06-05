package planner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// ---------------------------------------------------------------------------
// parsePlanResponse tests
// ---------------------------------------------------------------------------

func TestParsePlanResponse_ValidSingleJSON(t *testing.T) {
	raw := `{"intent":"sort algorithm","reasoning":"code task","strategy":"single","tasks":[{"agentName":"code-agent","capabilityIds":["code_generation"],"taskContent":"write a sort","expectedOutputs":["code"],"priority":1}]}`
	resp, err := parsePlanResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Strategy != "single" {
		t.Errorf("expected strategy=single, got %s", resp.Strategy)
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
	}
	if resp.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent, got %s", resp.Tasks[0].AgentName)
	}
}

func TestParsePlanResponse_ValidOrderedParallel(t *testing.T) {
	raw := `{"intent":"full stack app","reasoning":"needs both frontend and backend","strategy":"ordered_parallel","tasks":[{"agentName":"web-agent","capabilityIds":["web_generation"],"taskContent":"build UI","expectedOutputs":["webpage"],"priority":1},{"agentName":"code-agent","capabilityIds":["code_generation"],"taskContent":"build API","expectedOutputs":["code"],"priority":2}]}`
	resp, err := parsePlanResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Strategy != "ordered_parallel" {
		t.Errorf("expected ordered_parallel, got %s", resp.Strategy)
	}
	if len(resp.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.Tasks))
	}
	if resp.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected web-agent first, got %s", resp.Tasks[0].AgentName)
	}
	if resp.Tasks[1].AgentName != "code-agent" {
		t.Errorf("expected code-agent second, got %s", resp.Tasks[1].AgentName)
	}
}

func TestParsePlanResponse_InvalidJSON(t *testing.T) {
	_, err := parsePlanResponse(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParsePlanResponse_EmptyString(t *testing.T) {
	_, err := parsePlanResponse("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParsePlanResponse_MissingStrategy(t *testing.T) {
	_, err := parsePlanResponse(`{"intent":"test","tasks":[{"agentName":"code-agent","capabilityIds":["code_generation"],"taskContent":"do it","expectedOutputs":["code"]}]}`)
	if err == nil {
		t.Error("expected error for missing strategy")
	}
}

func TestParsePlanResponse_NoTasks(t *testing.T) {
	_, err := parsePlanResponse(`{"intent":"test","strategy":"single","tasks":[]}`)
	if err == nil {
		t.Error("expected error for empty tasks")
	}
}

// ---------------------------------------------------------------------------
// stripMarkdownFences tests
// ---------------------------------------------------------------------------

func TestStripMarkdownFences_JSON_Fenced(t *testing.T) {
	raw := "```json\n{\"hello\":\"world\"}\n```"
	got := stripMarkdownFences(raw)
	if got != `{"hello":"world"}` {
		t.Errorf("expected clean JSON, got %q", got)
	}
}

func TestStripMarkdownFences_Plain_Fenced(t *testing.T) {
	raw := "```\nsome text\n```"
	got := stripMarkdownFences(raw)
	if got != "some text" {
		t.Errorf("expected 'some text', got %q", got)
	}
}

func TestStripMarkdownFences_NoFence(t *testing.T) {
	raw := `{"plain":"json"}`
	got := stripMarkdownFences(raw)
	if got != raw {
		t.Errorf("expected unchanged, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// LLMPlanner convertToPlan tests (no actual LLM calls)
// ---------------------------------------------------------------------------

func TestLLMPlanner_ConvertToPlan_Single(t *testing.T) {
	p := NewLLMPlanner(nil, []string{"code-agent", "web-agent"})
	resp := &llmPlanResponse{
		Intent:    "generate code",
		Reasoning: "user wants Go code",
		Strategy:  "single",
		Tasks: []llmTaskPlan{
			{
				AgentName:       "code-agent",
				CapabilityIDs:   []string{"code_generation"},
				TaskContent:     "write Go code",
				ExpectedOutputs: []string{"code"},
				Priority:        1,
			},
		},
	}
	input := PlannerInput{
		RunID:          "run_1",
		ConversationID: "conv_1",
		UserMessage:    "write Go code",
	}

	orchPlan, err := p.convertToPlan(resp, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected single, got %s", orchPlan.Strategy)
	}
	if len(orchPlan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(orchPlan.Tasks))
	}
	if orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent, got %s", orchPlan.Tasks[0].AgentName)
	}
	if orchPlan.Aggregation.Mode != "none" {
		t.Errorf("expected none aggregation, got %s", orchPlan.Aggregation.Mode)
	}
}

func TestLLMPlanner_ConvertToPlan_OrderedParallel(t *testing.T) {
	p := NewLLMPlanner(nil, []string{"code-agent", "web-agent"})
	resp := &llmPlanResponse{
		Intent:    "full stack app",
		Reasoning: "both frontend and backend needed",
		Strategy:  "ordered_parallel",
		Tasks: []llmTaskPlan{
			{AgentName: "web-agent", CapabilityIDs: []string{"web_generation"}, TaskContent: "build UI", ExpectedOutputs: []string{"webpage"}, Priority: 1},
			{AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, TaskContent: "build API", ExpectedOutputs: []string{"code"}, Priority: 2},
		},
	}
	input := PlannerInput{RunID: "run_1", ConversationID: "conv_1", UserMessage: "build app"}

	orchPlan, err := p.convertToPlan(resp, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Strategy != plan.StrategyOrderedParallel {
		t.Errorf("expected ordered_parallel, got %s", orchPlan.Strategy)
	}
	if len(orchPlan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(orchPlan.Tasks))
	}
	if orchPlan.Aggregation.Mode != "summary" {
		t.Errorf("expected summary aggregation, got %s", orchPlan.Aggregation.Mode)
	}
	if !orchPlan.Aggregation.Required {
		t.Error("expected aggregation required=true")
	}
}

func TestLLMPlanner_ConvertToPlan_UnknownAgent(t *testing.T) {
	p := NewLLMPlanner(nil, []string{"code-agent"})
	resp := &llmPlanResponse{
		Intent:   "UI task",
		Strategy: "single",
		Tasks: []llmTaskPlan{
			{AgentName: "web-agent", CapabilityIDs: []string{"web_generation"}, TaskContent: "make UI", ExpectedOutputs: []string{"webpage"}, Priority: 1},
		},
	}
	input := PlannerInput{RunID: "run_1", ConversationID: "conv_1", UserMessage: "make UI"}

	orchPlan, err := p.convertToPlan(resp, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to available agent (code-agent) since web-agent unknown.
	if orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected fallback to code-agent, got %s", orchPlan.Tasks[0].AgentName)
	}
}

func TestLLMPlanner_ConvertToPlan_UnknownStrategy(t *testing.T) {
	p := NewLLMPlanner(nil, []string{"code-agent"})
	resp := &llmPlanResponse{
		Intent:   "test",
		Strategy: "invalid_strategy",
		Tasks: []llmTaskPlan{
			{AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, TaskContent: "do it", ExpectedOutputs: []string{"code"}, Priority: 1},
		},
	}
	input := PlannerInput{RunID: "run_1", ConversationID: "conv_1", UserMessage: "test"}

	_, err := p.convertToPlan(resp, input)
	if err == nil {
		t.Error("expected error for unknown strategy")
	}
}

func TestLLMPlanner_ConvertToPlan_EmptyAgentName(t *testing.T) {
	p := NewLLMPlanner(nil, []string{"code-agent"})
	resp := &llmPlanResponse{
		Intent:   "test",
		Strategy: "single",
		Tasks: []llmTaskPlan{
			{AgentName: "", CapabilityIDs: []string{"code_generation"}, TaskContent: "do it", ExpectedOutputs: []string{"code"}, Priority: 1},
		},
	}
	input := PlannerInput{RunID: "run_1", ConversationID: "conv_1", UserMessage: "test"}

	_, err := p.convertToPlan(resp, input)
	if err == nil {
		t.Error("expected error for empty agent name")
	}
}

// ---------------------------------------------------------------------------
// Fallback tests
// ---------------------------------------------------------------------------

func TestLLMPlanner_FallbackPlan(t *testing.T) {
	p := NewLLMPlanner(nil, []string{"code-agent", "web-agent"})
	input := PlannerInput{
		RunID:          "run_1",
		ConversationID: "conv_1",
		UserMessage:    "write Go code",
	}

	orchPlan := p.fallbackPlan(input, "test_error", "test-model")
	if orchPlan == nil {
		t.Fatal("expected plan, got nil")
	}
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("expected PlannerSource=fallback, got %s", orchPlan.PlannerSource)
	}
	if orchPlan.PlannerReasoning == "" {
		t.Error("expected PlannerReasoning to be non-empty")
	}
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected single strategy in fallback, got %s", orchPlan.Strategy)
	}
	if len(orchPlan.Tasks) == 0 {
		t.Error("expected at least one task in fallback")
	}
}

// ---------------------------------------------------------------------------
// Planner metadata stamping
// ---------------------------------------------------------------------------

func TestLLMPlanner_MetadataStamping(t *testing.T) {
	// Create a planner with a known model name but no real LLM client.
	// We test conversion since we can't make real LLM calls.
	p := NewLLMPlanner(nil, []string{"code-agent"})
	resp := &llmPlanResponse{
		Intent:    "test",
		Reasoning: "because it's a test",
		Strategy:  "single",
		Tasks: []llmTaskPlan{
			{AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, TaskContent: "test", ExpectedOutputs: []string{"code"}, Priority: 1},
		},
	}
	input := PlannerInput{RunID: "run_1", ConversationID: "conv_1", UserMessage: "test"}

	// Plan() would fail without a real LLM, so test the conversion path.
	plan1, err := p.convertToPlan(resp, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Planner metadata is NOT set in convertToPlan — it's set in Plan().
	if plan1.PlannerSource != "" {
		t.Error("PlannerSource should be empty in convertToPlan (set by Plan)")
	}

	// Test fallback metadata stamping.
	plan2 := p.fallbackPlan(input, "llm_error", "claude-haiku")
	if plan2.PlannerSource != "fallback" {
		t.Errorf("expected fallback source, got %s", plan2.PlannerSource)
	}
	if plan2.PlannerModel != "claude-haiku" {
		t.Errorf("expected model 'claude-haiku', got %s", plan2.PlannerModel)
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestFuzzyMatchAgent(t *testing.T) {
	tests := []struct {
		name      string
		available []string
		want      string
	}{
		{"codeagent", []string{"code-agent", "web-agent"}, "code-agent"},
		{"code_agent", []string{"code-agent", "web-agent"}, ""},
		{"web", []string{"code-agent", "web-agent"}, "web-agent"},
		{"vision", []string{"code-agent", "web-agent"}, ""},
		{"code-agent", []string{"code-agent"}, "code-agent"},
	}
	for _, tt := range tests {
		got := fuzzyMatchAgent(tt.name, tt.available)
		if got != tt.want {
			t.Errorf("fuzzyMatchAgent(%q, %v): got %q, want %q", tt.name, tt.available, got, tt.want)
		}
	}
}

func TestTaskPlan_TaskID(t *testing.T) {
	tests := []struct {
		agentName string
		want      string
	}{
		{"code-agent", "task_code-agent"},
		{"web-agent", "task_web-agent"},
		{"", "task_001"},
	}
	for _, tt := range tests {
		tp := llmTaskPlan{AgentName: tt.agentName}
		if got := tp.TaskID(); got != tt.want {
			t.Errorf("TaskID() for %q: got %q, want %q", tt.agentName, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// LLM response JSON round-trip test
// ---------------------------------------------------------------------------

func TestLLMPlanResponse_JSONRoundTrip(t *testing.T) {
	// Ensure the LLM response struct marshals and unmarshals correctly.
	original := llmPlanResponse{
		Intent:    "build a full-stack app",
		Reasoning: "User requested both a login page and a Go API. web-agent handles the frontend, code-agent handles the backend.",
		Strategy:  "ordered_parallel",
		Tasks: []llmTaskPlan{
			{
				AgentName:       "web-agent",
				CapabilityIDs:   []string{"web_generation"},
				TaskContent:     "Create a login page with email and password fields",
				ExpectedOutputs: []string{"webpage"},
				Priority:        1,
			},
			{
				AgentName:       "code-agent",
				CapabilityIDs:   []string{"code_generation"},
				TaskContent:     "Implement a Go HTTP API with login endpoint",
				ExpectedOutputs: []string{"code"},
				Priority:        2,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	parsed, err := parsePlanResponse(string(data))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.Intent != original.Intent {
		t.Errorf("intent mismatch: got %q, want %q", parsed.Intent, original.Intent)
	}
	if parsed.Strategy != original.Strategy {
		t.Errorf("strategy mismatch: got %q, want %q", parsed.Strategy, original.Strategy)
	}
	if len(parsed.Tasks) != len(original.Tasks) {
		t.Fatalf("task count mismatch: got %d, want %d", len(parsed.Tasks), len(original.Tasks))
	}
}

// ---------------------------------------------------------------------------
// Interface compliance
// ---------------------------------------------------------------------------

func TestLLMPlanner_ImplementsPlanner(t *testing.T) {
	// Compile-time check via var _ Planner = (*LLMPlanner)(nil) at package level.
	// This test just confirms we can assign.
	var p Planner = NewLLMPlanner(nil, []string{"code-agent"})
	if p == nil {
		t.Error("expected non-nil Planner")
	}
}
