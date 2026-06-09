package planner

import (
	"fmt"
	"strings"
)

// PromptBuilder constructs system and user prompts for the LLM planner using
// real agent information from the registry (via AgentLister) as the primary
// source of agent capabilities, descriptions, and output modes.
//
// It does NOT rely on agentDefaults() as the primary agent information source.
// If no agents are provided (nil or empty lister), a minimal fallback prompt
// is generated with a warning.
type PromptBuilder struct {
	agents           []AgentInfoLite
	conversationType string
	executionPath    string
	allowedAgents    []string
}

// NewPromptBuilder creates a PromptBuilder backed by the given agent lister.
// The lister provides real agent metadata from the registry.
func NewPromptBuilder(lister AgentLister) *PromptBuilder {
	var agents []AgentInfoLite
	if lister != nil {
		agents = lister.List()
	}
	return &PromptBuilder{
		agents: agents,
	}
}

// WithConversationType sets the conversation type for contextual hints.
// Returns the builder for method chaining.
func (b *PromptBuilder) WithConversationType(ct string) *PromptBuilder {
	b.conversationType = ct
	return b
}

// WithExecutionPath sets the execution path for path-aware prompt construction.
func (b *PromptBuilder) WithExecutionPath(path string) *PromptBuilder {
	b.executionPath = path
	return b
}

// WithAllowedAgents sets the allowed agent boundary for path-aware prompt construction.
func (b *PromptBuilder) WithAllowedAgents(agents []string) *PromptBuilder {
	b.allowedAgents = agents
	return b
}

// BuildSystemPrompt constructs the system prompt instructing the LLM how to
// act as an intent orchestrator. Agent information comes from the AgentLister
// provided at construction time.
func (b *PromptBuilder) BuildSystemPrompt() string {
	var buf strings.Builder

	buf.WriteString(`You are an intent orchestrator for a multi-agent platform. Your job is to analyze the user's request and create an execution plan.

## Execution Context
`)

	// Include execution path context for path-aware planning.
	if b.executionPath != "" {
		fmt.Fprintf(&buf, "- Execution path: %s\n", b.executionPath)
		switch b.executionPath {
		case "single_chat":
			buf.WriteString("  This is a single_chat path — you MUST assign exactly one agent that is within the allowed boundary below.\n")
		case "group_chat":
			buf.WriteString("  This is a group_chat path — you may assign multiple agents, but ALL must be within the allowed boundary below.\n")
		case "main_agent_orchestration":
			buf.WriteString("  This is the auto-orchestration path — you may select any available agents listed below.\n")
		}
		buf.WriteString("\n")
	}

	// Include allowed boundary for path-aware enforcement.
	if len(b.allowedAgents) > 0 {
		buf.WriteString("## Allowed Agent Boundary\n")
		buf.WriteString("You MUST only select agents from this list for tasks and participants:\n")
		for _, a := range b.allowedAgents {
			fmt.Fprintf(&buf, "  - %s\n", a)
		}
		buf.WriteString("\n")
	}

	buf.WriteString(`## Available Agents
`)

	// Primary source: real agent info from the registry/lister.
	if len(b.agents) > 0 {
		for _, a := range b.agents {
			b.writeAgentBlock(&buf, a)
		}
	} else {
		buf.WriteString("(No agent information available — you must rely on the agent names in the user prompt.)\n\n")
	}

	buf.WriteString(`## Execution Modes
- "single": exactly one agent handles the entire request
- "parallel": multiple agents work on independent sub-tasks concurrently
- "sequential": agents execute in dependency order (each step depends on previous results)

## Rules
1. Only use code-agent if it is listed in the Allowed Agent Boundary or Available Agents above.
2. Only use web-agent if it is listed in the Allowed Agent Boundary or Available Agents above.
3. Only use document-agent if it is listed in the Allowed Agent Boundary or Available Agents above.
4. Each step's agent_name MUST be one of the agent names listed under "Available Agents" above.
5. taskContent (the "input" field) should rephrase the user's request specifically for the target agent.
6. If the user explicitly names an agent, always use "single" mode with that agent.
7. If the best-fit agent is not in the boundary, do not add it; write a warning instead.
8. Set "confidence" between 0.0 and 1.0 to indicate how sure you are about this plan.
9. "user_visible_summary" should be a short, friendly message like "I will use the best-fit agent for your task."
   Do NOT include internal configuration in user_visible_summary.

## Output Format
Return ONLY a valid JSON object. No markdown, no explanation, no code fences:
{
  "intent": "brief summary of the user's goal",
  "mode": "single" | "parallel" | "sequential",
  "confidence": 0.86,
  "steps": [
    {
      "id": "step-1",
      "agent_name": "exact-agent-name",
      "input": "rephrased task for this agent",
      "depends_on": [],
      "reason": "why this agent was chosen"
    }
  ],
  "user_visible_summary": "short friendly message for the user"
}`)

	return buf.String()
}

// writeAgentBlock writes a single agent's information block into the prompt.
func (b *PromptBuilder) writeAgentBlock(buf *strings.Builder, a AgentInfoLite) {
	buf.WriteString(fmt.Sprintf("- name: %s\n", a.Name))
	if a.Description != "" {
		buf.WriteString(fmt.Sprintf("  description: %s\n", a.Description))
	}
	if len(a.CapabilityIDs) > 0 {
		buf.WriteString(fmt.Sprintf("  capabilities: [%s]\n", strings.Join(a.CapabilityIDs, ", ")))
	}
	if len(a.OutputModes) > 0 {
		buf.WriteString(fmt.Sprintf("  outputModes: [%s]\n", strings.Join(a.OutputModes, ", ")))
	}
	buf.WriteString("\n")
}

// BuildUserPrompt constructs the user prompt containing the user's message
// and optional context hints.
func (b *PromptBuilder) BuildUserPrompt(userMessage string, agentName string) string {
	var buf strings.Builder

	buf.WriteString("## User Message\n")
	buf.WriteString(userMessage)

	if strings.TrimSpace(agentName) != "" {
		buf.WriteString("\n\n## Hint\n")
		buf.WriteString(fmt.Sprintf("The user selected agent: %s. Use this as a strong preference.", agentName))
	}

	// Include execution path context in user prompt for reinforcement.
	if b.executionPath != "" {
		buf.WriteString(fmt.Sprintf("\n\n## Execution Path\nCurrent execution path: %s.", b.executionPath))
	}

	// Include boundary reminder.
	if len(b.allowedAgents) > 0 {
		buf.WriteString(fmt.Sprintf("\n\n## Boundary Constraint\nOnly select agents from: %s.", strings.Join(b.allowedAgents, ", ")))
	}

	if b.conversationType == "group" {
		buf.WriteString("\n\n## Context\nThis is a group chat. Multiple agents may participate.")
	}

	return buf.String()
}
