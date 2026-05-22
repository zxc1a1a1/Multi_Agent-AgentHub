package model

type AGUIRunRequest struct {
	ThreadID string        `json:"threadId"`
	RunID    string        `json:"runId"`
	Messages []AGUIMessage `json:"messages"`
	Tools    []AGUITool    `json:"tools"`
}

type AGUIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AGUITool struct {
	Name string `json:"name"`
}

type AGUIEvent struct {
	Type       string `json:"type"`
	MessageID  string `json:"messageId,omitempty"`
	RunID      string `json:"runId,omitempty"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	Error      string `json:"error,omitempty"`
}
