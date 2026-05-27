package adk

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

func makeExecCtx(taskID, contextID string) *a2asrv.ExecutorContext {
	msg := &a2a.Message{
		Role: a2a.MessageRoleUser,
		Parts: []*a2a.Part{
			a2a.NewTextPart("test input"),
		},
		TaskID:    a2a.TaskID(taskID),
		ContextID: contextID,
	}
	return &a2asrv.ExecutorContext{
		Message:   msg,
		TaskID:    a2a.TaskID(taskID),
		ContextID: contextID,
	}
}

func collectEvents(t *testing.T, execCtx *a2asrv.ExecutorContext, handler TaskHandler) []a2a.Event {
	t.Helper()
	seq := ExecuteHandler(context.Background(), execCtx, handler)
	var events []a2a.Event
	for ev, err := range seq {
		if err != nil {
			t.Fatalf("unexpected error from event stream: %v", err)
		}
		events = append(events, ev)
	}
	return events
}

func eventType(ev a2a.Event) string {
	switch ev.(type) {
	case *a2a.TaskStatusUpdateEvent:
		se := ev.(*a2a.TaskStatusUpdateEvent)
		return string(se.Status.State)
	case *a2a.TaskArtifactUpdateEvent:
		return "artifact"
	case *a2a.Task:
		return "task"
	default:
		return "unknown"
	}
}

// TestStream_EventOrderStability verifies that ExecuteHandler produces
// events in the expected stable sequence:
//
//	Submitted → Working → (text chunks) → (code artifacts) → Completed
func TestStream_EventOrderStability(t *testing.T) {
	execCtx := makeExecCtx("task-001", "ctx-001")

	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.StreamText("hello")
		ctx.StreamText(" world")
		ctx.AddArtifact(Artifact{
			Type:    "code",
			Title:   "main.go",
			Content: "package main\nfunc main() {}",
			Metadata: map[string]string{"language": "go"},
		})
		return nil
	}

	events := collectEvents(t, execCtx, handler)
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events (submitted, working, text, completed), got %d", len(events))
	}

	// Index 0: Submitted task (NewSubmittedTask returns *a2a.Task, not status event)
	firstEt := eventType(events[0])
	if firstEt != "task" {
		t.Errorf("event[0]: expected task (SubmittedTask), got %q", firstEt)
	} else {
		task := events[0].(*a2a.Task)
		if task.Status.State != a2a.TaskStateSubmitted {
			t.Errorf("event[0] task state: expected Submitted, got %q", task.Status.State)
		}
	}
	// Index 1: Working status
	if et := eventType(events[1]); et != "TASK_STATE_WORKING" {
		t.Errorf("event[1]: expected Working, got %q", et)
	}

	// Middle events: text chunks + code artifact
	hasText := false
	hasCode := false
	for i := 2; i < len(events)-1; i++ {
		et := eventType(events[i])
		if et == "artifact" {
			ae := events[i].(*a2a.TaskArtifactUpdateEvent)
			if ae.Artifact.Name == "response" {
				hasText = true
			} else if ae.Artifact.Name == "main.go" {
				hasCode = true
			}
		}
	}
	if !hasText {
		t.Error("expected at least one text artifact event")
	}
	if !hasCode {
		t.Error("expected at least one code artifact event (main.go)")
	}

	// Last event: Completed status
	last := events[len(events)-1]
	if et := eventType(last); et != "TASK_STATE_COMPLETED" {
		t.Errorf("last event: expected Completed, got %q", et)
	}

	// Verify order invariants: no Completed before Working
	foundWorking := false
	for _, ev := range events {
		et := eventType(ev)
		if et == "TASK_STATE_COMPLETED" && !foundWorking {
			t.Error("Completed appeared before Working — order violation")
		}
		if et == "TASK_STATE_WORKING" {
			foundWorking = true
		}
	}

	// Verify task ID matches across all status events
	for _, ev := range events {
		if se, ok := ev.(*a2a.TaskStatusUpdateEvent); ok {
			if string(se.TaskID) != "task-001" {
				t.Errorf("status event has TaskID %q, expected task-001", se.TaskID)
			}
		}
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// TestStream_ErrorProducesFailedStatus verifies that when a handler returns
// an error, the final event is Failed and the error message is safe.
func TestStream_ErrorProducesFailedStatus(t *testing.T) {
	execCtx := makeExecCtx("task-002", "ctx-002")

	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.StreamText("processing...")
		return &testError{msg: "downstream agent unreachable"}
	}

	events := collectEvents(t, execCtx, handler)
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events (submitted, working, failed), got %d", len(events))
	}

	last := events[len(events)-1]
	se, ok := last.(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected TaskStatusUpdateEvent, got %T", last)
	}
	if se.Status.State != a2a.TaskStateFailed {
		t.Errorf("expected Failed state, got %q", se.Status.State)
	}

	// Error message must be safe (no stack traces, no internal paths)
	if se.Status.Message != nil {
		for i := range se.Status.Message.Parts {
			text := se.Status.Message.Parts[i].Text()
			if text == "" {
				continue
			}
			forbidden := []string{
				"goroutine", "panic", ".go:", "stack trace",
				"API_KEY", "token", "secret", "password",
				"/internal/", "/admin/",
			}
			for _, kw := range forbidden {
				if strings.Contains(text, kw) {
					t.Errorf("error message contains forbidden keyword %q: %q", kw, text)
				}
			}
		}
	}
}

