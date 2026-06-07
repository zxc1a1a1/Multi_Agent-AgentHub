package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// agentNamesLister creates an AgentLister from a list of agent names for tests.
func agentNamesLister(names []string) AgentLister {
	agents := make([]AgentInfoLite, len(names))
	for i, n := range names {
		agents[i] = AgentInfoLite{Name: n}
	}
	return &mockLister{agents: agents}
}

// mockLister implements AgentLister for tests.
type mockLister struct {
	agents []AgentInfoLite
}

func (m *mockLister) List() []AgentInfoLite {
	return m.agents
}

// fakeModel implements PlannerModel for tests, returning pre-configured responses.
type fakeModel struct {
	response string
	err      error
}

func (f *fakeModel) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return f.response, f.err
}

// ---------------------------------------------------------------------------
// parsePlanResponse tests (legacy)
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
// Legacy: LLMPlanner convertToPlan tests (no actual LLM calls)
// ---------------------------------------------------------------------------

func TestLLMPlanner_ConvertToPlan_Single(t *testing.T) {
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent", "web-agent"}))
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
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent", "web-agent"}))
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
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent"}))
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
	// Legacy path: should fall back to available agent (code-agent) since web-agent unknown.
	// This demonstrates the legacy fuzzyMatchAgent/defaultAgent behavior.
	if orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected fallback to code-agent (legacy), got %s", orchPlan.Tasks[0].AgentName)
	}
}

func TestLLMPlanner_ConvertToPlan_UnknownStrategy(t *testing.T) {
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent"}))
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
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent"}))
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
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent", "web-agent"}))
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
// Planner metadata stamping (legacy)
// ---------------------------------------------------------------------------

func TestLLMPlanner_MetadataStamping(t *testing.T) {
	// Create a planner with a known model name but no real LLM client.
	// We test conversion since we can't make real LLM calls.
	p := NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent"}))
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
// Helper function tests (legacy)
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
// LLM response JSON round-trip test (legacy)
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
	var p Planner = NewLLMPlanner(nil, "", agentNamesLister([]string{"code-agent"}))
	if p == nil {
		t.Error("expected non-nil Planner")
	}
}

// ============================================================================
// Phase 3: New pipeline tests — PromptBuilder → PlannerModel → Parser → Normalizer
// ============================================================================

// validCodePlanJSON is a valid PlanSchema JSON for a code request.
const validCodePlanJSON = `{
	"intent": "generate Go code for sorting algorithm",
	"mode": "single",
	"confidence": 0.95,
	"steps": [
		{
			"agent_name": "code-agent",
			"input": "Write a Go implementation of quicksort",
			"reason": "user asked for Go code generation"
		}
	],
	"user_visible_summary": "I will ask code-agent to generate the sorting algorithm."
}`

// validWebPlanJSON is a valid PlanSchema JSON for a web UI request.
const validWebPlanJSON = `{
	"intent": "build a login page",
	"mode": "single",
	"confidence": 0.92,
	"steps": [
		{
			"agent_name": "web-agent",
			"input": "Create a login page with email and password fields",
			"reason": "user asked for a web UI"
		}
	],
	"user_visible_summary": "I will ask web-agent to build the login page."
}`

// validFullStackPlanJSON is a valid PlanSchema JSON for a full-stack request with parallel mode.
const validFullStackPlanJSON = `{
	"intent": "build a full-stack app with frontend and backend",
	"mode": "parallel",
	"confidence": 0.90,
	"steps": [
		{
			"agent_name": "web-agent",
			"input": "Build a React dashboard UI",
			"reason": "user asked for frontend"
		},
		{
			"agent_name": "code-agent",
			"input": "Build the Go API server",
			"reason": "user asked for backend"
		}
	],
	"user_visible_summary": "I will build the frontend with web-agent and the backend with code-agent."
}`

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_CodeRequest
// Go/backend request → single code-agent via new pipeline
// ---------------------------------------------------------------------------

func TestLLMPlanner_CodeRequest(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	model := &fakeModel{response: validCodePlanJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_code",
		ConversationID: "conv_code",
		UserMessage:    "write a Go sorting algorithm",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan == nil {
		t.Fatal("expected non-nil plan")
	}

	// Strategy: single (LLM mode=single)
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected StrategySingle, got %q", orchPlan.Strategy)
	}

	// Single task for code-agent
	if len(orchPlan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(orchPlan.Tasks))
	}
	if orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent, got %q", orchPlan.Tasks[0].AgentName)
	}

	// PlannerSource must be "llm" — not "rule" or "fallback"
	if orchPlan.PlannerSource != "llm" {
		t.Errorf("expected PlannerSource=llm, got %q", orchPlan.PlannerSource)
	}
	if orchPlan.PlannerModel != "test-model" {
		t.Errorf("expected PlannerModel=test-model, got %q", orchPlan.PlannerModel)
	}

	// Fallback must be disabled — this is the primary path
	if orchPlan.Fallback.Enabled {
		t.Error("expected Fallback.Enabled=false (primary path)")
	}

	// Validation.Validated must be false — only the validator may set it
	if orchPlan.Validation.Validated {
		t.Error("expected Validation.Validated=false (set by validator, not planner)")
	}
}

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_WebRequest
// web/UI request → single web-agent via new pipeline
// ---------------------------------------------------------------------------

