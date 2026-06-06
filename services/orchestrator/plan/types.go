// Package plan defines the OrchestrationPlan and TaskPlan types for intent orchestration.
package plan

// Strategy values.
const (
	StrategySingle         = "single"
	StrategyOrderedParallel = "ordered_parallel"
	StrategySequential      = "sequential"
)

// OrchestrationPlan is the structured result of intent orchestration.
type OrchestrationPlan struct {
	Version        string      `json:"version"`
	PlanID         string      `json:"planId"`
	RunID          string      `json:"runId"`
	ConversationID string      `json:"conversationId"`
	PlanningMode   string      `json:"planningMode"`
	Strategy       string      `json:"strategy"`
	IntentSummary  string      `json:"intentSummary"`
	Tasks          []TaskPlan  `json:"tasks"`
	Aggregation    Aggregation `json:"aggregation"`
	Fallback       Fallback    `json:"fallback"`
	Validation     Validation  `json:"validation"`

	// Planner metadata — populated by LLMPlanner, zero-value for RulePlanner.
	PlannerReasoning string `json:"plannerReasoning,omitempty"` // LLM reasoning
	PlannerModel     string `json:"plannerModel,omitempty"`     // model name used
	PlannerSource    string `json:"plannerSource,omitempty"`    // "llm" | "rule" | "fallback"
	RepairCount      int    `json:"repairCount,omitempty"`      // number of repair attempts (max 1)
}

// TaskPlan is a single execution unit within an OrchestrationPlan.
type TaskPlan struct {
	TaskID          string   `json:"taskId"`
	AgentName       string   `json:"agentName"`
	CapabilityIDs   []string `json:"capabilityIds"`
	TaskContent     string   `json:"taskContent"`
	ExpectedOutputs []string `json:"expectedOutputs"`
	DependsOn       []string `json:"dependsOn"`
	Priority        int      `json:"priority"`
	TimeoutMs       int64    `json:"timeoutMs"`
	RiskLevel       string   `json:"riskLevel"`
}

// Aggregation defines how task outputs should be combined.
type Aggregation struct {
	Required bool   `json:"required"`
	Mode     string `json:"mode"`
}

// Fallback defines fallback behavior when a plan fails.
type Fallback struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

// Validation holds the local validation result. Validated must only be set by
// a local validator, never by the Planner.
type Validation struct {
	Validated bool `json:"validated"`
}
