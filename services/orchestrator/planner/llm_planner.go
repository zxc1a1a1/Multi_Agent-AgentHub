package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// PlannerModel abstracts the LLM call for intent planning.
// It decouples the LLMPlanner from any specific provider implementation.
// Fake implementations enable testing without real API keys.
type PlannerModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

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

// ---------------------------------------------------------------------------
// Legacy types — preserved for backward compatibility, not used by new pipeline
// ---------------------------------------------------------------------------

// llmPlanResponse is the legacy JSON structure for the old parsePlanResponse path.
type llmPlanResponse struct {
	Intent    string        `json:"intent"`
	Reasoning string        `json:"reasoning"`
	Strategy  string        `json:"strategy"`
	Tasks     []llmTaskPlan `json:"tasks"`
}

type llmTaskPlan struct {
	AgentName       string   `json:"agentName"`
	CapabilityIDs   []string `json:"capabilityIds"`
	TaskContent     string   `json:"taskContent"`
	ExpectedOutputs []string `json:"expectedOutputs"`
	Priority        int      `json:"priority"`
}

// ---------------------------------------------------------------------------
// Validation error codes for plan validation failures
// ---------------------------------------------------------------------------

const (
	ValCodeUnknownAgent          = "unknown_agent"
	ValCodeUnsupportedSequential = "unsupported_sequential"
	ValCodeEmptyTasks            = "empty_tasks"
	ValCodeEmptyAgent            = "empty_agent"
)

// ValError is a single validation/pipeline failure with a code and message.
type ValError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	TaskIndex int    `json:"task_index,omitempty"` // -1 if not task-specific
}

// ValResult is the outcome of plan validation.
type ValResult struct {
	Valid  bool       `json:"valid"`
	Errors []ValError `json:"errors,omitempty"`
}

// ---------------------------------------------------------------------------
// PlanValidator — in-planner validation interface
// ---------------------------------------------------------------------------

// PlanValidator validates an OrchestrationPlan produced by the LLM.
// Returns a ValResult with structured error codes that the Repairer can use.
// The concrete implementation is auto-wired from the AgentLister in NewLLMPlanner.
type PlanValidator interface {
	Validate(p *plan.OrchestrationPlan) *ValResult
}

// ---------------------------------------------------------------------------
// LLMPlanner — new pipeline: PromptBuilder → PlannerModel → Parser → Normalizer → Validator
// ---------------------------------------------------------------------------

// LLMPlanner implements the Planner interface using an LLM.
// On failure it falls back to RulePlanner (deprecated transitional fallback)
// unless DisableFallback has been called.
// The pipeline (Phase 4):
//
//	PromptBuilder → PlannerModel → Parser → Normalizer → Validator
//	  → (if fail) Repairer once → Parser → Normalizer → Validator
//	  → (if still fail) RulePlanner fallback (or error if noFallback is true)
//
// Unknown agent names are never fuzzy-matched or defaulted; they are rejected
// by the Validator, triggering a repair attempt, then RulePlanner fallback.
type LLMPlanner struct {
	model       PlannerModel
	modelName   string
	lister      AgentLister
	validator   PlanValidator
	repairer    *PlanRepairer
	fallback    *RulePlanner
	noFallback  bool
}

// DisableFallback disables the built-in RulePlanner fallback.
// When fallback is disabled, Plan() returns an error on LLM/pipeline failure
// instead of silently falling back to deterministic routing.
func (p *LLMPlanner) DisableFallback() {
	p.noFallback = true
}

// NewLLMPlanner creates an LLMPlanner.
// model is the LLM backend; modelName is used for trace metadata.
// lister provides registry agent info for prompt construction and validation.
// The PlanValidator is auto-created from the lister — no separate injection needed.
func NewLLMPlanner(model PlannerModel, modelName string, lister AgentLister) *LLMPlanner {
	var availableAgents []string
	if lister != nil {
		for _, a := range lister.List() {
			availableAgents = append(availableAgents, a.Name)
		}
	}
	return &LLMPlanner{
		model:     model,
		modelName: modelName,
		lister:    lister,
		validator: newListerPlanValidator(lister),
		repairer:  NewPlanRepairer(model),
		fallback:  NewRulePlanner(availableAgents),
	}
}

