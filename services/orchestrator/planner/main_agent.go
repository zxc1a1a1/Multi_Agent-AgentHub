package planner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// ErrPlannerUnavailable is returned when no model is configured.
var ErrPlannerUnavailable = errors.New("planner model is not configured")

// MainAgent is the only semantic planning entry. It uses
// PromptBuilder → PlannerModel → Parser → Normalizer → Validator with at most
// one model-driven repair. There is no keyword or rule fallback.
type MainAgent struct {
	model     PlannerModel
	modelName string
	lister    AgentLister
	validator PlanValidator
	repairer  *PlanRepairer
}

// NewMainAgent creates a MainAgent. model and lister are required for production.
func NewMainAgent(model PlannerModel, modelName string, lister AgentLister) *MainAgent {
	return &MainAgent{
		model:     model,
		modelName: modelName,
		lister:    lister,
		validator: newListerPlanValidator(lister),
		repairer:  NewPlanRepairer(model),
	}
}

// Plan generates an OrchestrationPlan using the model. It returns an error when
// the model is missing, the call fails, or the output is invalid after one
// repair attempt. It never invents an agent or task.
func (m *MainAgent) Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	if m == nil || m.model == nil {
		return nil, ErrPlannerUnavailable
	}

	isNonAuto := input.ExecutionPath == "single_chat" || input.ExecutionPath == "group_chat"
	promptAgents := m.buildPromptAgents(input, isNonAuto)
	pb := m.buildPromptBuilder(promptAgents, input)

	if input.Budget != nil {
		if err := input.Budget.ReserveModel(1); err != nil {
			return nil, fmt.Errorf("planner model budget: %w", err)
		}
	}
	raw, err := m.model.Generate(ctx, pb.BuildSystemPrompt(), pb.BuildUserPrompt(input.UserMessage, input.AgentName))
	if err != nil {
		return nil, fmt.Errorf("planner model call failed: %w", err)
	}

	orchPlan, failures := m.runPipeline(raw, input)
	if orchPlan == nil {
		if input.Budget != nil {
			if err := input.Budget.RecordRepair(); err != nil {
				return nil, fmt.Errorf("planner repair budget: %w", err)
			}
		}
		repaired, repairErr := m.repairer.Repair(ctx, raw, failures, &staticLister{agents: promptAgents})
		if repairErr != nil {
			return nil, fmt.Errorf("planner output rejected (%s) and repair failed: %w", summarizeFailures(failures), repairErr)
		}
		orchPlan, failures = m.runPipeline(repaired, input)
		if orchPlan == nil {
			return nil, fmt.Errorf("planner output rejected after repair: %s", summarizeFailures(failures))
		}
		orchPlan.RepairCount = 1
	}

	orchPlan.PlannerSource = "main_agent"
	orchPlan.PlannerModel = m.modelName
	if orchPlan.PlanOwner == nil {
		orchPlan.PlanOwner = &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true}
	}
	orchPlan.ExecutionPath = input.ExecutionPath
	orchPlan.AllowedAgents = input.AllowedAgents
	orchPlan.ExecutionOwner = deriveExecutionOwner(input.ExecutionPath, input.AllowedAgents)
	if orchPlan.Revision < 1 {
		orchPlan.Revision = 1
	}
	orchPlan.Fallback = plan.Fallback{Enabled: false}

	participants := m.buildParticipants(orchPlan.Tasks, input)
	orchPlan.Participants = participants
	orchPlan.CandidateParticipants = participants
	orchPlan.DefaultSelectedParticipants = participants
	return orchPlan, nil
}

