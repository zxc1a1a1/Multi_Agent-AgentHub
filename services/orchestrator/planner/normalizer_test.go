package planner

import (
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// ---------------------------------------------------------------------------
// Mode mapping tests
// ---------------------------------------------------------------------------

func TestNormalize_ModeSingle(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "code generation",
		Mode:   "single",
		Steps: []PlanStep{
			{AgentName: "code-agent", Input: "write a function"},
		},
	}
	orchPlan, err := n.Normalize(schema, "run_1", "conv_1", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Strategy != plan.StrategySingle {
		t.Errorf("expected StrategySingle, got %q", orchPlan.Strategy)
	}
	if len(orchPlan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(orchPlan.Tasks))
	}
	if !orchPlan.Validation.Validated {
		// Good — validated must be false coming out of the normalizer.
	}
}

func TestNormalize_ModeParallel_MapsToOrderedParallel(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "full stack app",
		Mode:   "parallel",
		Steps: []PlanStep{
			{AgentName: "web-agent", Input: "build UI"},
			{AgentName: "code-agent", Input: "build API"},
		},
	}
	orchPlan, err := n.Normalize(schema, "run_2", "conv_2", "rule", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Strategy != plan.StrategyOrderedParallel {
		t.Errorf("expected StrategyOrderedParallel (compat map from parallel), got %q", orchPlan.Strategy)
	}
	if !orchPlan.Aggregation.Required {
		t.Error("expected aggregation.required=true for ordered_parallel")
	}
}

func TestNormalize_ModeSequential_MapsToSequentialForValidatorRejection(t *testing.T) {
	// sequential is mapped to StrategySequential explicitly so the validator
	// can produce a clear, deterministic rejection.
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "multi-step pipeline",
		Mode:   "sequential",
		Steps: []PlanStep{
			{ID: "step-1", AgentName: "code-agent", Input: "step 1"},
			{ID: "step-2", AgentName: "code-agent", Input: "step 2", DependsOn: []string{"step-1"}},
		},
	}
	orchPlan, err := n.Normalize(schema, "run_seq", "conv_seq", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Strategy != plan.StrategySequential {
		t.Errorf("expected StrategySequential, got %q", orchPlan.Strategy)
	}
	// The validator will reject this StrategySequential.
}

// ---------------------------------------------------------------------------
// Step normalization tests
// ---------------------------------------------------------------------------

func TestNormalize_GeneratesStepIDs(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "parallel",
		Steps: []PlanStep{
			{AgentName: "web-agent", Input: "do UI"},
			{AgentName: "code-agent", Input: "do backend"},
			{AgentName: "code-agent", Input: "do tests"},
		},
	}
	orchPlan, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Tasks[0].TaskID != "step-1" {
		t.Errorf("expected step-1, got %q", orchPlan.Tasks[0].TaskID)
	}
	if orchPlan.Tasks[1].TaskID != "step-2" {
		t.Errorf("expected step-2, got %q", orchPlan.Tasks[1].TaskID)
	}
	if orchPlan.Tasks[2].TaskID != "step-3" {
		t.Errorf("expected step-3, got %q", orchPlan.Tasks[2].TaskID)
	}
}

func TestNormalize_PreservesExplicitStepIDs(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "parallel",
		Steps: []PlanStep{
			{ID: "custom-id", AgentName: "code-agent", Input: "do it"},
		},
	}
	orchPlan, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Tasks[0].TaskID != "custom-id" {
		t.Errorf("expected 'custom-id', got %q", orchPlan.Tasks[0].TaskID)
	}
}

func TestNormalize_NilDependsOnBecomesEmptySlice(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "single",
		Steps: []PlanStep{
			{AgentName: "code-agent", Input: "do it", DependsOn: nil},
		},
	}
	orchPlan, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Tasks[0].DependsOn == nil {
		t.Error("expected non-nil DependsOn (normalized to empty slice)")
	}
	if len(orchPlan.Tasks[0].DependsOn) != 0 {
		t.Errorf("expected empty DependsOn, got %v", orchPlan.Tasks[0].DependsOn)
	}
}

func TestNormalize_TrimsFields(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "  test intent  ",
		Mode:   "single",
		Steps: []PlanStep{
			{AgentName: "  code-agent  ", Input: "  do it  "},
		},
	}
	orchPlan, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected trimmed 'code-agent', got %q", orchPlan.Tasks[0].AgentName)
	}
	if orchPlan.Tasks[0].TaskContent != "do it" {
		t.Errorf("expected trimmed 'do it', got %q", orchPlan.Tasks[0].TaskContent)
	}
}

// ---------------------------------------------------------------------------
// Unknown agent is NOT guessed
// ---------------------------------------------------------------------------

func TestNormalize_UnknownAgentPreserved(t *testing.T) {
	// The normalizer MUST NOT fuzzy-match, default, or fallback the agent name.
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "single",
		Steps: []PlanStep{
			{AgentName: "gibberish-agent-xyz", Input: "do something"},
		},
	}
	orchPlan, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Agent name must be preserved exactly (lower-cased but NOT replaced).
	if orchPlan.Tasks[0].AgentName != "gibberish-agent-xyz" {
		t.Errorf("expected 'gibberish-agent-xyz' preserved as-is, got %q", orchPlan.Tasks[0].AgentName)
	}
}

func TestNormalize_NoFuzzyMatch(t *testing.T) {
	// "codeagent" (no hyphen) must NOT be corrected to "code-agent".
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "single",
		Steps: []PlanStep{
			{AgentName: "codeagent", Input: "do it"},
		},
	}
	orchPlan, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Tasks[0].AgentName != "codeagent" {
		t.Errorf("expected 'codeagent' preserved (no fuzzy match), got %q", orchPlan.Tasks[0].AgentName)
	}
}

