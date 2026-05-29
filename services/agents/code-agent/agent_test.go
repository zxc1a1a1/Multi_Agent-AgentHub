package codeagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var _ adk.Agent = (*CodeAgent)(nil)

func TestNewCodeAgent_Defaults(t *testing.T) {
	agent := NewCodeAgent(Config{})
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.name != defaultAgentName {
		t.Fatalf("unexpected default name: got=%q want=%q", agent.name, defaultAgentName)
	}
	if agent.description != defaultAgentDescription {
		t.Fatalf("unexpected default description: got=%q want=%q", agent.description, defaultAgentDescription)
	}
	if agent.version != defaultAgentVersion {
		t.Fatalf("unexpected default version: got=%q want=%q", agent.version, defaultAgentVersion)
	}
}

func TestCodeAgent_InterfaceCompliance(t *testing.T) {
	agent := NewCodeAgent(Config{})
	if agent == nil {
		t.Fatal("expected non-nil code agent")
	}
}

func TestCodeAgent_Name(t *testing.T) {
	agent := NewCodeAgent(Config{Name: "custom-code-agent"})
	if got := agent.Name(); got != "custom-code-agent" {
		t.Fatalf("unexpected name: got=%q want=%q", got, "custom-code-agent")
	}
}

func TestCodeAgent_GenerateText(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "mock_key_should_not_be_used")

	agent := NewCodeAgent(Config{})
	resp, err := agent.Generate(context.Background(), buildRequest("please explain this"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}
	if !strings.Contains(textPart.Text, "code-agent v0.1 mock response") {
		t.Fatalf("unexpected text response: %q", textPart.Text)
	}
	if strings.Contains(textPart.Text, "mock_key_should_not_be_used") {
		t.Fatalf("response leaked env secret: %q", textPart.Text)
	}
}

func TestCodeAgent_GenerateUsesLastUserMessage(t *testing.T) {
	agent := NewCodeAgent(Config{})
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleUser,
				Parts: []adk.Part{
					adk.TextPart{Text: "just chat"},
				},
			},
			{
				Role: adk.RoleAssistant,
				Parts: []adk.Part{
					adk.TextPart{Text: "assistant reply"},
				},
			},
			{
				Role: adk.RoleUser,
				Parts: []adk.Part{
					adk.TextPart{Text: "请给我 go http server 代码"},
				},
			},
		},
	}

	resp, err := agent.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}
	if !strings.Contains(textPart.Text, "package main") {
		t.Fatalf("expected code-like response built from last user message, got=%q", textPart.Text)
	}
}

func TestCodeAgent_GenerateNoUserText(t *testing.T) {
	agent := NewCodeAgent(Config{})
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleAssistant,
				Parts: []adk.Part{
					adk.TextPart{Text: "assistant only"},
				},
			},
		},
	}

	_, err := agent.Generate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing user text")
	}
}

func TestCodeAgent_DoesNotExposeThinking(t *testing.T) {
	agent := NewCodeAgent(Config{})
	resp, err := agent.Generate(context.Background(), buildRequest("hello"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	for _, part := range resp.Parts {
		if _, ok := part.(adk.ThinkingPart); ok {
			t.Fatalf("thinking part should not be exposed: %+v", part)
		}
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response failed: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "thinking") {
		t.Fatalf("response should not include thinking payload: %s", string(raw))
	}
}

func buildRequest(text string) *adk.GenerateRequest {
	return &adk.GenerateRequest{
		Contents: []*adk.Content{
			{
				Role: adk.RoleUser,
				Parts: []adk.Part{
					adk.TextPart{Text: text},
				},
			},
		},
	}
}

func firstTextPart(parts []adk.Part) (adk.TextPart, bool) {
	for _, part := range parts {
		if value, ok := part.(adk.TextPart); ok {
			return value, true
		}
	}
	return adk.TextPart{}, false
}
