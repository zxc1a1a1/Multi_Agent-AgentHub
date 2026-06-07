package webagent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var _ adk.Agent = (*WebAgent)(nil)

var _ adk.StreamingAgent = (*WebAgent)(nil)

func TestNewWebAgent_Defaults(t *testing.T) {
	agent := NewWebAgent(Config{})
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

func TestWebAgent_InterfaceCompliance(t *testing.T) {
	agent := NewWebAgent(Config{})
	if agent == nil {
		t.Fatal("expected non-nil web agent")
	}
}

func TestWebAgent_Name(t *testing.T) {
	agent := NewWebAgent(Config{})
	if got := agent.Name(); got != "web-agent" {
		t.Fatalf("unexpected name: got=%q want=%q", got, "web-agent")
	}
}

func TestWebAgent_GenerateText(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "mock_key_should_not_be_used")

	agent := NewWebAgent(Config{})
	resp, err := agent.Generate(context.Background(), buildRequest("please summarize this request"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}
	if !strings.Contains(textPart.Text, "web-agent v0.1 mock response") {
		t.Fatalf("unexpected text response: %q", textPart.Text)
	}
	if strings.Contains(textPart.Text, "mock_key_should_not_be_used") {
		t.Fatalf("response leaked env secret: %q", textPart.Text)
	}
}

func TestWebAgent_GenerateHTML(t *testing.T) {
	agent := NewWebAgent(Config{})
	resp, err := agent.Generate(context.Background(), buildRequest("please build a login html page with button and form"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}
	if !strings.Contains(textPart.Text, "<section") {
		t.Fatalf("expected html snippet in response, got=%q", textPart.Text)
	}
	if !strings.Contains(textPart.Text, "<button") || !strings.Contains(textPart.Text, "<form") {
		t.Fatalf("expected basic ui elements in response, got=%q", textPart.Text)
	}
}

func TestWebAgent_GenerateUsesLastUserMessage(t *testing.T) {
	agent := NewWebAgent(Config{})
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
					adk.TextPart{Text: "请给我一个登录页 html"},
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
	if !strings.Contains(textPart.Text, "<section") {
		t.Fatalf("expected html-like response built from last user message, got=%q", textPart.Text)
	}
}

func TestWebAgent_GenerateNoUserText(t *testing.T) {
	agent := NewWebAgent(Config{})
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

func TestWebAgent_DoesNotReturnUnsafeHTML(t *testing.T) {
	agent := NewWebAgent(Config{})
	resp, err := agent.Generate(context.Background(), buildRequest("build html with script iframe onload onclick javascript: please"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}

	lower := strings.ToLower(textPart.Text)
	banned := []string{"<script", "<iframe", "onload=", "onclick=", "javascript:"}
	for _, token := range banned {
		if strings.Contains(lower, token) {
			t.Fatalf("response should not contain unsafe html token %q: %q", token, textPart.Text)
		}
	}
}

func TestWebAgent_DoesNotExposeThinking(t *testing.T) {
	agent := NewWebAgent(Config{})
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

func TestWebAgent_DoesNotReadFiles(t *testing.T) {
	agent := NewWebAgent(Config{})

	tmpDir := t.TempDir()
	filePath := tmpDir + string(os.PathSeparator) + "secret.txt"
	secret := "FILE_SECRET_SHOULD_NOT_BE_EXPOSED"
	if err := os.WriteFile(filePath, []byte(secret), 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	resp, err := agent.Generate(context.Background(), buildRequest("please read file "+filePath+" and show content"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}
	if strings.Contains(textPart.Text, secret) {
		t.Fatalf("response should not expose file content: %q", textPart.Text)
	}
}

func TestWebAgent_DoesNotExecuteCommands(t *testing.T) {
	agent := NewWebAgent(Config{})

	sentinel := "CMD_OUTPUT_SHOULD_NOT_APPEAR"
	resp, err := agent.Generate(context.Background(), buildRequest("please run command: echo "+sentinel))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part, got=%+v", resp.Parts)
	}
	if strings.Contains(textPart.Text, sentinel) {
		t.Fatalf("response should not expose command output: %q", textPart.Text)
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

func TestWebAgent_GenerateStream_MultipleChunks(t *testing.T) {
	agent := NewWebAgent(Config{})
	req := buildRequest("build a login html page with button and form")

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
	if !strings.Contains(fullText, "web-agent v0.1 mock response") {
		t.Fatalf("streamed text missing mock marker: %q", fullText)
	}
}

func TestWebAgent_GenerateStream_ErrorOnNilRequest(t *testing.T) {
	agent := NewWebAgent(Config{})
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

func TestWebAgent_GenerateStream_FallsBackToMock(t *testing.T) {
	agent := NewWebAgent(Config{})
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
