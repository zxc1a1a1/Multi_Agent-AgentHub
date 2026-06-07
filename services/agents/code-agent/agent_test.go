package codeagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var _ adk.Agent = (*CodeAgent)(nil)

var _ adk.StreamingAgent = (*CodeAgent)(nil)

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

func TestCodeAgent_GenerateStream_MultipleChunks(t *testing.T) {
	agent := NewCodeAgent(Config{})
	req := buildRequest("write a go http server")

	var chunks []string
	for resp, err := range agent.GenerateStream(context.Background(), req) {
		if err != nil {
			t.Fatalf("GenerateStream returned error: %v", err)
		}
		for _, part := range resp.Parts {
			textPart, ok := part.(adk.TextPart)
			if ok && textPart.Text != "" {
				chunks = append(chunks, textPart.Text)
			}
		}
	}

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 stream chunks, got %d", len(chunks))
	}

	fullText := strings.Join(chunks, "")
	if !strings.Contains(fullText, "code-agent v0.1 mock response") {
		t.Fatalf("streamed text missing mock marker: %q", fullText)
	}
}

func TestCodeAgent_GenerateStream_ErrorOnNilRequest(t *testing.T) {
	agent := NewCodeAgent(Config{})
	for resp, err := range agent.GenerateStream(context.Background(), nil) {
		if err == nil {
			t.Fatal("expected error for nil request")
		}
		if resp != nil {
			t.Fatal("expected nil response with error")
		}
		return
	}
	t.Fatal("expected at least one yield from GenerateStream")
}

func TestCodeAgent_GenerateStream_NoUserText(t *testing.T) {
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

	for resp, err := range agent.GenerateStream(context.Background(), req) {
		if err == nil {
			t.Fatal("expected error for missing user text")
		}
		if resp != nil {
			t.Fatal("expected nil response with error")
		}
		return
	}
}

func TestCodeAgent_GenerateStream_FallsBackToMock(t *testing.T) {
	// Without an LLM model configured, GenerateStream uses the mock path.
	agent := NewCodeAgent(Config{})
	req := buildRequest("hello")

	var chunkCount int
	for resp, err := range agent.GenerateStream(context.Background(), req) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != nil && len(resp.Parts) > 0 {
			chunkCount++
		}
	}

	if chunkCount == 0 {
		t.Fatal("expected at least one chunk from mock streaming")
	}
}
