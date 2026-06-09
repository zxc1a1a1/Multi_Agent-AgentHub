package planner

import (
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// PlanNormalizer converts a raw LLM PlanSchema into an internal OrchestrationPlan.
// It is a strict translator, not a guesser:
//   - Unknown agent names are preserved as-is for the validator to reject.
//   - Missing step IDs are generated (step-1, step-2, ...).
//   - nil depends_on is normalized to an empty slice.
//   - confidence is validated to be in [0,1].
//   - LLM mode=single → StrategySingle
//   - LLM mode=parallel → StrategyOrderedParallel (compatibility mapping)
//   - LLM mode=sequential → StrategySequential (validator rejects it)
//
// The normalizer does NOT call RulePlanner, fuzzyMatchAgent, defaultAgent,
// or any other guessing logic. The output always has Validation.Validated=false;
// only the validator may set it to true.
type PlanNormalizer struct{}

// NewPlanNormalizer creates a PlanNormalizer.
func NewPlanNormalizer() *PlanNormalizer {
	return &PlanNormalizer{}
}

// Normalize converts a PlanSchema into an OrchestrationPlan.
// runID, conversationID, planningMode, and executionPath are carried through from the request.
func (n *PlanNormalizer) Normalize(schema *PlanSchema, runID, conversationID, planningMode, executionPath string) (*plan.OrchestrationPlan, error) {
	if schema == nil {
		return nil, fmt.Errorf("normalize: nil schema")
	}

	// 1. Validate confidence range (done here because OrchestrationPlan
	//    does not carry a confidence field).
	if schema.Confidence < 0 || schema.Confidence > 1 {
		return nil, fmt.Errorf("normalize: confidence %f out of range [0,1]", schema.Confidence)
	}

	// 2. Map LLM mode to internal strategy.
	strategy, err := mapModeToStrategy(schema.Mode)
	if err != nil {
		return nil, err
	}

	// 3. Normalize each step — no agent name guessing.
	tasks := make([]plan.TaskPlan, 0, len(schema.Steps))
	for i, s := range schema.Steps {
		task := n.normalizeStep(s, i)
		tasks = append(tasks, task)
	}

	// 4. Build aggregation config.
	aggregation := plan.Aggregation{Required: false, Mode: "none"}
	if strategy == plan.StrategyOrderedParallel {
		aggregation = plan.Aggregation{Required: true, Mode: "summary"}
	}

	// 5. Assemble the OrchestrationPlan with Validation.Validated=false.
	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         plan.NewPlanID(),
		RunID:          runID,
		ConversationID: conversationID,
		PlanningMode:   planningMode,
		ExecutionPath:  executionPath,
		Strategy:       strategy,
		IntentSummary:  truncateToLength(strings.TrimSpace(schema.Intent), 120),
		Tasks:          tasks,
		Aggregation:    aggregation,
		Fallback:       plan.Fallback{Enabled: false},
		Validation:     plan.Validation{Validated: false},
		Warnings:       schema.Warnings,
	}, nil
}

// normalizeStep translates a single PlanStep into a TaskPlan.
// Agent name is lower-cased but never guessed or replaced.
func (n *PlanNormalizer) normalizeStep(s PlanStep, index int) plan.TaskPlan {
	// Generate ID if missing.
	id := strings.TrimSpace(s.ID)
	if id == "" {
		id = fmt.Sprintf("step-%d", index+1)
	}

	agentName := strings.ToLower(strings.TrimSpace(s.AgentName))
	input := strings.TrimSpace(s.Input)

	dependsOn := s.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}

	priority := index + 1
	if s.ID != "" {
		// If LLM provided an explicit step id, keep the index-based priority.
		// The priority is informational for the executor sort order.
	}

	return plan.TaskPlan{
		TaskID:          id,
		AgentName:       agentName,
		CapabilityIDs:   []string{}, // filled by Phase 3 registry wiring
		TaskContent:     input,
		ExpectedOutputs: []string{}, // filled by Phase 3 registry wiring
		DependsOn:       dependsOn,
		Priority:        priority,
		TimeoutMs:       120000,
		RiskLevel:       "low",
		Reason:          strings.TrimSpace(s.Reason),
	}
}

// mapModeToStrategy maps the LLM-facing mode to an internal strategy constant.
// sequential is mapped to StrategySequential so the validator can reject it
// deterministically.
func mapModeToStrategy(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case LLMModeSingle:
		return plan.StrategySingle, nil
	case LLMModeParallel:
		// Compatibility mapping per design standard §3.
		return plan.StrategyOrderedParallel, nil
	case LLMModeSequential:
		// Mapped to the constant so the validator produces a clear rejection.
		return plan.StrategySequential, nil
	default:
		return "", fmt.Errorf("normalize: unknown mode %q", mode)
	}
}

func truncateToLength(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
