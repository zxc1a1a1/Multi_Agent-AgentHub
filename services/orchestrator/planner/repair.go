package planner

import (
	"context"
	"fmt"
	"strings"
)

// PlanRepairer attempts to fix an invalid LLM plan output via a single LLM call.
// It constructs a focused repair prompt containing the original output, the
// specific errors found, the list of available agents, the expected JSON schema,
// and an instruction to return only corrected JSON.
//
// Repair is one-shot only — never loop.
type PlanRepairer struct {
	model PlannerModel
}

// NewPlanRepairer creates a PlanRepairer backed by the given PlannerModel.
func NewPlanRepairer(model PlannerModel) *PlanRepairer {
	return &PlanRepairer{model: model}
}

// Repair asks the LLM to fix the errors in the original output and returns the
// corrected JSON. It returns an error if the model call fails.
//
// The repair prompt includes:
//   - The original raw LLM output.
//   - A list of failures (parse errors, normalization errors, validation errors).
//   - Available agents and their descriptions.
//   - The expected JSON schema.
//   - An instruction to return corrected JSON only.
func (r *PlanRepairer) Repair(ctx context.Context, originalOutput string, failures []ValError, lister AgentLister) (string, error) {
	if r.model == nil {
		return "", fmt.Errorf("repair: no model available")
	}

	systemPrompt := "You are a JSON repair assistant. Your ONLY job is to fix the JSON based on the errors and constraints provided. Output ONLY the corrected JSON object — no markdown, no code fences, no explanation."

	userPrompt := r.buildRepairPrompt(originalOutput, failures, lister)

	return r.model.Generate(ctx, systemPrompt, userPrompt)
}

// buildRepairPrompt constructs the user-facing repair prompt.
func (r *PlanRepairer) buildRepairPrompt(originalOutput string, failures []ValError, lister AgentLister) string {
	var buf strings.Builder

	buf.WriteString("You previously generated an orchestration plan, but it contained errors.\n")
	buf.WriteString("Please fix ALL errors listed below and return ONLY the corrected JSON.\n\n")

	// 1. Original output.
	buf.WriteString("## Original Output\n")
	buf.WriteString(originalOutput)
	buf.WriteString("\n\n")

	// 2. Errors found.
	buf.WriteString("## Errors Found\n")
	for _, f := range failures {
		buf.WriteString(fmt.Sprintf("- [%s] %s\n", f.Code, f.Message))
	}
	buf.WriteString("\n")

	// 3. Available agents.
	buf.WriteString("## Available Agents\n")
	if lister != nil {
		for _, a := range lister.List() {
			desc := a.Description
			if desc == "" {
				desc = a.Name
			}
			buf.WriteString(fmt.Sprintf("- name: %s\n  description: %s\n\n", a.Name, desc))
		}
	} else {
		buf.WriteString("(No agent registry available — fix the agent names in the plan directly.)\n\n")
	}

	// 4. Expected JSON schema.
	buf.WriteString(`## Expected JSON Schema
{
  "intent": "brief summary of the user's goal",
  "mode": "single" | "parallel" | "sequential",
  "confidence": 0.86,
  "steps": [
    {
      "id": "step-1",
      "agent_name": "exact-agent-name-from-above-list",
      "input": "rephrased task for this agent",
      "depends_on": [],
      "reason": "why this agent was chosen"
    }
  ],
  "user_visible_summary": "short friendly message for the user"
}

## Instructions
1. Return ONLY the corrected JSON object. No markdown, no explanation, no code fences.
2. Every step's "agent_name" MUST exactly match one of the agents listed under "Available Agents" above.
3. Valid modes are: "single" (one agent), "parallel" (multiple independent agents), "sequential" (agents with dependencies).
   However, "sequential" is currently NOT SUPPORTED — do NOT use it.
4. Every step must have a non-empty "input" field.
5. The "steps" array must not be empty.
`)

	return buf.String()
}
