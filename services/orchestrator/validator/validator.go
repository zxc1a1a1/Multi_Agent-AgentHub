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
		r.Errors = append(r.Errors, ValidationError{Field: "plan", Message: "plan is nil"})
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

	// 4. strategy must be single or ordered_parallel.
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

	taskIDs := make(map[string]bool, len(p.Tasks))
	for _, t := range p.Tasks {
		taskIDs[t.TaskID] = true
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
	}

	return r
}

func (r *ValidationResult) add(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Field: field, Message: message})
}

func containsCI(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
