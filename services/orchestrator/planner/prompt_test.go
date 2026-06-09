package planner

import (
	"strings"
	"testing"
)

// mockAgentLister implements AgentLister for tests.
type mockAgentLister struct {
	agents []AgentInfoLite
}

func (m *mockAgentLister) List() []AgentInfoLite {
	return m.agents
}

func testAgentInfos() []AgentInfoLite {
	return []AgentInfoLite{
		{
			Name:          "code-agent",
			Description:   "Generates and explains code",
			CapabilityIDs: []string{"code_generation", "code_explanation"},
			OutputModes:   []string{"text", "code", "artifact_ref"},
		},
		{
			Name:          "web-agent",
			Description:   "Generates webpages and HTML previews",
			CapabilityIDs: []string{"web_generation", "html_generation"},
			OutputModes:   []string{"text", "webpage", "html", "artifact_ref"},
		},
	}
}

// ---------------------------------------------------------------------------
// System prompt tests
// ---------------------------------------------------------------------------

func TestPrompt_IncludesAgentNames(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// Agent names from the lister must appear.
	if !strings.Contains(prompt, "code-agent") {
		t.Error("expected prompt to contain 'code-agent' from lister")
	}
	if !strings.Contains(prompt, "web-agent") {
		t.Error("expected prompt to contain 'web-agent' from lister")
	}
}

func TestPrompt_IncludesAgentDescriptions(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// Agent descriptions from the lister must appear.
	if !strings.Contains(prompt, "Generates and explains code") {
		t.Error("expected prompt to contain code-agent description from lister")
	}
	if !strings.Contains(prompt, "Generates webpages and HTML previews") {
		t.Error("expected prompt to contain web-agent description from lister")
	}
}

func TestPrompt_IncludesAgentCapabilities(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// Capabilities from the lister must appear.
	if !strings.Contains(prompt, "code_generation") {
		t.Error("expected prompt to contain 'code_generation' from lister")
	}
	if !strings.Contains(prompt, "web_generation") {
		t.Error("expected prompt to contain 'web_generation' from lister")
	}
}

func TestPrompt_IncludesAgentOutputModes(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// Output modes from the lister must appear.
	if !strings.Contains(prompt, "artifact_ref") {
		t.Error("expected prompt to contain 'artifact_ref' from lister outputModes")
	}
	if !strings.Contains(prompt, "webpage") {
		t.Error("expected prompt to contain 'webpage' from lister outputModes")
	}
}

func TestPrompt_IncludesSchema(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// The output schema must be present.
	if !strings.Contains(prompt, `"intent"`) {
		t.Error("expected prompt to include output schema (intent field)")
	}
	if !strings.Contains(prompt, `"mode"`) {
		t.Error("expected prompt to include output schema (mode field)")
	}
	if !strings.Contains(prompt, `"steps"`) {
		t.Error("expected prompt to include output schema (steps field)")
	}
	if !strings.Contains(prompt, `"confidence"`) {
		t.Error("expected prompt to include output schema (confidence field)")
	}
	if !strings.Contains(prompt, `"user_visible_summary"`) {
		t.Error("expected prompt to include output schema (user_visible_summary field)")
	}
}

func TestPrompt_IncludesExecutionModes(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// All three modes should be documented.
	if !strings.Contains(prompt, `"single"`) {
		t.Error("expected prompt to document 'single' mode")
	}
	if !strings.Contains(prompt, `"parallel"`) {
		t.Error("expected prompt to document 'parallel' mode")
	}
	if !strings.Contains(prompt, `"sequential"`) {
		t.Error("expected prompt to document 'sequential' mode")
	}
}

func TestPrompt_NoSecretsLeaked(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// Secrets that must never appear.
	forbidden := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"sk-",
		"http://127.0.0.1",
		"localhost:8081",
		"localhost:8082",
		"mysql://",
		"postgres://",
	}
	for _, f := range forbidden {
		if strings.Contains(prompt, f) {
			t.Errorf("prompt leaked forbidden string: %q", f)
		}
	}
}

func TestPrompt_EmptyListerGraceful(t *testing.T) {
	// Nil lister should not panic.
	pb := NewPromptBuilder(nil)
	prompt := pb.BuildSystemPrompt()

	if prompt == "" {
		t.Error("expected non-empty prompt even with nil lister")
	}
	if !strings.Contains(prompt, "No agent information available") {
		t.Error("expected warning about missing agent info")
	}
}

