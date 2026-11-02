package planner

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestListerPlanValidatorAcceptsSequential(t *testing.T) {
	validator := newListerPlanValidator(&mockAgentLister{agents: testAgentInfos()})
	result := validator.Validate(&plan.OrchestrationPlan{
		Strategy: plan.StrategySequential,
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent"},
			{TaskID: "t2", AgentName: "code-agent", DependsOn: []string{"t1"}},
		},
	})
	if !result.Valid {
		t.Fatalf("sequential rejected: %#v", result.Errors)
	}
}
