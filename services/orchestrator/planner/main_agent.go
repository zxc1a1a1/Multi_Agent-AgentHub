package planner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// MainAgent is an internal Orchestrator component (NOT an A2A agent, NOT in registry).
// It implements Planner for the main_agent_orchestration execution path:
// recommends candidateParticipants, defaultSelectedParticipants, requiredParticipants,
// and generates a production plan based on rule-based capability matching.
type MainAgent struct {
	availableAgents []string
}

// NewMainAgent creates a rule-based MainAgent that matches user intent to
// available agent capabilities.
func NewMainAgent(availableAgents []string) *MainAgent {
	names := make([]string, len(availableAgents))
	copy(names, availableAgents)
	return &MainAgent{availableAgents: names}
}

// capabilityKeywords maps agent names to keywords that indicate a user needs that agent.
var capabilityKeywords = map[string][]string{
	"code-agent":     {"code", "api", "server", "backend", "go", "golang", "rest", "http", "json", "function", "script", "program", "database"},
	"web-agent":      {"web", "frontend", "page", "ui", "html", "css", "react", "vue", "component", "browser", "javascript", "typescript"},
	"document-agent": {"document", "doc", "report", "write", "article", "readme", "summary", "guide", "manual"},
}

// Plan examines the user message against available agent capabilities and produces
// an orchestration plan with candidate participant lists and task assignments.
func (m *MainAgent) Plan(_ context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	if m == nil {
		return nil, fmt.Errorf("main agent is nil")
	}

	userText := strings.ToLower(input.UserMessage)
	if input.Feedback != "" {
		userText = strings.ToLower(input.UserMessage + " " + input.Feedback)
	}

	// Score each available agent against the user's request.
	scores := m.scoreAgents(userText)
	if len(scores) == 0 {
		// No capability match — generate a basic plan with all available agents as candidates.
		return m.fallbackPlan(input)
	}

	// Build candidate participants from scored agents.
	candidateParticipants := make([]plan.PlanParticipant, 0, len(scores))
	for _, sc := range scores {
		candidateParticipants = append(candidateParticipants, plan.PlanParticipant{
			AgentName: sc.name,
			Required:  sc.score >= 3,
			Selected:  sc.score >= 2,
		})
	}

	// Determine required agents (high-confidence matches with score >= 3).
	requiredParticipants := make([]string, 0)
	for _, sc := range scores {
		if sc.score >= 3 {
			requiredParticipants = append(requiredParticipants, sc.name)
		}
	}
	if len(requiredParticipants) == 0 && len(scores) > 0 {
		// At least one agent is always required.
		requiredParticipants = append(requiredParticipants, scores[0].name)
		if !candidateParticipants[0].Required {
			candidateParticipants[0].Required = true
		}
	}

	// Default selected: agents with score >= 2, but always at least one.
	defaultSelected := make([]plan.PlanParticipant, 0)
	for _, sc := range scores {
		if sc.score >= 2 {
			defaultSelected = append(defaultSelected, plan.PlanParticipant{
				AgentName: sc.name,
				Required:  sc.score >= 3,
				Selected:  true,
			})
		}
	}
	if len(defaultSelected) == 0 && len(scores) > 0 {
		defaultSelected = append(defaultSelected, plan.PlanParticipant{
			AgentName: scores[0].name,
			Required:  true,
			Selected:  true,
		})
	}

	// Build tasks for default selected agents.
	tasks := make([]plan.TaskPlan, 0, len(defaultSelected))
	for i, ds := range defaultSelected {
		tasks = append(tasks, plan.TaskPlan{
			TaskID:      fmt.Sprintf("task-%d", i+1),
			AgentName:   ds.AgentName,
			TaskContent: input.UserMessage,
			Priority:    i + 1,
			TimeoutMs:   60000,
			RiskLevel:   "low",
		})
	}

	// Build participants list (all candidates).
	participants := make([]plan.PlanParticipant, 0, len(candidateParticipants))
	for _, c := range candidateParticipants {
		participants = append(participants, plan.PlanParticipant{
			AgentName: c.AgentName,
			Role:      "executor",
			Required:  c.Required,
			Selected:  c.Selected,
		})
	}

	revision := input.Revision
	if revision < 1 {
		revision = 1
	}

	// Choose strategy based on task count.
	strategy := plan.StrategyOrderedParallel
	if len(tasks) == 1 {
		strategy = plan.StrategySingle
	}

	return &plan.OrchestrationPlan{
		Version:                    "1.0",
		PlanID:                     fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		RunID:                      input.RunID,
		ConversationID:             input.ConversationID,
		ExecutionPath:              input.ExecutionPath,
		Strategy:                   strategy,
		IntentSummary:              fmt.Sprintf("Auto-orchestrated plan with %d agent(s)", len(defaultSelected)),
		Tasks:                      tasks,
		PlannerSource:              "main_agent",
		AllowedAgents:              input.AllowedAgents,
		Revision:                   revision,
		PlanOwner:                  &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants:               participants,
		CandidateParticipants:      candidateParticipants,
		DefaultSelectedParticipants: defaultSelected,
	}, nil
}

type agentScore struct {
	name  string
	score int
}

// scoreAgents computes relevance scores for each available agent against the user text.
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

// fallbackPlan generates a basic plan when no capability match is found.
func (m *MainAgent) fallbackPlan(input PlannerInput) (*plan.OrchestrationPlan, error) {
	participants := make([]plan.PlanParticipant, 0, len(m.availableAgents))
	for _, name := range m.availableAgents {
		participants = append(participants, plan.PlanParticipant{
			AgentName: name,
			Required:  false,
			Selected:  false,
		})
	}

	revision := input.Revision
	if revision < 1 {
		revision = 1
	}

	return &plan.OrchestrationPlan{
		Version:              "1.0",
		PlanID:               fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		RunID:                input.RunID,
		ConversationID:       input.ConversationID,
		ExecutionPath:        input.ExecutionPath,
		Strategy:             plan.StrategyConversational,
		IntentSummary:        "General assistance — specify your task for better agent matching",
		Tasks:                nil,
		PlannerSource:        "main_agent",
		AllowedAgents:        input.AllowedAgents,
		Revision:             revision,
		PlanOwner:            &plan.PlanOwner{Type: "main_agent", AgentName: "main-agent", IsMainAgent: true},
		Participants:         participants,
		CandidateParticipants: participants,
	}, nil
}
