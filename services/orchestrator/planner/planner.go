// Package planner defines the Planner interface and input types for intent orchestration.
package planner

import (
	"context"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// PlannerInput is the standard input for all Planner implementations.
type PlannerInput struct {
	RunID              string
	ConversationID     string
	ConversationType   string
	UserMessage        string
	AgentName          string
	SelectedAgentNames []string
	Mentions           []string
	PlanningMode       string
	TraceID            string
	AvailableAgents    []string
}

// Planner produces an OrchestrationPlan from a PlannerInput.
// All implementations must return a structured plan with validation.validated=false.
type Planner interface {
	Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error)
}