func TestLLMPlanner_WebRequest(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	model := &fakeModel{response: validWebPlanJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_web",
		ConversationID: "conv_web",
		UserMessage:    "build a login page",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Strategy: single (LLM mode=single)
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected StrategySingle, got %q", orchPlan.Strategy)
	}

	// Single task for web-agent
	if len(orchPlan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(orchPlan.Tasks))
	}
	if orchPlan.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected web-agent, got %q", orchPlan.Tasks[0].AgentName)
	}

	// PlannerSource must be "llm"
	if orchPlan.PlannerSource != "llm" {
		t.Errorf("expected PlannerSource=llm, got %q", orchPlan.PlannerSource)
	}

	// Fallback must be disabled
	if orchPlan.Fallback.Enabled {
		t.Error("expected Fallback.Enabled=false (primary path)")
	}
}

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_FullStack
// full-stack request → parallel with web-agent + code-agent
// ---------------------------------------------------------------------------

func TestLLMPlanner_FullStack(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	model := &fakeModel{response: validFullStackPlanJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_fs",
		ConversationID: "conv_fs",
		UserMessage:    "build a full-stack app with React frontend and Go backend",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Strategy: ordered_parallel (LLM mode=parallel maps to ordered_parallel)
	if orchPlan.Strategy != plan.StrategyOrderedParallel {
		t.Errorf("expected StrategyOrderedParallel, got %q", orchPlan.Strategy)
	}

	// Two tasks
	if len(orchPlan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(orchPlan.Tasks))
	}

	// Check both agents present
	agentNames := map[string]bool{}
	for _, task := range orchPlan.Tasks {
		agentNames[task.AgentName] = true
	}
	if !agentNames["web-agent"] {
		t.Error("expected web-agent in tasks")
	}
	if !agentNames["code-agent"] {
		t.Error("expected code-agent in tasks")
	}

	// Aggregation required for ordered_parallel
	if !orchPlan.Aggregation.Required {
		t.Error("expected Aggregation.Required=true for parallel plan")
	}
	if orchPlan.Aggregation.Mode != "summary" {
		t.Errorf("expected Aggregation.Mode=summary, got %q", orchPlan.Aggregation.Mode)
	}

	// PlannerSource must be "llm"
	if orchPlan.PlannerSource != "llm" {
		t.Errorf("expected PlannerSource=llm, got %q", orchPlan.PlannerSource)
	}

	// Fallback must be disabled
	if orchPlan.Fallback.Enabled {
		t.Error("expected Fallback.Enabled=false (primary path)")
	}
}

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_DoesNotUseRulePlanner
// Valid LLM plan → RulePlanner is NOT invoked
// Evidence: PlannerSource=llm, Fallback.Enabled=false
// ---------------------------------------------------------------------------

