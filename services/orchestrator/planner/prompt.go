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

// BuildSystemPrompt constructs the system prompt instructing the LLM how to
// act as an intent orchestrator. Agent information comes from the AgentLister
// provided at construction time.
func (b *PromptBuilder) BuildSystemPrompt() string {
	var buf strings.Builder

	buf.WriteString(`You are an intent orchestrator for a multi-agent platform. Your job is to analyze the user's request and create an execution plan.

## Available Agents
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
1. For code / backend / API / algorithm requests → use code-agent
2. For web UI / frontend / page / HTML / CSS requests → use web-agent
3. If the request needs BOTH frontend UI AND backend code → use "parallel" with both agents
4. Each step's agent_name MUST be one of the agent names listed under "Available Agents" above
5. taskContent (the "input" field) should rephrase the user's request specifically for the target agent
6. If the user explicitly names an agent, always use "single" mode with that agent
7. If unsure, default to code-agent with mode "single"
8. Set "confidence" between 0.0 and 1.0 to indicate how sure you are about this plan
9. "user_visible_summary" should be a short, friendly message like "I will ask code-agent to generate the server."
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

	if b.conversationType == "group" {
		buf.WriteString("\n\n## Context\nThis is a group chat. Multiple agents may participate.")
	}

	return buf.String()
}
