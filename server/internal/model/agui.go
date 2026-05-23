package model

type AGUIRunRequest struct {
	ThreadID string        `json:"threadId" binding:"required,max=128"`
	RunID    string        `json:"runId" binding:"required,max=128"`
	Messages []AGUIMessage `json:"messages" binding:"required,min=1,dive"`
	Tools    []AGUITool    `json:"tools" binding:"omitempty,dive"`
}

type AGUIMessage struct {
	Role    string `json:"role" binding:"required,oneof=user agent system"`
	Content string `json:"content" binding:"required,max=20000"`
}

type AGUITool struct {
	Name string `json:"name" binding:"required,printascii,max=64"`
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
