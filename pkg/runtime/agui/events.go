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
