package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// SingleExecutor executes a single-task OrchestrationPlan.
// It only supports StrategySingle plans with exactly one task.
// It MUST only execute validated plans (Validation.Validated == true).
type SingleExecutor struct {
	registry    AgentRegistry
	dispatcher  AgentDispatcher
	synthesizer synthesizer.Synthesizer
}

// SingleExecutorOption customizes a SingleExecutor.
type SingleExecutorOption func(*SingleExecutor)

// WithSingleSynthesizer injects a synthesizer for result aggregation.
func WithSingleSynthesizer(s synthesizer.Synthesizer) SingleExecutorOption {
	return func(e *SingleExecutor) {
		if e == nil {
			return
		}
		e.synthesizer = s
	}
}

// NewSingleExecutor creates a SingleExecutor wired to the given registry and dispatcher.
func NewSingleExecutor(reg AgentRegistry, disp AgentDispatcher, opts ...SingleExecutorOption) *SingleExecutor {
	e := &SingleExecutor{
		registry:   reg,
		dispatcher: disp,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

// Execute runs the single task in the plan. It returns an error only when the plan
// is not validated. Dispatch failures are returned as error events, not as error
// returns, so the caller can still emit them.
func (e *SingleExecutor) Execute(ctx context.Context, p *plan.OrchestrationPlan, msgID string) ([]ExecutionEvent, error) {
	if e == nil {
		return nil, fmt.Errorf("single executor is nil")
	}
	if p == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	// Reject unvalidated plans.
	if !p.Validation.Validated {
		return nil, ErrPlanNotValidated
	}

	if p.Strategy != plan.StrategySingle {
		return []ExecutionEvent{{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: fmt.Sprintf("single executor only supports strategy=%q, got %q", plan.StrategySingle, p.Strategy),
			},
		}}, nil
	}

	if len(p.Tasks) == 0 {
		return []ExecutionEvent{{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_BAD_REQUEST",
				Message: "plan has no tasks",
			},
		}}, nil
	}

	task := p.Tasks[0]
	agentName := task.AgentName

	// Resolve agent from registry.
	endpoint, ok := e.registry.Get(agentName)
	if !ok {
		return []ExecutionEvent{{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + agentName,
			},
		}}, nil
	}

	var events []ExecutionEvent

	events = append(events, taskStartedEvent(p, task, msgID, 0))

	// Dispatch to the remote agent.
	input := dispatcher.DispatchInput{
		AgentURL:       endpoint.URL,
		AgentName:      agentName,
		ConversationID: p.ConversationID,
		RunID:          p.RunID,
		Message:        task.TaskContent,
	}

	result, err := e.dispatcher.Dispatch(ctx, input)
	if err != nil {
		events = append(events, ExecutionEvent{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent dispatch failed: " + sanitizeAgentError(err.Error()),
			},
		})
		return events, nil
	}

	// message_delta with response text.
	if result != nil && result.Text != "" {
		events = append(events, taskContentEvent(p, task, msgID, 0, result.Text))
	}

	events = append(events, taskFinishedEvent(p, task, msgID, 0, resultText(result), "completed"))

	// run_finished
	events = append(events, ExecutionEvent{
		Type:  agui.InternalTypeRunFinished,
		RunID: p.RunID,
		State: map[string]any{"status": "completed"},
	})

	return events, nil
}

// ExecuteStream runs the single task and emits events incrementally. Each
// streamed dispatcher chunk produces its own message_delta, so the client sees
// partial output as it arrives rather than one buffered block.
func (e *SingleExecutor) ExecuteStream(ctx context.Context, p *plan.OrchestrationPlan, msgID string, emit EventSink) error {
	if e == nil {
		return fmt.Errorf("single executor is nil")
	}
	if p == nil {
		return fmt.Errorf("plan is nil")
	}
	if emit == nil {
		return fmt.Errorf("emit sink is nil")
	}
	if !p.Validation.Validated {
		return ErrPlanNotValidated
	}

	if p.Strategy != plan.StrategySingle {
		emit(ExecutionEvent{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: fmt.Sprintf("single executor only supports strategy=%q, got %q", plan.StrategySingle, p.Strategy),
			},
		})
		return nil
	}
	if len(p.Tasks) == 0 {
		emit(ExecutionEvent{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{Code: "ORCHESTRATOR_BAD_REQUEST", Message: "plan has no tasks"},
		})
		return nil
	}

	task := p.Tasks[0]
	agentName := task.AgentName

	endpoint, ok := e.registry.Get(agentName)
	if !ok {
		emit(ExecutionEvent{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + agentName,
			},
		})
		return nil
	}

	if !emit(taskStartedEvent(p, task, msgID, 0)) {
		return nil
	}

	input := dispatcher.DispatchInput{
		AgentURL:       endpoint.URL,
		AgentName:      agentName,
		ConversationID: p.ConversationID,
		RunID:          p.RunID,
		Message:        task.TaskContent,
		TimeoutMs:      task.TimeoutMs,
		TraceID:        p.TraceID,
	}

	var sb strings.Builder
	var dispatchErr error
	e.dispatcher.DispatchStream(ctx, input)(func(c dispatcher.DispatchChunk) bool {
		if c.Err != nil {
			dispatchErr = c.Err
			return false
		}
		if c.Text == "" {
			return true
		}
		sb.WriteString(c.Text)
		return emit(taskContentEvent(p, task, msgID, 0, c.Text))
	})

	if dispatchErr != nil {
		emit(ExecutionEvent{
			Type:  agui.InternalTypeRunError,
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent dispatch failed: " + sanitizeAgentError(dispatchErr.Error()),
			},
		})
		return nil
	}

	if !emit(taskFinishedEvent(p, task, msgID, 0, sb.String(), "completed")) {
		return nil
	}

	// Synthesize if the plan requests aggregation.
	synthesizeIfNeeded(ctx, p, msgID, []taskResult{{TaskID: task.TaskID, AgentName: agentName, Text: sb.String()}}, e.synthesizer, emit)

	emit(ExecutionEvent{
		Type:  agui.InternalTypeRunFinished,
		RunID: p.RunID,
		State: map[string]any{"status": "completed"},
	})
	return nil
}
