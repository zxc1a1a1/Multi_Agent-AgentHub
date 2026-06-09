package validator

import (
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// PathValidationError is a path-aware validation failure.
type PathValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PathValidationResult holds the outcome of path-aware validation.
// Warning-only errors do NOT set Valid=false.
type PathValidationResult struct {
	Valid    bool                 `json:"valid"`
	Errors   []PathValidationError `json:"errors,omitempty"`
	Warnings []PathValidationError `json:"warnings,omitempty"`
}

func (r *PathValidationResult) addError(field, code, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, PathValidationError{Field: field, Code: code, Message: message})
}

func (r *PathValidationResult) addWarning(field, code, message string) {
	r.Warnings = append(r.Warnings, PathValidationError{Field: field, Code: code, Message: message})
}

// PathAwareValidator validates an OrchestrationPlan against execution-path-specific
// constraints. It is a SECONDARY defense — the primary defense is the pre-planner
// ValidateAgentSelection in the executionpath package.
//
// PathAwareValidator checks:
//   - participants must come from AllowedAgents (availableBoundary)
//   - every task's AgentName must be within AllowedAgents
//   - single_chat: exactly one participant, must match the single selected agent
//   - group_chat: no boundary-external agents, at least one participant
//   - main_agent_orchestration: participants must not be empty
//   - missing participants → hard error
//   - missing steps/tasks → hard error
//   - strategy missing or unknown → WARNING only (not blocking)
//
// PathAwareValidator does NOT check agent existence in registry (that's
// PlanValidator's job), duplicate task IDs, DAG cycles, risk levels, or
// content security — those are PlanValidator responsibilities.
type PathAwareValidator struct {
	allowedAgents []string
	executionPath string
}

// NewPathAwareValidator creates a validator for the given execution path and
// allowed agent list (the availableBoundary).
func NewPathAwareValidator(executionPath string, allowedAgents []string) *PathAwareValidator {
	return &PathAwareValidator{
		executionPath:  executionPath,
		allowedAgents:  allowedAgents,
	}
}

// Validate checks the plan against path-specific constraints.
func (v *PathAwareValidator) Validate(p *plan.OrchestrationPlan) *PathValidationResult {
	r := &PathValidationResult{Valid: true}

	if p == nil {
		r.addError("plan", "PATH_VALIDATION", "plan is nil")
		return r
	}

	// Build allowed set for fast lookup.
	allowedSet := make(map[string]bool, len(v.allowedAgents))
	for _, a := range v.allowedAgents {
		allowedSet[strings.ToLower(strings.TrimSpace(a))] = true
	}

	// 1. Strategy: missing or unknown → WARNING only.
	if strings.TrimSpace(p.Strategy) == "" {
		r.addWarning("strategy", "PATH_STRATEGY_MISSING",
			"strategy is empty — execution may fall back to default")
	} else if !isKnownStrategy(p.Strategy) {
		r.addWarning("strategy", "PATH_STRATEGY_UNKNOWN",
			fmt.Sprintf("strategy %q is not recognized — may be treated as sequential", p.Strategy))
	}

	// 2. Tasks/steps must not be empty (except conversational).
	if p.Strategy != plan.StrategyConversational && len(p.Tasks) == 0 {
		r.addError("tasks", "PATH_MISSING_STEPS",
			"plan has no tasks/steps — must have at least one for non-conversational paths")
	}

	// 3. Validate each task's AgentName is within AllowedAgents.
	for i, t := range p.Tasks {
		field := fmt.Sprintf("tasks[%d].agentName", i)
		name := strings.ToLower(strings.TrimSpace(t.AgentName))
		if name == "" {
			r.addError(field, "PATH_EMPTY_AGENT",
				"task has empty agentName — all tasks must have an assigned agent")
			continue
		}
		if len(allowedSet) > 0 && !allowedSet[name] {
			r.addError(field, "PATH_AGENT_OUT_OF_BOUNDARY",
				fmt.Sprintf("agent %q is outside the allowed boundary %v", t.AgentName, v.allowedAgents))
		}
	}

	// 4. Validate participants against path constraints.
	v.validateParticipants(p, allowedSet, r)

	return r
}

// validateParticipants checks participant list against path-specific rules.
func (v *PathAwareValidator) validateParticipants(p *plan.OrchestrationPlan, allowedSet map[string]bool, r *PathValidationResult) {
	// 4a. Participants must not be empty for non-conversational paths.
	if p.Strategy != plan.StrategyConversational && len(p.Participants) == 0 {
		// Build participants from tasks as fallback info.
		taskAgents := make(map[string]bool)
		for _, t := range p.Tasks {
			if n := strings.TrimSpace(t.AgentName); n != "" {
				taskAgents[n] = true
			}
		}
		if len(taskAgents) > 0 {
			// Tasks exist but participants missing — still a hard error.
			r.addError("participants", "PATH_MISSING_PARTICIPANTS",
				"participants list is empty but tasks reference agents — participants are required")
		} else {
			r.addError("participants", "PATH_MISSING_PARTICIPANTS",
				"participants list is empty — at least one participant is required")
		}
		return
	}

	// 4b. Each participant must be within AllowedAgents.
	for i, part := range p.Participants {
		field := fmt.Sprintf("participants[%d].agentName", i)
		name := strings.ToLower(strings.TrimSpace(part.AgentName))
		if name == "" {
			r.addError(field, "PATH_EMPTY_PARTICIPANT",
				"participant has empty agentName")
			continue
		}
		if len(allowedSet) > 0 && !allowedSet[name] {
			r.addError(field, "PATH_PARTICIPANT_OUT_OF_BOUNDARY",
				fmt.Sprintf("participant %q is outside the allowed boundary %v", part.AgentName, v.allowedAgents))
		}
	}

	// 4c. single_chat: only one participant allowed.
	if v.executionPath == "single_chat" {
		if len(p.Participants) > 1 {
			r.addError("participants", "PATH_SINGLE_CHAT_TOO_MANY",
				fmt.Sprintf("single_chat requires exactly 1 participant, got %d", len(p.Participants)))
		}
		if len(v.allowedAgents) == 1 {
			// The sole participant must match the sole allowed agent.
			for _, part := range p.Participants {
				if !allowedSet[strings.ToLower(strings.TrimSpace(part.AgentName))] {
					r.addError("participants", "PATH_SINGLE_CHAT_WRONG_AGENT",
						fmt.Sprintf("single_chat participant %q does not match the expected agent %q",
							part.AgentName, v.allowedAgents[0]))
				}
			}
		}
	}

	// 4d. group_chat: must have at least one participant.
	if v.executionPath == "group_chat" && len(p.Participants) == 0 {
		r.addError("participants", "PATH_GROUP_CHAT_NO_PARTICIPANTS",
			"group_chat requires at least one participant")
	}

	// 4e. main_agent_orchestration: at least one participant must be selected.
	if v.executionPath == "main_agent_orchestration" && len(p.Participants) == 0 {
		r.addError("participants", "PATH_AUTO_NO_PARTICIPANTS",
			"main_agent_orchestration requires at least one participant")
	}
}

// isKnownStrategy reports whether the strategy string is a recognized value.
func isKnownStrategy(s string) bool {
	switch s {
	case plan.StrategySingle, plan.StrategyOrderedParallel, plan.StrategySequential, plan.StrategyConversational:
		return true
	default:
		return false
	}
}
