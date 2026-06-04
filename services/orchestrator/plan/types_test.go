package plan

import (
	"encoding/json"
	"testing"
)

func TestOrchestrationPlanJSONRoundTrip(t *testing.T) {
	p := OrchestrationPlan{
		Version:        "v1",
		PlanID:         "plan_001",
		RunID:          "run_001",
		ConversationID: "conv_001",
		PlanningMode:   "rule",
		Strategy:       StrategySingle,
		IntentSummary:  "User wants to generate code",
		Tasks: []TaskPlan{
			{
				TaskID:          "task_001",
				AgentName:       "code-agent",
				CapabilityIDs:   []string{"code_generation"},
				TaskContent:     "Write a Go HTTP server",
				ExpectedOutputs: []string{"code"},
				DependsOn:       []string{},
				Priority:        1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			},
		},
		Aggregation: Aggregation{Required: false, Mode: "none"},
		Fallback:    Fallback{Enabled: true, Reason: "rule_default"},
		Validation:  Validation{Validated: false},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got OrchestrationPlan
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != "v1" {
		t.Errorf("expected v1, got %q", got.Version)
	}
	if got.PlanID != "plan_001" {
		t.Errorf("expected plan_001, got %q", got.PlanID)
	}
	if got.Strategy != StrategySingle {
		t.Errorf("expected single, got %q", got.Strategy)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got.Tasks))
	}
	if got.Tasks[0].AgentName != "code-agent" {
		t.Errorf("expected code-agent, got %q", got.Tasks[0].AgentName)
	}
	if got.Validation.Validated {
		t.Error("expected validated=false")
	}
}

func TestStrategyConstants(t *testing.T) {
	if StrategySingle != "single" {
		t.Errorf("expected 'single', got %q", StrategySingle)
	}
	if StrategyOrderedParallel != "ordered_parallel" {
		t.Errorf("expected 'ordered_parallel', got %q", StrategyOrderedParallel)
	}
	if StrategySequential != "sequential" {
		t.Errorf("expected 'sequential', got %q", StrategySequential)
	}
}

func TestNewPlanID(t *testing.T) {
	id1 := NewPlanID()
	id2 := NewPlanID()
	if id1 == "" {
		t.Error("expected non-empty plan id")
	}
	if id1 == id2 {
		t.Error("expected unique plan ids")
	}
}

func TestOrchestrationPlanEmptyTasks(t *testing.T) {
	p := OrchestrationPlan{
		Version: "v1",
		PlanID:  "plan_001",
		Tasks:   nil,
	}
	if len(p.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(p.Tasks))
	}
}

func TestTaskPlanDependsOn(t *testing.T) {
	tp := TaskPlan{
		TaskID:    "task_002",
		AgentName: "code-agent",
		DependsOn: []string{"task_001"},
	}
	if len(tp.DependsOn) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(tp.DependsOn))
	}
	if tp.DependsOn[0] != "task_001" {
		t.Errorf("expected task_001, got %q", tp.DependsOn[0])
	}
}
