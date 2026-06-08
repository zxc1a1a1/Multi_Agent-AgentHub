package agui

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

	// Timestamp and tracing
	Timestamp string `json:"timestamp,omitempty"`
	TraceID   string `json:"traceId,omitempty"`

	// Lifecycle flags
	Final   bool `json:"final,omitempty"`
	Partial bool `json:"partial,omitempty"`
}

// EventSender identifies the sender of a message in AG-UI events.
type EventSender struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
}

// SafeError is AG-UI v1.0 standard error payload.
type SafeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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
	ActivityID                string            `json:"activityId"`
	ActivityType              string            `json:"activityType"`
	Status                    string            `json:"status"`
	ExecutionPath             string            `json:"executionPath"`
	PlanID                    string            `json:"planId"`
	Revision                  int               `json:"revision"`
	PlanOwner                 *PlanOwner        `json:"planOwner"`
	Participants              []PlanParticipant `json:"participants"`
	CandidateParticipants     []PlanParticipant `json:"candidateParticipants,omitempty"`
	DefaultSelectedParticipants []PlanParticipant `json:"defaultSelectedParticipants,omitempty"`
	RequiredParticipants      []string          `json:"requiredParticipants,omitempty"`
	Title                     string            `json:"title,omitempty"`
	Summary                   string            `json:"summary,omitempty"`
	Tasks                     []TaskSummary     `json:"tasks"`
	AllowedActions            []string          `json:"allowedActions"`
	Warnings                  []string          `json:"warnings,omitempty"`
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
