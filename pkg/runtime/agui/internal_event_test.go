package agui

import (
	"encoding/json"
	"testing"
)

func TestInternalStreamEventRoundTrip(t *testing.T) {
	activity := ActivitySnapshot{
		ActivityID:    "plan_001",
		ActivityType:  "plan_approval",
		Status:        "awaiting_confirmation",
		ExecutionPath: "single_chat",
		PlanID:        "plan_001",
		Revision:      1,
		PlanOwner:     &PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants: []PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
		},
		Tasks: []TaskSummary{
			{TaskID: "task_001", AgentName: "code-agent", Content: "Write code", Priority: 1, RiskLevel: "low"},
		},
		AllowedActions: []string{"approve", "revise", "cancel"},
	}

	original := InternalStreamEvent{
		Type:      InternalTypeActivitySnapshot,
		RunID:     "run_001",
		MessageID: "msg_001",
		Sender:    &EventSender{Type: "agent", Name: "orchestrator"},
		Activity:  &activity,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var roundTrip InternalStreamEvent
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if roundTrip.Type != InternalTypeActivitySnapshot {
		t.Errorf("expected type=%s, got %s", InternalTypeActivitySnapshot, roundTrip.Type)
	}
	if roundTrip.RunID != "run_001" {
		t.Errorf("expected runId=run_001, got %s", roundTrip.RunID)
	}
	if roundTrip.Activity == nil {
		t.Fatal("expected non-nil Activity after round-trip")
	}
	if roundTrip.Activity.PlanID != "plan_001" {
		t.Errorf("expected planId=plan_001, got %s", roundTrip.Activity.PlanID)
	}
	if roundTrip.Activity.PlanOwner == nil {
		t.Fatal("expected non-nil PlanOwner after round-trip")
	}
	if roundTrip.Activity.PlanOwner.AgentName != "main-agent" {
		t.Errorf("expected agentName=main-agent, got %s", roundTrip.Activity.PlanOwner.AgentName)
	}
}

func TestInternalStreamEventRoundTrip_AgentTurnFields(t *testing.T) {
	original := InternalStreamEvent{
		Type:      InternalTypeAgentTurnStarted,
		RunID:     "run_001",
		MessageID: "msg_001",
		TurnIndex: 2,
		StepID:    "step_1",
		AgentName: "code-agent",
		Sender:    &EventSender{Type: "agent", Name: "code-agent"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var roundTrip InternalStreamEvent
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if roundTrip.TurnIndex != 2 {
		t.Errorf("expected turnIndex=2, got %d", roundTrip.TurnIndex)
	}
	if roundTrip.StepID != "step_1" {
		t.Errorf("expected stepId=step_1, got %s", roundTrip.StepID)
	}
	if roundTrip.AgentName != "code-agent" {
		t.Errorf("expected agentName=code-agent, got %s", roundTrip.AgentName)
	}
}

func TestInternalStreamEventIncludesZeroTurnIndex(t *testing.T) {
	evt := InternalStreamEvent{
		Type:      InternalTypeAgentTurnStarted,
		RunID:     "run_001",
		MessageID: "msg_001",
		TurnIndex: 0,
		StepID:    "step_0",
		AgentName: "code-agent",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := parsed["turnIndex"]; !ok {
		t.Fatalf("internal AGENT_TURN wire event must include turnIndex even when it is 0; json=%s", string(data))
	}
}