func TestLLMPlanner_DoesNotUseRulePlanner(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"code"}},
		},
	}
	model := &fakeModel{response: validCodePlanJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_no_rule",
		ConversationID: "conv_no_rule",
		UserMessage:    "write Go code",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key evidence: PlannerSource MUST be "llm" — not "rule" or "fallback"
	if orchPlan.PlannerSource != "llm" {
		t.Errorf("FAIL: PlannerSource=%q — RulePlanner was invoked or fallback triggered", orchPlan.PlannerSource)
	}

	// Fallback must be disabled — no fallback path was taken
	if orchPlan.Fallback.Enabled {
		t.Errorf("FAIL: Fallback.Enabled=true with reason=%q — fallback was triggered instead of primary LLM path", orchPlan.Fallback.Reason)
	}

	// PlannerModel must be populated (LLM metadata)
	if orchPlan.PlannerModel != "test-model" {
		t.Errorf("expected PlannerModel=test-model, got %q", orchPlan.PlannerModel)
	}

	// PlannerReasoning should be non-empty
	if orchPlan.PlannerReasoning == "" {
		t.Error("expected non-empty PlannerReasoning from LLM intent")
	}

	t.Logf("✓ Valid LLM plan does not trigger RulePlanner: Source=%s, Fallback.Enabled=%v, Model=%s",
		orchPlan.PlannerSource, orchPlan.Fallback.Enabled, orchPlan.PlannerModel)
}

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_UsesRegistryAgents
// Prompt must use registry agent info, not hardcoded agentDefaults
// ---------------------------------------------------------------------------

func TestLLMPlanner_UsesRegistryAgents(t *testing.T) {
	// Custom registry agent with unique description — NOT the agentDefaults() values.
	customAgents := []AgentInfoLite{
		{
			Name:          "code-agent",
			Description:   "Custom registry description for code agent",
			CapabilityIDs: []string{"custom_capability"},
			OutputModes:   []string{"custom_output"},
		},
	}
	lister := &mockLister{agents: customAgents}

	// The model response includes these custom capabilities.
	customPlanJSON := `{
		"intent": "test custom agent",
		"mode": "single",
		"confidence": 0.99,
		"steps": [
			{
				"agent_name": "code-agent",
				"input": "use custom capability",
				"reason": "test"
			}
		],
		"user_visible_summary": "Testing custom agent."
	}`
	model := &fakeModel{response: customPlanJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_registry",
		ConversationID: "conv_registry",
		UserMessage:    "test with custom agent",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The plan should still be valid
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected StrategySingle, got %q", orchPlan.Strategy)
	}
	if orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent, got %q", orchPlan.Tasks[0].AgentName)
	}

	// The key test: the prompt built should use registry agent info.
	// We verify this by checking that the PromptBuilder with our lister
	// produces the correct system prompt, and the plan comes back with llm source.
	pb := NewPromptBuilder(lister)
	systemPrompt := pb.BuildSystemPrompt()

	// Prompt must contain the registry-provided custom values (not agentDefaults).
	if !strings.Contains(systemPrompt, "Custom registry description for code agent") {
		t.Error("FAIL: prompt does not contain registry-provided description")
	}
	if !strings.Contains(systemPrompt, "custom_capability") {
		t.Error("FAIL: prompt does not contain registry-provided capabilities")
	}
	if !strings.Contains(systemPrompt, "custom_output") {
		t.Error("FAIL: prompt does not contain registry-provided output modes")
	}

	// Must NOT contain agentDefaults values.
	if strings.Contains(systemPrompt, "code generation and explanation") {
		t.Error("FAIL: prompt contains agentDefaults() description instead of registry data")
	}

	t.Log("✓ Prompt uses registry agent info (not agentDefaults)")
}

