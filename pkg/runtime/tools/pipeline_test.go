package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// TestToolPipelineFromDescriptorToEvent covers the P3a closure:
// schema → broker authorization → execution → bounded event.
func TestToolPipelineFromDescriptorToEvent(t *testing.T) {
	w, _ := newTestWorkspace(t)
	tool := w.ReadTool()
	descriptor := tool.Descriptor()
	if err := descriptor.Validate(); err != nil {
		t.Fatalf("descriptor invalid: %v", err)
	}
	envelope := Envelope{
		RunID: "run-1", ToolCallID: "tc-1", ToolName: tool.Name(), ToolVersion: descriptor.Version,
		ArgsHash: "hash", PrincipalID: "principal", FencingVersion: 1,
	}
	if err := (Guard{}).Authorize(context.Background(), envelope, descriptor); err != nil {
		t.Fatalf("authorize read-only tool: %v", err)
	}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt"}`))
	if err != nil || result.IsError {
		t.Fatalf("execute: err=%v result=%#v", err, result)
	}
	event := EventFromResult(envelope.ToolCallID, tool.Name(), result, 4096)
	if event.Status != "succeeded" || event.Truncated || len(event.OutputHash) != 64 {
		t.Fatalf("unexpected tool event: %#v", event)
	}
	if !strings.Contains(event.Output, "hello world") {
		t.Fatalf("event output lost content: %s", event.Output)
	}
}

func TestEventFromResultTruncatesAndHashesFullOutput(t *testing.T) {
	full := strings.Repeat("x", 100)
	event := EventFromResult("tc", "workspace.read", &adk.ToolResult{Content: full}, 16)
	if !event.Truncated || len(event.Output) != 16 {
		t.Fatalf("expected 16-byte truncation, got %#v", event)
	}
	if len(event.OutputHash) != 64 {
		t.Fatalf("hash must cover full output: %s", event.OutputHash)
	}
}
