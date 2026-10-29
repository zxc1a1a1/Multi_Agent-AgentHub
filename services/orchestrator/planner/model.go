package planner

import (
	"context"
	"log"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// PlannerModel abstracts the LLM call for intent planning.
// Implementations must be injected; production never constructs a plan without one.
type PlannerModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// AgentInfoLite is the minimal agent information needed by the planner.
type AgentInfoLite struct {
	Name          string
	Description   string
	CapabilityIDs []string
	OutputModes   []string
}

// AgentLister provides the agent catalogue for prompt construction and validation.
type AgentLister interface {
	List() []AgentInfoLite
}

const (
	ValCodeUnknownAgent          = "unknown_agent"
	ValCodeUnsupportedSequential = "unsupported_sequential"
	ValCodeEmptyTasks            = "empty_tasks"
	ValCodeEmptyAgent            = "empty_agent"
	ValCodeOutOfBoundary         = "agent_out_of_boundary"
)

// ValError is a single validation/pipeline failure with a code and message.
type ValError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	TaskIndex int    `json:"task_index,omitempty"`
}

// ValResult is the outcome of plan validation.
type ValResult struct {
	Valid  bool       `json:"valid"`
	Errors []ValError `json:"errors,omitempty"`
}

// PlanValidator validates an OrchestrationPlan produced by the model.
type PlanValidator interface {
	Validate(p *plan.OrchestrationPlan) *ValResult
}

// listerPlanValidator rejects plans referencing agents outside the catalogue.
type listerPlanValidator struct {
	lister AgentLister
}

func newListerPlanValidator(lister AgentLister) PlanValidator {
	if lister == nil {
		return nil
	}
	return &listerPlanValidator{lister: lister}
}

func (v *listerPlanValidator) Validate(p *plan.OrchestrationPlan) *ValResult {
	r := &ValResult{Valid: true}

	if p == nil {
		r.Valid = false
		r.Errors = append(r.Errors, ValError{Code: "invalid_plan", Message: "plan is nil", TaskIndex: -1})
		return r
	}

	agents := v.lister.List()
	known := make(map[string]bool, len(agents))
	for _, a := range agents {
		known[a.Name] = true
	}

	for i, task := range p.Tasks {
		if task.AgentName == "" {
			log.Printf("planner: validation reject: task[%d] has empty agentName", i)
			r.Valid = false
			r.Errors = append(r.Errors, ValError{
				Code:      ValCodeEmptyAgent,
				Message:   "task has empty agentName",
				TaskIndex: i,
			})
			continue
		}
		if !known[task.AgentName] {
			log.Printf("planner: validation reject: agent %q not in registry", task.AgentName)
			r.Valid = false
			r.Errors = append(r.Errors, ValError{
				Code:      ValCodeUnknownAgent,
				Message:   "agent \"" + task.AgentName + "\" not found in registry",
				TaskIndex: i,
			})
		}
	}

	if p.Strategy != plan.StrategyConversational && len(p.Tasks) == 0 {
		log.Printf("planner: validation reject: plan has no tasks")
		r.Valid = false
		r.Errors = append(r.Errors, ValError{
			Code:      ValCodeEmptyTasks,
			Message:   "plan has no tasks",
			TaskIndex: -1,
		})
	}

	if p.Strategy == plan.StrategySequential {
		log.Printf("planner: validation reject: sequential strategy not supported")
		r.Valid = false
		r.Errors = append(r.Errors, ValError{
			Code:      ValCodeUnsupportedSequential,
			Message:   "sequential strategy is not yet supported",
			TaskIndex: -1,
		})
	}

	return r
}