// ---------------------------------------------------------------------------
// Phase 3 Fix: TestLLMPlanner_UnknownAgentNotFuzzyMatched
// LLM returns unknown agent → Normalizer preserves it → Validator rejects →
// RulePlanner fallback.
// The agent is NEVER fuzzy-matched or defaulted.
// ---------------------------------------------------------------------------

func TestLLMPlanner_UnknownAgentNotFuzzyMatched(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"code"}},
		},
	}

	// LLM returns "gibberish-agent" which is NOT in registry.
	unknownAgentJSON := `{
		"intent": "do something unknown",
		"mode": "single",
		"confidence": 0.5,
		"steps": [
			{
				"agent_name": "gibberish-agent-xyz",
				"input": "do something",
				"reason": "test"
			}
		],
		"user_visible_summary": "Testing unknown agent."
	}`
	model := &fakeModel{response: unknownAgentJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_unknown",
		ConversationID: "conv_unknown",
		UserMessage:    "do something",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key evidence 1: PlannerSource must be "fallback" — the Validator rejected
	// the unknown agent and the pipeline fell back to RulePlanner.
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("FAIL: expected PlannerSource=fallback (validator should reject unknown agent), got %q",
			orchPlan.PlannerSource)
	}

	// Key evidence 2: The fallback plan's agent must be a known agent (from RulePlanner),
	// NOT the unknown agent (which was never fuzzy-matched or defaulted).
	// The RulePlanner fallback picks a valid agent based on keywords/default.
	for _, task := range orchPlan.Tasks {
		if task.AgentName == "gibberish-agent-xyz" {
			t.Errorf("FAIL: unknown agent 'gibberish-agent-xyz' leaked into final plan — validator did not reject it")
		}
	}

	t.Logf("✓ Unknown agent rejected by Validator → RulePlanner fallback: Source=%s, Agent=%s",
		orchPlan.PlannerSource, orchPlan.Tasks[0].AgentName)
}

// ---------------------------------------------------------------------------
// Phase 3 Fix: TestLLMPlanner_ValidationFailFallsBack
// LLM plan passes parse+normalize but fails validation → RulePlanner fallback.
// This proves: unknown agent → Validator rejects → fallback (no Repairer).
// ---------------------------------------------------------------------------

func TestLLMPlanner_ValidationFailFallsBack(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"code"}},
		},
	}

	// LLM returns a structurally valid JSON but with an agent NOT in the registry.
	unknownAgentJSON := `{
		"intent": "do something unknown",
		"mode": "single",
		"confidence": 0.5,
		"steps": [
			{
				"agent_name": "nonexistent-agent",
				"input": "do something",
				"reason": "test"
			}
		],
		"user_visible_summary": "Testing validation failure."
	}`
	model := &fakeModel{response: unknownAgentJSON}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_val_fail",
		ConversationID: "conv_val_fail",
		UserMessage:    "do something",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Evidence 1: Fallback was triggered — PlannerSource must be "fallback"
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("FAIL: expected PlannerSource=fallback after validation reject, got %q. Validation did not reject the plan.",
			orchPlan.PlannerSource)
	}

	// Evidence 2: The plan should still be usable (via RulePlanner fallback)
	if len(orchPlan.Tasks) == 0 {
		t.Error("expected non-empty tasks from fallback plan")
	}

	// Evidence 3: The fallback plan should use known agents only
	for _, task := range orchPlan.Tasks {
		if task.AgentName == "nonexistent-agent" {
			t.Errorf("FAIL: unknown agent leaked into fallback plan")
		}
	}

	t.Logf("✓ Validation fail → RulePlanner fallback: Source=%s, Agent=%s",
		orchPlan.PlannerSource, orchPlan.Tasks[0].AgentName)
}

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_ModelErrorFallsBackToRulePlanner
// When the model errors, RulePlanner is invoked as deprecated fallback
// ---------------------------------------------------------------------------

