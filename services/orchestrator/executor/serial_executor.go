package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// SerialExecutor executes tasks in original plan.Tasks order, one by one.
// No sorting, no goroutines, no topological reordering.
type SerialExecutor struct {
	registry    AgentRegistry
	dispatcher  AgentDispatcher
	synthesizer synthesizer.Synthesizer
}

// SerialExecutorOption customizes a SerialExecutor.
type SerialExecutorOption func(*SerialExecutor)

// WithSerialSynthesizer injects a synthesizer for result aggregation.
func WithSerialSynthesizer(s synthesizer.Synthesizer) SerialExecutorOption {
	return func(e *SerialExecutor) {
		if e == nil {
			return
		}
		e.synthesizer = s
	}
}

// NewSerialExecutor creates a SerialExecutor.
func NewSerialExecutor(reg AgentRegistry, disp AgentDispatcher, opts ...SerialExecutorOption) *SerialExecutor {
	e := &SerialExecutor{
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

// ExecuteStream executes tasks serially in plan.Tasks order, streaming each
// task's output as it arrives. SelectedParticipants filtering is done by the
// caller (handler_run_stream.go) before the executor is invoked.
func (e *SerialExecutor) ExecuteStream(ctx context.Context, p *plan.OrchestrationPlan, msgID string, emit EventSink) error {
	if e == nil {
		return fmt.Errorf("serial executor is nil")
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
	if len(p.Tasks) == 0 {
		emit(ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{Code: "ORCHESTRATOR_BAD_REQUEST", Message: "plan has no tasks"},
		})
		return nil
	}

	taskResults := make([]taskResult, 0, len(p.Tasks))
	allSucceeded := true

	for i, task := range p.Tasks {
		if err := ctx.Err(); err != nil {
			return nil
		}

		taskMsgID := fmt.Sprintf("%s_%d", msgID, i)
		result, ok := e.streamOneTask(ctx, p, task, taskMsgID, emit)
		if !ok {
			allSucceeded = false
			continue
		}
		if result != nil {
			taskResults = append(taskResults, *result)
		}
	}

	// Emit summary.
	if allSucceeded && len(taskResults) > 0 {
		if !synthesizeIfNeeded(ctx, p, msgID, taskResults, e.synthesizer, emit) {
			summaryMsgID := msgID + "_summary"
			summary := buildSummary(taskResults)
			if !emit(ExecutionEvent{Type: "message_start", RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator"}) {
				return nil
			}
			if !emit(ExecutionEvent{Type: "message_delta", RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator", Delta: summary}) {
				return nil
			}
			if !emit(ExecutionEvent{Type: "message_end", RunID: p.RunID, MessageID: summaryMsgID, AgentName: "orchestrator"}) {
				return nil
			}
		}
	}

	status := "completed"
	if !allSucceeded {
		status = "partial_failure"
	}
	emit(ExecutionEvent{
		Type:  "run_finished",
		RunID: p.RunID,
		State: map[string]any{"status": status, "taskCount": len(p.Tasks)},
	})
	return nil
}

// streamOneTask dispatches a single task in streaming mode.
func (e *SerialExecutor) streamOneTask(ctx context.Context, p *plan.OrchestrationPlan, task plan.TaskPlan, msgID string, emit EventSink) (*taskResult, bool) {
	agentName := task.AgentName

	endpoint, ok := e.registry.Get(agentName)
	if !ok {
		emit(ExecutionEvent{
			Type:   "run_error",
			RunID:  p.RunID,
			TaskID: task.TaskID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + agentName,
			},
		})
		return nil, false
	}

	if !emit(ExecutionEvent{Type: "message_start", RunID: p.RunID, MessageID: msgID, TaskID: task.TaskID, AgentName: agentName}) {
		return nil, false
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
		return emit(ExecutionEvent{
			Type:      "message_delta",
			RunID:     p.RunID,
			MessageID: msgID,
			TaskID:    task.TaskID,
			AgentName: agentName,
			Delta:     c.Text,
		})
	})

	if dispatchErr != nil {
		emit(ExecutionEvent{
			Type:   "run_error",
			RunID:  p.RunID,
			TaskID: task.TaskID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent dispatch failed for " + agentName + ": " + sanitizeAgentError(dispatchErr.Error()),
			},
		})
		return nil, false
	}

	emit(ExecutionEvent{Type: "message_end", RunID: p.RunID, MessageID: msgID, TaskID: task.TaskID, AgentName: agentName})

	return &taskResult{TaskID: task.TaskID, AgentName: agentName, Text: sb.String()}, true
}
