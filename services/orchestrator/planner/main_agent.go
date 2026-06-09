package planner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// MainAgent is the primary Planner for ALL execution paths (single_chat, group_chat,
// main_agent_orchestration). It uses the LLM pipeline (PromptBuilder → PlannerModel →
// Parser → Normalizer → Validator) as the primary plan generation path.
//
// On LLM failure (model error, parse error, validation rejection), it falls back to
// keyword-based capability matching with a WARNING log. The fallback MUST NOT be silent.
//
// MainAgent is the ONLY valid type for Server.mainAgentPlanner. Old RulePlanner and
// LLMPlanner must NOT be injected as mainAgentPlanner.
type MainAgent struct {
	model     PlannerModel
	modelName string
	lister    AgentLister
	validator PlanValidator

	// availableAgents holds all agent names for fallback keyword matching.
	availableAgents []string
}

// NewMainAgent creates a MainAgent that uses the LLM pipeline for plan generation.
// model is the LLM backend (must not be nil in production). modelName is used for
// trace metadata. lister provides registry agent info for prompt construction.
//
// When model is nil, Plan() will fall back to keyword-based matching immediately
// (useful for testing without LLM).
func NewMainAgent(model PlannerModel, modelName string, lister AgentLister) *MainAgent {
	var availableAgents []string
	if lister != nil {
		for _, a := range lister.List() {
			availableAgents = append(availableAgents, a.Name)
		}
	}
	return &MainAgent{
		model:           model,
		modelName:       modelName,
		lister:          lister,
		validator:       newListerPlanValidator(lister),
		availableAgents: availableAgents,
	}
}

// Plan generates an OrchestrationPlan using the LLM pipeline.
//
// For ALL execution paths (single_chat, group_chat, main_agent_orchestration),
// the LLM is the primary path. The prompt is path-aware:
//   - single_chat / group_chat: only agents within AllowedAgents are shown
//   - main_agent_orchestration: all enabled agents are shown
//
// On LLM failure, a WARNING is logged and keyword-based fallback is used.
func (m *MainAgent) Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	if m == nil {
		return nil, fmt.Errorf("main agent is nil")
	}

	isNonAuto := input.ExecutionPath == "single_chat" || input.ExecutionPath == "group_chat"

	// Build path-aware agent list for the prompt.
	promptAgents := m.buildPromptAgents(input, isNonAuto)

	// Try LLM pipeline if we have a model.
	if m.model != nil {
		orchPlan, failures := m.tryLLMPipeline(ctx, input, promptAgents)
		if orchPlan != nil {
			// LLM success — stamp metadata.
			orchPlan.PlannerSource = "main_agent"
			orchPlan.PlannerModel = m.modelName
			if orchPlan.PlanOwner == nil {
				orchPlan.PlanOwner = &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true}
			}
			orchPlan.ExecutionPath = input.ExecutionPath
			orchPlan.AllowedAgents = input.AllowedAgents
			if orchPlan.Revision < 1 {
				orchPlan.Revision = 1
			}
			orchPlan.Fallback = plan.Fallback{Enabled: false}

			// Build participants from tasks (LLM path does not set these in normalizer).
			participants := m.buildParticipants(orchPlan.Tasks, input)
			orchPlan.Participants = participants
			orchPlan.CandidateParticipants = participants
			orchPlan.DefaultSelectedParticipants = participants
			return orchPlan, nil
		}

		// LLM pipeline failed — log WARNING and fall back.
		log.Printf("main_agent: LLM pipeline failed for path=%s with %d error(s), falling back to keyword-based plan",
			input.ExecutionPath, len(failures))
		for _, f := range failures {
			log.Printf("main_agent:   failure: code=%s message=%s", f.Code, f.Message)
		}
	} else {
		log.Printf("main_agent: no PlannerModel configured, using keyword-based fallback for path=%s", input.ExecutionPath)
	}

	// Fallback: keyword-based capability matching (not silent).
	return m.fallbackKeywordPlan(input, isNonAuto), nil
}

// buildPromptAgents returns the AgentInfoLite list to include in the LLM prompt,
// filtered by execution path constraints.
func (m *MainAgent) buildPromptAgents(input PlannerInput, isNonAuto bool) []AgentInfoLite {
	if m.lister == nil {
		return nil
	}

	allAgents := m.lister.List()

	// For non-auto paths (single_chat, group_chat): only show agents within AllowedAgents.
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

	// Auto path: show all enabled agents.
	return allAgents
}