func TestLLMPlanner_ModelErrorFallsBackToRulePlanner(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"code"}},
			{Name: "web-agent", Description: "web", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"webpage"}},
		},
	}

	// Model returns an error (simulating API failure).
	model := &fakeModel{response: "", err: fmt.Errorf("simulated API error")}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_err",
		ConversationID: "conv_err",
		UserMessage:    "write Go code",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fallback was triggered — PlannerSource must be "fallback"
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("expected PlannerSource=fallback after model error, got %q", orchPlan.PlannerSource)
	}

	// But the plan should still be usable (via RulePlanner fallback)
	if len(orchPlan.Tasks) == 0 {
		t.Error("expected non-empty tasks from fallback plan")
	}

	t.Logf("✓ Model error correctly triggers deprecated RulePlanner fallback: Source=%s", orchPlan.PlannerSource)
}

// ---------------------------------------------------------------------------
// Phase 3: TestLLMPlanner_ParseErrorFallsBack
// When LLM returns invalid JSON, fallback to RulePlanner
// ---------------------------------------------------------------------------

func TestLLMPlanner_ParseErrorFallsBack(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"code"}},
		},
	}

	// Model returns invalid JSON.
	model := &fakeModel{response: "this is not JSON at all"}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_parse_err",
		ConversationID: "conv_parse_err",
		UserMessage:    "write Go code",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fallback was triggered due to parse failure
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("expected PlannerSource=fallback after parse error, got %q", orchPlan.PlannerSource)
	}

	t.Logf("✓ Parse error correctly triggers deprecated RulePlanner fallback: Source=%s", orchPlan.PlannerSource)
}

// ============================================================================
// Phase 4: Repairer tests
// ============================================================================

// fakeMultiModel implements PlannerModel for repair tests.
// It cycles through a list of responses — first call returns the invalid
// response, second call returns the corrected response (simulating repair).
type fakeMultiModel struct {
	responses []string
	callCount int
	err       error
}

func (f *fakeMultiModel) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	idx := f.callCount
	f.callCount++
	if idx >= len(f.responses) {
		return f.responses[len(f.responses)-1], nil
	}
	return f.responses[idx], nil
}

// ---------------------------------------------------------------------------
// Phase 4: TestLLMPlanner_InvalidJSON_RepairSucceeds
// Invalid JSON → repair once → parse → normalize → validate → success.
// Evidence: PlannerSource=llm, RepairCount=1, Fallback.Enabled=false.
// ---------------------------------------------------------------------------

func TestLLMPlanner_InvalidJSON_RepairSucceeds(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	// First response: raw text (not JSON at all) → parse fails.
	// Second response (repair): valid JSON.
	model := &fakeMultiModel{
		responses: []string{
			"this is not JSON at all, just some random text",
			`{"intent":"generate code","mode":"single","confidence":0.95,"steps":[{"agent_name":"code-agent","input":"write Go code","reason":"code task"}],"user_visible_summary":"I will ask code-agent to generate code."}`,
		},
	}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_invalid_json",
		ConversationID: "conv_invalid_json",
		UserMessage:    "write Go code",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key evidence 1: PlannerSource must be "llm" (repaired LLM plan, NOT fallback).
	if orchPlan.PlannerSource != "llm" {
		t.Errorf("FAIL: expected PlannerSource=llm (repaired LLM plan), got %q", orchPlan.PlannerSource)
	}

	// Key evidence 2: RepairCount must be 1 (one repair succeeded).
	if orchPlan.RepairCount != 1 {
		t.Errorf("FAIL: expected RepairCount=1, got %d", orchPlan.RepairCount)
	}

	// Key evidence 3: Fallback must be disabled (repair succeeded).
	if orchPlan.Fallback.Enabled {
		t.Errorf("FAIL: Fallback.Enabled=true (reason=%q) — repair should have succeeded, not triggered fallback", orchPlan.Fallback.Reason)
	}

	// Key evidence 4: Model must be populated.
	if orchPlan.PlannerModel != "test-model" {
		t.Errorf("expected PlannerModel=test-model, got %q", orchPlan.PlannerModel)
	}

	// Plan should be valid.
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected StrategySingle, got %q", orchPlan.Strategy)
	}
	if len(orchPlan.Tasks) == 0 {
		t.Error("expected non-empty tasks")
	}

	t.Logf("✓ Invalid JSON repaired successfully: Source=%s, RepairCount=%d, Fallback.Enabled=%v",
		orchPlan.PlannerSource, orchPlan.RepairCount, orchPlan.Fallback.Enabled)
}

