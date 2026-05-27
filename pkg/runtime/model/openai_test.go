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

func TestOpenAIProvider_Name(t *testing.T) {
	p := openAIProvider{}
	if got := p.Name(); got != openAIProviderName {
		t.Fatalf("unexpected provider name: %s", got)
	}
}

func TestOpenAIProvider_CreateModel(t *testing.T) {
	t.Setenv("TEST_OPENAI_TOKEN", "test-token")
	p := openAIProvider{}

	model, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Name:      "openai-main",
		Provider:  openAIProviderName,
		APIKeyEnv: "TEST_OPENAI_TOKEN",
		Model:     "gpt-test",
		BaseURL:   "http://127.0.0.1:18080/v1/",
		MaxTokens: 256,
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

func TestOpenAIProvider_CreateModelMissingAPIKeyEnv(t *testing.T) {
	p := openAIProvider{}
	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Model: "gpt-test",
	})
	if err == nil {
		t.Fatalf("expected api_key_env validation error")
	}
}

func TestOpenAIProvider_CreateModelMissingEnvValue(t *testing.T) {
	t.Setenv("TEST_OPENAI_TOKEN", "")
	p := openAIProvider{}
	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		APIKeyEnv: "TEST_OPENAI_TOKEN",
		Model:     "gpt-test",
	})
	if err == nil {
		t.Fatalf("expected missing env value error")
	}
}

func TestOpenAIModel_GenerateText(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotContentType string
	var gotReq openAIChatRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"hello from openai"},"finish_reason":"stop"}],"usage":{"prompt_tokens":6,"completion_tokens":2,"total_tokens":8}}`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL+"/v1")
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleSystem, Parts: []adk.Part{adk.TextPart{Text: "system"}}},
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "question"}}},
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("unexpected endpoint: %s", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected authorization: %q", gotAuth)
	}
	if !strings.Contains(strings.ToLower(gotContentType), "application/json") {
		t.Fatalf("unexpected content-type: %q", gotContentType)
	}
	if gotReq.Model != "gpt-test" {
		t.Fatalf("unexpected request model: %s", gotReq.Model)
	}
	if len(gotReq.Messages) != 2 {
		t.Fatalf("unexpected message count: %d", len(gotReq.Messages))
	}
	if gotReq.Messages[0].Role != "system" || gotReq.Messages[1].Role != "user" {
		t.Fatalf("unexpected roles: %+v", gotReq.Messages)
	}
	if resp.FinishReason != adk.FinishStop {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if len(resp.Parts) != 1 {
		t.Fatalf("unexpected part count: %d", len(resp.Parts))
	}
	text, ok := resp.Parts[0].(adk.TextPart)
	if !ok || text.Text != "hello from openai" {
		t.Fatalf("unexpected text part: %#v", resp.Parts[0])
	}
}

func TestOpenAIModel_GenerateToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"lookup","arguments":"{\"q\":\"go\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
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
	tc, ok := resp.Parts[0].(adk.ToolCallPart)
	if !ok {
		t.Fatalf("expected ToolCallPart, got %T", resp.Parts[0])
	}
	if tc.ID != "call-1" || tc.Name != "lookup" {
		t.Fatalf("unexpected tool call: %+v", tc)
	}
	if strings.TrimSpace(string(tc.Arguments)) != `{"q":"go"}` {
		t.Fatalf("unexpected tool args: %s", string(tc.Arguments))
	}
}

func TestOpenAIModel_GenerateMaxTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"truncated"},"finish_reason":"length"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "long"}}}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if resp.FinishReason != adk.FinishMaxToken {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
}

func TestOpenAIModel_GenerateUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":9,"completion_tokens":4,"total_tokens":13}}`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
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

func TestOpenAIModel_RequestIncludesConfig(t *testing.T) {
	var gotReq openAIChatRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
	temperature := 0.4
	topP := 0.8
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{
			{Role: adk.RoleSystem, Parts: []adk.Part{adk.TextPart{Text: "sys"}}},
			{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "u"}}},
			{Role: adk.RoleAssistant, Parts: []adk.Part{
				adk.TextPart{Text: "a"},
				adk.ToolCallPart{ID: "call-1", Name: "lookup", Arguments: json.RawMessage(`{"q":"gpt"}`)},
			}},
			{Role: adk.RoleTool, Parts: []adk.Part{
				adk.ToolResultPart{CallID: "call-1", Name: "lookup", Content: "result"},
			}},
		},
		Config: &adk.GenerateConfig{
			Temperature:   &temperature,
			MaxTokens:     77,
			TopP:          &topP,
			StopSequences: []string{"END", "STOP"},
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response")
	}
	if gotReq.MaxTokens != 77 {
		t.Fatalf("unexpected max_tokens: %d", gotReq.MaxTokens)
	}
	if gotReq.Temperature == nil || *gotReq.Temperature != 0.4 {
		t.Fatalf("unexpected temperature: %#v", gotReq.Temperature)
	}
	if gotReq.TopP == nil || *gotReq.TopP != 0.8 {
		t.Fatalf("unexpected top_p: %#v", gotReq.TopP)
	}
	if len(gotReq.Stop) != 2 || gotReq.Stop[0] != "END" || gotReq.Stop[1] != "STOP" {
		t.Fatalf("unexpected stop sequences: %#v", gotReq.Stop)
	}
	if len(gotReq.Messages) != 4 {
		t.Fatalf("unexpected message count: %d", len(gotReq.Messages))
	}
	if gotReq.Messages[2].Role != "assistant" || len(gotReq.Messages[2].ToolCalls) != 1 {
		t.Fatalf("assistant tool calls not captured: %+v", gotReq.Messages[2])
	}
	if gotReq.Messages[3].Role != "tool" || gotReq.Messages[3].ToolCallID != "call-1" {
		t.Fatalf("tool message not captured: %+v", gotReq.Messages[3])
	}
}

func TestOpenAIModel_HTTPErrorSanitized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `invalid token=fake-token`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
	_, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
	})
	if err == nil {
		t.Fatalf("expected http error")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected status in error, got %v", err)
	}
	if strings.Contains(err.Error(), "fake-token") {
		t.Fatalf("error leaked body: %v", err)
	}
}

func TestOpenAIModel_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
	_, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
	})
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestOpenAIModel_GenerateStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"stream"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer srv.Close()

	model := newOpenAITestModel(t, srv.URL)
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

func newOpenAITestModel(t *testing.T, baseURL string) *openAICompatibleModel {
	t.Helper()
	t.Setenv("TEST_OPENAI_TOKEN", "test-token")

	p := openAIProvider{}
	m, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Name:      "openai-test",
		Provider:  openAIProviderName,
		APIKeyEnv: "TEST_OPENAI_TOKEN",
		Model:     "gpt-test",
		BaseURL:   baseURL,
		MaxTokens: 128,
	})
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	model, ok := m.(*openAICompatibleModel)
	if !ok {
		t.Fatalf("unexpected model type: %T", m)
	}
	return model
}
