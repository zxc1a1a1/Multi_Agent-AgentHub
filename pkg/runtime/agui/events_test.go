package agui

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestActivitySnapshotJSONSchema(t *testing.T) {
	snapshot := ActivitySnapshot{
		ActivityID:    "plan_001",
		ActivityType:  "plan_approval",
		Status:        "awaiting_confirmation",
		ExecutionPath: "main_agent_orchestration",
		PlanID:        "plan_001",
		Revision:      1,
		PlanOwner:     &PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants: []PlanParticipant{
			{AgentName: "code-agent", Required: true, Selected: true},
			{AgentName: "web-agent", Required: false, Selected: false},
		},
		Tasks: []TaskSummary{
			{TaskID: "task_001", AgentName: "code-agent", Content: "Build API", Priority: 1, RiskLevel: "low"},
		},
		AllowedActions: []string{"approve", "revise", "cancel"},
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	requiredFields := []string{"activityId", "activityType", "status", "executionPath", "planId", "revision", "participants", "tasks", "allowedActions"}
	for _, f := range requiredFields {
		if _, ok := parsed[f]; !ok {
			t.Errorf("required field %q missing", f)
		}
	}

	if parsed["activityId"] != "plan_001" {
		t.Errorf("expected activityId=plan_001, got %v", parsed["activityId"])
	}
}

func TestAgentTurnStartedJSONSchema(t *testing.T) {
	evt := AgentTurnStarted{
		Type:      PublicTypeAgentTurnStarted,
		RunID:     "run_001",
		MessageID: "msg_001",
		TurnIndex: 0,
		StepID:    "step_1",
		AgentName: "code-agent",
		Sender:    &EventSender{Type: "agent", Name: "code-agent"},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["turnIndex"].(float64) != 0 {
		t.Errorf("expected turnIndex=0, got %v", parsed["turnIndex"])
	}
	if parsed["agentName"] != "code-agent" {
		t.Errorf("expected agentName=code-agent, got %v", parsed["agentName"])
	}
}

func TestAgentTurnContentJSONSchema(t *testing.T) {
	evt := AgentTurnContent{
		Type:      PublicTypeAgentTurnContent,
		RunID:     "run_001",
		MessageID: "msg_001",
		TurnIndex: 1,
		Delta:     "Here is some code",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["turnIndex"].(float64) != 1 {
		t.Errorf("expected turnIndex=1, got %v", parsed["turnIndex"])
	}
	if parsed["delta"] != "Here is some code" {
		t.Errorf("expected delta, got %v", parsed["delta"])
	}
}

func TestAgentTurnFinishedJSONSchema(t *testing.T) {
	evt := AgentTurnFinished{
		Type:      PublicTypeAgentTurnFinished,
		RunID:     "run_001",
		MessageID: "msg_001",
		TurnIndex: 1,
		AgentName: "code-agent",
		Status:    "completed",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", parsed["status"])
	}
}

func TestAgentTurnEventFieldsAsTopLevel(t *testing.T) {
	evt := Event{
		Type:      PublicTypeAgentTurnStarted,
		RunID:     "run_001",
		MessageID: "msg_001",
		TurnIndex: 3,
		StepID:    "step_2",
		AgentName: "web-agent",
		Sender:    &EventSender{Type: "agent", Name: "web-agent"},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// turnIndex must be a top-level field
	if ti, ok := parsed["turnIndex"].(float64); !ok || int(ti) != 3 {
		t.Errorf("turnIndex must be top-level=3, got %v", parsed["turnIndex"])
	}
	if parsed["stepId"] != "step_2" {
		t.Errorf("stepId must be top-level=step_2, got %v", parsed["stepId"])
	}
	if parsed["agentName"] != "web-agent" {
		t.Errorf("agentName must be top-level=web-agent, got %v", parsed["agentName"])
	}
}

func TestTaskSummaryJSONExcludesDependsOn(t *testing.T) {
	ts := TaskSummary{
		TaskID:    "task_001",
		AgentName: "code-agent",
		Content:   "Write code",
		Priority:  1,
		RiskLevel: "low",
	}

	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if strings.Contains(string(data), "dependsOn") {
		t.Fatal("TaskSummary JSON must NOT contain dependsOn")
	}
}

func TestActivitySnapshotTasksExcludesDependsOn(t *testing.T) {
	snapshot := ActivitySnapshot{
		ActivityID: "plan_001",
		Tasks: []TaskSummary{
			{TaskID: "task_001", AgentName: "code-agent", Content: "Test", Priority: 1, RiskLevel: "low"},
		},
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if strings.Contains(string(data), "dependsOn") {
		t.Fatal("ActivitySnapshot JSON must NOT contain dependsOn in tasks")
	}
}

func TestAgentTurnEventIncludesZeroTurnIndex(t *testing.T) {
	evt := Event{
		Type:      PublicTypeAgentTurnStarted,
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
		t.Fatalf("AGENT_TURN event must include turnIndex even when it is 0; json=%s", string(data))
	}
	if parsed["turnIndex"].(float64) != 0 {
		t.Fatalf("expected turnIndex=0, got %v", parsed["turnIndex"])
	}
}