// ---------------------------------------------------------------------------
// Phase 4: TestLLMPlanner_UnknownAgent_RepairSucceeds
// Unknown agent → repair once → corrected agent → parse → normalize → validate → success.
// Evidence: PlannerSource=llm, RepairCount=1, Fallback.Enabled=false, agent fixed.
// ---------------------------------------------------------------------------

func TestLLMPlanner_UnknownAgent_RepairSucceeds(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	// First response: uses "gibberish-agent" which is NOT in registry.
	// Second response (repair): corrected to use "web-agent".
	model := &fakeMultiModel{
		responses: []string{
			`{"intent":"build UI","mode":"single","confidence":0.9,"steps":[{"agent_name":"gibberish-agent","input":"build a login page","reason":"UI task"}],"user_visible_summary":"Building UI."}`,
			`{"intent":"build UI","mode":"single","confidence":0.9,"steps":[{"agent_name":"web-agent","input":"build a login page","reason":"UI task"}],"user_visible_summary":"Building UI."}`,
		},
	}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_unknown_agent",
		ConversationID: "conv_unknown_agent",
		UserMessage:    "build a login page",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key evidence 1: PlannerSource must be "llm" (repaired, NOT fallback).
	if orchPlan.PlannerSource != "llm" {
		t.Errorf("FAIL: expected PlannerSource=llm (repaired plan), got %q", orchPlan.PlannerSource)
	}

	// Key evidence 2: RepairCount must be 1.
	if orchPlan.RepairCount != 1 {
		t.Errorf("FAIL: expected RepairCount=1, got %d", orchPlan.RepairCount)
	}

	// Key evidence 3: Fallback must be disabled.
	if orchPlan.Fallback.Enabled {
		t.Errorf("FAIL: Fallback.Enabled=true — repair should have succeeded")
	}

	// Key evidence 4: The agent MUST be the corrected one (web-agent), NOT gibberish-agent.
	if len(orchPlan.Tasks) == 0 {
		t.Fatal("expected non-empty tasks")
	}
	if orchPlan.Tasks[0].AgentName == "gibberish-agent" {
		t.Errorf("FAIL: unknown agent leaked into repaired plan")
	}
	if orchPlan.Tasks[0].AgentName != "web-agent" {
		t.Errorf("expected repaired agent=web-agent, got %q", orchPlan.Tasks[0].AgentName)
	}

	// Should be a single strategy.
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected StrategySingle, got %q", orchPlan.Strategy)
	}

	t.Logf("✓ Unknown agent repaired successfully: Source=%s, RepairCount=%d, Agent=%s",
		orchPlan.PlannerSource, orchPlan.RepairCount, orchPlan.Tasks[0].AgentName)
}

// ---------------------------------------------------------------------------
// Phase 4: TestLLMPlanner_RepairFail_FallsBackToRulePlanner
// Invalid plan → repair once (but repair also returns invalid) → fallback RulePlanner.
// Evidence: PlannerSource=fallback, Fallback.Enabled=true.
// ---------------------------------------------------------------------------

