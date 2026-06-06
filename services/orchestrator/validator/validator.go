// Package validator provides PlanValidator for OrchestrationPlan validation.
// All OrchestrationPlans must pass validation before execution.
package validator

import (
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

// Registry is the subset of agent registry that PlanValidator needs.
type Registry interface {
	Get(name string) (registry.AgentEndpoint, bool)
	Names() []string
}

// ValidationError is a single validation failure with field path and message.
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationResult holds the outcome of plan validation.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// PlanValidator validates an OrchestrationPlan against structural rules and
// agent registry capabilities.
type PlanValidator struct {
	registry Registry
}

// New creates a PlanValidator backed by the given registry.
func New(reg Registry) *PlanValidator {
	return &PlanValidator{registry: reg}
}

// Validate checks the plan against all required rules. If the plan passes, the
// caller must set plan.Validation.Validated = true before execution.
func (v *PlanValidator) Validate(p *plan.OrchestrationPlan) *ValidationResult {
	r := &ValidationResult{Valid: true}

	if p == nil {
		r.Valid = false
		r.Errors = append(r.Errors, ValidationError{Field: "plan", Code: "INVALID", Message: "plan is nil"})
		return r
	}

	// 1. planId required.
	if strings.TrimSpace(p.PlanID) == "" {
		r.add("planId", "planId is required")
	}

	// 2. runId required.
	if strings.TrimSpace(p.RunID) == "" {
		r.add("runId", "runId is required")
	}

	// 3. conversationId required.
	if strings.TrimSpace(p.ConversationID) == "" {
		r.add("conversationId", "conversationId is required")
	}

	// 4a. sequential is explicitly rejected — not yet supported by executor/httpapi.
	if p.Strategy == plan.StrategySequential {
		r.add("strategy", "sequential strategy is not yet supported (executor/httpapi does not handle sequential execution)")
	}

	// 4b. strategy must be single or ordered_parallel (catches unknown strategies).
	if p.Strategy != plan.StrategySingle && p.Strategy != plan.StrategyOrderedParallel {
		r.add("strategy", fmt.Sprintf("strategy must be %q or %q, got %q",
			plan.StrategySingle, plan.StrategyOrderedParallel, p.Strategy))
	}

	// 5. tasks non-empty.
	if len(p.Tasks) == 0 {
		r.add("tasks", "tasks must not be empty")
	}

	// 6. v1.0 max 3 tasks.
	if len(p.Tasks) > 3 {
		r.add("tasks", fmt.Sprintf("v1.0 supports at most 3 tasks, got %d", len(p.Tasks)))
	}

	// 7. single strategy requires exactly 1 task.
	if p.Strategy == plan.StrategySingle && len(p.Tasks) != 1 {
		r.add("tasks", fmt.Sprintf("single strategy requires exactly 1 task, got %d", len(p.Tasks)))
	}

	// 8. ordered_parallel strategy requires at least 2 tasks.
	if p.Strategy == plan.StrategyOrderedParallel && len(p.Tasks) < 2 {
		r.add("tasks", fmt.Sprintf("ordered_parallel strategy requires at least 2 tasks, got %d", len(p.Tasks)))
	}

	// Build taskID set with duplicate detection.
	taskIDs := make(map[string]bool, len(p.Tasks))
	for _, t := range p.Tasks {
		id := strings.TrimSpace(t.TaskID)
		if id == "" {
			continue // caught by per-task taskId check below
		}
		if taskIDs[id] {
			r.add("tasks", fmt.Sprintf("duplicate taskId %q", id))
		}
		taskIDs[id] = true
	}

	for i, t := range p.Tasks {
		prefix := fmt.Sprintf("tasks[%d]", i)
		id := strings.TrimSpace(t.TaskID)

		// Each task must have a taskId.
		if id == "" {
			r.add(prefix+".taskId", "taskId is required")
		}

		// 7. agentName must exist in registry.
		agentName := strings.TrimSpace(t.AgentName)
		if agentName == "" {
			r.add(prefix+".agentName", "agentName is required")
		} else if v.registry != nil {
			agent, ok := v.registry.Get(agentName)
			if !ok {
				r.add(prefix+".agentName", fmt.Sprintf("agent %q not found in registry", agentName))
			} else {
				// 8. capabilityIds must belong to target agent.
				for _, cid := range t.CapabilityIDs {
					if !containsCI(agent.CapabilityIDs, cid) {
						r.add(prefix+".capabilityIds",
							fmt.Sprintf("capability %q is not supported by agent %q", cid, agentName))
					}
				}

				// 9. expectedOutputs must be supported by target agent.
				for _, eo := range t.ExpectedOutputs {
					if !containsCI(agent.OutputTypes, eo) {
						r.add(prefix+".expectedOutputs",
							fmt.Sprintf("output type %q is not supported by agent %q", eo, agentName))
					}
				}
			}
		}

		// 10. dependsOn must reference existing taskIds.
		for _, dep := range t.DependsOn {
			if !taskIDs[dep] {
				r.add(prefix+".dependsOn",
					fmt.Sprintf("dependsOn references unknown task %q", dep))
			}
		}

		// 10a. dependsOn must not reference self.
		for _, dep := range t.DependsOn {
			if strings.EqualFold(strings.TrimSpace(dep), id) {
				r.add(prefix+".dependsOn",
					fmt.Sprintf("task must not depend on itself (%q)", id))
			}
		}

		// 10b. ordered_parallel tasks must not have depends_on.
		if p.Strategy == plan.StrategyOrderedParallel && len(t.DependsOn) > 0 {
			r.add(prefix+".dependsOn",
				"depends_on is not allowed in ordered_parallel strategy")
		}

		// 10c. taskContent must not be empty.
		if strings.TrimSpace(t.TaskContent) == "" {
			r.add(prefix+".taskContent", "taskContent must not be empty")
		}

		// 11. timeoutMs must be between 5000 and 180000.
		if t.TimeoutMs < 5000 || t.TimeoutMs > 180000 {
			r.add(prefix+".timeoutMs",
				fmt.Sprintf("timeoutMs must be between 5000 and 180000, got %d", t.TimeoutMs))
		}

		// 12. riskLevel must be low / medium / high.
		rl := strings.ToLower(strings.TrimSpace(t.RiskLevel))
		if rl != "low" && rl != "medium" && rl != "high" {
			r.add(prefix+".riskLevel",
				fmt.Sprintf("riskLevel must be low/medium/high, got %q", t.RiskLevel))
		}

		// 13. high risk tasks are not allowed for auto execution in v1.0.
		if rl == "high" {
			r.add(prefix+".riskLevel",
				"high risk tasks are not allowed for auto execution in v1.0")
		}

		// 14. taskContent must not contain internal URLs.
		if containsURL(t.TaskContent) {
			r.add(prefix+".taskContent",
				"taskContent must not contain internal URLs")
		}

		// 15. taskContent must not contain API keys / tokens / secrets.
		if containsSecret(t.TaskContent) {
			r.add(prefix+".taskContent",
				"taskContent must not contain secrets or tokens")
		}

		// 16. taskContent must not contain database DSNs.
		if containsDSN(t.TaskContent) {
			r.add(prefix+".taskContent",
				"taskContent must not contain database connection strings")
		}

		// 17. taskContent must not contain system prompts.
		if containsSystemPrompt(t.TaskContent) {
			r.add(prefix+".taskContent",
				"taskContent must not contain system prompts")
		}
	}

	return r
}

func (r *ValidationResult) add(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Field: field, Code: "INVALID", Message: message})
}

func containsCI(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}

// containsURL returns true if s contains internal URL patterns.
func containsURL(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{
		"http://", "https://",
		"localhost", "127.0.0.1", "0.0.0.0",
	}
	for _, pat := range patterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// containsSecret returns true if s contains API key or token-like strings.
func containsSecret(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{
		"sk-",          // common LLM API key prefix
		"api_key", "apikey", "api-key",
		"bearer ",      // token prefix
		"token=", "token:", "token ",
		"secret=", "secret:", "secret ",
		"password=", "password:", "password ",
		"credential",
	}
	for _, pat := range patterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// containsDSN returns true if s contains database connection string patterns.
func containsDSN(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{
		"mysql://", "postgres://", "postgresql://",
		"mongodb://", "sqlite://", "redis://",
		"jdbc:", "dsn=",
	}
	for _, pat := range patterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// containsSystemPrompt returns true if s contains system prompt-like text.
func containsSystemPrompt(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{
		"you are a", "you are an",
		"system prompt", "system instruction",
		"system:", "system message",
	}
	for _, pat := range patterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}
