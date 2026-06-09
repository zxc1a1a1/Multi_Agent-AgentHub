package planner

// PlanSchema is the canonical LLM-facing orchestration plan schema.
// The LLM is expected to return a JSON object conforming to this structure.
// It is the raw contract between the LLM and the Parser — Normalizer converts
// it into an internal OrchestrationPlan separately.
type PlanSchema struct {
	// Intent is a brief summary of the user's goal. Required.
	Intent string `json:"intent"`

	// Mode is the execution mode: "single", "parallel", or "sequential". Required.
	Mode string `json:"mode"`

	// Confidence is an optional 0..1 score expressing the LLM's confidence.
	// It is informational only and MUST NOT be the sole execution gate.
	Confidence float64 `json:"confidence,omitempty"`

	// Steps is the ordered list of execution steps. Required, non-empty.
	Steps []PlanStep `json:"steps"`

	// UserVisibleSummary is a human-readable summary safe for display to users.
	// It MUST NOT contain internal information such as URLs, keys, or raw prompts.
	UserVisibleSummary string `json:"user_visible_summary,omitempty"`

	// Warnings is an optional list of non-blocking plan-level warnings.
	Warnings []string `json:"warnings,omitempty"`
}

// PlanStep is a single execution step within a PlanSchema.
type PlanStep struct {
	// ID is an optional identifier. When missing, the Normalizer will generate one.
	ID string `json:"id,omitempty"`

	// AgentName is the target agent for this step. Required.
	AgentName string `json:"agent_name"`

	// Input is the task content for the agent. Required.
	Input string `json:"input"`

	// DependsOn lists step IDs that must complete before this step starts.
	// Optional; nil is normalized to an empty slice.
	DependsOn []string `json:"depends_on,omitempty"`

	// Reason explains why this agent was chosen. Optional.
	Reason string `json:"reason,omitempty"`
}

// LLMSchemaMode values define the allowed PlanSchema.Mode values.
const (
	LLMModeSingle     = "single"
	LLMModeParallel   = "parallel"
	LLMModeSequential = "sequential"
)
