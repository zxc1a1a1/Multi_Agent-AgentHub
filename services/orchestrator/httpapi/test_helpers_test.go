package httpapi

import (
	"context"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
)

// FakeMainAgent is a test stub that implements planner.Planner.
// It must NOT be used in production code.
type FakeMainAgent struct {
	// PlanToReturn is the plan returned by Plan() on every call.
	PlanToReturn *plan.OrchestrationPlan
	// PlanErr is the error returned by Plan().
	PlanErr error
	// PlanCalled is set to true after Plan() is called.
	PlanCalled bool
	// PlanCallCount tracks how many times Plan() was called.
	PlanCallCount int
	// LastInput records the last PlannerInput passed to Plan().
	LastInput planner.PlannerInput
	// Plans is a sequence of plans returned on successive calls (used for revise tests).
	// When non-empty, each call returns Plans[callN % len(Plans)].
	Plans []*plan.OrchestrationPlan
}

// Plan implements planner.Planner.
func (f *FakeMainAgent) Plan(_ context.Context, input planner.PlannerInput) (*plan.OrchestrationPlan, error) {
	f.PlanCalled = true
	f.PlanCallCount++
	f.LastInput = input
	if f.PlanErr != nil {
		return nil, f.PlanErr
	}
	if len(f.Plans) > 0 {
		idx := (f.PlanCallCount - 1) % len(f.Plans)
		return f.Plans[idx], nil
	}
	return f.PlanToReturn, nil
}

// Ensure FakeMainAgent implements planner.Planner.
var _ planner.Planner = (*FakeMainAgent)(nil)

// makeTestPlan creates a minimal valid OrchestrationPlan for testing.
func makeTestPlan(runID, strategy string, tasks []plan.TaskPlan) *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		Version:        "1.0",
		PlanID:         "plan-test-" + runID,
		RunID:          runID,
		ConversationID: "conv-" + runID,
		ExecutionPath:  "main_agent_orchestration",
		Strategy:       strategy,
		IntentSummary:  "test plan",
		Tasks:          tasks,
		PlannerSource:  "main_agent",
		PlanOwner: &plan.PlanOwner{
			Type:        "main_agent",
			AgentName:   "main-agent",
			IsMainAgent: true,
		},
	}
}

// makeTaskPlan creates a TaskPlan with defaults for testing.
func makeTaskPlan(taskID, agentName, taskContent string) plan.TaskPlan {
	return plan.TaskPlan{
		TaskID:      taskID,
		AgentName:   agentName,
		TaskContent: taskContent,
		Priority:    1,
		TimeoutMs:   60000,
		RiskLevel:   "low",
	}
}
