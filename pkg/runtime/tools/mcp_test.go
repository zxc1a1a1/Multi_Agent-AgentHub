package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeMCPClient struct {
	tools []MCPToolInfo
	calls map[string]int
}

func (f *fakeMCPClient) ListTools(context.Context) ([]MCPToolInfo, error) {
	return f.tools, nil
}

func (f *fakeMCPClient) CallTool(_ context.Context, name string, _ json.RawMessage) (MCPCallResult, error) {
	f.calls[name]++
	return MCPCallResult{Content: "result-" + name}, nil
}

func TestMCPAdapterExposesOnlyAllowListedTools(t *testing.T) {
	client := &fakeMCPClient{
		tools: []MCPToolInfo{
			{Name: "search", Description: "search", InputSchema: json.RawMessage(`{"type":"object"}`), ReadOnly: true},
			{Name: "delete", Description: "delete", InputSchema: json.RawMessage(`{"type":"object"}`), ReadOnly: false},
		},
		calls: map[string]int{},
	}
	adapter, err := NewMCPAdapter(context.Background(), client, "docs-server", "1.2.3", map[string]MCPToolPolicy{
		"search": {ReadOnly: true, Approval: ApprovalNone},
	})
	if err != nil {
		t.Fatal(err)
	}
	tools := adapter.Tools()
	if len(tools) != 1 || tools[0].Name() != "search" {
		t.Fatalf("only allow-listed tool should be exposed: %#v", tools)
	}
	if tools[0].Descriptor().Provider != "mcp:docs-server" || tools[0].Descriptor().Version != "1.2.3" {
		t.Fatalf("provider/version missing: %#v", tools[0].Descriptor())
	}
	res, err := tools[0].Execute(context.Background(), json.RawMessage(`{"q":"x"}`))
	if err != nil || res.IsError || res.Content != "result-search" {
		t.Fatalf("MCP call failed: %v %#v", err, res)
	}
	if client.calls["delete"] != 0 {
		t.Fatal("unlisted MCP tool was called")
	}
}

func TestMCPPolicyOverridesServerAnnotations(t *testing.T) {
	client := &fakeMCPClient{
		tools: []MCPToolInfo{{Name: "write", InputSchema: json.RawMessage(`{"type":"object"}`), ReadOnly: true}},
		calls: map[string]int{},
	}
	adapter, err := NewMCPAdapter(context.Background(), client, "server", "1", map[string]MCPToolPolicy{
		"write": {ReadOnly: false, Approval: ApprovalAlways},
	})
	if err != nil {
		t.Fatal(err)
	}
	tool := adapter.Tools()[0]
	if tool.Descriptor().ReadOnly {
		t.Fatal("server readOnly annotation must not grant read-only policy")
	}
	env := Envelope{RunID: "r", ToolCallID: "tc", ToolName: "write", ToolVersion: "1", ArgsHash: "h", PrincipalID: "p", FencingVersion: 1}
	if _, err := Execute(context.Background(), Guard{}, env, tool.Descriptor(), tool, json.RawMessage(`{}`)); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("unapproved MCP write must be blocked, got %v", err)
	}
	if client.calls["write"] != 0 {
		t.Fatal("unapproved MCP write reached the server")
	}
}

func TestMCPAdapterRejectsInvalidSchemaAndUnknownPolicy(t *testing.T) {
	bad := &fakeMCPClient{tools: []MCPToolInfo{{Name: "bad", InputSchema: json.RawMessage(`not-json`)}}}
	if _, err := NewMCPAdapter(context.Background(), bad, "s", "1", nil); err == nil {
		t.Fatal("invalid schema must be rejected")
	}
	empty := &fakeMCPClient{tools: []MCPToolInfo{}}
	if _, err := NewMCPAdapter(context.Background(), empty, "s", "1", map[string]MCPToolPolicy{"ghost": {ReadOnly: true}}); err == nil {
		t.Fatal("policy for an unadvertised tool must be rejected")
	}
}
