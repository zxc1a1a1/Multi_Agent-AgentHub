package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

func TestAnthropicProvider_Name(t *testing.T) {
	p := anthropicProvider{}
	if got := p.Name(); got != anthropicProviderName {
		t.Fatalf("unexpected provider name: %s", got)
	}
}

func TestAnthropicProvider_CreateModel(t *testing.T) {
	t.Setenv("TEST_ANTHROPIC_TOKEN", "test-token")
	p := anthropicProvider{}

	model, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Name:      "anthropic-main",
		Provider:  anthropicProviderName,
		APIKeyEnv: "TEST_ANTHROPIC_TOKEN",
		Model:     "claude-test",
		BaseURL:   "http://127.0.0.1:18080/",
		MaxTokens: 512,
	})
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	if model == nil {
		t.Fatalf("expected non-nil model")
	}
	if _, ok := model.(adk.Model); !ok {
		t.Fatalf("created model does not implement adk.Model: %T", model)
	}
}

func TestAnthropicProvider_CreateModelMissingAPIKeyEnv(t *testing.T) {
	p := anthropicProvider{}
	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Model: "claude-test",
	})
	if err == nil {
		t.Fatalf("expected api_key_env validation error")
	}
}

func TestAnthropicProvider_CreateModelMissingEnvValue(t *testing.T) {
	t.Setenv("TEST_ANTHROPIC_TOKEN", "")
	p := anthropicProvider{}

	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		APIKeyEnv: "TEST_ANTHROPIC_TOKEN",
		Model:     "claude-test",
	})
	if err == nil {
		t.Fatalf("expected missing env value error")
	}
}

func TestAnthropicModel_GenerateText(t *testing.T) {
	var gotReq anthropicRequest
	var gotPath string
	var gotAPIKey string
	var gotVersion string
	var gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotContentType = r.Header.Get("content-type")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"hello from anthropic"}],"stop_reason":"end_turn","usage":{"input_tokens":7,"output_tokens":3}}`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	req := &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleSystem, Parts: []adk.Part{adk.TextPart{Text: "you are safe"}}},
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "question"}}},
			{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "previous answer"}}},
			{Role: adk.RoleUser, Parts: []adk.Part{adk.ToolResultPart{CallID: "call-1", Name: "x", Content: "tool data"}}},
		},
	}

	resp, err := model.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("unexpected endpoint: %s", gotPath)
	}
	if gotAPIKey != "test-token" {
		t.Fatalf("unexpected x-api-key header: %q", gotAPIKey)
	}
	if gotVersion != anthropicAPIVersion {
		t.Fatalf("unexpected anthropic-version header: %q", gotVersion)
	}
	if !strings.Contains(strings.ToLower(gotContentType), "application/json") {
		t.Fatalf("unexpected content-type: %q", gotContentType)
	}
	if gotReq.Model != "claude-test" {
		t.Fatalf("unexpected request model: %q", gotReq.Model)
	}
	if gotReq.System != "you are safe" {
		t.Fatalf("unexpected system prompt: %q", gotReq.System)
	}
	if len(gotReq.Messages) != 3 {
		t.Fatalf("unexpected message count: %d", len(gotReq.Messages))
	}
	if gotReq.Messages[0].Role != "user" || !strings.Contains(gotReq.Messages[0].Content, "question") {
		t.Fatalf("unexpected first message: %+v", gotReq.Messages[0])
	}
	if gotReq.Messages[1].Role != "assistant" {
		t.Fatalf("unexpected second role: %s", gotReq.Messages[1].Role)
	}
	if resp.FinishReason != adk.FinishStop {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if len(resp.Parts) != 1 {
		t.Fatalf("unexpected part count: %d", len(resp.Parts))
	}
	part, ok := resp.Parts[0].(adk.TextPart)
	if !ok || part.Text != "hello from anthropic" {
		t.Fatalf("unexpected text part: %#v", resp.Parts[0])
	}
}

func TestAnthropicModel_GenerateToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":[{"type":"tool_use","id":"tool-1","name":"lookup","input":{"q":"a"}}],"stop_reason":"tool_use","usage":{"input_tokens":5,"output_tokens":2}}`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "find"}}}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if resp.FinishReason != adk.FinishToolUse {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if len(resp.Parts) != 1 {
		t.Fatalf("unexpected part count: %d", len(resp.Parts))
	}
	toolPart, ok := resp.Parts[0].(adk.ToolCallPart)
	if !ok {
		t.Fatalf("expected ToolCallPart, got %T", resp.Parts[0])
	}
	if toolPart.ID != "tool-1" || toolPart.Name != "lookup" {
		t.Fatalf("unexpected tool call: %+v", toolPart)
	}
	if strings.TrimSpace(string(toolPart.Arguments)) != `{"q":"a"}` {
		t.Fatalf("unexpected tool args: %s", string(toolPart.Arguments))
	}
}

func TestAnthropicModel_GenerateMaxTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"truncated"}],"stop_reason":"max_tokens","usage":{"input_tokens":10,"output_tokens":10}}`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "long prompt"}}}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if resp.FinishReason != adk.FinishMaxToken {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
}

func TestAnthropicModel_GenerateUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":9,"output_tokens":4}}`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "usage?"}}}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if resp.Usage == nil {
		t.Fatalf("expected usage metadata")
	}
	if resp.Usage.InputTokens != 9 || resp.Usage.OutputTokens != 4 || resp.Usage.TotalTokens != 13 {
		t.Fatalf("unexpected usage: %#v", resp.Usage)
	}
}

func TestAnthropicModel_HTTPErrorSanitized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `upstream failed secret=fake-token`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	_, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
	})
	if err == nil {
		t.Fatalf("expected http error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected status in error, got %v", err)
	}
	if strings.Contains(err.Error(), "fake-token") {
		t.Fatalf("error leaked response body: %v", err)
	}
}

func TestAnthropicModel_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	_, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
	})
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestAnthropicModel_GenerateStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"stream result"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer srv.Close()

	model := newAnthropicTestModel(t, srv.URL)
	stream := model.GenerateStream(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "stream"}}}},
	})

	count := 0
	stream(func(resp *adk.GenerateResponse, err error) bool {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if resp == nil {
			t.Fatalf("expected non-nil response")
		}
		count++
		return true
	})
	if count != 1 {
		t.Fatalf("expected single stream yield, got %d", count)
	}
}

func newAnthropicTestModel(t *testing.T, baseURL string) *anthropicModel {
	t.Helper()
	t.Setenv("TEST_ANTHROPIC_TOKEN", "test-token")

	p := anthropicProvider{}
	m, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Name:      "anthropic-test",
		Provider:  anthropicProviderName,
		APIKeyEnv: "TEST_ANTHROPIC_TOKEN",
		Model:     "claude-test",
		BaseURL:   baseURL,
		MaxTokens: 128,
	})
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	model, ok := m.(*anthropicModel)
	if !ok {
		t.Fatalf("unexpected model type: %T", m)
	}
	return model
}
