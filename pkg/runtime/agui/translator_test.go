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

// ── AG-UI v1.0 metadata-driven tests ──

func TestTranslator_V1RunStarted(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType: "run_started",
			MetaRunID:     "run-001",
		},
		Actions: &adk.EventActions{
			StateDelta: map[string]any{"phase": "executing", "planId": "plan-1"},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "RUN_STARTED" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.RunID != "run-001" {
		t.Fatalf("runId not preserved: %q", e.RunID)
	}
	if e.State == nil || e.State["phase"] != "executing" {
		t.Fatalf("state not preserved: %#v", e.State)
	}
	if e.StateDelta == nil {
		t.Fatalf("stateDelta backward compat not set")
	}
}

func TestTranslator_V1RunFinished(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType: "run_finished",
			MetaRunID:     "run-001",
		},
		Final: true,
		Actions: &adk.EventActions{
			StateDelta: map[string]any{"status": "completed", "taskCount": float64(2)},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "RUN_FINISHED" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.RunID != "run-001" {
		t.Fatalf("runId not preserved: %q", e.RunID)
	}
	if !e.Final {
		t.Fatalf("expected final=true")
	}
}

func TestTranslator_V1RunError(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType: "run_error",
			MetaRunID:     "run-001",
		},
		Final: true,
		Actions: &adk.EventActions{
			StateDelta: map[string]any{
				"code":    "ORCHESTRATOR_INTERNAL",
				"message": "something went wrong",
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "RUN_ERROR" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.RunID != "run-001" {
		t.Fatalf("runId not preserved: %q", e.RunID)
	}
	if e.Error == nil {
		t.Fatal("expected error object")
	}
	if e.Error.Code != "ORCHESTRATOR_INTERNAL" {
		t.Fatalf("unexpected error code: %q", e.Error.Code)
	}
}

func TestTranslator_V1MessageStart(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "web-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_start",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-0",
			MetaTaskID:     "task_web",
			MetaSenderType: "agent",
			MetaSenderName: "web-agent",
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "TEXT_MESSAGE_START" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.RunID != "run-001" {
		t.Fatalf("runId not preserved: %q", e.RunID)
	}
	if e.MessageID != "msg-0" {
		t.Fatalf("messageId not preserved: %q", e.MessageID)
	}
	if e.TaskID != "task_web" {
		t.Fatalf("taskId not preserved: %q", e.TaskID)
	}
	if e.Sender == nil || e.Sender.Name != "web-agent" || e.Sender.Type != "agent" {
		t.Fatalf("sender not preserved: %#v", e.Sender)
	}
	if e.Author != "web-agent" {
		t.Fatalf("author backward compat not set: %q", e.Author)
	}
}

func TestTranslator_V1MessageDelta(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "web-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_delta",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-0",
			MetaSenderType: "agent",
			MetaSenderName: "web-agent",
		},
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "hello from web"}},
		},
		Partial: true,
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "TEXT_MESSAGE_CONTENT" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.Delta != "hello from web" {
		t.Fatalf("delta not set: %q", e.Delta)
	}
	if e.Content != "hello from web" {
		t.Fatalf("content backward compat not set: %q", e.Content)
	}
	if e.Text != "hello from web" {
		t.Fatalf("text backward compat not set: %q", e.Text)
	}
	if e.RunID != "run-001" {
		t.Fatalf("runId not preserved: %q", e.RunID)
	}
	if e.MessageID != "msg-0" {
		t.Fatalf("messageId not preserved: %q", e.MessageID)
	}
}

func TestTranslator_V1MessageEnd(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "web-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_end",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-0",
			MetaSenderType: "agent",
			MetaSenderName: "web-agent",
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "TEXT_MESSAGE_END" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.MessageID != "msg-0" {
		t.Fatalf("messageId not preserved: %q", e.MessageID)
	}
	if e.RunID != "run-001" {
		t.Fatalf("runId not preserved: %q", e.RunID)
	}
}

