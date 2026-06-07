package executor

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// OrderedParallelExecutor executes an ordered_parallel OrchestrationPlan by
// running tasks serially, sorted by priority (ascending), with independent
// message_start / message_delta / message_end events per task. After all
// tasks succeed, it emits an orchestrator summary.
//
// Phase 7: this is a controlled serial execution — no real goroutine fan-out.
type OrderedParallelExecutor struct {
	registry    AgentRegistry
	dispatcher  AgentDispatcher
	synthesizer synthesizer.Synthesizer
}

// OrderedParallelExecutorOption customizes an OrderedParallelExecutor.
type OrderedParallelExecutorOption func(*OrderedParallelExecutor)

// WithOrderedParallelSynthesizer injects a synthesizer for result aggregation.
func WithOrderedParallelSynthesizer(s synthesizer.Synthesizer) OrderedParallelExecutorOption {
	return func(e *OrderedParallelExecutor) {
		if e == nil {
			return
		}
		e.synthesizer = s
	}
}

// NewOrderedParallelExecutor creates an OrderedParallelExecutor.
func NewOrderedParallelExecutor(reg AgentRegistry, disp AgentDispatcher, opts ...OrderedParallelExecutorOption) *OrderedParallelExecutor {
	e := &OrderedParallelExecutor{
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

// Execute runs all tasks in priority order, then emits an orchestrator summary.
func (e *OrderedParallelExecutor) Execute(ctx context.Context, p *plan.OrchestrationPlan, msgID string) ([]ExecutionEvent, error) {
	if e == nil {
		return nil, fmt.Errorf("ordered_parallel executor is nil")
	}
	if p == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	if !p.Validation.Validated {
		return nil, ErrPlanNotValidated
	}

	if p.Strategy != plan.StrategyOrderedParallel {
		return []ExecutionEvent{{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: fmt.Sprintf("ordered_parallel executor only supports strategy=%q, got %q", plan.StrategyOrderedParallel, p.Strategy),
			},
		}}, nil
	}

	if len(p.Tasks) == 0 {
		return []ExecutionEvent{{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_BAD_REQUEST",
				Message: "plan has no tasks",
			},
		}}, nil
	}

	// Sort tasks by priority ascending. Stable sort preserves insertion order
	// for equal priorities, keeping RulePlanner's web-before-code default.
	tasks := make([]plan.TaskPlan, len(p.Tasks))
	copy(tasks, p.Tasks)
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority
	})

	var events []ExecutionEvent
	taskResults := make([]taskResult, 0, len(tasks))
	allSucceeded := true

	for i, task := range tasks {
		// Per-task message ID: append task index to keep messages distinct.
		taskMsgID := fmt.Sprintf("%s_%d", msgID, i)

		taskEvents, result, err := e.executeOneTask(ctx, p, task, taskMsgID)
		events = append(events, taskEvents...)

		if err != nil {
			allSucceeded = false
			events = append(events, ExecutionEvent{
				Type:  "run_error",
				RunID: p.RunID,
				TaskID: task.TaskID,
				Error: &ExecutionError{
					Code:    "ORCHESTRATOR_AGENT_FAILED",
					Message: "Task " + task.TaskID + " failed: " + sanitizeSummary(err.Error()),
				},
			})
		} else if result != nil {
			taskResults = append(taskResults, *result)
		}
	}

	// Orchestrator summary: a separate message with sender="orchestrator".
	if allSucceeded {
		summaryMsgID := msgID + "_summary"
		summary := buildSummary(p, taskResults)
		events = append(events, ExecutionEvent{
			Type:      "message_start",
			RunID:     p.RunID,
			MessageID: summaryMsgID,
			AgentName: "orchestrator",
		})
		events = append(events, ExecutionEvent{
			Type:      "message_delta",
			RunID:     p.RunID,
			MessageID: summaryMsgID,
			AgentName: "orchestrator",
			Delta:     summary,
		})
		events = append(events, ExecutionEvent{
			Type:      "message_end",
			RunID:     p.RunID,
			MessageID: summaryMsgID,
			AgentName: "orchestrator",
		})
	}

	// run_finished
	status := "completed"
	if !allSucceeded {
		status = "partial_failure"
	}
	events = append(events, ExecutionEvent{
		Type:  "run_finished",
		RunID: p.RunID,
		State: map[string]any{
			"status":    status,
			"taskCount": len(tasks),
		},
	})

	return events, nil
}

// taskResult is the accumulated output of one executed task.
type taskResult struct {
	TaskID    string
	AgentName string
	Text      string
}

