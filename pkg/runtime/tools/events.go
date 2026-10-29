package tools

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// ToolEvent is the sanitized, bounded event sent to the frontend. It never
// carries the full unbounded tool output when the limit is exceeded.
type ToolEvent struct {
	Type       string `json:"type"` // tool_call_result
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Status     string `json:"status"` // succeeded|failed
	Output     string `json:"output,omitempty"`
	OutputHash string `json:"outputHash"`
	Truncated  bool   `json:"truncated"`
}

// EventFromResult converts a tool result into a bounded event with a hash for
// audit/replay without storing the full payload in the event stream.
func EventFromResult(callID, toolName string, result *adk.ToolResult, maxBytes int) ToolEvent {
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	content := ""
	isError := false
	if result != nil {
		content = result.Content
		isError = result.IsError
	}
	sum := sha256.Sum256([]byte(content))
	truncated := false
	if len(content) > maxBytes {
		content = content[:maxBytes]
		truncated = true
	}
	status := "succeeded"
	if isError {
		status = "failed"
	}
	return ToolEvent{
		Type:       "tool_call_result",
		ToolCallID: callID,
		ToolName:   toolName,
		Status:     status,
		Output:     content,
		OutputHash: hex.EncodeToString(sum[:]),
		Truncated:  truncated,
	}
}
