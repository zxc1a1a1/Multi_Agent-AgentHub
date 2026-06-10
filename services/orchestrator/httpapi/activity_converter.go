package httpapi

import (
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// toAGUIActivitySnapshot converts an Orchestrator internal plan model to the
// canonical AG-UI ActivitySnapshot for SSE wire emission.
func toAGUIActivitySnapshot(orchPlan *plan.OrchestrationPlan, status string) agui.ActivitySnapshot {
	if orchPlan == nil {
		return agui.ActivitySnapshot{}
	}

	snapshot := agui.ActivitySnapshot{
		ActivityID:     orchPlan.PlanID,
		ActivityType:   "plan_approval",
		Status:         status,
		ExecutionPath:  orchPlan.ExecutionPath,
		PlanID:         orchPlan.PlanID,
		Revision:       orchPlan.Revision,
		PlanOwner:      toAGUIPlanOwner(orchPlan.PlanOwner),
		ExecutionOwner: toAGUIExecutionOwner(orchPlan.ExecutionOwner),
		Participants:   toAGUIParticipants(orchPlan.Participants),
		Title:          orchPlan.IntentSummary,
		Summary:        orchPlan.IntentSummary,
		Tasks:          toAGUITaskSummaries(orchPlan.Tasks),
		AllowedActions: []string{"approve", "revise", "cancel"},
		Warnings:       orchPlan.Warnings,
	}

	if len(orchPlan.CandidateParticipants) > 0 {
		snapshot.CandidateParticipants = toAGUIParticipants(orchPlan.CandidateParticipants)
	}
	if len(orchPlan.DefaultSelectedParticipants) > 0 {
		snapshot.DefaultSelectedParticipants = toAGUIParticipants(orchPlan.DefaultSelectedParticipants)
	}

	// Derive required participant names for frontend validation
	var required []string
	for _, p := range orchPlan.Participants {
		if p.Required {
			required = append(required, p.AgentName)
		}
	}
	if len(required) > 0 {
		snapshot.RequiredParticipants = required
	}

	return snapshot
}

// toAGUIPlanOwner converts plan.PlanOwner to agui.PlanOwner.
// Pure type mapping — does not normalize or change values.
func toAGUIPlanOwner(owner *plan.PlanOwner) *agui.PlanOwner {
	if owner == nil {
		return nil
	}
	return &agui.PlanOwner{
		Type:        owner.Type,
		AgentName:   owner.AgentName,
		IsMainAgent: owner.IsMainAgent,
	}
}

// toAGUIExecutionOwner converts plan.ExecutionOwner to agui.ExecutionOwner.
func toAGUIExecutionOwner(owner *plan.ExecutionOwner) *agui.ExecutionOwner {
	if owner == nil {
		return nil
	}
	return &agui.ExecutionOwner{
		Type:                 owner.Type,
		AgentName:            owner.AgentName,
		AgentNames:           owner.AgentNames,
		SelectedParticipants: owner.SelectedParticipants,
	}
}

// toAGUIParticipants converts plan.PlanParticipant slice to agui.PlanParticipant slice.
// Role and Reason are intentionally dropped — agui wire schema does not include them.
func toAGUIParticipants(ps []plan.PlanParticipant) []agui.PlanParticipant {
	if len(ps) == 0 {
		return nil
	}
	out := make([]agui.PlanParticipant, len(ps))
	for i, p := range ps {
		out[i] = agui.PlanParticipant{
			AgentName: p.AgentName,
			Required:  p.Required,
			Selected:  p.Selected,
		}
	}
	return out
}

// toAGUITaskSummaries converts plan.TaskPlan slice to agui.TaskSummary slice.
// DependsOn is intentionally dropped — agui.TaskSummary and frontend ActivityTask
// do not include dependsOn.
func toAGUITaskSummaries(tasks []plan.TaskPlan) []agui.TaskSummary {
	if len(tasks) == 0 {
		return nil
	}
	out := make([]agui.TaskSummary, len(tasks))
	for i, t := range tasks {
		out[i] = agui.TaskSummary{
			TaskID:    t.TaskID,
			AgentName: t.AgentName,
			Content:   t.TaskContent,
			Priority:  t.Priority,
			RiskLevel: t.RiskLevel,
		}
	}
	return out
}
