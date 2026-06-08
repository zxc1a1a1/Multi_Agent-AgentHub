// Package plan defines the OrchestrationPlan and TaskPlan types for intent orchestration.
package plan

// Strategy values.
const (
	StrategySingle         = "single"
	StrategyOrderedParallel = "ordered_parallel"
	StrategySequential      = "sequential"
	StrategyConversational  = "conversational"
)

// OrchestrationPlan is the structured result of intent orchestration.
type OrchestrationPlan struct {
	Version        string      `json:"version"`
	PlanID         string      `json:"planId"`
	RunID          string      `json:"runId"`
	ConversationID string      `json:"conversationId"`
	PlanningMode   string      `json:"planningMode"`   // legacy — prefer ExecutionPath
	ExecutionPath  string      `json:"executionPath"`  // ChatExecutionPath used for this plan
	Strategy       string      `json:"strategy"`
	IntentSummary  string      `json:"intentSummary"`
	Tasks          []TaskPlan  `json:"tasks"`
	Aggregation    Aggregation `json:"aggregation"`
	Fallback       Fallback    `json:"fallback"`
	Validation     Validation  `json:"validation"`
	TraceID        string      `json:"traceId,omitempty"` // propagated from request through dispatcher to agents

	// Planner metadata — populated by LLMPlanner, zero-value for RulePlanner.
	PlannerReasoning string `json:"plannerReasoning,omitempty"` // LLM reasoning
	PlannerModel     string `json:"plannerModel,omitempty"`     // model name used
	PlannerSource    string `json:"plannerSource,omitempty"`    // "llm" | "rule" | "fallback"
	RepairCount      int    `json:"repairCount,omitempty"`      // number of repair attempts (max 1)

	// AllowedAgents carries the participant boundary that was enforced at plan time.
	AllowedAgents []string `json:"allowedAgents,omitempty"`

	// Revision is the monotonic plan revision counter, starting at 1.
	// Incremented on each REQUEST_PLAN_REVISION.
	Revision int `json:"revision,omitempty"`

	// PlanOwner identifies who generated this plan (agent or main_agent).
	PlanOwner *PlanOwner `json:"planOwner,omitempty"`

	// Participants is the list of agents involved in this plan.
	Participants []PlanParticipant `json:"participants,omitempty"`

	// CandidateParticipants lists all agents the MainAgent recommends as eligible.
	// The user may select a subset. MainAgent orchestration only.
	CandidateParticipants []PlanParticipant `json:"candidateParticipants,omitempty"`

	// DefaultSelectedParticipants is the MainAgent's recommended default selection.
	// MainAgent orchestration only.
	DefaultSelectedParticipants []PlanParticipant `json:"defaultSelectedParticipants,omitempty"`
}

// PlanOwner identifies the plan author.
type PlanOwner struct {
	Type        string `json:"type"`        // "agent" | "main_agent"
	AgentName   string `json:"agentName"`   // agent name (not code-agent for main_agent)
	IsMainAgent bool   `json:"isMainAgent,omitempty"`
}

// PlanParticipant describes one agent participant in a plan.
type PlanParticipant struct {
	AgentName string `json:"agentName"`
	Role      string `json:"role,omitempty"`
	Required  bool   `json:"required,omitempty"`
	Selected  bool   `json:"selected,omitempty"`
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
