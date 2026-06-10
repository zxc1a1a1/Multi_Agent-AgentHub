package validator

import (
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func makeTestPlanWithParticipants(path string, tasks []plan.TaskPlan, participants []plan.PlanParticipant, allowedAgents []string) *plan.OrchestrationPlan {
	return &plan.OrchestrationPlan{
		Version:        "1.0",
		PlanID:         "plan-test",
		RunID:          "run-test",
		ConversationID: "conv-test",
		ExecutionPath:  path,
		Strategy:       plan.StrategySingle,
		IntentSummary:  "test",
		Tasks:          tasks,
		Participants:   participants,
		AllowedAgents:  allowedAgents,
	}
}

func makeTask(agentName, taskID string) plan.TaskPlan {
	return plan.TaskPlan{
		TaskID:      taskID,
		AgentName:   agentName,
		TaskContent: "do something",
		Priority:    1,
		TimeoutMs:   60000,
		RiskLevel:   "low",
	}
}

func makeParticipant(agentName string, required, selected bool) plan.PlanParticipant {
	return plan.PlanParticipant{
		AgentName: agentName,
		Role:      "executor",
		Required:  required,
		Selected:  selected,
	}
}

// ---------------------------------------------------------------------------
// Participants from allowed boundary
// ---------------------------------------------------------------------------

func TestPathValidator_ParticipantsWithinBoundary(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true)},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: participant within boundary")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestPathValidator_ParticipantOutsideBoundary(t *testing.T) {
	v := NewPathAwareValidator("group_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("group_chat",
		[]plan.TaskPlan{makeTask("web-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("web-agent", true, true)},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: participant outside boundary")
	}
	// Both task and participant should be flagged.
	foundTask := false
	foundParticipant := false
	for _, e := range result.Errors {
		if strings.Contains(e.Field, "tasks[") && e.Code == "PATH_AGENT_OUT_OF_BOUNDARY" {
			foundTask = true
		}
		if strings.Contains(e.Field, "participants[") && e.Code == "PATH_PARTICIPANT_OUT_OF_BOUNDARY" {
			foundParticipant = true
		}
	}
	if !foundTask {
		t.Error("expected task boundary error")
	}
	if !foundParticipant {
		t.Error("expected participant boundary error")
	}
}

// ---------------------------------------------------------------------------
// Tasks assignedAgent within boundary
// ---------------------------------------------------------------------------

func TestPathValidator_TaskAgentWithinBoundary(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent", "web-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("code-agent", "t1"), makeTask("web-agent", "t2")},
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true), makeParticipant("web-agent", true, true)},
		[]string{"code-agent", "web-agent"},
	)
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: all task agents within boundary")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestPathValidator_TaskAgentOutsideBoundary(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("unknown-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("unknown-agent", true, true)},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: task agent outside boundary")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_AGENT_OUT_OF_BOUNDARY" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_AGENT_OUT_OF_BOUNDARY error")
	}
}

// ---------------------------------------------------------------------------
// single_chat: participants must be only the current agent
// ---------------------------------------------------------------------------

func TestPathValidator_SingleChatValid(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true)},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: single_chat with correct agent")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestPathValidator_SingleChatTooManyParticipants(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1"), makeTask("web-agent", "t2")},
		[]plan.PlanParticipant{
			makeParticipant("code-agent", true, true),
			makeParticipant("web-agent", true, true),
		},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: single_chat with too many participants")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_SINGLE_CHAT_TOO_MANY" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_SINGLE_CHAT_TOO_MANY error")
	}
}

// ---------------------------------------------------------------------------
// group_chat: no boundary-external agents
// ---------------------------------------------------------------------------

func TestPathValidator_GroupChatAllWithinBoundary(t *testing.T) {
	v := NewPathAwareValidator("group_chat", []string{"code-agent", "web-agent"})
	p := makeTestPlanWithParticipants("group_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1"), makeTask("web-agent", "t2")},
		[]plan.PlanParticipant{
			makeParticipant("code-agent", true, true),
			makeParticipant("web-agent", true, true),
		},
		[]string{"code-agent", "web-agent"},
	)
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: group_chat with all agents within boundary")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

func TestPathValidator_GroupChatBoundaryExternal(t *testing.T) {
	v := NewPathAwareValidator("group_chat", []string{"code-agent", "web-agent"})
	p := makeTestPlanWithParticipants("group_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1"), makeTask("document-agent", "t2")},
		[]plan.PlanParticipant{
			makeParticipant("code-agent", true, true),
			makeParticipant("document-agent", true, true),
		},
		[]string{"code-agent", "web-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: group_chat with boundary-external agent")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_AGENT_OUT_OF_BOUNDARY" || e.Code == "PATH_PARTICIPANT_OUT_OF_BOUNDARY" {
			found = true
		}
	}
	if !found {
		t.Error("expected boundary violation error")
	}
}

// ---------------------------------------------------------------------------
// auto (main_agent_orchestration): no unknown/disabled agents
// ---------------------------------------------------------------------------

func TestPathValidator_AutoPathUnknownAgent(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent", "web-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("gibberish-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("gibberish-agent", true, true)},
		[]string{"code-agent", "web-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: auto path with unknown agent")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_AGENT_OUT_OF_BOUNDARY" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_AGENT_OUT_OF_BOUNDARY for auto path unknown agent")
	}
}

func TestPathValidator_AutoPathValidAgents(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent", "web-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("code-agent", "t1"), makeTask("web-agent", "t2")},
		[]plan.PlanParticipant{
			makeParticipant("code-agent", true, true),
			makeParticipant("web-agent", false, true),
		},
		[]string{"code-agent", "web-agent"},
	)
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: auto path with known agents")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// Missing participants → failed
// ---------------------------------------------------------------------------

