package agui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func TestTranslator_TextPart(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e1",
		Author: "agent-a",
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "hello"}},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "message" {
		t.Fatalf("unexpected type: %q", events[0].Type)
	}
	if events[0].Text != "hello" {
		t.Fatalf("unexpected text: %q", events[0].Text)
	}
	if events[0].Role != string(adk.RoleAssistant) {
		t.Fatalf("unexpected role: %q", events[0].Role)
	}
}

func TestTranslator_ToolCallPart(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e2",
		Author: "agent-a",
		Content: &adk.Content{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.ToolCallPart{
					ID:        "call-1",
					Name:      "get_skill",
					Arguments: json.RawMessage(`{"name":"alpha"}`),
				},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != "tool.call" || ev.ToolCall == nil {
		t.Fatalf("unexpected event: %#v", ev)
	}
	if ev.ToolCall.ID != "call-1" || ev.ToolCall.Name != "get_skill" {
		t.Fatalf("unexpected tool call: %#v", ev.ToolCall)
	}
}

func TestTranslator_ToolResultPart(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e3",
		Author: "agent-a",
		Content: &adk.Content{
			Role: adk.RoleTool,
			Parts: []adk.Part{
				adk.ToolResultPart{
					CallID:  "call-1",
					Name:    "get_skill",
					Content: "ok",
					IsError: false,
				},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != "tool.result" || ev.ToolResult == nil {
		t.Fatalf("unexpected event: %#v", ev)
	}
	if ev.ToolResult.Content != "ok" {
		t.Fatalf("unexpected tool result content: %q", ev.ToolResult.Content)
	}
}

func TestTranslator_ThinkingPartRedacted(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e4",
		Author: "agent-a",
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.ThinkingPart{Thinking: "internal secret"}},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "thinking" {
		t.Fatalf("unexpected type: %q", events[0].Type)
	}
	if events[0].Text != "" {
		t.Fatalf("thinking content should not be exposed: %q", events[0].Text)
	}
}

func TestTranslator_StateDelta(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e5",
		Author: "agent-a",
		Actions: &adk.EventActions{
			StateDelta: map[string]any{"step": "running"},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "state.delta" {
		t.Fatalf("unexpected type: %q", events[0].Type)
	}
	if events[0].StateDelta["step"] != "running" {
		t.Fatalf("unexpected state delta: %#v", events[0].StateDelta)
	}
}

func TestTranslator_ArtifactDelta(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e6",
		Author: "agent-a",
		Actions: &adk.EventActions{
			ArtifactDelta: []adk.Artifact{
				{
					Type:     "code",
					Title:    "main.go",
					Content:  "package main",
					Metadata: map[string]string{"language": "go"},
				},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != "artifact.delta" || ev.Artifact == nil {
		t.Fatalf("unexpected event: %#v", ev)
	}
	if ev.Artifact.Type != "code" {
		t.Fatalf("unexpected artifact type: %#v", ev.Artifact)
	}
}

func TestTranslator_Final(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e7",
		Author: "agent-a",
		Final:  true,
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "done"}},
		},
	})

	if len(events) != 2 {
		t.Fatalf("expected message + message.end, got %d", len(events))
	}
	if events[1].Type != "message.end" || !events[1].Final {
		t.Fatalf("unexpected final event: %#v", events[1])
	}
}

func TestTranslator_Partial(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:      "e8",
		Author:  "agent-a",
		Partial: true,
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "delta"}},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "message.delta" || !events[0].Partial {
		t.Fatalf("unexpected partial event: %#v", events[0])
	}
}

func TestTranslator_MultipleParts(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e9",
		Author: "agent-a",
		Content: &adk.Content{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.TextPart{Text: "hello"},
				adk.ToolCallPart{
					ID:        "call-1",
					Name:      "list_skills",
					Arguments: json.RawMessage(`{}`),
				},
				adk.ToolResultPart{CallID: "call-1", Name: "list_skills", Content: "ok"},
			},
		},
	})

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Type != "message" || events[1].Type != "tool.call" || events[2].Type != "tool.result" {
		t.Fatalf("unexpected event sequence: %#v", events)
	}
}

func TestTranslator_UsesTextStreamFilter(t *testing.T) {
	filter := NewTextStreamFilter(WithMaxTextChars(64))
	tr := NewTranslator(WithTextStreamFilter(filter))
	events := tr.Translate(adk.Event{
		ID:     "e10",
		Author: "agent-a",
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "OPENAI_API_KEY=abcdef123456"}},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if strings.Contains(events[0].Text, "abcdef123456") {
		t.Fatalf("sensitive text leaked: %q", events[0].Text)
	}
	if !strings.Contains(events[0].Text, "[redacted]") {
		t.Fatalf("expected redacted text, got %q", events[0].Text)
	}
}

func TestTranslator_InvalidToolArguments(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		ID:     "e11",
		Author: "agent-a",
		Content: &adk.Content{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.ToolCallPart{
					ID:        "call-1",
					Name:      "bad_args",
					Arguments: json.RawMessage(`{"name":`),
				},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "tool.call" || events[0].ToolCall == nil {
		t.Fatalf("unexpected event: %#v", events[0])
	}
	if _, ok := events[0].ToolCall.Arguments.(string); !ok {
		t.Fatalf("invalid json args should fallback to raw string, got %T", events[0].ToolCall.Arguments)
	}
}

func TestTranslator_NilContent(t *testing.T) {
	tr := NewTranslator()

	actionsOnly := tr.Translate(adk.Event{
		ID:     "e12",
		Author: "agent-a",
		Actions: &adk.EventActions{
			StateDelta: map[string]any{"k": "v"},
		},
	})
	if len(actionsOnly) != 1 || actionsOnly[0].Type != "state.delta" {
		t.Fatalf("expected one state.delta event, got %#v", actionsOnly)
	}

	empty := tr.Translate(adk.Event{ID: "e13", Author: "agent-a"})
	if len(empty) != 0 {
		t.Fatalf("expected empty events for nil content and nil actions, got %#v", empty)
	}
}
