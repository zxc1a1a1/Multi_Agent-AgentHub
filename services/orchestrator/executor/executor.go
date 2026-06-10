// Package executor provides plan execution for the Orchestrator.
// All executors must check plan.Validation.Validated before execution.
package executor

import (
	"context"
	"errors"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// ErrPlanNotValidated is returned when an executor receives a plan without
// Validation.Validated == true.
var ErrPlanNotValidated = errors.New("plan not validated: validation.validated must be true before execution")

// InternalTypeTaskRefRegistered is an executor-internal event used by httpapi
// to register the real remote A2A task id for cancellation/tool-result routing.
// It is never forwarded as an AG-UI event.
const InternalTypeTaskRefRegistered = "__agenthub_task_ref_registered"

// AgentRegistry is the subset of the agent registry that executors need.
type AgentRegistry interface {
	ResolveURL(ctx context.Context, name string) (string, bool, error)
}

// AgentDispatcher is the subset of the A2A dispatcher that executors need.
type AgentDispatcher interface {
	Dispatch(ctx context.Context, input dispatcher.DispatchInput) (*dispatcher.DispatchResult, error)
	DispatchStream(ctx context.Context, input dispatcher.DispatchInput) func(yield func(dispatcher.DispatchChunk) bool)
}

// Executor executes a validated OrchestrationPlan and returns execution events.
type Executor interface {
	Execute(ctx context.Context, p *plan.OrchestrationPlan, msgID string) ([]ExecutionEvent, error)
}

// EventSink receives execution events as they are produced. Returning false
// signals the executor to stop (e.g. the client disconnected).
type EventSink func(ExecutionEvent) bool

// StreamingExecutor executes a validated OrchestrationPlan and emits events
// incrementally through the sink, enabling per-chunk streaming to the client.
type StreamingExecutor interface {
	ExecuteStream(ctx context.Context, p *plan.OrchestrationPlan, msgID string, emit EventSink) error
}

// ExecutionEvent is a single event produced during execution.
type ExecutionEvent struct {
	Type         string
	RunID        string
	MessageID    string
	TaskID       string
	StepID       string
	AgentName    string
	RemoteTaskID string
	AgentURL     string
	TurnIndex    int
	Delta        string
	Status       string
	Summary      string
	Error        *ExecutionError
	State        map[string]any
	ArtifactMeta *dispatcher.ArtifactMeta // non-nil when this event carries artifact metadata
}

// ExecutionError is a sanitized error carried in execution events.
type ExecutionError struct {
	Code    string
	Message string
}

// sanitizeAgentError applies a final length cap to dispatch error messages
// before they are included in user-facing execution events. The A2A client
// already sanitizes sensitive content; this is a defence-in-depth cap.
func sanitizeAgentError(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:197] + "..."
	}
	return s
}

// synthesizeIfNeeded runs the synthesizer when the plan requests aggregation.
// It returns the synthesized text (or the static summary) and emits it as a
// streaming message. It returns true if synthesis occurred.
func synthesizeIfNeeded(ctx context.Context, p *plan.OrchestrationPlan, msgID string, results []taskResult, synth synthesizer.Synthesizer, emit EventSink) bool {
	if p == nil || emit == nil || len(results) == 0 {
		return false
	}
	if !p.Aggregation.Required || p.Aggregation.Mode != "summary" {
		return false
	}

	summaryMsgID := msgID + "_summary"

	// Build task outputs for the synthesizer.
	outputs := make([]synthesizer.TaskOutput, len(results))
	for i, r := range results {
		outputs[i] = synthesizer.TaskOutput{
			TaskID:    r.TaskID,
			AgentName: r.AgentName,
			Output:    r.Text,
		}
	}

	var summary string
	if synth != nil {
		s, err := synth.Synthesize(ctx, p.IntentSummary, outputs)
		if err == nil && s != "" {
			summary = s
		}
	}
	if summary == "" {
		// Fall back to static summary.
		static := synthesizer.NewStaticSynthesizer()
		summary, _ = static.Synthesize(ctx, p.IntentSummary, outputs)
	}

	if !emit(ExecutionEvent{Type: agui.InternalTypeMessageStart, RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator"}) {
		return false
	}
	if !emit(ExecutionEvent{Type: agui.InternalTypeMessageDelta, RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator", Delta: summary}) {
		return false
	}
	if !emit(ExecutionEvent{Type: agui.InternalTypeMessageEnd, RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator"}) {
		return false
	}
	return true
}