func TestTranslator_V1StateUpdate(t *testing.T) {
	tr := NewTranslator()
	events := tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType: "state_update",
			MetaRunID:     "run-001",
		},
		Actions: &adk.EventActions{
			StateDelta: map[string]any{
				"phase":   "dispatching",
				"message": "dispatching to 2 agents",
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "STATE_UPDATE" {
		t.Fatalf("unexpected type: %q", e.Type)
	}
	if e.State == nil || e.State["phase"] != "dispatching" {
		t.Fatalf("state not preserved: %#v", e.State)
	}
	if e.StateDelta == nil {
		t.Fatalf("stateDelta backward compat not set")
	}
}

func TestTranslator_V1MultiAgentOrderedParallel(t *testing.T) {
	tr := NewTranslator()

	// Simulate ordered_parallel: web-agent message, code-agent message, orchestrator summary
	events := make([]Event, 0)

	// Web-agent message_start
	events = append(events, tr.Translate(adk.Event{
		Author: "web-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_start",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-0",
			MetaSenderType: "agent",
			MetaSenderName: "web-agent",
		},
	})...)

	// Web-agent message_delta
	events = append(events, tr.Translate(adk.Event{
		Author: "web-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_delta",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-0",
			MetaSenderType: "agent",
			MetaSenderName: "web-agent",
		},
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "web output"}},
		},
	})...)

	// Web-agent message_end
	events = append(events, tr.Translate(adk.Event{
		Author: "web-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_end",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-0",
			MetaSenderType: "agent",
			MetaSenderName: "web-agent",
		},
	})...)

	// Code-agent message_start (different messageId!)
	events = append(events, tr.Translate(adk.Event{
		Author: "code-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_start",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-1",
			MetaSenderType: "agent",
			MetaSenderName: "code-agent",
		},
	})...)

	// Code-agent message_delta
	events = append(events, tr.Translate(adk.Event{
		Author: "code-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_delta",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-1",
			MetaSenderType: "agent",
			MetaSenderName: "code-agent",
		},
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "code output"}},
		},
	})...)

	// Code-agent message_end
	events = append(events, tr.Translate(adk.Event{
		Author: "code-agent",
		Metadata: map[string]any{
			MetaEventType:  "message_end",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-1",
			MetaSenderType: "agent",
			MetaSenderName: "code-agent",
		},
	})...)

	// Orchestrator summary
	events = append(events, tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType:  "message_start",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-2",
			MetaSenderType: "orchestrator",
			MetaSenderName: "orchestrator",
		},
	})...)
	events = append(events, tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType:  "message_delta",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-2",
			MetaSenderType: "orchestrator",
			MetaSenderName: "orchestrator",
		},
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "All tasks completed"}},
		},
	})...)
	events = append(events, tr.Translate(adk.Event{
		Author: "orchestrator",
		Metadata: map[string]any{
			MetaEventType:  "message_end",
			MetaRunID:      "run-001",
			MetaMessageID:  "msg-2",
			MetaSenderType: "orchestrator",
			MetaSenderName: "orchestrator",
		},
	})...)

	// Count message_start, content, end events
	msgStarts := 0
	msgContents := 0
	msgEnds := 0
	senders := make(map[string]bool)
	messageIDs := make(map[string]bool)

	for _, e := range events {
		switch e.Type {
		case "TEXT_MESSAGE_START":
			msgStarts++
			if e.Sender != nil {
				senders[e.Sender.Name] = true
			}
			messageIDs[e.MessageID] = true
		case "TEXT_MESSAGE_CONTENT":
			msgContents++
		case "TEXT_MESSAGE_END":
			msgEnds++
		}
	}

	if msgStarts != 3 {
		t.Fatalf("expected 3 TEXT_MESSAGE_START events, got %d", msgStarts)
	}
	if msgContents != 3 {
		t.Fatalf("expected 3 TEXT_MESSAGE_CONTENT events, got %d", msgContents)
	}
	if msgEnds != 3 {
		t.Fatalf("expected 3 TEXT_MESSAGE_END events, got %d", msgEnds)
	}
	if len(senders) != 3 {
		t.Fatalf("expected 3 distinct senders, got %d: %v", len(senders), senders)
	}
	if !senders["web-agent"] || !senders["code-agent"] || !senders["orchestrator"] {
		t.Fatalf("missing expected senders: %v", senders)
	}
	if len(messageIDs) != 3 {
		t.Fatalf("expected 3 distinct messageIds, got %d: %v", len(messageIDs), messageIDs)
	}
}

func TestTranslator_LegacyFallbackWithoutMetadata(t *testing.T) {
	tr := NewTranslator()
	// Without metadata, should still produce legacy event types
	events := tr.Translate(adk.Event{
		ID:     "e1",
		Author: "agent-a",
		Content: &adk.Content{
			Role:  adk.RoleAssistant,
			Parts: []adk.Part{adk.TextPart{Text: "legacy text"}},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "message" {
		t.Fatalf("legacy fallback: expected 'message' type, got %q", events[0].Type)
	}
	if events[0].Text != "legacy text" {
		t.Fatalf("legacy fallback: text not preserved")
	}
	// Legacy events should have backward-compat fields
	if events[0].Delta == "" {
		t.Fatalf("legacy fallback: delta should be set for forward compat")
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
