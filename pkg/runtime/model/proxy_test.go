package model

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

func TestProxyProvider_Name(t *testing.T) {
	p := proxyProvider{}
	if got := p.Name(); got != proxyProviderName {
		t.Fatalf("unexpected provider name: %s", got)
	}
}

func TestProxyProvider_CreateModel(t *testing.T) {
	t.Setenv("TEST_PROXY_TOKEN", "fake-token")
	p := proxyProvider{}

	model, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Name:      "proxy-main",
		Provider:  proxyProviderName,
		APIKeyEnv: "TEST_PROXY_TOKEN",
		Model:     "proxy-model",
		BaseURL:   "http://127.0.0.1:18081/proxy",
		MaxTokens: 128,
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

func TestProxyProvider_CreateModelMissingBaseURL(t *testing.T) {
	t.Setenv("TEST_PROXY_TOKEN", "fake-token")
	p := proxyProvider{}
	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		APIKeyEnv: "TEST_PROXY_TOKEN",
		Model:     "proxy-model",
	})
	if err == nil {
		t.Fatalf("expected missing base_url error")
	}
}

func TestProxyProvider_CreateModelMissingAPIKeyEnv(t *testing.T) {
	p := proxyProvider{}
	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Model:   "proxy-model",
		BaseURL: "http://127.0.0.1:18081/proxy",
	})
	if err == nil {
		t.Fatalf("expected missing api_key_env error")
	}
}

func TestProxyProvider_CreateModelMissingEnvValue(t *testing.T) {
	t.Setenv("TEST_PROXY_TOKEN", "")
	p := proxyProvider{}
	_, err := p.CreateModel(context.Background(), registry.ModelConfig{
		APIKeyEnv: "TEST_PROXY_TOKEN",
		Model:     "proxy-model",
		BaseURL:   "http://127.0.0.1:18081/proxy",
	})
	if err == nil {
		t.Fatalf("expected missing env value error")
	}
}

func TestProxyModel_GenerateUsesCustomBaseURL(t *testing.T) {
	var gotPath string
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer srv.Close()

	model := newProxyTestModel(t, srv.URL+"/custom")
	_, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if gotPath != "/custom/chat/completions" {
		t.Fatalf("unexpected endpoint path: %s", gotPath)
	}
	if gotAuth != "Bearer fake-token" {
		t.Fatalf("unexpected authorization: %q", gotAuth)
	}
}

func TestProxyModel_GenerateText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"proxy text"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":3,"total_tokens":7}}`)
	}))
	defer srv.Close()

	model := newProxyTestModel(t, srv.URL)
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.Parts) != 1 {
		t.Fatalf("unexpected part count: %d", len(resp.Parts))
	}
	part, ok := resp.Parts[0].(adk.TextPart)
	if !ok || part.Text != "proxy text" {
		t.Fatalf("unexpected text part: %#v", resp.Parts[0])
	}
}

func TestProxyModel_GenerateToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"proxy_lookup","arguments":"{\"k\":1}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":2,"total_tokens":4}}`)
	}))
	defer srv.Close()

	model := newProxyTestModel(t, srv.URL)
	resp, err := model.Generate(context.Background(), &adk.GenerateRequest{
		Contents: []*adk.Content{{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "go"}}}},
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
	if tc.ID != "call-1" || tc.Name != "proxy_lookup" {
		t.Fatalf("unexpected tool call: %+v", tc)
	}
}

func TestProxyModel_GenerateStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, `data: {"choices":[{"delta":{"content":"proxy stream"},"finish_reason":"stop"}]}`+"\n\n")
		_, _ = io.WriteString(w, `data: [DONE]`+"\n\n")
	}))
	defer srv.Close()

	model := newProxyTestModel(t, srv.URL)
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
	if count < 1 {
		t.Fatalf("expected at least 1 stream yield, got %d", count)
	}
}

func newProxyTestModel(t *testing.T, baseURL string) *openAICompatibleModel {
	t.Helper()
	t.Setenv("TEST_PROXY_TOKEN", "fake-token")

	p := proxyProvider{}
	m, err := p.CreateModel(context.Background(), registry.ModelConfig{
		Name:      "proxy-test",
		Provider:  proxyProviderName,
		APIKeyEnv: "TEST_PROXY_TOKEN",
		Model:     "proxy-model",
		BaseURL:   strings.TrimSpace(baseURL),
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