// Plan generates an OrchestrationPlan using the LLM pipeline with one-shot repair:
//
//	PromptBuilder (registry agents) → PlannerModel →
//	  Parser → Normalizer → Validator → return plan (primary path)
//
// On pipeline failure (parse/normalize/validate):
//
//	Repairer (one shot) → Parser → Normalizer → Validator → return repaired plan
//	  → (if repair fails) RulePlanner fallback
//
// Repair success: PlannerSource=llm, RepairCount=1, Fallback.Enabled=false.
// Repair failure: fallback RulePlanner.
//
// Unknown agent names are never fuzzy-matched or defaulted — the Normalizer
// preserves them as-is, and the PlanValidator/Repairer handles them.
func (p *LLMPlanner) Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	// 1. Build prompts using PromptBuilder with real registry agent info.
	pb := NewPromptBuilder(p.lister)
	if input.ConversationType != "" {
		pb = pb.WithConversationType(input.ConversationType)
	}
	systemPrompt := pb.BuildSystemPrompt()
	userPrompt := pb.BuildUserPrompt(input.UserMessage, input.AgentName)

	// 2. Call the PlannerModel.
	raw, err := p.model.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		if p.noFallback {
			return nil, fmt.Errorf("llm_planner: LLM call failed (fallback disabled): %w", err)
		}
		log.Printf("llm_planner: LLM call failed: %v, falling back to RulePlanner", err)
		return p.fallbackPlan(input, "llm_error", p.modelName), nil
	}

	// 3. Try pipeline: parse → normalize → validate.
	orchPlan, failures := p.runPipeline(raw, input)
	if orchPlan != nil {
		// Primary path success — stamp metadata.
		orchPlan.PlannerSource = "llm"
		orchPlan.PlannerModel = p.modelName
		orchPlan.PlannerReasoning = truncateToLength(strings.TrimSpace(orchPlan.IntentSummary), 120)
		orchPlan.Fallback = plan.Fallback{Enabled: false}
		return orchPlan, nil
	}

	// 4. Pipeline failed — attempt one-shot repair.
	log.Printf("llm_planner: pipeline failed with %d error(s), attempting repair", len(failures))
	repairedRaw, repairErr := p.repairer.Repair(ctx, raw, failures, p.lister)
	if repairErr != nil {
		if p.noFallback {
			return nil, fmt.Errorf("llm_planner: repair failed (fallback disabled): %w", repairErr)
		}
		log.Printf("llm_planner: repair model call failed: %v, falling back to RulePlanner", repairErr)
		return p.fallbackPlan(input, "repair_failed", p.modelName), nil
	}

	// 5. Re-run pipeline on repaired output.
	orchPlan, _ = p.runPipeline(repairedRaw, input)
	if orchPlan != nil {
		// Repair success — stamp metadata with repair count.
		log.Printf("llm_planner: repair succeeded, returning repaired plan")
		orchPlan.PlannerSource = "llm"
		orchPlan.PlannerModel = p.modelName
		orchPlan.PlannerReasoning = truncateToLength(strings.TrimSpace(orchPlan.IntentSummary), 120)
		orchPlan.RepairCount = 1
		orchPlan.Fallback = plan.Fallback{Enabled: false}
		return orchPlan, nil
	}

	// 6. Repair did not fix the issue — fallback to RulePlanner (or error).
	if p.noFallback {
		return nil, fmt.Errorf("llm_planner: pipeline failed after repair (fallback disabled): %d validation errors", len(failures))
	}
	log.Printf("llm_planner: pipeline still failed after repair, falling back to RulePlanner")
	return p.fallbackPlan(input, "repair_failed", p.modelName), nil
}

// runPipeline runs the parse → normalize → validate pipeline on raw LLM output.
// Returns (plan, nil) on success, or (nil, failures) on any pipeline error.
// The returned plan does NOT have PlannerSource/PlannerModel/Fallback set —
// the caller (Plan) is responsible for metadata stamping.
func (p *LLMPlanner) runPipeline(raw string, input PlannerInput) (*plan.OrchestrationPlan, []ValError) {
	// Parse raw output into PlanSchema.
	parser := NewPlanParser()
	schema, err := parser.Parse(raw)
	if err != nil {
		return nil, []ValError{{Code: "parse_error", Message: err.Error(), TaskIndex: -1}}
	}

	// Normalize PlanSchema → OrchestrationPlan.
	// Unknown agent names are preserved as-is; no fuzzyMatchAgent/defaultAgent.
	normalizer := NewPlanNormalizer()
	orchPlan, err := normalizer.Normalize(schema, input.RunID, input.ConversationID, input.PlanningMode)
	if err != nil {
		return nil, []ValError{{Code: "normalize_error", Message: err.Error(), TaskIndex: -1}}
	}

	// Validate the normalized plan against registry agents.
	// Unknown agents are rejected here — NOT fuzzy-matched or defaulted.
	if p.validator != nil {
		result := p.validator.Validate(orchPlan)
		if !result.Valid {
			return nil, result.Errors
		}
	}

	return orchPlan, nil
}