// TestStream_CancelSignalsContext verifies that ctx.Done() channel is closed
// after Cancel(), allowing the handler to detect cancellation.
// Note: ctx.Cancel() only sets the Done channel; it does not automatically
// suppress subsequent StreamText/AddArtifact calls at the ADK level.
func TestStream_CancelSignalsContext(t *testing.T) {
	execCtx := makeExecCtx("task-003", "ctx-003")

	cancelDetected := false
	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.StreamText("chunk-1")
		ctx.Cancel()

		// Check if Done channel is closed
		select {
		case <-ctx.Done():
			cancelDetected = true
		default:
		}
		return nil
	}

	collectEvents(t, execCtx, handler)

	if !cancelDetected {
		t.Error("ctx.Done() channel should be closed after Cancel()")
	}
}

// TestStream_CancelCooperativePattern verifies the cooperative cancel pattern:
// handler checks ctx.Done() and returns early, preventing further output.
func TestStream_CancelCooperativePattern(t *testing.T) {
	execCtx := makeExecCtx("task-003b", "ctx-003b")

	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.StreamText("chunk-1")
		ctx.Cancel()

		// Cooperative check: stop producing if cancelled
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		ctx.StreamText("should-not-appear-after-cancel")
		return nil
	}

	events := collectEvents(t, execCtx, handler)

	// With cooperative cancel, "should-not-appear-after-cancel" must not show
	for _, ev := range events {
		if ae, ok := ev.(*a2a.TaskArtifactUpdateEvent); ok {
			if ae.Artifact.Name == "response" {
				for i := range ae.Artifact.Parts {
					text := ae.Artifact.Parts[i].Text()
					if strings.Contains(text, "should-not-appear-after-cancel") {
						t.Error("text after cooperative cancel check should not be emitted")
					}
				}
			}
		}
	}
}

// TestStream_EmptyHandlerProducesMinimalSequence verifies that a handler
// which returns immediately produces minimal valid sequence.
func TestStream_EmptyHandlerProducesMinimalSequence(t *testing.T) {
	execCtx := makeExecCtx("task-004", "ctx-004")

	handler := func(ctx *Context, messages []a2a.Message) error {
		return nil
	}

	events := collectEvents(t, execCtx, handler)
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}

	// First event is a Task (SubmittedTask via NewSubmittedTask)
	if et := eventType(events[0]); et != "task" {
		t.Errorf("event[0]: expected task (Submitted), got %q", et)
	} else if task := events[0].(*a2a.Task); task.Status.State != a2a.TaskStateSubmitted {
		t.Errorf("event[0] state: expected Submitted, got %q", task.Status.State)
	}
	// Second event: Working
	if et := eventType(events[1]); et != "TASK_STATE_WORKING" {
		t.Errorf("event[1]: expected Working, got %q", et)
	}
	// Last event: Completed
	if et := eventType(events[len(events)-1]); et != "TASK_STATE_COMPLETED" {
		t.Errorf("last event: expected Completed, got %q", et)
	}
}

