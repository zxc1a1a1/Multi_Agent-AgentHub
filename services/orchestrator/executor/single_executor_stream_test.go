package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// streamStubDispatcher implements AgentDispatcher with a scripted set of
// streamed text chunks per call.
type streamStubDispatcher struct {
	chunks []string
	err    error
}

func (d *streamStubDispatcher) Dispatch(ctx context.Context, input dispatcher.DispatchInput) (*dispatcher.DispatchResult, error) {
	return &dispatcher.DispatchResult{Text: strings.Join(d.chunks, "")}, d.err
}

func (d *streamStubDispatcher) DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool) {
	return func(yield func(dispatcher.DispatchChunk) bool) {
		for _, c := range d.chunks {
			if !yield(dispatcher.DispatchChunk{Text: c}) {
				return
			}
		}
		if d.err != nil {
			yield(dispatcher.DispatchChunk{Err: d.err})
		}
	}
}

// collectStreamEvents drains an ExecuteStream call into a slice of events.
func collectStreamEvents(t *testing.T, exec StreamingExecutor, p *plan.OrchestrationPlan, msgID string) []ExecutionEvent {
	t.Helper()
	var events []ExecutionEvent
	err := exec.ExecuteStream(context.Background(), p, msgID, func(e ExecutionEvent) bool {
		events = append(events, e)
		return true
	})
	if err != nil {
		t.Fatalf("ExecuteStream returned error: %v", err)
	}
	return events
}

// TestSingleExecutorStream_EmitsPerChunkDeltas verifies each streamed chunk
// produces a distinct message_delta event (true streaming, not buffered).
func TestSingleExecutorStream_EmitsPerChunkDeltas(t *testing.T) {
	disp := &streamStubDispatcher{chunks: []string{"alpha", "beta", "gamma"}}
	e := NewSingleExecutor(newStubRegistry(), disp)

	events := collectStreamEvents(t, e, validSinglePlan(), "msg_1")

	var deltas []string
	var sawStart, sawEnd, sawFinished bool
	for _, ev := range events {
		switch ev.Type {
		case "message_start":
			sawStart = true
		case "message_delta":
			deltas = append(deltas, ev.Delta)
		case "message_end":
			sawEnd = true
		case "run_finished":
			sawFinished = true
		}
	}

	if !sawStart || !sawEnd || !sawFinished {
		t.Fatalf("missing lifecycle events: start=%v end=%v finished=%v", sawStart, sawEnd, sawFinished)
	}
	if len(deltas) != 3 {
		t.Fatalf("expected 3 separate deltas (one per chunk), got %d: %v", len(deltas), deltas)
	}
	if strings.Join(deltas, "") != "alphabetagamma" {
		t.Fatalf("unexpected concatenated delta text: %v", deltas)
	}
}