// ExecuteStream runs all tasks in priority order and emits events incrementally
// through the sink. Each streamed dispatcher chunk produces its own
// message_delta so partial output reaches the client as it arrives. After all
// tasks complete it emits the orchestrator summary and run_finished.
func (e *OrderedParallelExecutor) ExecuteStream(ctx context.Context, p *plan.OrchestrationPlan, msgID string, emit EventSink) error {
	if e == nil {
		return fmt.Errorf("ordered_parallel executor is nil")
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
	if p.Strategy != plan.StrategyOrderedParallel {
		emit(ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: fmt.Sprintf("ordered_parallel executor only supports strategy=%q, got %q", plan.StrategyOrderedParallel, p.Strategy),
			},
		})
		return nil
	}
	if len(p.Tasks) == 0 {
		emit(ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{Code: "ORCHESTRATOR_BAD_REQUEST", Message: "plan has no tasks"},
		})
		return nil
	}

	tasks := make([]plan.TaskPlan, len(p.Tasks))
	copy(tasks, p.Tasks)
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority
	})

	taskResults := make([]taskResult, 0, len(tasks))
	allSucceeded := true

	for i, task := range tasks {
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

	if allSucceeded {
		// Use synthesizer if available and plan requests aggregation; otherwise static summary.
		if !synthesizeIfNeeded(ctx, p, msgID, taskResults, e.synthesizer, emit) {
			summaryMsgID := msgID + "_summary"
			summary := buildSummary(p, taskResults)
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
		State: map[string]any{"status": status, "taskCount": len(tasks)},
	})
	return nil
}

// streamOneTask dispatches a single task in streaming mode. It returns the
// accumulated task result and whether the task succeeded.
func (e *OrderedParallelExecutor) streamOneTask(ctx context.Context, p *plan.OrchestrationPlan, task plan.TaskPlan, msgID string, emit EventSink) (*taskResult, bool) {
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

// executeOneTask dispatches a single task and returns message_start/delta/end events.
func (e *OrderedParallelExecutor) executeOneTask(ctx context.Context, p *plan.OrchestrationPlan, task plan.TaskPlan, msgID string) ([]ExecutionEvent, *taskResult, error) {
	agentName := task.AgentName

	endpoint, ok := e.registry.Get(agentName)
	if !ok {
		return []ExecutionEvent{{
			Type:  "run_error",
			RunID: p.RunID,
			TaskID: task.TaskID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + agentName,
			},
		}}, nil, fmt.Errorf("agent %s unavailable", agentName)
	}

	var events []ExecutionEvent

	// message_start
	events = append(events, ExecutionEvent{
		Type:      "message_start",
		RunID:     p.RunID,
		MessageID: msgID,
		TaskID:    task.TaskID,
		AgentName: agentName,
	})

	input := dispatcher.DispatchInput{
		AgentURL:       endpoint.URL,
		AgentName:      agentName,
		ConversationID: p.ConversationID,
		RunID:          p.RunID,
		Message:        task.TaskContent,
		TraceID:        p.TraceID,
	}

	result, err := e.dispatcher.Dispatch(ctx, input)
	if err != nil {
		events = append(events, ExecutionEvent{
			Type:  "run_error",
			RunID: p.RunID,
			TaskID: task.TaskID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent dispatch failed for " + agentName + ": " + sanitizeAgentError(err.Error()),
			},
		})
		return events, nil, err
	}

	if result != nil && result.Text != "" {
		events = append(events, ExecutionEvent{
			Type:      "message_delta",
			RunID:     p.RunID,
			MessageID: msgID,
			TaskID:    task.TaskID,
			AgentName: agentName,
			Delta:     result.Text,
		})
	}

	// message_end
	events = append(events, ExecutionEvent{
		Type:      "message_end",
		RunID:     p.RunID,
		MessageID: msgID,
		TaskID:    task.TaskID,
		AgentName: agentName,
	})

	tr := &taskResult{
		TaskID:    task.TaskID,
		AgentName: agentName,
	}
	if result != nil {
		tr.Text = result.Text
	}
	return events, tr, nil
}

func buildSummary(p *plan.OrchestrationPlan, results []taskResult) string {
	var b strings.Builder
	b.WriteString("All ")
	b.WriteString(fmt.Sprintf("%d", len(results)))
	b.WriteString(" task(s) completed successfully.")

	for _, r := range results {
		b.WriteString("\n- ")
		b.WriteString(r.TaskID)
		b.WriteString(" (")
		b.WriteString(r.AgentName)
		b.WriteString(")")
		if r.Text != "" {
			b.WriteString(" produced output")
		}
	}
	return b.String()
}

func sanitizeSummary(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	return s
}
