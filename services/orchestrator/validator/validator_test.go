package validator

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// stubRegistry implements Registry for tests.
type stubRegistry struct {
	agents map[string]registry.AgentEndpoint
}

func newStubRegistry() *stubRegistry {
	return &stubRegistry{
		agents: map[string]registry.AgentEndpoint{
			"code-agent": {
				Name:          "code-agent",
				URL:           "http://code:8080",
				CapabilityIDs: []string{"code_generation"},
				OutputTypes:   []string{"code", "text"},
			},
			"web-agent": {
				Name:          "web-agent",
				URL:           "http://web:8080",
				CapabilityIDs: []string{"web_generation"},
				OutputTypes:   []string{"webpage", "html", "text"},
			},
		},
	}
}

func (s *stubRegistry) Get(name string) (registry.AgentEndpoint, bool) {
	ep, ok := s.agents[name]
	return ep, ok
}

func (s *stubRegistry) Names() []string {
	names := make([]string, 0, len(s.agents))
	for n := range s.agents {
		names = append(names, n)
	}
	return names
}

func validSinglePlan() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         "plan_001",
		RunID:          "run_001",
		ConversationID: "conv_001",
		PlanningMode:   "auto",
		Strategy:       plan.StrategySingle,
		IntentSummary:  "test summary",
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_001",
				AgentName:       "code-agent",
				CapabilityIDs:   []string{"code_generation"},
				TaskContent:     "write Go code",
				ExpectedOutputs: []string{"code"},
				DependsOn:       []string{},
				Priority:        1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			},
		},
		Aggregation: plan.Aggregation{Required: false, Mode: "none"},
		Fallback:    plan.Fallback{Enabled: true, Reason: "rule_default"},
		Validation:  plan.Validation{Validated: false},
	}
}

func validOrderedParallelPlan() *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         "plan_op_001",
		RunID:          "run_op_001",
		ConversationID: "conv_op_001",
		PlanningMode:   "rule",
		Strategy:       plan.StrategyOrderedParallel,
		IntentSummary:  "mixed task",
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_web",
				AgentName:       "web-agent",
				CapabilityIDs:   []string{"web_generation"},
				TaskContent:     "make a page",
				ExpectedOutputs: []string{"webpage"},
				DependsOn:       []string{},
				Priority:        1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			},
			{
				TaskID:          "task_code",
				AgentName:       "code-agent",
				CapabilityIDs:   []string{"code_generation"},
				TaskContent:     "make an API",
				ExpectedOutputs: []string{"code"},
				DependsOn:       []string{},
				Priority:        2,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			},
		},
		Aggregation: plan.Aggregation{Required: true, Mode: "summary"},
		Fallback:    plan.Fallback{Enabled: true, Reason: "rule_default"},
		Validation:  plan.Validation{Validated: false},
	}
}

func TestValidateValidSinglePlan(t *testing.T) {
	v := New(newStubRegistry())
	result := v.Validate(validSinglePlan())
	if !result.Valid {
		t.Error("expected valid plan")
		for _, e := range result.Errors {
			t.Logf("  unexpected error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestValidateValidOrderedParallelPlan(t *testing.T) {
	v := New(newStubRegistry())
	result := v.Validate(validOrderedParallelPlan())
	if !result.Valid {
		t.Error("expected valid ordered_parallel plan")
		for _, e := range result.Errors {
			t.Logf("  unexpected error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestValidateNilReturnsInvalid(t *testing.T) {
	v := New(newStubRegistry())
	result := v.Validate(nil)
	if result.Valid {
		t.Error("expected invalid result for nil plan")
	}
}

func TestValidateMissingPlanId(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.PlanID = ""
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: missing planId")
	}
}

func TestValidateMissingRunId(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.RunID = ""
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: missing runId")
	}
}

func TestValidateMissingConversationId(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.ConversationID = ""
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: missing conversationId")
	}
}

func TestValidateInvalidStrategy(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Strategy = "parallel"
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: unsupported strategy")
	}
}

func TestValidateEmptyTasks(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks = nil
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: empty tasks")
	}
}

func TestValidateTooManyTasks(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks = []plan.TaskPlan{
		{TaskID: "t1", AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, ExpectedOutputs: []string{"code"}, TimeoutMs: 120000, RiskLevel: "low"},
		{TaskID: "t2", AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, ExpectedOutputs: []string{"code"}, TimeoutMs: 120000, RiskLevel: "low"},
		{TaskID: "t3", AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, ExpectedOutputs: []string{"code"}, TimeoutMs: 120000, RiskLevel: "low"},
		{TaskID: "t4", AgentName: "code-agent", CapabilityIDs: []string{"code_generation"}, ExpectedOutputs: []string{"code"}, TimeoutMs: 120000, RiskLevel: "low"},
	}
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: too many tasks")
	}
}