func TestPrompt_EmptyAgentListGraceful(t *testing.T) {
	lister := &mockAgentLister{agents: nil}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	if prompt == "" {
		t.Error("expected non-empty prompt even with empty agent list")
	}
	if !strings.Contains(prompt, "No agent information available") {
		t.Error("expected warning about missing agent info")
	}
}

// ---------------------------------------------------------------------------
// User prompt tests
// ---------------------------------------------------------------------------

func TestPrompt_UserPromptIncludesMessage(t *testing.T) {
	pb := NewPromptBuilder(nil)
	prompt := pb.BuildUserPrompt("build a REST API", "")
	if !strings.Contains(prompt, "build a REST API") {
		t.Errorf("expected user message in prompt, got %q", prompt)
	}
}

func TestPrompt_UserPromptIncludesAgentHint(t *testing.T) {
	pb := NewPromptBuilder(nil)
	prompt := pb.BuildUserPrompt("build a UI", "web-agent")
	if !strings.Contains(prompt, "web-agent") {
		t.Errorf("expected agent hint in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "strong preference") {
		t.Error("expected 'strong preference' language for explicit agent hint")
	}
}

func TestPrompt_UserPromptNoAgentHintWhenEmpty(t *testing.T) {
	pb := NewPromptBuilder(nil)
	prompt := pb.BuildUserPrompt("do something", "")
	if strings.Contains(prompt, "strong preference") {
		t.Error("expected no agent hint when agent name is empty")
	}
}

func TestPrompt_UserPromptGroupChatContext(t *testing.T) {
	pb := NewPromptBuilder(nil).WithConversationType("group")
	prompt := pb.BuildUserPrompt("something", "")
	if !strings.Contains(prompt, "group chat") {
		t.Errorf("expected group chat context, got %q", prompt)
	}
}

func TestPrompt_UserPromptNoGroupContextForDirect(t *testing.T) {
	pb := NewPromptBuilder(nil).WithConversationType("direct")
	prompt := pb.BuildUserPrompt("something", "")
	if strings.Contains(prompt, "group chat") {
		t.Error("expected no group chat context for direct conversation")
	}
}

func TestPrompt_UsesListerNotDefaults(t *testing.T) {
	// The primary source of agent info must be the lister, not agentDefaults().
	// agentDefaults("code-agent") uses description "code generation and explanation"
	// which is different from what we put in the mock.
	lister := &mockAgentLister{agents: []AgentInfoLite{
		{
			Name:          "code-agent",
			Description:   "Custom registry description for code agent",
			CapabilityIDs: []string{"custom_capability"},
			OutputModes:   []string{"custom_output"},
		},
	}}
	pb := NewPromptBuilder(lister)
	prompt := pb.BuildSystemPrompt()

	// Must contain the registry-provided custom values.
	if !strings.Contains(prompt, "Custom registry description for code agent") {
		t.Error("expected prompt to use registry-provided description, not agentDefaults()")
	}
	if !strings.Contains(prompt, "custom_capability") {
		t.Error("expected prompt to use registry-provided capabilities, not agentDefaults()")
	}
	if !strings.Contains(prompt, "custom_output") {
		t.Error("expected prompt to use registry-provided outputModes, not agentDefaults()")
	}
}

// ---------------------------------------------------------------------------
// Execution path and boundary tests (P0 #5)
// ---------------------------------------------------------------------------

func TestMainAgentPromptContainsExecutionPathAndBoundary(t *testing.T) {
	lister := &mockAgentLister{agents: testAgentInfos()}

	t.Run("system prompt contains execution path", func(t *testing.T) {
		pb := NewPromptBuilder(lister).
			WithExecutionPath("single_chat").
			WithAllowedAgents([]string{"code-agent"})
		prompt := pb.BuildSystemPrompt()

		if !strings.Contains(prompt, "Execution Context") {
			t.Error("expected 'Execution Context' section in system prompt")
		}
		if !strings.Contains(prompt, "Execution path: single_chat") {
			t.Error("expected 'Execution path: single_chat' in system prompt")
		}
		if !strings.Contains(prompt, "single_chat path") {
			t.Error("expected path-specific instructions for single_chat")
		}
	})

	t.Run("system prompt contains allowed agent boundary", func(t *testing.T) {
		pb := NewPromptBuilder(lister).
			WithExecutionPath("group_chat").
			WithAllowedAgents([]string{"code-agent", "web-agent"})
		prompt := pb.BuildSystemPrompt()

		if !strings.Contains(prompt, "Allowed Agent Boundary") {
			t.Error("expected 'Allowed Agent Boundary' section in system prompt")
		}
		if !strings.Contains(prompt, "code-agent") {
			t.Error("expected 'code-agent' in boundary section")
		}
		if !strings.Contains(prompt, "web-agent") {
			t.Error("expected 'web-agent' in boundary section")
		}
		if !strings.Contains(prompt, "MUST only select agents from this list") {
			t.Error("expected boundary enforcement instruction")
		}
	})

	t.Run("user prompt contains execution path", func(t *testing.T) {
		pb := NewPromptBuilder(lister).
			WithExecutionPath("main_agent_orchestration").
			WithAllowedAgents([]string{"code-agent", "web-agent"})
		prompt := pb.BuildUserPrompt("build a full-stack app", "")

		if !strings.Contains(prompt, "Execution Path") {
			t.Error("expected 'Execution Path' in user prompt")
		}
		if !strings.Contains(prompt, "main_agent_orchestration") {
			t.Error("expected execution path value in user prompt")
		}
	})

	t.Run("user prompt contains boundary constraint", func(t *testing.T) {
		pb := NewPromptBuilder(lister).
			WithExecutionPath("single_chat").
			WithAllowedAgents([]string{"code-agent"})
		prompt := pb.BuildUserPrompt("write code", "")

		if !strings.Contains(prompt, "Boundary Constraint") {
			t.Error("expected 'Boundary Constraint' in user prompt")
		}
		if !strings.Contains(prompt, "code-agent") {
			t.Error("expected boundary agent in user prompt constraint")
		}
	})

	t.Run("prompts without boundary omit sections", func(t *testing.T) {
		pb := NewPromptBuilder(lister)
		prompt := pb.BuildSystemPrompt()
		// The boundary section header uses double hash; rule text may mention
		// "Allowed Agent Boundary" inline, so check for the section header.
		if strings.Contains(prompt, "## Allowed Agent Boundary") {
			t.Error("should NOT have boundary section when no boundary is set")
		}
		if strings.Contains(prompt, "Execution path:") {
			t.Error("should NOT have execution path when not set")
		}
	})
}

// TestPrompt_BoundaryOnlyReviewAgent_NoCodeAgentHint verifies that when the
// boundary only contains "review-agent", the prompt does NOT hardcode hints
// suggesting code-agent or web-agent must be used.
func TestPrompt_BoundaryOnlyReviewAgent_NoCodeAgentHint(t *testing.T) {
	lister := &mockAgentLister{agents: []AgentInfoLite{
		{
			Name:          "review-agent",
			Description:   "Reviews code for security and style issues",
			CapabilityIDs: []string{"code_review", "security_audit"},
			OutputModes:   []string{"text"},
		},
	}}

	pb := NewPromptBuilder(lister).
		WithExecutionPath("single_chat").
		WithAllowedAgents([]string{"review-agent"})

	systemPrompt := pb.BuildSystemPrompt()
	userPrompt := pb.BuildUserPrompt("review my code", "")

	// The old hardcoded rule "If unsure, default to code-agent" must NOT appear.
	if strings.Contains(systemPrompt, "default to code-agent") {
		t.Error("prompt must NOT contain 'default to code-agent' — boundary is review-agent only")
	}
	// The old hardcoded rule "For code / backend / API → use code-agent" must NOT appear.
	// (The new boundary-aware rule "Only use code-agent if it is listed..." may use
	// "use code-agent" in a different form, which is fine — it references the boundary.)
	if strings.Contains(systemPrompt, "For code / backend") {
		t.Error("prompt must NOT contain old hardcoded 'For code requests → use code-agent' rule")
	}
	if strings.Contains(systemPrompt, "For web UI") {
		t.Error("prompt must NOT contain old hardcoded 'For web UI → use web-agent' rule")
	}
	// Should contain the boundary-aware rule instead.
	if !strings.Contains(systemPrompt, "best-fit agent is not in the boundary") {
		t.Error("prompt must contain boundary-aware fallback rule")
	}

	// Boundary must list review-agent.
	if !strings.Contains(systemPrompt, "review-agent") {
		t.Error("prompt must contain review-agent in boundary and agent listing")
	}

	// User prompt must contain boundary constraint.
	if !strings.Contains(userPrompt, "review-agent") {
		t.Error("user prompt must mention review-agent in boundary constraint")
	}
}
