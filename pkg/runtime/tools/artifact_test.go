package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestArtifactPublish(t *testing.T) {
	store := NewInMemoryArtifactStore()
	tool := ArtifactPublishTool(store)
	if tool.Descriptor().Approval != ApprovalConfirm || tool.Descriptor().ReadOnly {
		t.Fatalf("artifact descriptor wrong: %#v", tool.Descriptor())
	}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"name":"report.md","mimeType":"text/markdown","content":"# hi"}`))
	if err != nil || res.IsError {
		t.Fatalf("publish failed: %v %#v", err, res)
	}
	out := decodeResult(t, res.Content)
	id, _ := out["artifactId"].(string)
	artifact, ok := store.Get(id)
	if !ok || artifact.Content != "# hi" || artifact.Size != 4 {
		t.Fatalf("artifact not stored: %#v", artifact)
	}
	res, _ = tool.Execute(context.Background(), json.RawMessage(`{"name":"x","content":""}`))
	if !res.IsError {
		t.Fatal("empty artifact must be rejected")
	}
	res, _ = tool.Execute(context.Background(), json.RawMessage(`{"name":"big","content":"`+strings.Repeat("a", maxArtifactBytes+1)+`"}`))
	if !res.IsError {
		t.Fatal("oversized artifact must be rejected")
	}
}

func TestArtifactRequiresApproval(t *testing.T) {
	store := NewInMemoryArtifactStore()
	tool := ArtifactPublishTool(store)
	d := tool.Descriptor()
	env := Envelope{RunID: "r", ToolCallID: "tc", ToolName: d.Name, ToolVersion: d.Version, ArgsHash: "h", PrincipalID: "p", FencingVersion: 1}
	if _, err := Execute(context.Background(), Guard{}, env, d, tool, json.RawMessage(`{"name":"x","content":"y"}`)); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("expected approval requirement, got %v", err)
	}
}
