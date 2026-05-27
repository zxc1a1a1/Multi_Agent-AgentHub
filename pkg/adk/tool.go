package adk

import (
	"context"
	"encoding/json"
)

// Tool defines a callable capability exposed to a model during generation.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

// ToolResult is the normalized output payload returned by a Tool execution.
type ToolResult struct {
	Content string
	IsError bool
}
