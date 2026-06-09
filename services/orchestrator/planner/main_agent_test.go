package planner

import (
	"context"
	"strings"
	"testing"
)

// fakePlannerModel implements PlannerModel for testing.
type fakePlannerModel struct {
	output  string
	err     error
	latestSystemPrompt string
	latestUserPrompt   string
}

func (f *fakePlannerModel) Generate(_ context.Context, systemPrompt, userPrompt string) (string, error) {
	f.latestSystemPrompt = systemPrompt
	f.latestUserPrompt = userPrompt
	if f.err != nil {
		return "", f.err
	}
	return f.output, nil
}

func validLLMResponse() string {
	return `{"intent":"write a Go HTTP server","mode":"single","confidence":0.9,"steps":[{"agent_name":"code-agent","input":"write a Go HTTP server","reason":"code-agent handles code generation"}],"user_visible_summary":"I will ask code-agent to generate the server."}`
}

// TestProductionMainAgentUsesPlannerModel verifies that MainAgent.Plan calls the
// PlannerModel when model != nil, and falls back to keyword-based matching when
// model is nil.
func TestProductionMainAgentUsesPlannerModel(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}

	t.Run("calls PlannerModel when model is set", func(t *testing.T) {
		fakeModel := &fakePlannerModel{output: validLLMResponse()}
		ma := NewMainAgent(fakeModel, "test-model", lister)

		input := PlannerInput{
			RunID:          "run-1",
			ConversationID: "conv-1",
			UserMessage:    "write a Go HTTP server",
			ExecutionPath:  "main_agent_orchestration",
		}
		orchPlan, err := ma.Plan(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if orchPlan == nil {
			t.Fatal("expected non-nil plan")
		}
		if orchPlan.PlannerSource != "main_agent" {
			t.Errorf("expected PlannerSource='main_agent', got %q", orchPlan.PlannerSource)
		}
		if orchPlan.PlannerModel != "test-model" {
			t.Errorf("expected PlannerModel='test-model', got %q", orchPlan.PlannerModel)
		}
		if orchPlan.Fallback.Enabled {
			t.Error("fallback should NOT be enabled when LLM succeeds")
		}
		if orchPlan.PlanOwner == nil || !orchPlan.PlanOwner.IsMainAgent {
			t.Error("PlanOwner.IsMainAgent must be true")
		}
	})

	t.Run("LLM output includes path and boundary in prompts", func(t *testing.T) {
		fakeModel := &fakePlannerModel{output: validLLMResponse()}
		ma := NewMainAgent(fakeModel, "test-model", lister)

		input := PlannerInput{
			RunID:          "run-2",
			ConversationID: "conv-2",
			UserMessage:    "write Go code",
			ExecutionPath:  "single_chat",
			AllowedAgents:  []string{"code-agent"},
		}
		_, err := ma.Plan(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(fakeModel.latestSystemPrompt, "Execution Context") {
			t.Error("system prompt must contain 'Execution Context'")
		}
		if !strings.Contains(fakeModel.latestSystemPrompt, "single_chat") {
			t.Error("system prompt must contain execution path 'single_chat'")
		}
		if !strings.Contains(fakeModel.latestSystemPrompt, "Allowed Agent Boundary") {
			t.Error("system prompt must contain 'Allowed Agent Boundary'")
		}
		if !strings.Contains(fakeModel.latestUserPrompt, "Boundary Constraint") {
			t.Error("user prompt must contain 'Boundary Constraint'")
		}
	})

	t.Run("falls back to keyword matching when model is nil", func(t *testing.T) {
		ma := NewMainAgent(nil, "", lister)

		input := PlannerInput{
			RunID:          "run-3",
			ConversationID: "conv-3",
			UserMessage:    "write Go code for an HTTP API",
			ExecutionPath:  "main_agent_orchestration",
		}
		orchPlan, err := ma.Plan(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if orchPlan == nil {
			t.Fatal("expected non-nil plan from fallback")
		}
		if orchPlan.PlannerSource != "fallback" {
			t.Errorf("expected PlannerSource='fallback' when model is nil, got %q", orchPlan.PlannerSource)
		}
		if !orchPlan.Fallback.Enabled {
			t.Error("fallback.Enabled should be true when model is nil")
		}
	})

	t.Run("falls back on model error", func(t *testing.T) {
		fakeModel := &fakePlannerModel{err: context.DeadlineExceeded}
		ma := NewMainAgent(fakeModel, "test-model", lister)

		input := PlannerInput{
			RunID:          "run-4",
			ConversationID: "conv-4",
			UserMessage:    "write code",
			ExecutionPath:  "main_agent_orchestration",
		}
		orchPlan, err := ma.Plan(context.Background(), input)
		if err != nil {
			t.Fatalf("fallback should not return error: %v", err)
		}
		if orchPlan.PlannerSource != "fallback" {
			t.Errorf("expected PlannerSource='fallback' on model error, got %q", orchPlan.PlannerSource)
		}
		if !orchPlan.Fallback.Enabled {
			t.Error("fallback.Enabled should be true on model error")
		}
	})
}

// TestMainAgentSchemaCarriesParticipantReasons verifies that participant reasons
// flow from LLM schema through normalizer to MainAgent's buildParticipants.
func TestMainAgentSchemaCarriesParticipantReasons(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}

	// LLM output with reason fields.
	llmOutput := `{"intent":"build full stack","mode":"parallel","confidence":0.85,"steps":[{"agent_name":"code-agent","input":"write API","reason":"chosen for backend work"},{"agent_name":"web-agent","input":"build UI","reason":"chosen for frontend work"}],"user_visible_summary":"Building the app with 2 agents."}`
	fakeModel := &fakePlannerModel{output: llmOutput}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	input := PlannerInput{
		RunID:          "run-reasons",
		ConversationID: "conv-1",
		UserMessage:    "build a full stack app",
		ExecutionPath:  "main_agent_orchestration",
	}
	orchPlan, err := ma.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orchPlan.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(orchPlan.Participants))
	}

	// Participant reasons must come from LLM step reasons.
	if orchPlan.Participants[0].Reason != "chosen for backend work" {
		t.Errorf("participant[0].Reason mismatch: %q", orchPlan.Participants[0].Reason)
	}
	if orchPlan.Participants[1].Reason != "chosen for frontend work" {
		t.Errorf("participant[1].Reason mismatch: %q", orchPlan.Participants[1].Reason)
	}
}