// ---------------------------------------------------------------------------
// PlanValidator implementation — validates against AgentLister
// ---------------------------------------------------------------------------

// listerPlanValidator validates an OrchestrationPlan against the agent registry.
// It checks that every task's agent exists in the registry and enforces basic
// structural rules. Unknown agents are rejected (no fuzzy-match, no default).
type listerPlanValidator struct {
	lister AgentLister
}

// newListerPlanValidator creates a validator from an AgentLister.
// Returns nil when lister is nil (validation is skipped).
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

	// Build set of known agent names from registry.
	agents := v.lister.List()
	known := make(map[string]bool, len(agents))
	for _, a := range agents {
		known[a.Name] = true
	}

	// Check every task's agent exists in the registry.
	for i, task := range p.Tasks {
		if task.AgentName == "" {
			log.Printf("llm_planner: validation reject: task[%d] has empty agentName", i)
			r.Valid = false
			r.Errors = append(r.Errors, ValError{
				Code:      ValCodeEmptyAgent,
				Message:   "task has empty agentName",
				TaskIndex: i,
			})
			continue
		}
		if !known[task.AgentName] {
			log.Printf("llm_planner: validation reject: agent %q not in registry", task.AgentName)
			r.Valid = false
			r.Errors = append(r.Errors, ValError{
				Code:      ValCodeUnknownAgent,
				Message:   "agent \"" + task.AgentName + "\" not found in registry",
				TaskIndex: i,
			})
		}
	}

	// Structural: tasks must not be empty.
	if len(p.Tasks) == 0 {
		log.Printf("llm_planner: validation reject: plan has no tasks")
		r.Valid = false
		r.Errors = append(r.Errors, ValError{
			Code:      ValCodeEmptyTasks,
			Message:   "plan has no tasks",
			TaskIndex: -1,
		})
	}

	// Structural: sequential strategy is not supported.
	if p.Strategy == plan.StrategySequential {
		log.Printf("llm_planner: validation reject: sequential strategy not supported")
		r.Valid = false
		r.Errors = append(r.Errors, ValError{
			Code:      ValCodeUnsupportedSequential,
			Message:   "sequential strategy is not yet supported",
			TaskIndex: -1,
		})
	}

	return r
}

// ---------------------------------------------------------------------------
// Legacy: System / User prompt builders (from Phase 1, used by old parsePlanResponse path)
// ---------------------------------------------------------------------------

// buildSystemPrompt is the legacy prompt builder used by the old parsePlanResponse path.
// Deprecated: new pipeline uses PromptBuilder with registry agent info.
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

// buildUserPrompt is the legacy user prompt builder.
// Deprecated: new pipeline uses PromptBuilder.BuildUserPrompt.
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
// Legacy: Response parsing (Phase 1) — not used by new pipeline
// ---------------------------------------------------------------------------

// parsePlanResponse is the legacy JSON parser for the llmPlanResponse format.
// Deprecated: new pipeline uses PlanParser → PlanSchema.
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
// Legacy: LLM response → OrchestrationPlan conversion (Phase 1)
// Deprecated: new pipeline uses PlanNormalizer.
// ---------------------------------------------------------------------------

// convertToPlan converts a legacy llmPlanResponse into an OrchestrationPlan.
// It calls fuzzyMatchAgent / defaultAgent for unknown agent names.
// Deprecated: new pipeline uses PlanNormalizer which preserves unknown agents.
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
		TraceID:        input.TraceID,
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
			TraceID:        input.TraceID,
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
// Legacy: Defaults / helpers — preserved for backward compatibility
// Deprecated: new pipeline does NOT use fuzzyMatchAgent or defaultAgent.
//             Agent validation is handled by the Validator.
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
// Deprecated: new pipeline preserves unknown agent names as-is.
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
// Deprecated: new pipeline preserves unknown agent names as-is.
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
