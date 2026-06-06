package planner

// PlannerSource values for PlannerTrace.Source.
const (
	TraceSourceLLM      = "llm"
	TraceSourceRule     = "rule"
	TraceSourceFallback = "fallback"
)

// PlannerTrace records metadata about a single planning invocation.
// It is used for logging, observability, and populating the run_started
// SSE event state metadata. It MUST NOT contain raw prompts, API keys,
// internal service URLs, or other secrets.
type PlannerTrace struct {
	// Source indicates which planner produced this plan: llm, rule, or fallback.
	Source string

	// Model is the LLM model name used, e.g. "claude-haiku-4-5-20251001".
	// Empty for rule-based planning.
	Model string

	// Provider is the LLM provider: "anthropic" or "openai".
	// Empty for rule-based planning.
	Provider string

	// Fallback is true when the primary planner failed and a fallback was used.
	Fallback bool

	// FallbackReason explains why a fallback was triggered.
	FallbackReason string

	// RepairCount is the number of repair attempts made by the Repairer.
	RepairCount int

	// ParseError records the last parse error, if any.
	ParseError string

	// ValidationErrors records validation errors found, if any.
	ValidationErrors []string

	// Intent is the final intent summary applied to the plan.
	Intent string

	// Mode is the LLM-facing mode that was returned (single/parallel/sequential).
	Mode string

	// TaskCount is the number of tasks in the resulting plan.
	TaskCount int

	// Agents lists the agent names involved in the plan.
	Agents []string

	// LatencyMS is the planning duration in milliseconds.
	LatencyMS int64
}
