package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// AgentInfoLite is the minimal agent information needed by the LLMPlanner.
// We use our own type instead of importing registry to keep planner
// decoupled from the concrete registry implementation.
type AgentInfoLite struct {
	Name          string
	Description   string
	CapabilityIDs []string
	OutputModes   []string
}

// AgentLister is the interface the LLMPlanner needs from a registry.
type AgentLister interface {
	List() []AgentInfoLite
}

// llmPlanResponse is the JSON structure we expect the LLM to return.
type llmPlanResponse struct {
	Intent    string         `json:"intent"`
	Reasoning string         `json:"reasoning"`
	Strategy  string         `json:"strategy"`
	Tasks     []llmTaskPlan  `json:"tasks"`
}

type llmTaskPlan struct {
	AgentName       string   `json:"agentName"`
	CapabilityIDs   []string `json:"capabilityIds"`
	TaskContent     string   `json:"taskContent"`
	ExpectedOutputs []string `json:"expectedOutputs"`
	Priority        int      `json:"priority"`
}

// LLMPlanner implements the Planner interface using an LLM.
// On failure it automatically falls back to RulePlanner.
type LLMPlanner struct {
	llm      *PlannerLLM
	fallback *RulePlanner
}

// NewLLMPlanner creates an LLMPlanner.
func NewLLMPlanner(llm *PlannerLLM, availableAgents []string) *LLMPlanner {
	return &LLMPlanner{
		llm:      llm,
		fallback: NewRulePlanner(availableAgents),
	}
}

// Plan generates an OrchestrationPlan using LLM intent analysis.
// If the LLM call fails or returns invalid JSON, the method falls back
// to the RulePlanner automatically.
func (p *LLMPlanner) Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	systemPrompt := p.buildSystemPrompt(input.AvailableAgents)
	userPrompt := p.buildUserPrompt(input)

	raw, err := p.llm.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("llm_planner: LLM call failed: %v, falling back to RulePlanner", err)
		return p.fallbackPlan(input, "llm_error", ""), nil
	}

	resp, err := parsePlanResponse(raw)
	if err != nil {
		log.Printf("llm_planner: failed to parse LLM response (%v), raw=%s, falling back", err, truncate(raw, 200))
		return p.fallbackPlan(input, "parse_error", ""), nil
	}

	// Convert LLM response into an OrchestrationPlan.
	orchPlan, convErr := p.convertToPlan(resp, input)
	if convErr != nil {
		log.Printf("llm_planner: conversion error: %v, falling back", convErr)
		return p.fallbackPlan(input, "conversion_error", ""), nil
	}

	// Stamp planner metadata.
	orchPlan.PlannerSource = "llm"
	orchPlan.PlannerModel = p.llm.cfg.Model
	orchPlan.PlannerReasoning = resp.Reasoning

	return orchPlan, nil
}

// ---------------------------------------------------------------------------
// System / User prompt builders
// ---------------------------------------------------------------------------

func (p *LLMPlanner) buildSystemPrompt(availableAgents []string) string {
	var b strings.Builder
	b.WriteString(`You are an intent orchestrator for a multi-agent platform. Your job is to analyze the user's request and create an execution plan.

## Available Agents
`)
	for _, name := range availableAgents {
		desc, caps, outs := agentDefaults(name)
		b.WriteString(fmt.Sprintf("- name: %s\n  description: %s\n  capabilities: [%s]\n  outputModes: [%s]\n\n",
			name, desc, strings.Join(caps, ", "), strings.Join(outs, ", ")))
	}

	b.WriteString(`## Execution Strategies
- "single": exactly one agent handles the entire request
- "ordered_parallel": multiple agents work on independent sub-tasks in priority order

## Rules
1. For code / backend / API / algorithm requests → use code-agent
2. For web UI / frontend / page / HTML / CSS requests → use web-agent
3. If the request needs BOTH frontend UI AND backend code → use "ordered_parallel" with both agents
4. In ordered_parallel, web-agent runs BEFORE code-agent (priority 1 = web, priority 2 = code)
5. Each task's capabilityIds MUST come from the agent's actual capabilities listed above
6. Each task's expectedOutputs MUST come from the agent's actual outputModes listed above
7. taskContent should rephrase the user's request specifically for the target agent
8. If the user explicitly names an agent, always use "single" strategy with that agent
9. If unsure, default to code-agent with strategy "single"

## Output Format
Return ONLY a valid JSON object. No markdown, no explanation, no code fences:
{
  "intent": "brief summary of what the user wants",
  "reasoning": "brief explanation of why this strategy and these agents were chosen",
  "strategy": "single" | "ordered_parallel",
  "tasks": [
    {
      "agentName": "exact-agent-name",
      "capabilityIds": ["capability-from-above-list"],
      "taskContent": "rephrased request specific for this agent",
      "expectedOutputs": ["output-mode-from-above-list"],
      "priority": 1
    }
  ]
}`)
	return b.String()
}

