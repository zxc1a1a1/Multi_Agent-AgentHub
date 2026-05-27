package adk

import (
	"context"
	"encoding/json"
	"testing"
)

type mockTool struct {
	name        string
	description string
	schema      json.RawMessage
	execute     func(context.Context, json.RawMessage) (*ToolResult, error)
}

func (m mockTool) Name() string {
	return m.name
}

func (m mockTool) Description() string {
	return m.description
}

func (m mockTool) Schema() json.RawMessage {
	return m.schema
}

func (m mockTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	if m.execute != nil {
		return m.execute(ctx, args)
	}
	return &ToolResult{}, nil
}

func TestTool_InterfaceCompliance(t *testing.T) {
	var _ Tool = mockTool{}

	expectedSchema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	tool := mockTool{
		name:        "search",
		description: "searches documentation",
		schema:      expectedSchema,
	}

	var i Tool = tool

	if i.Name() != "search" {
		t.Fatalf("unexpected tool name: %q", i.Name())
	}
	if i.Description() != "searches documentation" {
		t.Fatalf("unexpected tool description: %q", i.Description())
	}
	if string(i.Schema()) != string(expectedSchema) {
		t.Fatalf("unexpected schema: %s", string(i.Schema()))
	}
}

func TestTool_ExecuteReceivesArguments(t *testing.T) {
	args := json.RawMessage(`{"language":"go","strict":true}`)
	tool := mockTool{
		name:        "generate_code",
		description: "generates code",
		schema:      json.RawMessage(`{"type":"object"}`),
		execute: func(ctx context.Context, gotArgs json.RawMessage) (*ToolResult, error) {
			if ctx == nil {
				t.Fatal("context must not be nil")
			}

			var payload map[string]any
			if err := json.Unmarshal(gotArgs, &payload); err != nil {
				t.Fatalf("failed to unmarshal execute args: %v", err)
			}
			if payload["language"] != "go" {
				t.Fatalf("unexpected language: %#v", payload["language"])
			}
			strictValue, ok := payload["strict"].(bool)
			if !ok || !strictValue {
				t.Fatalf("unexpected strict value: %#v", payload["strict"])
			}

			return &ToolResult{Content: "ok", IsError: false}, nil
		},
	}

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if result == nil {
		t.Fatal("execute returned nil result")
	}
	if result.Content != "ok" {
		t.Fatalf("unexpected result content: %q", result.Content)
	}
	if result.IsError {
		t.Fatal("expected result.IsError to be false")
	}
}

func TestToolResult_Error(t *testing.T) {
	result := ToolResult{
		Content: "tool failed",
		IsError: true,
	}

	if result.Content != "tool failed" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
}

func TestTool_SchemaIsJSON(t *testing.T) {
	tool := mockTool{
		name:        "validate_schema",
		description: "validates schema",
		schema:      json.RawMessage(`{"type":"object","required":["input"]}`),
	}

	var decoded map[string]any
	if err := json.Unmarshal(tool.Schema(), &decoded); err != nil {
		t.Fatalf("schema is not valid json: %v", err)
	}
	if decoded["type"] != "object" {
		t.Fatalf("unexpected schema type: %#v", decoded["type"])
	}
}
