package adk

import "context"

// Plugin hooks into agent generation and tool call lifecycle.
type Plugin interface {
	BeforeGenerate(ctx context.Context, state *SessionState, req *GenerateRequest) error
	AfterGenerate(ctx context.Context, state *SessionState, resp *GenerateResponse) error
	BeforeTool(ctx context.Context, state *SessionState, call *ToolCallPart) error
	AfterTool(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error
}

// BasePlugin provides no-op default hooks.
type BasePlugin struct{}

func (BasePlugin) BeforeGenerate(context.Context, *SessionState, *GenerateRequest) error {
	return nil
}

func (BasePlugin) AfterGenerate(context.Context, *SessionState, *GenerateResponse) error {
	return nil
}

func (BasePlugin) BeforeTool(context.Context, *SessionState, *ToolCallPart) error {
	return nil
}

func (BasePlugin) AfterTool(context.Context, *SessionState, *ToolCallPart, *ToolResult) error {
	return nil
}