func (p *LLMPlanner) buildUserPrompt(input PlannerInput) string {
	var b strings.Builder
	b.WriteString("## User Message\n")
	b.WriteString(input.UserMessage)
	if input.AgentName != "" {
		b.WriteString("\n\n## Hint\n")
		b.WriteString(fmt.Sprintf("The user selected agent: %s. Use this as a strong preference.", input.AgentName))
	}
	if input.ConversationType == "group" {
		b.WriteString("\n\n## Context\nThis is a group chat. Multiple agents may participate.")
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Response parsing
// ---------------------------------------------------------------------------

func parsePlanResponse(raw string) (*llmPlanResponse, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("empty response")
	}
	var resp llmPlanResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	if resp.Strategy == "" {
		return nil, fmt.Errorf("missing strategy field")
	}
	if len(resp.Tasks) == 0 {
		return nil, fmt.Errorf("no tasks in plan")
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// LLM response → OrchestrationPlan conversion
// ---------------------------------------------------------------------------

func (p *LLMPlanner) convertToPlan(resp *llmPlanResponse, input PlannerInput) (*plan.OrchestrationPlan, error) {
	// Validate strategy.
	strategy := strings.ToLower(strings.TrimSpace(resp.Strategy))
	switch strategy {
	case plan.StrategySingle, plan.StrategyOrderedParallel:
		// ok
	default:
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}

	planID := plan.NewPlanID()

	var tasks []plan.TaskPlan
	for i, t := range resp.Tasks {
		agentName := strings.ToLower(strings.TrimSpace(t.AgentName))
		if agentName == "" {
			return nil, fmt.Errorf("task %d has empty agentName", i)
		}

		// Validate the agent is known.
		if !p.fallback.isKnown(agentName) {
			// Try fuzzy match: if LLM returned "codeagent" normalize to "code-agent".
			if fixed := fuzzyMatchAgent(agentName, input.AvailableAgents); fixed != "" {
				agentName = fixed
			} else {
				agentName = defaultAgent(input.AvailableAgents)
			}
		}

		taskID := t.TaskID()
		priority := t.Priority
		if priority <= 0 {
			priority = i + 1
		}
		timeout := int64(120000)
		taskContent := t.TaskContent
		if taskContent == "" {
			taskContent = input.UserMessage
		}

		tasks = append(tasks, plan.TaskPlan{
			TaskID:          taskID,
			AgentName:       agentName,
			CapabilityIDs:   nonEmptyStrings(t.CapabilityIDs, staticCapabilityIDs(agentName)),
			TaskContent:     taskContent,
			ExpectedOutputs: nonEmptyStrings(t.ExpectedOutputs, staticExpectedOutputs(agentName)),
			DependsOn:       []string{},
			Priority:        priority,
			TimeoutMs:       timeout,
			RiskLevel:       "low",
		})
	}

	aggregation := plan.Aggregation{Required: false, Mode: "none"}
	if strategy == plan.StrategyOrderedParallel {
		aggregation = plan.Aggregation{Required: true, Mode: "summary"}
	}

	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         planID,
		RunID:          input.RunID,
		ConversationID: input.ConversationID,
		PlanningMode:   input.PlanningMode,
		Strategy:       strategy,
		IntentSummary:  summarize(input.UserMessage),
		Tasks:          tasks,
		Aggregation:    aggregation,
		Fallback:       plan.Fallback{Enabled: true, Reason: "llm_default"},
		Validation:     plan.Validation{Validated: false},
	}, nil
}

func (t llmTaskPlan) TaskID() string {
	if t.AgentName == "" {
		return "task_001"
	}
	return "task_" + strings.ToLower(strings.TrimSpace(t.AgentName))
}

// ---------------------------------------------------------------------------
// Fallback
// ---------------------------------------------------------------------------

func (p *LLMPlanner) fallbackPlan(input PlannerInput, sourceCode, model string) *plan.OrchestrationPlan {
	orchPlan, err := p.fallback.Plan(context.Background(), input)
	if err != nil {
		// Ultimate fallback: single code-agent plan.
		agentName := defaultAgent(input.AvailableAgents)
		return &plan.OrchestrationPlan{
			Version:        "v1",
			PlanID:         plan.NewPlanID(),
			RunID:          input.RunID,
			ConversationID: input.ConversationID,
			PlanningMode:   input.PlanningMode,
			Strategy:       plan.StrategySingle,
			IntentSummary:  summarize(input.UserMessage),
			Tasks: []plan.TaskPlan{{
				TaskID:          "task_001",
				AgentName:       agentName,
				CapabilityIDs:   staticCapabilityIDs(agentName),
				TaskContent:     input.UserMessage,
				ExpectedOutputs: staticExpectedOutputs(agentName),
				Priority:        1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			}},
			Aggregation:     plan.Aggregation{Required: false, Mode: "none"},
			Fallback:        plan.Fallback{Enabled: true, Reason: "ultimate_fallback"},
			Validation:      plan.Validation{Validated: false},
			PlannerSource:   "fallback",
			PlannerModel:    model,
			PlannerReasoning: fmt.Sprintf("LLM call failed (code: %s), fell back to rule-based planner", sourceCode),
		}
	}
	orchPlan.PlannerSource = "fallback"
	orchPlan.PlannerModel = model
	orchPlan.PlannerReasoning = fmt.Sprintf("LLM call failed (code: %s), fell back to RulePlanner", sourceCode)
	return orchPlan
}

// ---------------------------------------------------------------------------
// Defaults / helpers
// ---------------------------------------------------------------------------

// agentDefaults returns static metadata for well-known agents.
func agentDefaults(name string) (description string, capabilities, outputModes []string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "code-agent":
		return "code generation and explanation",
			[]string{"code_generation", "code_explanation"},
			[]string{"text", "code", "artifact_ref"}
	case "web-agent":
		return "web UI and HTML generation",
			[]string{"web_generation", "html_generation", "ui_summarization"},
			[]string{"text", "webpage", "html", "artifact_ref"}
	case "vision-agent":
		return "visual analysis and image description",
			[]string{"vision_analysis", "image_description"},
			[]string{"text", "vision_analysis"}
	default:
		return name, []string{}, []string{"text"}
	}
}

// fuzzyMatchAgent tries to normalize a potentially malformed agent name.
func fuzzyMatchAgent(name string, available []string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	// Remove hyphens and compare.
	collapsed := strings.ReplaceAll(lower, "-", "")
	for _, a := range available {
		if strings.EqualFold(strings.ReplaceAll(a, "-", ""), collapsed) {
			return a
		}
	}
	// Partial match.
	for _, a := range available {
		if strings.Contains(strings.ToLower(a), lower) || strings.Contains(lower, strings.ToLower(a)) {
			return a
		}
	}
	return ""
}

// defaultAgent returns the first available agent or "code-agent".
func defaultAgent(available []string) string {
	for _, a := range available {
		if strings.EqualFold(a, "code-agent") {
			return "code-agent"
		}
	}
	if len(available) > 0 {
		return available[0]
	}
	return "code-agent"
}

// nonEmptyStrings returns the first non-empty slice.
func nonEmptyStrings(preferred, fallback []string) []string {
	if len(preferred) > 0 {
		return preferred
	}
	return fallback
}

// truncate limits a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure LLMPlanner implements Planner.
var _ Planner = (*LLMPlanner)(nil)
