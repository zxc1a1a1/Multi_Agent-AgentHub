package planner

import (
	"context"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// RulePlanner implements Planner using keyword-based rules.
// Mixed web+code keywords produce ordered_parallel; everything else is single.
type RulePlanner struct {
	availableAgents []string
}

// NewRulePlanner creates a RulePlanner backed by the given agent names.
func NewRulePlanner(availableAgents []string) *RulePlanner {
	return &RulePlanner{
		availableAgents: availableAgents,
	}
}

// Plan generates an OrchestrationPlan using keyword rules.
// Mixed web+code keywords produce an ordered_parallel plan;
// all other inputs produce a single-task plan.
func (p *RulePlanner) Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	msg := strings.ToLower(input.UserMessage)
	isWeb := containsAny(msg, webKeywords)
	isCode := containsAny(msg, codeKeywords)

	// Mixed: generate ordered_parallel plan with both agents.
	if isWeb && isCode && p.isKnown("web-agent") && p.isKnown("code-agent") {
		return &plan.OrchestrationPlan{
			Version:        "v1",
			PlanID:         plan.NewPlanID(),
			RunID:          input.RunID,
			ConversationID: input.ConversationID,
			PlanningMode:   input.PlanningMode,
			Strategy:       plan.StrategyOrderedParallel,
			IntentSummary:  summarize(input.UserMessage),
			Tasks: []plan.TaskPlan{
				{
					TaskID:          "task_web",
					AgentName:       "web-agent",
					CapabilityIDs:   []string{"web_generation"},
					TaskContent:     input.UserMessage,
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
					TaskContent:     input.UserMessage,
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
		}, nil
	}

	// Single agent dispatch.
	agentName := p.determineAgent(input)

	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         plan.NewPlanID(),
		RunID:          input.RunID,
		ConversationID: input.ConversationID,
		PlanningMode:   input.PlanningMode,
		Strategy:       plan.StrategySingle,
		IntentSummary:  summarize(input.UserMessage),
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_001",
				AgentName:       agentName,
				CapabilityIDs:   staticCapabilityIDs(agentName),
				TaskContent:     input.UserMessage,
				ExpectedOutputs: staticExpectedOutputs(agentName),
				DependsOn:       []string{},
				Priority:        1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			},
		},
		Aggregation: plan.Aggregation{Required: false, Mode: "none"},
		Fallback:    plan.Fallback{Enabled: true, Reason: "rule_default"},
		Validation:  plan.Validation{Validated: false},
	}, nil
}

var webKeywords = []string{
	"页面", "ui", "html", "react", "登录页", "前端",
	"page", "webpage", "css", "component", "layout",
}

var codeKeywords = []string{
	"go", "api", "后端", "接口", "server", "service",
	"golang", "handler", "endpoint", "database", "sql", "函数",
}

// determineAgent picks a single target agent based on priority:
//  1. Explicit agentName from request (mention/direct)
//  2. First element of selectedAgentNames
//  3. Keyword matching on userMessage
//  4. Default to "code-agent"
func (p *RulePlanner) determineAgent(input PlannerInput) string {
	if name := strings.TrimSpace(input.AgentName); name != "" {
		if p.isKnown(name) {
			return name
		}
	}
	if len(input.SelectedAgentNames) > 0 {
		if name := strings.TrimSpace(input.SelectedAgentNames[0]); name != "" && p.isKnown(name) {
			return name
		}
	}

	msg := strings.ToLower(input.UserMessage)
	isWeb := containsAny(msg, webKeywords)
	isCode := containsAny(msg, codeKeywords)

	// Mixed web+code is handled by Plan() as ordered_parallel.
	// Here in the single-agent fallback, treat mixed as web-first.
	if isWeb && isCode {
		if p.isKnown("web-agent") {
			return "web-agent"
		}
		if p.isKnown("code-agent") {
			return "code-agent"
		}
	}
	if isWeb && p.isKnown("web-agent") {
		return "web-agent"
	}
	if isCode && p.isKnown("code-agent") {
		return "code-agent"
	}

	// Default fallback.
	if p.isKnown("code-agent") {
		return "code-agent"
	}
	if len(p.availableAgents) > 0 {
		return p.availableAgents[0]
	}
	return "code-agent"
}

func (p *RulePlanner) isKnown(name string) bool {
	for _, a := range p.availableAgents {
		if strings.EqualFold(a, name) {
			return true
		}
	}
	return false
}

func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func summarize(text string) string {
	const maxLen = 120
	cleaned := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if len(cleaned) <= maxLen {
		return cleaned
	}
	return cleaned[:maxLen-3] + "..."
}

func staticCapabilityIDs(agentName string) []string {
	switch strings.ToLower(strings.TrimSpace(agentName)) {
	case "code-agent":
		return []string{"code_generation"}
	case "web-agent":
		return []string{"web_generation"}
	default:
		return nil
	}
}

func staticExpectedOutputs(agentName string) []string {
	switch strings.ToLower(strings.TrimSpace(agentName)) {
	case "code-agent":
		return []string{"code"}
	case "web-agent":
		return []string{"webpage", "markdown"}
	default:
		return nil
	}
}

// Ensure RulePlanner implements Planner.
var _ Planner = (*RulePlanner)(nil)
