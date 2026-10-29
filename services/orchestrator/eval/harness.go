// Package eval is the project-internal FakeModel evaluation harness. It proves
// planner contract behavior deterministically; real-model quality is separate.
package eval

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
)

type Scenario struct {
	Name              string
	Message           string
	ExecutionPath     string
	AllowedAgents     []string
	Outputs           []string
	BudgetLimits      *harness.Limits
	PreUsedModelCalls int
	WantError         bool
	WantAgents        []string
	WantStrategy      string
	WantRepair        int
}

type Result struct {
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	Detail     string `json:"detail"`
	ModelCalls int    `json:"modelCalls"`
}

type fakeModel struct {
	outputs []string
	calls   int
}

func (f *fakeModel) Generate(context.Context, string, string) (string, error) {
	f.calls++
	if len(f.outputs) == 0 {
		return "", errors.New("no output")
	}
	idx := f.calls - 1
	if idx >= len(f.outputs) {
		idx = len(f.outputs) - 1
	}
	return f.outputs[idx], nil
}

type lister struct{}

func (lister) List() []planner.AgentInfoLite {
	return []planner.AgentInfoLite{
		{Name: "code-agent", Description: "code"},
		{Name: "web-agent", Description: "web"},
		{Name: "document-agent", Description: "docs"},
	}
}

func Run(ctx context.Context, scenarios []Scenario) []Result {
	results := make([]Result, 0, len(scenarios))
	for _, sc := range scenarios {
		results = append(results, runOne(ctx, sc))
	}
	return results
}

func runOne(ctx context.Context, sc Scenario) Result {
	model := &fakeModel{outputs: sc.Outputs}
	agent := planner.NewMainAgent(model, "fake-model", lister{})
	input := planner.PlannerInput{
		RunID: "eval-" + sc.Name, ConversationID: "eval", UserMessage: sc.Message,
		ExecutionPath: sc.ExecutionPath, AllowedAgents: sc.AllowedAgents,
	}
	if sc.BudgetLimits != nil {
		budget, err := harness.NewBudget(*sc.BudgetLimits)
		if err != nil {
			return Result{Name: sc.Name, Detail: err.Error(), ModelCalls: model.calls}
		}
		if sc.PreUsedModelCalls > 0 {
			if err := budget.ReserveModel(sc.PreUsedModelCalls); err != nil {
				return Result{Name: sc.Name, Detail: err.Error(), ModelCalls: model.calls}
			}
		}
		input.Budget = budget
	}
	plan, err := agent.Plan(ctx, input)
	result := Result{Name: sc.Name, ModelCalls: model.calls}
	if sc.WantError {
		if err == nil {
			result.Detail = "expected error, got plan"
			return result
		}
		result.Passed = true
		result.Detail = "rejected as expected"
		return result
	}
	if err != nil {
		result.Detail = "unexpected error: " + err.Error()
		return result
	}
	if plan == nil {
		result.Detail = "nil plan"
		return result
	}
	if sc.WantRepair != plan.RepairCount {
		result.Detail = fmt.Sprintf("repairCount=%d want %d", plan.RepairCount, sc.WantRepair)
		return result
	}
	if sc.WantStrategy != "" && plan.Strategy != sc.WantStrategy {
		result.Detail = fmt.Sprintf("strategy=%s want %s", plan.Strategy, sc.WantStrategy)
		return result
	}
	var agents []string
	for _, task := range plan.Tasks {
		agents = append(agents, task.AgentName)
	}
	if strings.Join(agents, ",") != strings.Join(sc.WantAgents, ",") {
		result.Detail = fmt.Sprintf("agents=%v want %v", agents, sc.WantAgents)
		return result
	}
	result.Passed = true
	result.Detail = "ok"
	return result
}

func DefaultScenarios() []Scenario {
	return []Scenario{
		{
			Name: "single-code", Message: "write a Go API", ExecutionPath: "main_agent_orchestration",
			Outputs:    []string{`{"intent":"code","mode":"single","steps":[{"agent_name":"code-agent","input":"write a Go API"}]}`},
			WantAgents: []string{"code-agent"},
		},
		{
			Name: "answer-mode", Message: "hi", ExecutionPath: "main_agent_orchestration",
			Outputs:    []string{`{"intent":"greeting","mode":"answer","answer":"hello"}`},
			WantAgents: nil, WantStrategy: "conversational",
		},
		{
			Name: "repair-on-unknown-agent", Message: "write code", ExecutionPath: "main_agent_orchestration",
			Outputs: []string{
				`{"intent":"x","mode":"single","steps":[{"agent_name":"ghost","input":"x"}]}`,
				`{"intent":"x","mode":"single","steps":[{"agent_name":"code-agent","input":"x"}]}`,
			},
			WantAgents: []string{"code-agent"}, WantRepair: 1,
		},
		{
			Name: "reject-after-repair", Message: "write code", ExecutionPath: "main_agent_orchestration",
			Outputs: []string{
				`{"intent":"x","mode":"single","steps":[{"agent_name":"ghost","input":"x"}]}`,
				`{"intent":"x","mode":"single","steps":[{"agent_name":"ghost","input":"x"}]}`,
			},
			WantError: true,
		},
		{
			Name: "reject-out-of-boundary", Message: "write code", ExecutionPath: "single_chat",
			AllowedAgents: []string{"code-agent"},
			Outputs:       []string{`{"intent":"x","mode":"single","steps":[{"agent_name":"web-agent","input":"x"}]}`},
			WantError:     true,
		},
		{
			Name: "budget-exhausted", Message: "write code", ExecutionPath: "main_agent_orchestration",
			Outputs:           []string{`{"intent":"x","mode":"single","steps":[{"agent_name":"code-agent","input":"x"}]}`},
			BudgetLimits:      func() *harness.Limits { l := harness.DefaultLimits(); l.ModelCalls = 1; return &l }(),
			PreUsedModelCalls: 1,
			WantError:         true,
		},
	}
}