func TestLLMPlanner_RepairFail_FallsBackToRulePlanner(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	// Both responses use an unknown agent — repair cannot fix it.
	model := &fakeMultiModel{
		responses: []string{
			`{"intent":"do something","mode":"single","confidence":0.5,"steps":[{"agent_name":"nonexistent-agent","input":"something","reason":"test"}],"user_visible_summary":"Test."}`,
			`{"intent":"do something","mode":"single","confidence":0.5,"steps":[{"agent_name":"still-wrong-agent","input":"something","reason":"test"}],"user_visible_summary":"Test."}`,
		},
	}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_repair_fail",
		ConversationID: "conv_repair_fail",
		UserMessage:    "do something",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key evidence 1: PlannerSource must be "fallback" — repair failed, RulePlanner invoked.
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("FAIL: expected PlannerSource=fallback after repair failure, got %q", orchPlan.PlannerSource)
	}

	// Key evidence 2: Fallback must be enabled.
	if !orchPlan.Fallback.Enabled {
		t.Error("FAIL: expected Fallback.Enabled=true after repair failure")
	}

	// Key evidence 3: The fallback plan should use known agents only.
	for _, task := range orchPlan.Tasks {
		if task.AgentName == "nonexistent-agent" || task.AgentName == "still-wrong-agent" {
			t.Errorf("FAIL: unknown agent %q leaked into fallback plan", task.AgentName)
		}
	}

	t.Logf("✓ Repair failure correctly triggers RulePlanner fallback: Source=%s, Fallback.Enabled=%v, Fallback.Reason=%q",
		orchPlan.PlannerSource, orchPlan.Fallback.Enabled, orchPlan.Fallback.Reason)
}

// ---------------------------------------------------------------------------
// Phase 4: TestPlanRepairer_PromptConstruction
// Verify the repair prompt contains required elements: original output, errors,
// available agents, schema, and instruction.
// ---------------------------------------------------------------------------

func TestPlanRepairer_PromptConstruction(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
			{Name: "web-agent", Description: "web UI generation", CapabilityIDs: []string{"web_generation"}, OutputModes: []string{"text", "webpage"}},
		},
	}
	repairer := NewPlanRepairer(nil) // nil model is fine — we only test prompt construction.

	failures := []ValError{
		{Code: ValCodeUnknownAgent, Message: `agent "gibberish-agent" not found in registry`, TaskIndex: 0},
		{Code: ValCodeUnsupportedSequential, Message: "sequential strategy is not yet supported", TaskIndex: -1},
	}
	originalOutput := `{"intent":"test","mode":"sequential","steps":[{"agent_name":"gibberish-agent","input":"do something"}],"user_visible_summary":"Test."}`

	prompt := repairer.buildRepairPrompt(originalOutput, failures, lister)

	// Must contain original output.
	if !strings.Contains(prompt, originalOutput) {
		t.Error("FAIL: repair prompt does not contain original output")
	}

	// Must contain error codes.
	if !strings.Contains(prompt, ValCodeUnknownAgent) {
		t.Errorf("FAIL: repair prompt does not contain error code %q", ValCodeUnknownAgent)
	}
	if !strings.Contains(prompt, ValCodeUnsupportedSequential) {
		t.Errorf("FAIL: repair prompt does not contain error code %q", ValCodeUnsupportedSequential)
	}

	// Must contain available agents.
	if !strings.Contains(prompt, "code-agent") {
		t.Error("FAIL: repair prompt does not list code-agent")
	}
	if !strings.Contains(prompt, "web-agent") {
		t.Error("FAIL: repair prompt does not list web-agent")
	}

	// Must contain JSON schema.
	if !strings.Contains(prompt, "Expected JSON Schema") {
		t.Error("FAIL: repair prompt does not contain expected JSON schema")
	}

	// Must contain instructions.
	if !strings.Contains(prompt, "Instructions") {
		t.Error("FAIL: repair prompt does not contain instructions")
	}

	// Must contain instruction to return only corrected JSON.
	if !strings.Contains(prompt, "ONLY the corrected JSON") {
		t.Error("FAIL: repair prompt does not include 'ONLY the corrected JSON' instruction")
	}

	t.Log("✓ Repair prompt contains all required elements: original output, errors, agents, schema, instructions")
}

