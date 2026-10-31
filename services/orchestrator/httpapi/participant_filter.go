package httpapi

import (
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// filterPlanByConfirmedParticipants returns a shallow copy whose task list only
// contains agents the user confirmed. The original plan is left untouched
// because PendingPlan.FullPlan and revision bookkeeping reference it.
func filterPlanByConfirmedParticipants(p *plan.OrchestrationPlan) *plan.OrchestrationPlan {
	if p == nil || len(p.ConfirmedParticipantNames) == 0 {
		return p
	}
	selected := make(map[string]bool, len(p.ConfirmedParticipantNames))
	for _, name := range p.ConfirmedParticipantNames {
		selected[strings.ToLower(strings.TrimSpace(name))] = true
	}
	execPlan := *p
	execTasks := make([]plan.TaskPlan, 0, len(p.Tasks))
	for _, task := range p.Tasks {
		if selected[strings.ToLower(strings.TrimSpace(task.AgentName))] {
			execTasks = append(execTasks, task)
		}
	}
	execPlan.Tasks = execTasks
	return &execPlan
}