// tryLLMPipeline runs the full LLM pipeline and returns the plan on success.
// Returns (nil, failures) on any pipeline failure.
func (m *MainAgent) tryLLMPipeline(ctx context.Context, input PlannerInput, promptAgents []AgentInfoLite) (*plan.OrchestrationPlan, []ValError) {
	// 1. Build prompts with path-aware agent list.
	pb := m.buildPromptBuilder(promptAgents, input)
	systemPrompt := pb.BuildSystemPrompt()
	userPrompt := pb.BuildUserPrompt(input.UserMessage, input.AgentName)

	// 2. Call the PlannerModel.
	raw, err := m.model.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, []ValError{{Code: "model_error", Message: err.Error(), TaskIndex: -1}}
	}

	// 3. Parse → Normalize → Validate.
	parser := NewPlanParser()
	schema, err := parser.Parse(raw)
	if err != nil {
		return nil, []ValError{{Code: "parse_error", Message: err.Error(), TaskIndex: -1}}
	}

	normalizer := NewPlanNormalizer()
	orchPlan, err := normalizer.Normalize(schema, input.RunID, input.ConversationID, input.PlanningMode, input.ExecutionPath)
	if err != nil {
		return nil, []ValError{{Code: "normalize_error", Message: err.Error(), TaskIndex: -1}}
	}

	// 4. Validate against registry agents.
	if m.validator != nil {
		result := m.validator.Validate(orchPlan)
		if !result.Valid {
			return nil, result.Errors
		}
	}

	// 5. Build participants from tasks.
	orchPlan.Participants = m.buildParticipants(orchPlan.Tasks, input)
	orchPlan.PlanOwner = &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true}

	return orchPlan, nil
}

// buildPromptBuilder creates a PromptBuilder with the given agents, adding
// path-specific context including execution path and allowed boundary.
func (m *MainAgent) buildPromptBuilder(agents []AgentInfoLite, input PlannerInput) *PromptBuilder {
	filteredLister := &staticLister{agents: agents}
	pb := NewPromptBuilder(filteredLister)
	if input.ConversationType != "" {
		pb = pb.WithConversationType(input.ConversationType)
	}
	if input.ExecutionPath != "" {
		pb = pb.WithExecutionPath(input.ExecutionPath)
	}
	if len(input.AllowedAgents) > 0 {
		pb = pb.WithAllowedAgents(input.AllowedAgents)
	}
	return pb
}

// buildParticipants constructs participant list from tasks and input constraints.
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

// fallbackKeywordPlan generates a plan using keyword-based capability matching.
// This is the fallback path — it MUST log a warning before being called.
func (m *MainAgent) fallbackKeywordPlan(input PlannerInput, isNonAuto bool) *plan.OrchestrationPlan {
	// Non-auto paths with explicit AllowedAgents: create one task per agent.
	if isNonAuto && len(input.AllowedAgents) > 0 {
		return m.planWithFixedAgents(input)
	}

	userText := strings.ToLower(input.UserMessage)
	if input.Feedback != "" {
		userText = strings.ToLower(input.UserMessage + " " + input.Feedback)
	}

	scores := m.scoreAgents(userText)
	if len(scores) == 0 {
		return m.fallbackPlan(input)
	}

	// Build tasks for agents with score >= 2 (or at least one).
	tasks := make([]plan.TaskPlan, 0)
	participants := make([]plan.PlanParticipant, 0)
	candidateParticipants := make([]plan.PlanParticipant, 0)
	defaultSelected := make([]plan.PlanParticipant, 0)

	for _, sc := range scores {
		candidateParticipants = append(candidateParticipants, plan.PlanParticipant{
			AgentName: sc.name,
			Required:  sc.score >= 3,
			Selected:  sc.score >= 2,
		})
	}

	hasSelected := false
	for _, sc := range scores {
		if sc.score >= 2 {
			hasSelected = true
			break
		}
	}

	for i, sc := range scores {
		selected := sc.score >= 2
		if !hasSelected && i == 0 {
			selected = true
		}
		if !selected {
			continue
		}
		required := sc.score >= 3 || (!hasSelected && i == 0)
		tasks = append(tasks, plan.TaskPlan{
			TaskID:      fmt.Sprintf("task-%d", len(tasks)+1),
			AgentName:   sc.name,
			TaskContent: input.UserMessage,
			Priority:    len(tasks) + 1,
			TimeoutMs:   60000,
			RiskLevel:   "low",
		})
		participants = append(participants, plan.PlanParticipant{
			AgentName: sc.name,
			Role:      "executor",
			Required:  required,
			Selected:  true,
		})
		defaultSelected = append(defaultSelected, plan.PlanParticipant{
			AgentName: sc.name,
			Required:  required,
			Selected:  true,
		})
	}

	revision := input.Revision
	if revision < 1 {
		revision = 1
	}

	strategy := plan.StrategyOrderedParallel
	if len(tasks) == 1 {
		strategy = plan.StrategySingle
	}
	if len(tasks) == 0 {
		strategy = plan.StrategyConversational
	}

	return &plan.OrchestrationPlan{
		Version:                     "1.0",
		PlanID:                      fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		RunID:                       input.RunID,
		ConversationID:              input.ConversationID,
		ExecutionPath:               input.ExecutionPath,
		Strategy:                    strategy,
		IntentSummary:               fmt.Sprintf("Keyword-based plan with %d agent(s) [FALLBACK]", len(tasks)),
		Tasks:                       tasks,
		PlannerSource:               "fallback",
		AllowedAgents:               input.AllowedAgents,
		Revision:                    revision,
		PlanOwner:                   &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants:                participants,
		CandidateParticipants:       candidateParticipants,
		DefaultSelectedParticipants: defaultSelected,
		Fallback:                    plan.Fallback{Enabled: true, Reason: "llm_pipeline_failed"},
	}
}