// TestMainAgent_NonAutoPathFiltersAgents ensures that for single_chat/group_chat,
// only agents within AllowedAgents appear in the prompt.
func TestMainAgent_NonAutoPathFiltersAgents(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()} // code-agent, web-agent
	fakeModel := &fakePlannerModel{output: validLLMResponse()}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	input := PlannerInput{
		RunID:          "run-filter",
		ConversationID: "conv-1",
		UserMessage:    "write code",
		ExecutionPath:  "single_chat",
		AllowedAgents:  []string{"code-agent"}, // only code-agent allowed
	}
	_, err := ma.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The prompt should contain code-agent but NOT web-agent in the agent listing.
	// "web-agent" appears in rule descriptions (e.g. "For web UI..."), so we check
	// for "name: web-agent" which only appears in the agent listing block.
	if !strings.Contains(fakeModel.latestSystemPrompt, "name: code-agent") {
		t.Error("prompt must list code-agent in agent block")
	}
	if strings.Contains(fakeModel.latestSystemPrompt, "name: web-agent") {
		t.Error("prompt must NOT list web-agent in agent block for single_chat with code-agent boundary")
	}

	// Verify the boundary section lists only code-agent.
	if !strings.Contains(fakeModel.latestSystemPrompt, "Allowed Agent Boundary") {
		t.Fatal("expected Allowed Agent Boundary section")
	}
	// The boundary lists agents as "  - <name>".
	if !strings.Contains(fakeModel.latestSystemPrompt, "  - code-agent") {
		t.Error("boundary must contain code-agent")
	}
	if strings.Contains(fakeModel.latestSystemPrompt, "  - web-agent") {
		t.Error("boundary must NOT contain web-agent for single_chat with code-agent boundary")
	}
}

// Ensure MainAgent implements Planner.
var _ Planner = (*MainAgent)(nil)