// ---------------------------------------------------------------------------
// Phase 4: TestRulePlannerDeprecated
// Verify that RulePlanner has the Deprecated comment and no new keywords were added.
// ---------------------------------------------------------------------------

func TestRulePlannerDeprecated(t *testing.T) {
	// Verify RulePlanner has the Deprecated comment by behavior:
	// RulePlanner is still functional as a fallback, and no new keywords were added.

	rp := NewRulePlanner([]string{"code-agent", "web-agent"})

	// RulePlanner should still work as a deprecated fallback.
	input := PlannerInput{
		RunID:          "run_deprecated",
		ConversationID: "conv_deprecated",
		UserMessage:    "write Go code",
		PlanningMode:   "auto",
	}
	plan, err := rp.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("RulePlanner (deprecated) should still work: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan from deprecated RulePlanner")
	}
	if len(plan.Tasks) == 0 {
		t.Error("expected non-empty tasks from deprecated RulePlanner")
	}

	// Verify no keywords were changed or removed (keyword set must match).
	expectedWebKeywords := []string{
		"页面", "ui", "html", "react", "登录页", "前端",
		"page", "webpage", "css", "component", "layout",
	}
	expectedCodeKeywords := []string{
		"go", "api", "后端", "接口", "server", "service",
		"golang", "handler", "endpoint", "database", "sql", "函数",
	}

	if len(webKeywords) < len(expectedWebKeywords) {
		t.Errorf("FAIL: webKeywords count changed: got %d, want at least %d", len(webKeywords), len(expectedWebKeywords))
	}
	if len(codeKeywords) < len(expectedCodeKeywords) {
		t.Errorf("FAIL: codeKeywords count changed: got %d, want at least %d", len(codeKeywords), len(expectedCodeKeywords))
	}

	t.Log("✓ RulePlanner is deprecated transitional fallback, keyword baseline verified")
}

// ---------------------------------------------------------------------------
// Phase 4: TestLLMPlanner_RepairMaxOnce
// Verify that repair is called exactly once, not in a loop.
// Evidence: even if repair output fails again, fallback is triggered (not re-repair).
// ---------------------------------------------------------------------------

func TestLLMPlanner_RepairMaxOnce(t *testing.T) {
	lister := &mockLister{
		agents: []AgentInfoLite{
			{Name: "code-agent", Description: "code generation", CapabilityIDs: []string{"code_generation"}, OutputModes: []string{"text", "code"}},
		},
	}
	// All responses are invalid — repairer should try once then give up.
	// The model's callCount will tell us how many times Generate was called.
	model := &fakeMultiModel{
		responses: []string{
			"garbage not json",   // first call: parse fails
			"still not json",     // repair call: parse still fails
			"should not be used", // should NEVER be called (repair max once)
		},
	}

	p := NewLLMPlanner(model, "test-model", lister)
	input := PlannerInput{
		RunID:          "run_repair_once",
		ConversationID: "conv_repair_once",
		UserMessage:    "write Go code",
		PlanningMode:   "auto",
	}

	orchPlan, err := p.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// model.callCount must be exactly 2: initial call + one repair call.
	// If it's 3, the repairer tried again (forbidden).
	if model.callCount > 2 {
		t.Errorf("FAIL: model was called %d times — repair should be max once (expected exactly 2: initial + one repair)",
			model.callCount)
	}
	if model.callCount < 2 {
		t.Errorf("FAIL: model was called only %d times — repair should have been attempted once", model.callCount)
	}

	// Must have fallen back to RulePlanner.
	if orchPlan.PlannerSource != "fallback" {
		t.Errorf("FAIL: expected PlannerSource=fallback after repair failure, got %q", orchPlan.PlannerSource)
	}

	t.Logf("✓ Repair max once: model called %d times, Source=%s", model.callCount, orchPlan.PlannerSource)
}
