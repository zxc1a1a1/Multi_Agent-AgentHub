package adk

import (
	"encoding/json"
	"testing"
)

func TestTextPart_ImplementsPart(t *testing.T) {
	var part Part = TextPart{Text: "hello"}
	textPart, ok := part.(TextPart)
	if !ok {
		t.Fatalf("expected TextPart, got %T", part)
	}
	if textPart.Text != "hello" {
		t.Fatalf("unexpected text: %q", textPart.Text)
	}
}

func TestToolCallPart_ImplementsPart(t *testing.T) {
	arguments := json.RawMessage(`{"language":"go","strict":true}`)
	var part Part = ToolCallPart{
		ID:        "call-1",
		Name:      "generate_code",
		Arguments: arguments,
	}

	toolCallPart, ok := part.(ToolCallPart)
	if !ok {
		t.Fatalf("expected ToolCallPart, got %T", part)
	}

	var decoded map[string]any
	if err := json.Unmarshal(toolCallPart.Arguments, &decoded); err != nil {
		t.Fatalf("failed to unmarshal arguments: %v", err)
	}
	if decoded["language"] != "go" {
		t.Fatalf("unexpected language: %#v", decoded["language"])
	}
	strictValue, ok := decoded["strict"].(bool)
	if !ok || !strictValue {
		t.Fatalf("unexpected strict value: %#v", decoded["strict"])
	}
}

func TestToolResultPart_ImplementsPart(t *testing.T) {
	var part Part = ToolResultPart{
		CallID:  "call-1",
		Name:    "generate_code",
		Content: "ok",
		IsError: false,
	}
	toolResultPart, ok := part.(ToolResultPart)
	if !ok {
		t.Fatalf("expected ToolResultPart, got %T", part)
	}
	if toolResultPart.CallID != "call-1" || toolResultPart.Name != "generate_code" {
		t.Fatalf("unexpected tool result identity: %+v", toolResultPart)
	}
}

func TestThinkingPart_ImplementsPart(t *testing.T) {
	var part Part = ThinkingPart{Thinking: "internal marker"}
	thinkingPart, ok := part.(ThinkingPart)
	if !ok {
		t.Fatalf("expected ThinkingPart, got %T", part)
	}
	if thinkingPart.Thinking == "" {
		t.Fatal("thinking must not be empty in this test")
	}
}

func TestContent_Construction(t *testing.T) {
	content := Content{
		Role: RoleAssistant,
		Parts: []Part{
			TextPart{Text: "reply"},
			ToolResultPart{CallID: "call-1", Name: "generate_code", Content: "done", IsError: false},
		},
	}

	if content.Role != RoleAssistant {
		t.Fatalf("unexpected role: %q", content.Role)
	}
	if len(content.Parts) != 2 {
		t.Fatalf("unexpected parts count: %d", len(content.Parts))
	}
}

func TestContent_PartOrderPreserved(t *testing.T) {
	content := Content{
		Role: RoleAssistant,
		Parts: []Part{
			TextPart{Text: "first"},
			ToolResultPart{CallID: "call-1", Name: "step", Content: "second", IsError: false},
			ThinkingPart{Thinking: "third"},
		},
	}

	first, ok := content.Parts[0].(TextPart)
	if !ok || first.Text != "first" {
		t.Fatalf("unexpected first part: %#v", content.Parts[0])
	}
	second, ok := content.Parts[1].(ToolResultPart)
	if !ok || second.Content != "second" {
		t.Fatalf("unexpected second part: %#v", content.Parts[1])
	}
	third, ok := content.Parts[2].(ThinkingPart)
	if !ok || third.Thinking != "third" {
		t.Fatalf("unexpected third part: %#v", content.Parts[2])
	}
}

func TestEvent_FinalFlag(t *testing.T) {
	event := Event{Final: true}
	if !event.Final {
		t.Fatal("expected event.Final to be true")
	}
}

func TestEvent_StateDelta(t *testing.T) {
	event := Event{
		Actions: &EventActions{
			StateDelta: map[string]any{
				"phase": "1.2",
				"done":  true,
			},
		},
	}
	if event.Actions == nil {
		t.Fatal("expected event actions")
	}
	if event.Actions.StateDelta["phase"] != "1.2" {
		t.Fatalf("unexpected phase value: %#v", event.Actions.StateDelta["phase"])
	}
	if event.Actions.StateDelta["done"] != true {
		t.Fatalf("unexpected done value: %#v", event.Actions.StateDelta["done"])
	}
}

func TestEvent_Author(t *testing.T) {
	event := Event{Author: "code-agent"}
	if event.Author != "code-agent" {
		t.Fatalf("unexpected author: %q", event.Author)
	}
}

func TestArtifact_Metadata(t *testing.T) {
	artifact := Artifact{
		Type:    "code",
		Title:   "main.go",
		Content: "package main",
		Metadata: map[string]string{
			"language": "go",
			"mimeType": "text/plain",
		},
	}
	if artifact.Type != "code" || artifact.Title != "main.go" {
		t.Fatalf("unexpected artifact identity: %+v", artifact)
	}
	if artifact.Content == "" {
		t.Fatal("artifact content must not be empty in this test")
	}
	if artifact.Metadata["language"] != "go" {
		t.Fatalf("unexpected metadata language: %q", artifact.Metadata["language"])
	}
}
