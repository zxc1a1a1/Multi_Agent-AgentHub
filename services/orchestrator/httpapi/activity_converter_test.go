package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestToAGUIActivitySnapshot(t *testing.T) {
	orchPlan := &plan.OrchestrationPlan{
		PlanID:        "plan_001",
		ExecutionPath: "single_chat",
		Revision:      1,
		IntentSummary: "Test plan",
		PlanOwner: &plan.PlanOwner{
			Type:        "main_agent",
			AgentName:   "main-agent",
			IsMainAgent: true,
		},
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
		Tasks: []plan.TaskPlan{
			{TaskID: "task_001", AgentName: "code-agent", TaskContent: "Write code", Priority: 1, RiskLevel: "low"},
		},
		Warnings: []string{"test warning"},
	}

	snapshot := toAGUIActivitySnapshot(orchPlan, "awaiting_confirmation")

	if snapshot.ActivityID != "plan_001" {
		t.Errorf("expected ActivityID=plan_001, got %s", snapshot.ActivityID)
	}
	if snapshot.Status != "awaiting_confirmation" {
		t.Errorf("expected Status=awaiting_confirmation, got %s", snapshot.Status)
	}
	if snapshot.PlanOwner == nil {
		t.Fatal("expected non-nil PlanOwner")
	}
	if snapshot.PlanOwner.AgentName != "main-agent" {
		t.Errorf("expected AgentName=main-agent, got %s", snapshot.PlanOwner.AgentName)
	}
	if len(snapshot.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(snapshot.Participants))
	}
	if snapshot.Participants[0].AgentName != "code-agent" {
		t.Errorf("expected participant=code-agent, got %s", snapshot.Participants[0].AgentName)
	}
	if len(snapshot.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(snapshot.Tasks))
	}
	if len(snapshot.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(snapshot.Warnings))
	}
}

func TestToAGUIActivitySnapshot_Nil(t *testing.T) {
	snapshot := toAGUIActivitySnapshot(nil, "")
	if snapshot.ActivityID != "" {
		t.Errorf("expected empty for nil plan, got %s", snapshot.ActivityID)
	}
}

func TestToAGUITaskSummariesExcludesDependsOn(t *testing.T) {
	tasks := []plan.TaskPlan{
		{TaskID: "task_001", AgentName: "code-agent", TaskContent: "Write code", Priority: 1, RiskLevel: "low", DependsOn: []string{"task_000"}},
	}
	summaries := toAGUITaskSummaries(tasks)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "dependsOn") {
		t.Fatal("toAGUITaskSummaries JSON must NOT contain dependsOn")
	}
	if strings.Contains(string(data), "role") {
		t.Fatal("toAGUITaskSummaries JSON must NOT contain role")
	}
}

func TestToAGUIParticipantsExcludesRoleReason(t *testing.T) {
	ps := []plan.PlanParticipant{
		{AgentName: "code-agent", Role: "executor", Required: true, Selected: true, Reason: "best fit"},
	}
	participants := toAGUIParticipants(ps)
	if len(participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(participants))
	}

	data, err := json.Marshal(participants)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "role") {
		t.Fatal("toAGUIParticipants JSON must NOT contain role")
	}
	if strings.Contains(string(data), "reason") {
		t.Fatal("toAGUIParticipants JSON must NOT contain reason")
	}

	p := participants[0]
	if p.AgentName != "code-agent" {
		t.Errorf("expected agentName=code-agent, got %s", p.AgentName)
	}
	if !p.Required {
		t.Error("expected Required=true")
	}
	if !p.Selected {
		t.Error("expected Selected=true")
	}
}

func TestEmitActivitySnapshotUsesInternalStreamEvent(t *testing.T) {
	orchPlan := &plan.OrchestrationPlan{
		PlanID:        "plan_001",
		ExecutionPath: "single_chat",
		Revision:      1,
		PlanOwner: &plan.PlanOwner{
			Type:      "main_agent",
			AgentName: "main-agent",
		},
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
		Tasks: []plan.TaskPlan{
			{TaskID: "task_001", AgentName: "code-agent", TaskContent: "Write code", Priority: 1, RiskLevel: "low"},
		},
	}

	// Verify the function does not panic and constructs a valid snapshot
	activity := toAGUIActivitySnapshot(orchPlan, "awaiting_confirmation")
	if activity.PlanID != "plan_001" {
		t.Errorf("expected planId=plan_001, got %s", activity.PlanID)
	}
	if activity.Status != "awaiting_confirmation" {
		t.Errorf("expected status=awaiting_confirmation, got %s", activity.Status)
	}
}