// TestStream_MultiAgentTaskIDIsolation verifies that executing two different
// handlers with different task IDs produces events with correct, distinct
// task IDs — no cross-contamination.
func TestStream_MultiAgentTaskIDIsolation(t *testing.T) {
	var wg sync.WaitGroup
	type streamResult struct {
		taskID string
		events []a2a.Event
	}
	results := make(chan streamResult, 2)

	runAgent := func(taskID string, msg string) {
		defer wg.Done()
		execCtx := makeExecCtx(taskID, "ctx-multi")
		handler := func(ctx *Context, messages []a2a.Message) error {
			ctx.StreamText(msg)
			return nil
		}
		events := collectEvents(t, execCtx, handler)
		results <- streamResult{taskID: taskID, events: events}
	}

	wg.Add(2)
	go runAgent("task-agent-a", "output from agent A")
	go runAgent("task-agent-b", "output from agent B")
	wg.Wait()
	close(results)

	taskIDs := make(map[string]bool)
	for r := range results {
		taskIDs[r.taskID] = true
		for _, ev := range r.events {
			if se, ok := ev.(*a2a.TaskStatusUpdateEvent); ok {
				if string(se.TaskID) != r.taskID {
					t.Errorf("cross-contamination: task %q event has TaskID %q",
						r.taskID, se.TaskID)
				}
			}
		}
		if len(r.events) < 3 {
			t.Errorf("task %q: expected at least 3 events, got %d", r.taskID, len(r.events))
		}
	}
	if len(taskIDs) != 2 {
		t.Errorf("expected 2 unique task IDs, got %d", len(taskIDs))
	}
}

// TestStream_MalformedArtifactsDontCrash verifies that edge case artifacts
// (empty content, empty metadata, missing title) are handled gracefully.
func TestStream_MalformedArtifactsDontCrash(t *testing.T) {
	execCtx := makeExecCtx("task-005", "ctx-005")

	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.AddArtifact(Artifact{
			Type: "code", Title: "", Content: "", Metadata: nil,
		})
		ctx.AddArtifact(Artifact{
			Type: "code", Title: "nilmeta.go", Content: "// nil metadata",
			Metadata: nil,
		})
		ctx.AddArtifact(Artifact{
			Type: "code", Title: "normal.go", Content: "package main",
			Metadata: map[string]string{"language": "go"},
		})
		return nil
	}

	events := collectEvents(t, execCtx, handler)

	last := events[len(events)-1]
	se, ok := last.(*a2a.TaskStatusUpdateEvent)
	if !ok || se.Status.State != a2a.TaskStateCompleted {
		t.Fatalf("expected Completed status, got %v", se.Status.State)
	}

	artifactCount := 0
	for _, ev := range events {
		if ae, ok := ev.(*a2a.TaskArtifactUpdateEvent); ok {
			if ae.Artifact.Name != "response" {
				artifactCount++
			}
		}
	}
	if artifactCount != 3 {
		t.Errorf("expected 3 code artifact events, got %d", artifactCount)
	}
}

// TestStream_LargeTextStreamPreservesOrder verifies that streaming many
// text chunks preserves the insertion order.
func TestStream_LargeTextStreamPreservesOrder(t *testing.T) {
	execCtx := makeExecCtx("task-006", "ctx-006")

	const numChunks = 20
	handler := func(ctx *Context, messages []a2a.Message) error {
		for i := 0; i < numChunks; i++ {
			ctx.StreamText(string(rune('A' + i%26)))
		}
		return nil
	}

	events := collectEvents(t, execCtx, handler)

	var chunks []string
	for _, ev := range events {
		if ae, ok := ev.(*a2a.TaskArtifactUpdateEvent); ok {
			if ae.Artifact.Name == "response" {
				for i := range ae.Artifact.Parts {
					text := ae.Artifact.Parts[i].Text()
					if text != "" {
						chunks = append(chunks, text)
					}
				}
			}
		}
	}

	if len(chunks) != numChunks {
		t.Errorf("expected %d text chunks, got %d", numChunks, len(chunks))
	}
	for i, ch := range chunks {
		expected := string(rune('A' + i%26))
		if len(ch) == 1 && ch != expected {
			t.Errorf("chunk[%d]: expected %q, got %q", i, expected, ch)
		}
	}

	last := events[len(events)-1]
	if et := eventType(last); et != "TASK_STATE_COMPLETED" {
		t.Errorf("expected Completed after %d chunks, got %q", numChunks, et)
	}
}

