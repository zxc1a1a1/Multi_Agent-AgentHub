package orchestrator

import (
	"encoding/json"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func statusEvent(state a2a.TaskState) *a2a.TaskStatusUpdateEvent {
	return &a2a.TaskStatusUpdateEvent{
		Status: a2a.TaskStatus{State: state},
	}
}

func artifactEvent(typ, content, language, filename string) *a2a.TaskArtifactUpdateEvent {
	meta := map[string]any{
		"type": typ,
	}
	if language != "" {
		meta["language"] = language
	}
	if filename != "" {
		meta["filename"] = filename
	}
	return &a2a.TaskArtifactUpdateEvent{
		Artifact: &a2a.Artifact{
			Name:     filename,
			Metadata: meta,
			Parts: a2a.ContentParts{
				{Content: a2a.Text(content)},
			},
		},
	}
}

func messageEvent(parts ...string) *a2a.Message {
	contentParts := make(a2a.ContentParts, 0, len(parts))
	for _, p := range parts {
		contentParts = append(contentParts, &a2a.Part{Content: a2a.Text(p)})
	}
	return &a2a.Message{
		Parts: contentParts,
	}
}

func TestConverterWorkingTextAndCompleted(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	startEvents := c.ConvertA2AEvent(statusEvent(a2a.TaskStateWorking))
	if len(startEvents) != 1 {
		t.Fatalf("expected 1 start event, got %d", len(startEvents))
	}
	if startEvents[0].Type != "TEXT_MESSAGE_START" {
		t.Fatalf("expected TEXT_MESSAGE_START, got %s", startEvents[0].Type)
	}
	if startEvents[0].MessageID == "" {
		t.Fatalf("expected non-empty messageID")
	}

	textEvents := c.ConvertA2AEvent(artifactEvent("text", "hello", "", ""))
	if len(textEvents) != 1 {
		t.Fatalf("expected 1 text event, got %d", len(textEvents))
	}
	if textEvents[0].Type != "TEXT_MESSAGE_CONTENT" {
		t.Fatalf("expected TEXT_MESSAGE_CONTENT, got %s", textEvents[0].Type)
	}
	if textEvents[0].Content != "hello" {
		t.Fatalf("expected content hello, got %q", textEvents[0].Content)
	}

	doneEvents := c.ConvertA2AEvent(statusEvent(a2a.TaskStateCompleted))
	if len(doneEvents) != 2 {
		t.Fatalf("expected 2 completion events without code artifact, got %d", len(doneEvents))
	}
	if doneEvents[0].Type != "TEXT_MESSAGE_END" {
		t.Fatalf("expected TEXT_MESSAGE_END, got %s", doneEvents[0].Type)
	}
	if doneEvents[1].Type != "RUN_FINISHED" {
		t.Fatalf("expected RUN_FINISHED, got %s", doneEvents[1].Type)
	}
}

func TestConverterCodeArtifactFlushesToolCallOnCompleted(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	_ = c.ConvertA2AEvent(statusEvent(a2a.TaskStateWorking))

	codeUpdate := artifactEvent("code", "package main", "go", "main.go")
	bufferEvents := c.ConvertA2AEvent(codeUpdate)
	if len(bufferEvents) != 0 {
		t.Fatalf("expected code artifact to be buffered, got %d events", len(bufferEvents))
	}

	events := c.ConvertA2AEvent(statusEvent(a2a.TaskStateCompleted))
	if len(events) != 5 {
		t.Fatalf("expected 5 events on completion with one code artifact, got %d", len(events))
	}

	if events[0].Type != "TEXT_MESSAGE_END" {
		t.Fatalf("expected first event TEXT_MESSAGE_END, got %s", events[0].Type)
	}
	if events[1].Type != "TOOL_CALL_START" {
		t.Fatalf("expected TOOL_CALL_START, got %s", events[1].Type)
	}
	if events[1].ToolName != "code_preview" {
		t.Fatalf("expected tool name code_preview, got %q", events[1].ToolName)
	}
	if events[1].ToolCallID == "" {
		t.Fatalf("expected non-empty toolCallID")
	}
	if events[2].Type != "TOOL_CALL_ARGS" {
		t.Fatalf("expected TOOL_CALL_ARGS, got %s", events[2].Type)
	}
	if events[2].ToolCallID != events[1].ToolCallID {
		t.Fatalf("expected same toolCallID across start/args")
	}
	if events[3].Type != "TOOL_CALL_END" {
		t.Fatalf("expected TOOL_CALL_END, got %s", events[3].Type)
	}
	if events[3].ToolCallID != events[1].ToolCallID {
		t.Fatalf("expected same toolCallID across start/end")
	}
	if events[4].Type != "RUN_FINISHED" {
		t.Fatalf("expected RUN_FINISHED, got %s", events[4].Type)
	}

	var args map[string]string
	if err := json.Unmarshal([]byte(events[2].Content), &args); err != nil {
		t.Fatalf("failed to unmarshal TOOL_CALL_ARGS content: %v", err)
	}
	if args["code"] != "package main" {
		t.Fatalf("expected args.code package main, got %q", args["code"])
	}
	if args["language"] != "go" {
		t.Fatalf("expected args.language go, got %q", args["language"])
	}
	if args["filename"] != "main.go" {
		t.Fatalf("expected args.filename main.go, got %q", args["filename"])
	}
}

func TestConverterFailedAndCanceledToRunError(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	failedEvents := c.ConvertA2AEvent(statusEvent(a2a.TaskStateFailed))
	if len(failedEvents) != 1 || failedEvents[0].Type != "RUN_ERROR" {
		t.Fatalf("failed state should map to RUN_ERROR, got %#v", failedEvents)
	}

	cancelEvents := c.ConvertA2AEvent(statusEvent(a2a.TaskStateCanceled))
	if len(cancelEvents) != 1 || cancelEvents[0].Type != "RUN_ERROR" {
		t.Fatalf("canceled state should map to RUN_ERROR, got %#v", cancelEvents)
	}
}

func TestConverterNonCodeArtifactDoesNotGenerateToolCall(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	_ = c.ConvertA2AEvent(statusEvent(a2a.TaskStateWorking))
	_ = c.ConvertA2AEvent(artifactEvent("text", "plain text", "", ""))
	events := c.ConvertA2AEvent(statusEvent(a2a.TaskStateCompleted))

	for _, e := range events {
		if e.Type == "TOOL_CALL_START" || e.Type == "TOOL_CALL_ARGS" || e.Type == "TOOL_CALL_END" {
			t.Fatalf("did not expect tool call events for non-code artifact, got %s", e.Type)
		}
	}
}

func TestConverterNoCodePreviewSkillSkipsToolCallCurrentBehavior(t *testing.T) {
	c := NewConverter(nil)

	_ = c.ConvertA2AEvent(statusEvent(a2a.TaskStateWorking))
	_ = c.ConvertA2AEvent(artifactEvent("code", "print('x')", "python", "main.py"))
	events := c.ConvertA2AEvent(statusEvent(a2a.TaskStateCompleted))

	if len(events) != 2 {
		t.Fatalf("expected only TEXT_MESSAGE_END and RUN_FINISHED, got %d events", len(events))
	}
	if events[0].Type != "TEXT_MESSAGE_END" || events[1].Type != "RUN_FINISHED" {
		t.Fatalf("unexpected events without code_preview skill: %#v", events)
	}
}

func TestConverterNilEventReturnsNil(t *testing.T) {
	c := NewConverter([]string{"code_preview"})
	events := c.ConvertA2AEvent(nil)
	if events != nil {
		t.Fatalf("expected nil events for nil input, got %#v", events)
	}
}

func TestConverterTaskStateInputRequiredCurrentBehavior(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	events := c.ConvertA2AEvent(statusEvent(a2a.TaskStateInputRequired))
	if events != nil {
		t.Fatalf("expected nil events for TASK_STATE_INPUT_REQUIRED current behavior, got %#v", events)
	}
}

func TestConverterMessageBeforeStartCurrentBehavior(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	events := c.ConvertA2AEvent(messageEvent("text before start"))
	if events != nil {
		t.Fatalf("expected nil events when message arrives before start, got %#v", events)
	}
}

func TestConverterMultiTextPartCurrentBehaviorKeepsLastPart(t *testing.T) {
	c := NewConverter([]string{"code_preview"})
	_ = c.ConvertA2AEvent(statusEvent(a2a.TaskStateWorking))

	e := &a2a.TaskArtifactUpdateEvent{
		Artifact: &a2a.Artifact{
			Metadata: map[string]any{"type": "text"},
			Parts: a2a.ContentParts{
				{Content: a2a.Text("first")},
				{Content: a2a.Text("second")},
			},
		},
	}

	events := c.ConvertA2AEvent(e)
	if len(events) != 1 {
		t.Fatalf("expected one text content event, got %d", len(events))
	}
	if events[0].Type != "TEXT_MESSAGE_CONTENT" {
		t.Fatalf("expected TEXT_MESSAGE_CONTENT, got %s", events[0].Type)
	}
	// Lock current behavior: only the last text part is kept by converter.
	if events[0].Content != "second" {
		t.Fatalf("expected current behavior to keep last part(second), got %q", events[0].Content)
	}
}

func TestConverterUnsupportedTaskStateReturnsNil(t *testing.T) {
	c := NewConverter([]string{"code_preview"})

	events := c.ConvertA2AEvent(statusEvent(a2a.TaskStateSubmitted))
	if events != nil {
		t.Fatalf("expected nil for unsupported task state mapping, got %#v", events)
	}
}