// planWithFixedAgents creates one task per agent in AllowedAgents.
// Used ONLY as a fallback for non-auto paths (single_chat, group_chat).
func (m *MainAgent) planWithFixedAgents(input PlannerInput) *plan.OrchestrationPlan {
	agents := input.AllowedAgents

	participants := make([]plan.PlanParticipant, len(agents))
	candidateParticipants := make([]plan.PlanParticipant, len(agents))
	defaultSelected := make([]plan.PlanParticipant, len(agents))
	tasks := make([]plan.TaskPlan, len(agents))

	for i, name := range agents {
		participants[i] = plan.PlanParticipant{AgentName: name, Role: "executor", Required: true, Selected: true}
		candidateParticipants[i] = plan.PlanParticipant{AgentName: name, Required: true, Selected: true}
		defaultSelected[i] = plan.PlanParticipant{AgentName: name, Required: true, Selected: true}
		tasks[i] = plan.TaskPlan{
			TaskID:      fmt.Sprintf("task-%d", i+1),
			AgentName:   name,
			TaskContent: input.UserMessage,
			Priority:    i + 1,
			TimeoutMs:   60000,
			RiskLevel:   "low",
		}
	}

	revision := input.Revision
	if revision < 1 {
		revision = 1
	}

	strategy := plan.StrategyOrderedParallel
	if len(tasks) == 1 {
		strategy = plan.StrategySingle
	}
	if len(tasks) == 0 {
		strategy = plan.StrategyConversational
	}

	return &plan.OrchestrationPlan{
		Version:                     "1.0",
		PlanID:                      fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		RunID:                       input.RunID,
		ConversationID:              input.ConversationID,
		ExecutionPath:               input.ExecutionPath,
		Strategy:                    strategy,
		IntentSummary:               fmt.Sprintf("Fixed-agent plan with %d agent(s) [FALLBACK]", len(agents)),
		Tasks:                       tasks,
		PlannerSource:               "fallback",
		AllowedAgents:               input.AllowedAgents,
		Revision:                    revision,
		PlanOwner:                   &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants:                participants,
		CandidateParticipants:       candidateParticipants,
		DefaultSelectedParticipants: defaultSelected,
		Fallback:                    plan.Fallback{Enabled: true, Reason: "llm_pipeline_failed_fixed_agents"},
	}
}

// fallbackPlan generates a minimal plan when no capability match is found.
func (m *MainAgent) fallbackPlan(input PlannerInput) *plan.OrchestrationPlan {
	participants := make([]plan.PlanParticipant, 0, len(m.availableAgents))
	for _, name := range m.availableAgents {
		participants = append(participants, plan.PlanParticipant{AgentName: name, Required: false, Selected: false})
	}

	revision := input.Revision
	if revision < 1 {
		revision = 1
	}

	return &plan.OrchestrationPlan{
		Version:         "1.0",
		PlanID:          fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		RunID:           input.RunID,
		ConversationID:  input.ConversationID,
		ExecutionPath:   input.ExecutionPath,
		Strategy:        plan.StrategyConversational,
		IntentSummary:   "General assistance [FALLBACK] — specify your task for better agent matching",
		Tasks:           nil,
		PlannerSource:   "fallback",
		AllowedAgents:   input.AllowedAgents,
		Revision:        revision,
		PlanOwner:       &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants:    participants,
		CandidateParticipants: participants,
		Fallback:        plan.Fallback{Enabled: true, Reason: "no_capability_match"},
	}
}

// ---------------------------------------------------------------------------
// Keyword scoring (fallback only)
// ---------------------------------------------------------------------------

type agentScore struct {
	name  string
	score int
}

var capabilityKeywords = map[string][]string{
	"code-agent":     {"code", "api", "server", "backend", "go", "golang", "rest", "http", "json", "function", "script", "program", "database"},
	"web-agent":      {"web", "frontend", "page", "ui", "html", "css", "react", "vue", "component", "browser", "javascript", "typescript"},
	"document-agent": {"document", "doc", "report", "write", "article", "readme", "summary", "guide", "manual"},
}

func (m *MainAgent) scoreAgents(userText string) []agentScore {
	scores := make([]agentScore, 0, len(m.availableAgents))
	for _, name := range m.availableAgents {
		keywords, ok := capabilityKeywords[name]
		if !ok {
			continue
		}
		score := 0
		for _, kw := range keywords {
			if strings.Contains(userText, kw) {
				score++
			}
		}
		scores = append(scores, agentScore{name: name, score: score})
	}

	// Sort by score descending.
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Filter out zero-score agents.
	filtered := make([]agentScore, 0, len(scores))
	for _, s := range scores {
		if s.score > 0 {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// ---------------------------------------------------------------------------
// staticLister — helper for filtered agent lists
// ---------------------------------------------------------------------------

type staticLister struct {
	agents []AgentInfoLite
}

func (s *staticLister) List() []AgentInfoLite {
	return s.agents
}

// Ensure MainAgent implements Planner.
var _ Planner = (*MainAgent)(nil)
