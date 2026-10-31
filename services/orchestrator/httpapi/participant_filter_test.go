package httpapi

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestFilterPlanByConfirmedParticipants(t *testing.T) {
	original := &plan.OrchestrationPlan{
		PlanID: "plan-1",
		Tasks: []plan.TaskPlan{
			{TaskID: "t1", AgentName: "code-agent"},
			{TaskID: "t2", AgentName: "review-agent"},
			{TaskID: "t3", AgentName: "test-agent"},
		},
		ConfirmedParticipantNames: []string{" CODE-AGENT ", "test-agent"},
	}
	filtered := filterPlanByConfirmedParticipants(original)
	if filtered == original {
		t.Fatal("confirmed selection must produce a copy")
	}
	if len(filtered.Tasks) != 2 || filtered.Tasks[0].AgentName != "code-agent" || filtered.Tasks[1].AgentName != "test-agent" {
		t.Fatalf("unexpected filtered tasks: %#v", filtered.Tasks)
	}
	if len(original.Tasks) != 3 {
		t.Fatalf("original plan must not be mutated: %#v", original.Tasks)
	}
}

func TestFilterPlanWithoutSelectionKeepsAllTasks(t *testing.T) {
	original := &plan.OrchestrationPlan{
		Tasks: []plan.TaskPlan{{TaskID: "t1", AgentName: "code-agent"}},
	}
	if got := filterPlanByConfirmedParticipants(original); got != original || len(got.Tasks) != 1 {
		t.Fatalf("empty selection must keep the original plan: %#v", got)
	}
	if got := filterPlanByConfirmedParticipants(nil); got != nil {
		t.Fatal("nil plan must stay nil")
	}
}
