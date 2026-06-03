// Package executor provides plan execution for the Orchestrator.
// All executors must check plan.Validation.Validated before execution.
package executor

import (
	"context"
	"errors"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// ErrPlanNotValidated is returned when an executor receives a plan without
// Validation.Validated == true.
var ErrPlanNotValidated = errors.New("plan not validated: validation.validated must be true before execution")

// AgentRegistry is the subset of the agent registry that executors need.
type AgentRegistry interface {
	Get(name string) (registry.AgentEndpoint, bool)
}

// AgentDispatcher is the subset of the A2A dispatcher that executors need.
type AgentDispatcher interface {
	Dispatch(ctx context.Context, input dispatcher.DispatchInput) (*dispatcher.DispatchResult, error)
}

// Executor executes a validated OrchestrationPlan and returns execution events.
type Executor interface {
	Execute(ctx context.Context, p *plan.OrchestrationPlan, msgID string) ([]ExecutionEvent, error)
}

// ExecutionEvent is a single event produced during execution.
type ExecutionEvent struct {
	Type      string
	RunID     string
	MessageID string
	TaskID    string
	AgentName string
	Delta     string
	Error     *ExecutionError
	State     map[string]any
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
