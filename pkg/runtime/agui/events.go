package agui

// Event is a frontend-consumable AG-UI event DTO.
type Event struct {
	Type       string         `json:"type"`
	ID         string         `json:"id,omitempty"`
	Author     string         `json:"author,omitempty"`
	Role       string         `json:"role,omitempty"`
	Text       string         `json:"text,omitempty"`
	ToolCall   *ToolCall      `json:"toolCall,omitempty"`
	ToolResult *ToolResult    `json:"toolResult,omitempty"`
	Artifact   *Artifact      `json:"artifact,omitempty"`
	StateDelta map[string]any `json:"stateDelta,omitempty"`
	Final      bool           `json:"final,omitempty"`
	Partial    bool           `json:"partial,omitempty"`
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
