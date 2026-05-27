package adk

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

type orderTrackingPlugin struct {
	BasePlugin
	steps *[]string
}

func (p *orderTrackingPlugin) BeforeGenerate(ctx context.Context, state *SessionState, req *GenerateRequest) error {
	if p.steps != nil {
		*p.steps = append(*p.steps, "before_generate")
	}
	if state != nil {
		state.Set("before_generate_seen", true)
	}
	return nil
}

func (p *orderTrackingPlugin) AfterGenerate(ctx context.Context, state *SessionState, resp *GenerateResponse) error {
	if p.steps != nil {
		*p.steps = append(*p.steps, "after_generate")
	}
	if state != nil {
		state.Set("after_generate_seen", true)
	}
	return nil
}

func (p *orderTrackingPlugin) BeforeTool(ctx context.Context, state *SessionState, call *ToolCallPart) error {
	if p.steps != nil {
		*p.steps = append(*p.steps, "before_tool")
	}
	if state != nil {
		state.Set("before_tool_seen", true)
	}
	return nil
}

func (p *orderTrackingPlugin) AfterTool(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error {
	if p.steps != nil {
		*p.steps = append(*p.steps, "after_tool")
	}
	if state != nil {
		state.Set("after_tool_seen", true)
	}
	return nil
}

func TestBasePlugin_NoOp(t *testing.T) {
	plugin := BasePlugin{}
	ctx := context.Background()
	state := NewSessionState(nil)
	req := &GenerateRequest{
		Contents: []*Content{
			{
				Role:  RoleUser,
				Parts: []Part{TextPart{Text: "hello"}},
			},
		},
	}
	resp := &GenerateResponse{
		Parts: []Part{TextPart{Text: "done"}},
	}
	call := &ToolCallPart{
		ID:        "call-1",
		Name:      "unit_tool",
		Arguments: json.RawMessage(`{"ok":true}`),
	}
	result := &ToolResult{Content: "ok", IsError: false}

	if err := plugin.BeforeGenerate(ctx, state, req); err != nil {
		t.Fatalf("BeforeGenerate should be nil error, got: %v", err)
	}
	if err := plugin.AfterGenerate(ctx, state, resp); err != nil {
		t.Fatalf("AfterGenerate should be nil error, got: %v", err)
	}
	if err := plugin.BeforeTool(ctx, state, call); err != nil {
		t.Fatalf("BeforeTool should be nil error, got: %v", err)
	}
	if err := plugin.AfterTool(ctx, state, call, result); err != nil {
		t.Fatalf("AfterTool should be nil error, got: %v", err)
	}
}

func TestPlugin_InterfaceCompliance(t *testing.T) {
	var _ Plugin = BasePlugin{}
	var _ Plugin = &orderTrackingPlugin{}
}

func TestPlugin_OrderTracking(t *testing.T) {
	steps := make([]string, 0, 4)
	plugin := &orderTrackingPlugin{steps: &steps}
	state := NewSessionState(nil)
	ctx := context.Background()

	call := &ToolCallPart{
		ID:        "call-1",
		Name:      "format_code",
		Arguments: json.RawMessage(`{"language":"go"}`),
	}
	req := &GenerateRequest{
		Contents: []*Content{{Role: RoleUser, Parts: []Part{TextPart{Text: "format this"}}}},
	}
	resp := &GenerateResponse{
		Parts:        []Part{TextPart{Text: "formatted"}},
		FinishReason: FinishStop,
	}
	result := &ToolResult{Content: "ok"}

	if err := plugin.BeforeGenerate(ctx, state, req); err != nil {
		t.Fatalf("BeforeGenerate error: %v", err)
	}
	if err := plugin.AfterGenerate(ctx, state, resp); err != nil {
		t.Fatalf("AfterGenerate error: %v", err)
	}
	if err := plugin.BeforeTool(ctx, state, call); err != nil {
		t.Fatalf("BeforeTool error: %v", err)
	}
	if err := plugin.AfterTool(ctx, state, call, result); err != nil {
		t.Fatalf("AfterTool error: %v", err)
	}

	want := []string{"before_generate", "after_generate", "before_tool", "after_tool"}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("unexpected steps: got=%v want=%v", steps, want)
	}
}

func TestPlugin_CanAccessSessionState(t *testing.T) {
	plugin := &orderTrackingPlugin{}
	state := NewSessionState(map[string]any{"phase": "1.4+1.5"})
	ctx := context.Background()
	call := &ToolCallPart{
		ID:        "call-2",
		Name:      "lint",
		Arguments: json.RawMessage(`{"strict":true}`),
	}

	if err := plugin.BeforeGenerate(ctx, state, &GenerateRequest{}); err != nil {
		t.Fatalf("BeforeGenerate error: %v", err)
	}
	if err := plugin.AfterGenerate(ctx, state, &GenerateResponse{}); err != nil {
		t.Fatalf("AfterGenerate error: %v", err)
	}
	if err := plugin.BeforeTool(ctx, state, call); err != nil {
		t.Fatalf("BeforeTool error: %v", err)
	}
	if err := plugin.AfterTool(ctx, state, call, &ToolResult{}); err != nil {
		t.Fatalf("AfterTool error: %v", err)
	}

	phase, ok := state.Get("phase")
	if !ok || phase != "1.4+1.5" {
		t.Fatalf("unexpected phase from state: ok=%v value=%#v", ok, phase)
	}
	for _, key := range []string{"before_generate_seen", "after_generate_seen", "before_tool_seen", "after_tool_seen"} {
		value, ok := state.Get(key)
		if !ok {
			t.Fatalf("missing state key: %s", key)
		}
		boolValue, ok := value.(bool)
		if !ok || !boolValue {
			t.Fatalf("unexpected state value for %s: %#v", key, value)
		}
	}
}