func TestPathValidator_MissingParticipants(t *testing.T) {
	v := NewPathAwareValidator("group_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("group_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		nil, // no participants
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: missing participants")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_MISSING_PARTICIPANTS" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_MISSING_PARTICIPANTS error")
	}
}

func TestPathValidator_EmptyParticipantsWithTasks(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		[]plan.PlanParticipant{}, // empty participants
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: empty participants with tasks")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_MISSING_PARTICIPANTS" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_MISSING_PARTICIPANTS error for empty participants")
	}
}

// ---------------------------------------------------------------------------
// Missing steps → failed
// ---------------------------------------------------------------------------

func TestPathValidator_MissingSteps(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		nil, // no tasks
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true)},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: missing steps")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_MISSING_STEPS" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_MISSING_STEPS error")
	}
}

// ---------------------------------------------------------------------------
// Strategy missing → WARNING only (not blocking)
// ---------------------------------------------------------------------------

func TestPathValidator_StrategyMissingIsWarning(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true)},
		[]string{"code-agent"},
	)
	p.Strategy = "" // missing strategy
	result := v.Validate(p)
	// Must still be Valid=true (warning only)
	if !result.Valid {
		t.Error("expected valid: strategy missing should be WARNING only, not blocking")
	}
	found := false
	for _, w := range result.Warnings {
		if w.Code == "PATH_STRATEGY_MISSING" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_STRATEGY_MISSING warning")
	}
}

// ---------------------------------------------------------------------------
// Strategy unknown → WARNING only (not blocking)
// ---------------------------------------------------------------------------

func TestPathValidator_StrategyUnknownIsWarning(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true)},
		[]string{"code-agent"},
	)
	p.Strategy = "fancy_new_strategy" // unknown strategy
	result := v.Validate(p)
	// Must still be Valid=true (warning only)
	if !result.Valid {
		t.Error("expected valid: unknown strategy should be WARNING only, not blocking")
	}
	found := false
	for _, w := range result.Warnings {
		if w.Code == "PATH_STRATEGY_UNKNOWN" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_STRATEGY_UNKNOWN warning")
	}
}

// ---------------------------------------------------------------------------
// Empty agentName in tasks → fail
// ---------------------------------------------------------------------------

func TestPathValidator_EmptyTaskAgentName(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{makeTask("", "t1")}, // empty agentName
		[]plan.PlanParticipant{makeParticipant("code-agent", true, true)},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: empty task agentName")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "PATH_EMPTY_AGENT" {
			found = true
		}
	}
	if !found {
		t.Error("expected PATH_EMPTY_AGENT error")
	}
}

// ---------------------------------------------------------------------------
// Nil plan
// ---------------------------------------------------------------------------

func TestPathValidator_NilPlan(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	result := v.Validate(nil)
	if result.Valid {
		t.Error("expected invalid: nil plan")
	}
}

// ---------------------------------------------------------------------------
// Empty allowedAgents (boundary not enforced → all agents allowed)
// ---------------------------------------------------------------------------

func TestPathValidator_EmptyAllowedAgentsSkipsBoundaryCheck(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", nil)
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("any-agent", "t1")},
		[]plan.PlanParticipant{makeParticipant("any-agent", true, true)},
		nil,
	)
	result := v.Validate(p)
	// When allowedAgents is empty, boundary checks are skipped.
	if !result.Valid {
		t.Error("expected valid: empty allowedAgents skips boundary enforcement")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// Conversational path: no tasks is allowed
// ---------------------------------------------------------------------------

func TestPathValidator_ConversationalNoStepsOK(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		nil, // no tasks — allowed for conversational
		[]plan.PlanParticipant{makeParticipant("code-agent", false, false)},
		[]string{"code-agent"},
	)
	p.Strategy = plan.StrategyConversational
	result := v.Validate(p)
	if !result.Valid {
		t.Error("expected valid: conversational plan with no tasks")
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s", e.Field, e.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// Multiple errors collected
// ---------------------------------------------------------------------------

func TestPathValidator_MultipleErrorsCollected(t *testing.T) {
	v := NewPathAwareValidator("single_chat", []string{"code-agent"})
	p := makeTestPlanWithParticipants("single_chat",
		[]plan.TaskPlan{
			makeTask("web-agent", "t1"),   // outside boundary
			makeTask("unknown-xyz", "t2"), // outside boundary
			makeTask("", "t3"),            // empty agentName
		},
		[]plan.PlanParticipant{
			makeParticipant("web-agent", true, true),
			makeParticipant("code-agent", true, true),
		},
		[]string{"code-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: multiple violations")
	}
	// Should have: 2x agent out of boundary (tasks), 1x empty agent (task),
	// 1x participant out of boundary, 1x too many participants (single_chat)
	if len(result.Errors) < 4 {
		t.Errorf("expected at least 4 errors, got %d", len(result.Errors))
		for _, e := range result.Errors {
			t.Logf("  error: %s: %s: %s", e.Field, e.Code, e.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// MainAgent path requires at least one participant
// ---------------------------------------------------------------------------

func TestPathValidator_AutoPathNoParticipants(t *testing.T) {
	v := NewPathAwareValidator("main_agent_orchestration", []string{"code-agent", "web-agent"})
	p := makeTestPlanWithParticipants("main_agent_orchestration",
		[]plan.TaskPlan{makeTask("code-agent", "t1")},
		nil, // no participants
		[]string{"code-agent", "web-agent"},
	)
	result := v.Validate(p)
	if result.Valid {
		t.Error("expected invalid: auto path with no participants")
	}
}
