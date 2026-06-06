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