func TestValidateUnknownAgentName(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].AgentName = "unknown-agent"
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: unknown agent name")
	}
}

func TestValidateEmptyAgentName(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].AgentName = ""
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: empty agent name")
	}
}

func TestValidateUnknownCapability(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].CapabilityIDs = []string{"nonexistent_capability"}
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: unknown capability")
	}
}

func TestValidateUnsupportedOutput(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].ExpectedOutputs = []string{"video"}
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: unsupported output type")
	}
}

func TestValidateInvalidDependsOn(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].DependsOn = []string{"task_nonexistent"}
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: dependsOn references unknown task")
	}
}

func TestValidateValidDependsOn(t *testing.T) {
	v := New(newStubRegistry())
	p := validOrderedParallelPlan()
	// task_code depends on task_web (both exist in the plan).
	p.Tasks[1].DependsOn = []string{"task_web"}
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: dependsOn references existing task")
		for _, e := range result.Errors {
			t.Logf("  unexpected error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestValidateTimeoutTooLow(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].TimeoutMs = 1000
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: timeout too low")
	}
}

func TestValidateTimeoutTooHigh(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].TimeoutMs = 200000
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: timeout too high")
	}
}

func TestValidateTimeoutAtLowerBound(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].TimeoutMs = 5000
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: timeout at lower bound 5000")
	}
}

func TestValidateTimeoutAtUpperBound(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].TimeoutMs = 180000
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: timeout at upper bound 180000")
	}
}

func TestValidateInvalidRiskLevel(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].RiskLevel = "critical"
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: unsupported risk level")
	}
}

func TestValidateHighRiskRejected(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].RiskLevel = "high"
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: high risk tasks rejected in v1.0")
	}
}

func TestValidateMediumRiskAccepted(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].RiskLevel = "medium"
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: medium risk accepted")
	}
}

func TestValidateAllErrorsCollected(t *testing.T) {
	v := New(newStubRegistry())
	p := &plan.OrchestrationPlan{
		PlanID: "",
		RunID:  "",
		Strategy: "invalid",
		Tasks: []plan.TaskPlan{
			{
				AgentName:       "unknown-agent",
				CapabilityIDs:   []string{"fake_cap"},
				ExpectedOutputs: []string{"fake_output"},
				DependsOn:       []string{"task_missing"},
				TimeoutMs:       100,
				RiskLevel:       "extreme",
			},
		},
	}
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid")
	}
	// Should have accumulated at least 8 errors:
	// missing planId, missing runId, missing conversationId, invalid strategy,
	// unknown agent, unknown capability, unsupported output, invalid dependsOn,
	// timeout too low, invalid riskLevel
	if len(result.Errors) < 8 {
		t.Errorf("expected at least 8 errors, got %d", len(result.Errors))
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestValidateRegistryNilSkipsAgentChecks(t *testing.T) {
	// When registry is nil, agent/capability/output checks are skipped.
	v := New(nil)
	p := validSinglePlan()
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid when registry is nil (skips agent checks)")
		for _, e := range result.Errors {
			t.Logf("  unexpected error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestValidateRegistryNilStillCatchesStructuralErrors(t *testing.T) {
	v := New(nil)
	p := validSinglePlan()
	p.PlanID = ""
	p.Tasks[0].TimeoutMs = 100
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: structural errors caught even with nil registry")
	}
	foundPlanID := false
	foundTimeout := false
	for _, e := range result.Errors {
		if e.Field == "planId" {
			foundPlanID = true
		}
		if e.Field == "tasks[0].timeoutMs" {
			foundTimeout = true
		}
	}
	if !foundPlanID || !foundTimeout {
		t.Errorf("expected both planId and timeoutMs errors, got %d errors", len(result.Errors))
	}
}

func TestValidateWebAgentOutputTypes(t *testing.T) {
	v := New(newStubRegistry())
	p := validSinglePlan()
	p.Tasks[0].AgentName = "web-agent"
	p.Tasks[0].CapabilityIDs = []string{"web_generation"}
	p.Tasks[0].ExpectedOutputs = []string{"html", "text"}
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: web-agent supports html and text outputs")
		for _, e := range result.Errors {
			t.Logf("  unexpected error: %s: %s", e.Field, e.Message)
		}
	}
}