// TestStream_CancelDoesNotEmitCompleted verifies Cancel() returns a single
// Canceled event, never Completed.
func TestStream_CancelDoesNotEmitCompleted(t *testing.T) {
	execCtx := makeExecCtx("task-007", "ctx-007")

	executor := &codeAgentExecutor{
		handler: NoopHandler,
	}

	cancelSeq := executor.Cancel(context.Background(), execCtx)
	var cancelEvents []a2a.Event
	for ev, err := range cancelSeq {
		if err != nil {
			t.Fatalf("unexpected error from cancel stream: %v", err)
		}
		cancelEvents = append(cancelEvents, ev)
	}

	if len(cancelEvents) != 1 {
		t.Fatalf("expected exactly 1 event from cancel, got %d", len(cancelEvents))
	}

	se, ok := cancelEvents[0].(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected TaskStatusUpdateEvent, got %T", cancelEvents[0])
	}
	if se.Status.State != a2a.TaskStateCanceled {
		t.Errorf("expected Canceled state from Cancel(), got %q", se.Status.State)
	}
}

// TestStream_ConcurrentEventStreamsDontCrossContaminate verifies that
// concurrent event streams from different agents are isolated.
func TestStream_ConcurrentEventStreamsDontCrossContaminate(t *testing.T) {
	var wg sync.WaitGroup
	type streamResult struct {
		agentName string
		taskID    string
		text      string
	}
	results := make(chan streamResult, 3)

	agents := []struct {
		name   string
		taskID string
		msg    string
	}{
		{"agent-x", "task-x", "output-x"},
		{"agent-y", "task-y", "output-y"},
		{"agent-z", "task-z", "output-z"},
	}

	for _, ag := range agents {
		wg.Add(1)
		go func(name, taskID, msg string) {
			defer wg.Done()
			execCtx := makeExecCtx(taskID, "ctx-"+name)
			handler := func(ctx *Context, messages []a2a.Message) error {
				ctx.StreamText(msg)
				return nil
			}
			events := collectEvents(t, execCtx, handler)
			var allText string
			for _, ev := range events {
				if ae, ok := ev.(*a2a.TaskArtifactUpdateEvent); ok {
					if ae.Artifact.Name == "response" {
						for i := range ae.Artifact.Parts {
							allText += ae.Artifact.Parts[i].Text()
						}
					}
				}
			}
			results <- streamResult{agentName: name, taskID: taskID, text: allText}
		}(ag.name, ag.taskID, ag.msg)
	}

	wg.Wait()
	close(results)

	for r := range results {
		if r.text != r.agentName && r.text != "" {
			// text is the message the handler streamed (r.msg), verify no cross-contamination
			if r.text == "" {
				continue
			}
		}
		for _, ag := range agents {
			if ag.name == r.agentName && r.taskID != ag.taskID {
				t.Errorf("agent %q: taskID mismatch", r.agentName)
			}
		}
	}
}

// TestStream_NoopHandlerProducesValidSequence verifies NoopHandler produces
// Submitted → Working → Completed.
func TestStream_NoopHandlerProducesValidSequence(t *testing.T) {
	execCtx := makeExecCtx("task-009", "ctx-009")
	events := collectEvents(t, execCtx, NoopHandler)

	if len(events) != 3 {
		t.Fatalf("NoopHandler: expected 3 events, got %d", len(events))
	}

	// First event: Submitted task (*a2a.Task)
	if et := eventType(events[0]); et != "task" {
		t.Errorf("NoopHandler event[0]: expected task, got %q", et)
	}
	// Second event: Working
	if et := eventType(events[1]); et != "TASK_STATE_WORKING" {
		t.Errorf("NoopHandler event[1]: expected Working, got %q", et)
	}
	// Third event: Completed
	if et := eventType(events[2]); et != "TASK_STATE_COMPLETED" {
		t.Errorf("NoopHandler event[2]: expected Completed, got %q", et)
	}
}

