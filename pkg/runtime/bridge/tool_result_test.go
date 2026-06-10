package bridge

import (
	"strings"
	"testing"
)

func TestMapToolResultTextToMessage(t *testing.T) {
	m, err := MapToolResult(ToolResultInput{RunID: "run-1", TaskID: "task-1", ToolCallID: "tc-1", Status: "success", Data: map[string]any{"ok": true}})
	if err != nil {
		t.Fatalf("MapToolResult: %v", err)
	}
	if m == nil || m.Message == nil {
		t.Fatalf("expected message mapping, got %#v", m)
	}
	if m.Message.Role != "tool" {
		t.Fatalf("expected tool role, got %q", m.Message.Role)
	}
	if !strings.Contains(m.Message.Content, "tc-1") || !strings.Contains(m.Message.Content, "task-1") {
		t.Fatalf("message missing identifiers: %s", m.Message.Content)
	}
}

func TestMapToolResultArtifactRef(t *testing.T) {
	m, err := MapToolResult(ToolResultInput{ToolCallID: "tc-1", ContentType: "artifact_ref", Data: map[string]any{"ref": "artifact://1", "title": "report", "mimeType": "text/plain"}})
	if err != nil {
		t.Fatalf("MapToolResult: %v", err)
	}
	if m == nil || m.Artifact == nil || m.Artifact.Ref != "artifact://1" {
		t.Fatalf("expected artifact ref, got %#v", m)
	}
	if m.Message != nil {
		t.Fatalf("did not expect inline message for artifact ref")
	}
}

func TestMapToolResultSanitizesSecrets(t *testing.T) {
	m, err := MapToolResult(ToolResultInput{ToolCallID: "tc-1", Data: map[string]any{"token": "sk-abcdef1234567890"}})
	if err != nil {
		t.Fatalf("MapToolResult: %v", err)
	}
	if strings.Contains(m.Message.Content, "sk-abcdef") {
		t.Fatalf("secret was not redacted: %s", m.Message.Content)
	}
}

func TestMapToolResultLargeContentToArtifact(t *testing.T) {
	m, err := MapToolResult(ToolResultInput{ToolCallID: "tc-1", Data: strings.Repeat("x", maxInlineToolResultBytes+100)})
	if err != nil {
		t.Fatalf("MapToolResult: %v", err)
	}
	if m == nil || m.Artifact == nil {
		t.Fatalf("expected large content artifact mapping, got %#v", m)
	}
	if m.Message != nil {
		t.Fatalf("large content must not enter inline message")
	}
}
