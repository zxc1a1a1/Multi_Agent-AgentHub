package agui

import "encoding/json"

// Event is a frontend-consumable AG-UI event DTO.
// v1.0 fields align with docs/contracts/agui-events.md.
// Legacy fields (Author, Text, Content, StateDelta) are kept for backward compatibility.
type Event struct {
	Type string `json:"type"`

	// AG-UI v1.0 standard identity fields
	RunID     string `json:"runId,omitempty"`
	ThreadID  string `json:"threadId,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	TaskID    string `json:"taskId,omitempty"`

	// AG-UI v1.0 sender object (replaces bare Author string)
	Sender *EventSender `json:"sender,omitempty"`

	// Legacy identity fields — retained for backward compatibility
	ID     string `json:"id,omitempty"`
	Author string `json:"author,omitempty"`

	Role string `json:"role,omitempty"`

	// AG-UI v1.0 primary text fields
	Delta   string `json:"delta,omitempty"`
	Content string `json:"content,omitempty"`

	// Legacy text field — retained for backward compatibility
	Text string `json:"text,omitempty"`

	// Tool call fields
	ToolCall   *ToolCall   `json:"toolCall,omitempty"`
	ToolResult *ToolResult `json:"toolResult,omitempty"`

	// Artifact field
	Artifact *Artifact `json:"artifact,omitempty"`

	// State fields: State is AG-UI v1.0 standard; StateDelta is legacy compat
	State      map[string]any `json:"state,omitempty"`
	StateDelta map[string]any `json:"stateDelta,omitempty"`

	// Activity field for ACTIVITY_SNAPSHOT events (v1.3 plan approval)
	Activity *ActivitySnapshot `json:"activity,omitempty"`

	// Error field (AG-UI v1.0 standard)
	Error *SafeError `json:"error,omitempty"`

	// AGENT_TURN top-level fields (Phase 5-0 canonical — omitempty, non-AGENT_TURN events unaffected)
	TurnIndex int    `json:"turnIndex,omitempty"`
	StepID    string `json:"stepId,omitempty"`
	AgentName string `json:"agentName,omitempty"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`

	// Timestamp and tracing
	Timestamp string `json:"timestamp,omitempty"`
	TraceID   string `json:"traceId,omitempty"`

	// Lifecycle flags
	Final   bool `json:"final,omitempty"`
	Partial bool `json:"partial,omitempty"`
}

// EventSender identifies the sender of a message in AG-UI events.
// MarshalJSON preserves the required turnIndex field for AGENT_TURN events,
// including the first turn where turnIndex is 0. The struct tag keeps turnIndex
// omitted for non-turn events to avoid polluting unrelated AG-UI events.
func (e Event) MarshalJSON() ([]byte, error) {
	type eventAlias Event
	data, err := json.Marshal(eventAlias(e))
	if err != nil {
		return nil, err
	}
	if !isAgentTurnType(e.Type) {
		return data, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	obj["turnIndex"] = e.TurnIndex
	return json.Marshal(obj)
}

type EventSender struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
}

// SafeError is AG-UI v1.0 standard error payload.
type SafeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ToolCall is the AG-UI DTO for one tool call.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments any    `json:"arguments,omitempty"`
}

// ToolResult is the AG-UI DTO for one tool execution result.
type ToolResult struct {
	CallID  string `json:"callId"`
	Name    string `json:"name"`
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

// Artifact is the AG-UI DTO for one artifact delta.
type Artifact struct {
	Type     string            `json:"type"`
	Title    string            `json:"title,omitempty"`
	Content  string            `json:"content,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ActivitySnapshot is the AG-UI v1.3 DTO for plan approval and activity state.
// Replaces the deprecated confirm_plan TOOL_CALL events.
type ActivitySnapshot struct {
	ActivityID                  string            `json:"activityId"`
	ActivityType                string            `json:"activityType"`
	Status                      string            `json:"status"`
	ExecutionPath               string            `json:"executionPath"`
	PlanID                      string            `json:"planId"`
	Revision                    int               `json:"revision"`
	PlanOwner                   *PlanOwner        `json:"planOwner"`
	Participants                []PlanParticipant `json:"participants"`
	CandidateParticipants       []PlanParticipant `json:"candidateParticipants,omitempty"`
	DefaultSelectedParticipants []PlanParticipant `json:"defaultSelectedParticipants,omitempty"`
	RequiredParticipants        []string          `json:"requiredParticipants,omitempty"`
	Title                       string            `json:"title,omitempty"`
	Summary                     string            `json:"summary,omitempty"`
	Tasks                       []TaskSummary     `json:"tasks"`
	AllowedActions              []string          `json:"allowedActions"`
	Warnings                    []string          `json:"warnings,omitempty"`
}

// PlanOwner identifies the owner of an orchestration plan.
type PlanOwner struct {
	Type        string `json:"type"`
	AgentName   string `json:"agentName"`
	IsMainAgent bool   `json:"isMainAgent"`
}

// PlanParticipant represents an agent participant in a plan.
type PlanParticipant struct {
	AgentName string `json:"agentName"`
	Required  bool   `json:"required"`
	Selected  bool   `json:"selected"`
}

// TaskSummary is a lightweight task entry carried in ACTIVITY_SNAPSHOT.
type TaskSummary struct {
	TaskID    string `json:"taskId"`
	AgentName string `json:"agentName"`
	Content   string `json:"content"`
	Priority  int    `json:"priority"`
	RiskLevel string `json:"riskLevel"`
}

// AgentTurnStarted is the AG-UI event marking the start of an agent's execution turn.
// Required for group_chat and main_agent_orchestration execution paths.
type AgentTurnStarted struct {
	Type      string       `json:"type"`
	RunID     string       `json:"runId"`
	MessageID string       `json:"messageId"`
	TurnIndex int          `json:"turnIndex"`
	StepID    string       `json:"stepId"`
	AgentName string       `json:"agentName"`
	Sender    *EventSender `json:"sender"`
}

// AgentTurnContent is the AG-UI event for incremental text content within an agent's turn.
type AgentTurnContent struct {
	Type      string `json:"type"`
	RunID     string `json:"runId"`
	MessageID string `json:"messageId"`
	TurnIndex int    `json:"turnIndex"`
	Delta     string `json:"delta"`
}

// AgentTurnFinished is the AG-UI event marking the end of an agent's execution turn.
type AgentTurnFinished struct {
	Type      string `json:"type"`
	RunID     string `json:"runId"`
	MessageID string `json:"messageId"`
	TurnIndex int    `json:"turnIndex"`
	AgentName string `json:"agentName"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status,omitempty"` // "completed" or "failed"
}