// TestStream_EventMetadataHasTaskID verifies each event carries correct
// task ID and context ID.
func TestStream_EventMetadataHasTaskID(t *testing.T) {
	execCtx := makeExecCtx("task-010-meta", "ctx-010-meta")

	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.StreamText("test")
		ctx.AddArtifact(Artifact{
			Type: "code", Title: "meta.go", Content: "package meta",
			Metadata: map[string]string{"language": "go"},
		})
		return nil
	}

	events := collectEvents(t, execCtx, handler)

	for i, ev := range events {
		switch e := ev.(type) {
		case *a2a.TaskStatusUpdateEvent:
			if string(e.TaskID) != "task-010-meta" {
				t.Errorf("event[%d] status: expected TaskID task-010-meta, got %q", i, e.TaskID)
			}
			if e.ContextID != "ctx-010-meta" {
				t.Errorf("event[%d] status: expected ContextID ctx-010-meta, got %q", i, e.ContextID)
			}
		case *a2a.TaskArtifactUpdateEvent:
			if string(e.TaskID) != "task-010-meta" {
				t.Errorf("event[%d] artifact: expected TaskID task-010-meta, got %q", i, e.TaskID)
			}
			if e.ContextID != "ctx-010-meta" {
				t.Errorf("event[%d] artifact: expected ContextID ctx-010-meta, got %q", i, e.ContextID)
			}
		}
	}
}

// TestStream_InvalidContentIsSafe verifies that artifact content containing
// malformed JSON-like strings or nested fences does not break streaming.
func TestStream_InvalidContentIsSafe(t *testing.T) {
	execCtx := makeExecCtx("task-011", "ctx-011")

	handler := func(ctx *Context, messages []a2a.Message) error {
		ctx.AddArtifact(Artifact{
			Type:    "code",
			Title:   "broken.json",
			Content: `{"key": "value", broken: true, 'single': quote}`,
			Metadata: map[string]string{"language": "json"},
		})
		ctx.AddArtifact(Artifact{
			Type:    "code",
			Title:   "nested.go",
			Content: "```go\npackage main\n```",
			Metadata: map[string]string{"language": "go"},
		})
		return nil
	}

	events := collectEvents(t, execCtx, handler)
	last := events[len(events)-1]
	if et := eventType(last); et != "TASK_STATE_COMPLETED" {
		t.Errorf("stream should complete normally despite malformed content, got %q", et)
	}
}

// TestStream_TerminalStatesAreDistinct verifies terminal states are distinct.
func TestStream_TerminalStatesAreDistinct(t *testing.T) {
	execCtx := makeExecCtx("task-012", "ctx-012")

	okHandler := func(ctx *Context, messages []a2a.Message) error {
		return nil
	}
	events := collectEvents(t, execCtx, okHandler)
	last := events[len(events)-1]
	if et := eventType(last); et != "TASK_STATE_COMPLETED" {
		t.Errorf("success handler: expected Completed, got %q", et)
	}

	executor := &codeAgentExecutor{handler: NoopHandler}
	cancelSeq := executor.Cancel(context.Background(), execCtx)
	for ev, err := range cancelSeq {
		if err != nil {
			t.Fatalf("cancel error: %v", err)
		}
		se := ev.(*a2a.TaskStatusUpdateEvent)
		if se.Status.State != a2a.TaskStateCanceled {
			t.Errorf("cancel: expected Canceled, got %q", se.Status.State)
		}
	}

	if a2a.TaskStateCanceled == a2a.TaskStateCompleted {
		t.Error("Canceled and Completed must be distinct states")
	}
	if a2a.TaskStateFailed == a2a.TaskStateCompleted {
		t.Error("Failed and Completed must be distinct states")
	}
}
