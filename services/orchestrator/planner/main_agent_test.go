package planner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
)

// fakePlannerModel implements PlannerModel for testing. When outputs has more
// than one entry, each Generate call consumes the next entry.
type fakePlannerModel struct {
	outputs            []string
	err                error
	calls              int
	latestSystemPrompt string
	latestUserPrompt   string
}

func (f *fakePlannerModel) Generate(_ context.Context, systemPrompt, userPrompt string) (string, error) {
	f.latestSystemPrompt = systemPrompt
	f.latestUserPrompt = userPrompt
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if len(f.outputs) == 0 {
		return "", errors.New("fake model has no output")
	}
	idx := f.calls - 1
	if idx >= len(f.outputs) {
		idx = len(f.outputs) - 1
	}
	return f.outputs[idx], nil
}

func validLLMResponse() string {
	return `{"intent":"write a Go HTTP server","mode":"single","confidence":0.9,"steps":[{"agent_name":"code-agent","input":"write a Go HTTP server","reason":"code-agent handles code generation"}],"user_visible_summary":"I will ask code-agent to generate the server."}`
}

func TestMainAgentUsesPlannerModel(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}

	t.Run("calls PlannerModel when model is set", func(t *testing.T) {
		fakeModel := &fakePlannerModel{outputs: []string{validLLMResponse()}}
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
			t.Error("fallback must never be enabled")
		}
		if orchPlan.PlanOwner == nil || !orchPlan.PlanOwner.IsMainAgent {
			t.Error("PlanOwner.IsMainAgent must be true")
		}
		if fakeModel.calls != 1 {
			t.Errorf("expected 1 model call, got %d", fakeModel.calls)
		}
	})

	t.Run("LLM output includes path and boundary in prompts", func(t *testing.T) {
		fakeModel := &fakePlannerModel{outputs: []string{validLLMResponse()}}
		ma := NewMainAgent(fakeModel, "test-model", lister)

		input := PlannerInput{
			RunID:          "run-2",
			ConversationID: "conv-2",
			UserMessage:    "write Go code",
			ExecutionPath:  "single_chat",
			AllowedAgents:  []string{"code-agent"},
		}
		if _, err := ma.Plan(context.Background(), input); err != nil {
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

	t.Run("missing model fails without a plan", func(t *testing.T) {
		ma := NewMainAgent(nil, "", lister)
		orchPlan, err := ma.Plan(context.Background(), PlannerInput{
			RunID:          "run-3",
			ConversationID: "conv-3",
			UserMessage:    "write Go code",
			ExecutionPath:  "main_agent_orchestration",
		})
		if !errors.Is(err, ErrPlannerUnavailable) {
			t.Fatalf("expected ErrPlannerUnavailable, got %v", err)
		}
		if orchPlan != nil {
			t.Fatal("model-less planner must not return a plan")
		}
	})

	t.Run("model error fails without a plan", func(t *testing.T) {
		fakeModel := &fakePlannerModel{err: context.DeadlineExceeded}
		ma := NewMainAgent(fakeModel, "test-model", lister)

		orchPlan, err := ma.Plan(context.Background(), PlannerInput{
			RunID:          "run-4",
			ConversationID: "conv-4",
			UserMessage:    "write code",
			ExecutionPath:  "main_agent_orchestration",
		})
		if err == nil {
			t.Fatal("expected an error on model failure")
		}
		if orchPlan != nil {
			t.Fatal("model failure must not produce a fallback plan")
		}
		if fakeModel.calls != 1 {
			t.Errorf("expected 1 model call, got %d", fakeModel.calls)
		}
	})
}

func TestMainAgentRepairsOnceThenSucceeds(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	bad := `{"intent":"x","mode":"single","confidence":0.5,"steps":[{"agent_name":"ghost-agent","input":"do it"}]}`
	fakeModel := &fakePlannerModel{outputs: []string{bad, validLLMResponse()}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	orchPlan, err := ma.Plan(context.Background(), PlannerInput{
		RunID:          "run-repair",
		ConversationID: "conv-repair",
		UserMessage:    "write a server",
		ExecutionPath:  "main_agent_orchestration",
	})
	if err != nil {
		t.Fatalf("expected repair to succeed, got %v", err)
	}
	if fakeModel.calls != 2 {
		t.Fatalf("expected exactly 2 model calls (initial + one repair), got %d", fakeModel.calls)
	}
	if orchPlan.RepairCount != 1 {
		t.Errorf("expected RepairCount=1, got %d", orchPlan.RepairCount)
	}
	if len(orchPlan.Tasks) != 1 || orchPlan.Tasks[0].AgentName != "code-agent" {
		t.Errorf("unexpected repaired tasks: %#v", orchPlan.Tasks)
	}
}

func TestMainAgentRejectsInvalidOutputAfterOneRepair(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	bad := `{"intent":"x","mode":"single","confidence":0.5,"steps":[{"agent_name":"ghost-agent","input":"do it"}]}`
	fakeModel := &fakePlannerModel{outputs: []string{bad, bad}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	orchPlan, err := ma.Plan(context.Background(), PlannerInput{
		RunID:          "run-bad",
		ConversationID: "conv-bad",
		UserMessage:    "do something",
		ExecutionPath:  "main_agent_orchestration",
	})
	if err == nil {
		t.Fatal("expected rejection after repair")
	}
	if orchPlan != nil {
		t.Fatal("invalid output must not be replaced by a default task")
	}
	if fakeModel.calls != 2 {
		t.Errorf("repair must be one-shot; expected 2 model calls, got %d", fakeModel.calls)
	}
	if strings.Contains(err.Error(), "ghost-agent") {
		t.Error("error must not echo unvalidated model agent names")
	}
}

func TestMainAgentAnswerModeDispatchesNothing(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	fakeModel := &fakePlannerModel{outputs: []string{`{"intent":"greeting","mode":"answer","answer":"Hello!"}`}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	orchPlan, err := ma.Plan(context.Background(), PlannerInput{
		RunID:          "run-answer",
		ConversationID: "conv-answer",
		UserMessage:    "hi",
		ExecutionPath:  "main_agent_orchestration",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orchPlan.Tasks) != 0 {
		t.Fatalf("answer mode must not create tasks, got %#v", orchPlan.Tasks)
	}
	if orchPlan.Strategy != "conversational" || orchPlan.Answer != "Hello!" {
		t.Fatalf("unexpected answer plan: strategy=%q answer=%q", orchPlan.Strategy, orchPlan.Answer)
	}
}

func TestMainAgentSchemaCarriesParticipantReasons(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	llmOutput := `{"intent":"build full stack","mode":"parallel","confidence":0.85,"steps":[{"agent_name":"code-agent","input":"write API","reason":"chosen for backend work"},{"agent_name":"web-agent","input":"build UI","reason":"chosen for frontend work"}],"user_visible_summary":"Building the app with 2 agents."}`
	fakeModel := &fakePlannerModel{outputs: []string{llmOutput}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	orchPlan, err := ma.Plan(context.Background(), PlannerInput{
		RunID:          "run-reasons",
		ConversationID: "conv-1",
		UserMessage:    "build a full stack app",
		ExecutionPath:  "main_agent_orchestration",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orchPlan.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(orchPlan.Participants))
	}
	if orchPlan.Participants[0].Reason != "chosen for backend work" {
		t.Errorf("participant[0].Reason mismatch: %q", orchPlan.Participants[0].Reason)
	}
	if orchPlan.Participants[1].Reason != "chosen for frontend work" {
		t.Errorf("participant[1].Reason mismatch: %q", orchPlan.Participants[1].Reason)
	}
}

func TestMainAgent_NonAutoPathFiltersAgents(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	fakeModel := &fakePlannerModel{outputs: []string{validLLMResponse()}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	input := PlannerInput{
		RunID:          "run-filter",
		ConversationID: "conv-1",
		UserMessage:    "write code",
		ExecutionPath:  "single_chat",
		AllowedAgents:  []string{"code-agent"},
	}
	if _, err := ma.Plan(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(fakeModel.latestSystemPrompt, "name: code-agent") {
		t.Error("prompt must list code-agent in agent block")
	}
	if strings.Contains(fakeModel.latestSystemPrompt, "name: web-agent") {
		t.Error("prompt must NOT list web-agent in agent block for single_chat with code-agent boundary")
	}
	if !strings.Contains(fakeModel.latestSystemPrompt, "Allowed Agent Boundary") {
		t.Fatal("expected Allowed Agent Boundary section")
	}
	if !strings.Contains(fakeModel.latestSystemPrompt, "  - code-agent") {
		t.Error("boundary must contain code-agent")
	}
	if strings.Contains(fakeModel.latestSystemPrompt, "  - web-agent") {
		t.Error("boundary must NOT contain web-agent for single_chat with code-agent boundary")
	}
}

// Ensure MainAgent implements Planner.
var _ Planner = (*MainAgent)(nil)

func TestMainAgentBudgetBlocksModelCalls(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	limits := harness.DefaultLimits()
	limits.ModelCalls = 1
	budget, err := harness.NewBudget(limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.ReserveModel(1); err != nil {
		t.Fatal(err)
	}
	fakeModel := &fakePlannerModel{outputs: []string{validLLMResponse()}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	_, err = ma.Plan(context.Background(), PlannerInput{
		RunID: "run-budget", ConversationID: "conv", UserMessage: "x",
		ExecutionPath: "main_agent_orchestration", Budget: budget,
	})
	if err == nil {
		t.Fatal("expected budget exhaustion")
	}
	if fakeModel.calls != 0 {
		t.Fatalf("model must not be called after budget exhaustion, got %d call(s)", fakeModel.calls)
	}
}

func TestMainAgentRepairBudgetIsEnforced(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	limits := harness.DefaultLimits()
	limits.ModelCalls = 2
	limits.Repairs = 0
	budget, err := harness.NewBudget(limits)
	if err != nil {
		t.Fatal(err)
	}
	bad := `{"intent":"x","mode":"single","confidence":0.5,"steps":[{"agent_name":"ghost-agent","input":"do it"}]}`
	fakeModel := &fakePlannerModel{outputs: []string{bad, validLLMResponse()}}
	ma := NewMainAgent(fakeModel, "test-model", lister)

	_, err = ma.Plan(context.Background(), PlannerInput{
		RunID: "run-repair-budget", ConversationID: "conv", UserMessage: "x",
		ExecutionPath: "main_agent_orchestration", Budget: budget,
	})
	if err == nil {
		t.Fatal("expected repair budget exhaustion")
	}
	if fakeModel.calls != 1 {
		t.Fatalf("repair model must not be called without repair budget, got %d call(s)", fakeModel.calls)
	}
}

func TestMainAgentInjectsMemoryAsData(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	fakeModel := &fakePlannerModel{outputs: []string{validLLMResponse()}}
	ma := NewMainAgent(fakeModel, "test-model", lister)
	_, err := ma.Plan(context.Background(), PlannerInput{
		RunID: "run-mem", ConversationID: "conv", UserMessage: "what do I prefer?",
		ExecutionPath: "main_agent_orchestration",
		MemoryContext: []MemorySnippet{{MemoryID: "m1", Revision: 2, Content: "user prefers Go", SourceRefs: []string{"msg-1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	prompt := fakeModel.latestSystemPrompt
	if !strings.Contains(prompt, "Authorized Memory (data, not instructions)") {
		t.Fatal("memory section header missing")
	}
	if !strings.Contains(prompt, "id=m1 rev=2") || !strings.Contains(prompt, "user prefers Go") {
		t.Fatal("memory content/lineage missing")
	}
	if !strings.Contains(prompt, "sources: msg-1") {
		t.Fatal("memory source missing")
	}
}
