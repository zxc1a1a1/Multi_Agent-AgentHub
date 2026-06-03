package executor

import (
	"context"
	"fmt"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// SingleExecutor executes a single-task OrchestrationPlan.
// It only supports StrategySingle plans with exactly one task.
// It MUST only execute validated plans (Validation.Validated == true).
type SingleExecutor struct {
	registry   AgentRegistry
	dispatcher AgentDispatcher
}

// NewSingleExecutor creates a SingleExecutor wired to the given registry and dispatcher.
func NewSingleExecutor(reg AgentRegistry, disp AgentDispatcher) *SingleExecutor {
	return &SingleExecutor{
		registry:   reg,
		dispatcher: disp,
	}
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
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_NOT_IMPLEMENTED",
				Message: fmt.Sprintf("single executor only supports strategy=%q, got %q", plan.StrategySingle, p.Strategy),
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

	task := p.Tasks[0]
	agentName := task.AgentName

	// Resolve agent from registry.
	endpoint, ok := e.registry.Get(agentName)
	if !ok {
		return []ExecutionEvent{{
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + agentName,
			},
		}}, nil
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
			Type:  "run_error",
			RunID: p.RunID,
			Error: &ExecutionError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent execution failed",
			},
		})
		return events, nil
	}

	// message_delta with response text.
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

	// run_finished
	events = append(events, ExecutionEvent{
		Type:  "run_finished",
		RunID: p.RunID,
		State: map[string]any{"status": "completed"},
	})

	return events, nil
}
