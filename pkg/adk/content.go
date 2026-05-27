package adk

import "encoding/json"

// Role identifies the speaker role for a content message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

// Content is a role-tagged message composed from ordered parts.
type Content struct {
	Role  Role
	Parts []Part
}

// Part is a typed content segment in a Content message.
type Part interface {
	partMarker()
}

// TextPart carries plain text content.
type TextPart struct {
	Text string
}

func (TextPart) partMarker() {}

// ToolCallPart carries a model tool invocation request.
type ToolCallPart struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

func (ToolCallPart) partMarker() {}

// ToolResultPart carries the output from a tool call.
type ToolResultPart struct {
	CallID  string
	Name    string
	Content string
	IsError bool
}

func (ToolResultPart) partMarker() {}

// ThinkingPart is an optional model thinking signal for internal runtime use.
// It is an internal reasoning marker and should be handled safely by upper layers.
type ThinkingPart struct {
	Thinking string
}

func (ThinkingPart) partMarker() {}