// ---------------------------------------------------------------------------
// Confidence validation
// ---------------------------------------------------------------------------

func TestNormalize_ConfidenceInRange(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent:     "test",
		Mode:       "single",
		Confidence: 0.86,
		Steps:      []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	_, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error for in-range confidence: %v", err)
	}
}

func TestNormalize_ConfidenceNegative(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent:     "test",
		Mode:       "single",
		Confidence: -0.5,
		Steps:      []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	_, err := n.Normalize(schema, "r", "c", "auto", "")
	if err == nil {
		t.Error("expected error for negative confidence")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected 'out of range', got %v", err)
	}
}

func TestNormalize_ConfidenceTooHigh(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent:     "test",
		Mode:       "single",
		Confidence: 2.5,
		Steps:      []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	_, err := n.Normalize(schema, "r", "c", "auto", "")
	if err == nil {
		t.Error("expected error for confidence > 1")
	}
}

func TestNormalize_ConfidenceZero(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent:     "test",
		Mode:       "single",
		Confidence: 0,
		Steps:      []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	_, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error for confidence=0: %v", err)
	}
}

func TestNormalize_ConfidenceOne(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent:     "test",
		Mode:       "single",
		Confidence: 1.0,
		Steps:      []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	_, err := n.Normalize(schema, "r", "c", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error for confidence=1: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestNormalize_NilSchema(t *testing.T) {
	n := NewPlanNormalizer()
	_, err := n.Normalize(nil, "r", "c", "auto", "")
	if err == nil {
		t.Error("expected error for nil schema")
	}
}

func TestNormalize_UnknownMode(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "super_fast_mode",
		Steps:  []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	_, err := n.Normalize(schema, "r", "c", "auto", "")
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}

// ---------------------------------------------------------------------------
// Metadata checks
// ---------------------------------------------------------------------------

func TestNormalize_ValidationAlwaysFalse(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "single",
		Steps:  []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	orchPlan, _ := n.Normalize(schema, "r", "c", "auto", "")
	if orchPlan.Validation.Validated {
		t.Error("expected Validation.Validated=false from normalizer (only validator sets true)")
	}
}

// ---------------------------------------------------------------------------
// Warnings and participant reasons (P0 #6)
// ---------------------------------------------------------------------------

func TestMainAgentSchemaCarriesParticipantReasonsOrWarnings(t *testing.T) {
	n := NewPlanNormalizer()

	schema := &PlanSchema{
		Intent: "build a full-stack app",
		Mode:   "parallel",
		Steps: []PlanStep{
			{ID: "step-1", AgentName: "code-agent", Input: "write API", Reason: "code-agent handles backend logic"},
			{ID: "step-2", AgentName: "web-agent", Input: "build UI", Reason: "web-agent handles frontend rendering"},
		},
		Warnings: []string{"confidence below recommended threshold", "code-agent load may be high"},
	}

	orchPlan, err := n.Normalize(schema, "run_warn", "conv_warn", "auto", "main_agent_orchestration")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Warnings must be carried through.
	if len(orchPlan.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(orchPlan.Warnings))
	}
	if orchPlan.Warnings[0] != "confidence below recommended threshold" {
		t.Errorf("unexpected warning[0]: %q", orchPlan.Warnings[0])
	}
	if orchPlan.Warnings[1] != "code-agent load may be high" {
		t.Errorf("unexpected warning[1]: %q", orchPlan.Warnings[1])
	}

	// Step reasons must be carried to tasks.
	if orchPlan.Tasks[0].Reason != "code-agent handles backend logic" {
		t.Errorf("expected task[0].Reason='code-agent handles backend logic', got %q", orchPlan.Tasks[0].Reason)
	}
	if orchPlan.Tasks[1].Reason != "web-agent handles frontend rendering" {
		t.Errorf("expected task[1].Reason='web-agent handles frontend rendering', got %q", orchPlan.Tasks[1].Reason)
	}
}

func TestMainAgentSchema_WarningsEmptyWhenOmitted(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "simple task",
		Mode:   "single",
		Steps:  []PlanStep{{AgentName: "code-agent", Input: "write a function"}},
	}
	orchPlan, err := n.Normalize(schema, "run_no_warn", "conv_no_warn", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Warnings != nil {
		t.Errorf("expected nil Warnings when schema has none, got %v", orchPlan.Warnings)
	}
}

func TestMainAgentSchema_ReasonEmptyWhenOmitted(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "simple task",
		Mode:   "single",
		Steps:  []PlanStep{{AgentName: "code-agent", Input: "write a function"}},
	}
	orchPlan, err := n.Normalize(schema, "run_no_reason", "conv_no_reason", "auto", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orchPlan.Tasks[0].Reason != "" {
		t.Errorf("expected empty Reason, got %q", orchPlan.Tasks[0].Reason)
	}
}

func TestNormalize_GeneratesPlanID(t *testing.T) {
	n := NewPlanNormalizer()
	schema := &PlanSchema{
		Intent: "test",
		Mode:   "single",
		Steps:  []PlanStep{{AgentName: "code-agent", Input: "do it"}},
	}
	orchPlan, _ := n.Normalize(schema, "run_x", "conv_x", "auto", "")
	if orchPlan.PlanID == "" {
		t.Error("expected non-empty PlanID")
	}
	if orchPlan.RunID != "run_x" {
		t.Errorf("expected RunID 'run_x', got %q", orchPlan.RunID)
	}
	if orchPlan.ConversationID != "conv_x" {
		t.Errorf("expected ConversationID 'conv_x', got %q", orchPlan.ConversationID)
	}
}