func (m *MainAgent) runPipeline(raw string, input PlannerInput) (*plan.OrchestrationPlan, []ValError) {
	schema, err := NewPlanParser().Parse(raw)
	if err != nil {
		return nil, []ValError{{Code: "parse_error", Message: err.Error(), TaskIndex: -1}}
	}

	orchPlan, err := NewPlanNormalizer().Normalize(schema, input.RunID, input.ConversationID, input.PlanningMode, input.ExecutionPath)
	if err != nil {
		return nil, []ValError{{Code: "normalize_error", Message: err.Error(), TaskIndex: -1}}
	}

	if m.validator != nil {
		if result := m.validator.Validate(orchPlan); !result.Valid {
			return nil, result.Errors
		}
	}
	if len(input.AllowedAgents) > 0 {
		allowed := make(map[string]bool, len(input.AllowedAgents))
		for _, name := range input.AllowedAgents {
			allowed[strings.ToLower(strings.TrimSpace(name))] = true
		}
		var failures []ValError
		for i, task := range orchPlan.Tasks {
			if !allowed[strings.ToLower(strings.TrimSpace(task.AgentName))] {
				failures = append(failures, ValError{
					Code: ValCodeOutOfBoundary, TaskIndex: i,
					Message: "agent \"" + task.AgentName + "\" is outside the allowed boundary",
				})
			}
		}
		if len(failures) > 0 {
			return nil, failures
		}
	}
	return orchPlan, nil
}

func summarizeFailures(failures []ValError) string {
	if len(failures) == 0 {
		return "unknown validation error"
	}
	parts := make([]string, 0, len(failures))
	for _, f := range failures {
		parts = append(parts, f.Code)
	}
	return strings.Join(parts, ",")
}

func deriveExecutionOwner(executionPath string, allowedAgents []string) *plan.ExecutionOwner {
	switch executionPath {
	case "single_chat":
		agentName := ""
		if len(allowedAgents) > 0 {
			agentName = allowedAgents[0]
		}
		return &plan.ExecutionOwner{Type: "agent", AgentName: agentName}
	case "group_chat":
		return &plan.ExecutionOwner{Type: "group", AgentNames: allowedAgents}
	case "main_agent_orchestration":
		return &plan.ExecutionOwner{Type: "main_agent_orchestration", AgentName: "main-agent", SelectedParticipants: allowedAgents}
	default:
		return nil
	}
}

func (m *MainAgent) buildPromptAgents(input PlannerInput, isNonAuto bool) []AgentInfoLite {
	if m.lister == nil {
		return nil
	}
	allAgents := m.lister.List()
	if isNonAuto && len(input.AllowedAgents) > 0 {
		allowedSet := make(map[string]bool, len(input.AllowedAgents))
		for _, a := range input.AllowedAgents {
			allowedSet[strings.ToLower(strings.TrimSpace(a))] = true
		}
		filtered := make([]AgentInfoLite, 0, len(input.AllowedAgents))
		for _, a := range allAgents {
			if allowedSet[strings.ToLower(strings.TrimSpace(a.Name))] {
				filtered = append(filtered, a)
			}
		}
		return filtered
	}
	return allAgents
}

func (m *MainAgent) buildPromptBuilder(agents []AgentInfoLite, input PlannerInput) *PromptBuilder {
	pb := NewPromptBuilder(&staticLister{agents: agents})
	if input.ConversationType != "" {
		pb = pb.WithConversationType(input.ConversationType)
	}
	if input.ExecutionPath != "" {
		pb = pb.WithExecutionPath(input.ExecutionPath)
	}
	if len(input.AllowedAgents) > 0 {
		pb = pb.WithAllowedAgents(input.AllowedAgents)
	}
	if len(input.MemoryContext) > 0 {
		pb = pb.WithMemoryContext(input.MemoryContext)
	}
	return pb
}

func (m *MainAgent) buildParticipants(tasks []plan.TaskPlan, input PlannerInput) []plan.PlanParticipant {
	seen := make(map[string]bool, len(tasks))
	allowed := make(map[string]bool, len(input.AllowedAgents))
	for _, a := range input.AllowedAgents {
		allowed[strings.ToLower(strings.TrimSpace(a))] = true
	}

	participants := make([]plan.PlanParticipant, 0, len(tasks))
	for _, t := range tasks {
		name := strings.ToLower(strings.TrimSpace(t.AgentName))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		required := len(input.AllowedAgents) == 0 || allowed[name]
		participants = append(participants, plan.PlanParticipant{
			AgentName: t.AgentName,
			Role:      "executor",
			Required:  required,
			Selected:  true,
			Reason:    t.Reason,
		})
	}
	return participants
}

type staticLister struct {
	agents []AgentInfoLite
}

func (s *staticLister) List() []AgentInfoLite {
	return s.agents
}

// Ensure MainAgent implements Planner.
var _ Planner = (*MainAgent)(nil)
