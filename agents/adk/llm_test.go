package adk

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewLLMClient_UsesTimeoutHTTPClient_OpenAI(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://example.com/v1")

	client := NewLLMClient()
	if client == nil {
		t.Fatalf("NewLLMClient returned nil")
	}
	if client.httpClient == nil {
		t.Fatalf("httpClient should not be nil")
	}
	if got, want := client.httpClient.Timeout, defaultLLMRequestTimeout; got != want {
		t.Fatalf("timeout mismatch: got %v want %v", got, want)
	}
}

func TestNewLLMClient_UsesTimeoutHTTPClient_Anthropic(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "anthropic")
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_BASE_URL", "https://example.com")

	client := NewLLMClient()
	if client == nil {
		t.Fatalf("NewLLMClient returned nil")
	}
	if client.httpClient == nil {
		t.Fatalf("httpClient should not be nil")
	}
	if got, want := client.httpClient.Timeout, defaultLLMRequestTimeout; got != want {
		t.Fatalf("timeout mismatch: got %v want %v", got, want)
	}
}

func TestStreamOpenAI_UsesInjectedHTTPClient(t *testing.T) {
	called := false
	llm := &LLMClient{
		provider: "openai",
		apiKey:   "test-key",
		model:    "gpt-test",
		baseURL:  "https://example.com/v1",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				called = true
				if req.URL.Path != "/v1/chat/completions" {
					t.Fatalf("unexpected path: %s", req.URL.Path)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						"data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n" +
							"data: [DONE]\n\n",
					)),
					Header: make(http.Header),
				}, nil
			}),
		},
	}

	var got strings.Builder
	err := llm.StreamCompletion(
		context.Background(),
		"system",
		[]LLMMessage{{Role: "user", Content: "hi"}},
		func(text string) { got.WriteString(text) },
	)
	if err != nil {
		t.Fatalf("StreamCompletion returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected injected http client to be called")
	}
	if got.String() != "hello" {
		t.Fatalf("unexpected chunk content: %q", got.String())
	}
}

func TestStreamAnthropic_UsesInjectedHTTPClient(t *testing.T) {
	called := false
	llm := &LLMClient{
		provider: "anthropic",
		apiKey:   "test-key",
		model:    "claude-test",
		baseURL:  "https://example.com",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				called = true
				if req.URL.Path != "/v1/messages" {
					t.Fatalf("unexpected path: %s", req.URL.Path)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\n" +
							"data: [DONE]\n\n",
					)),
					Header: make(http.Header),
				}, nil
			}),
		},
	}

	var got strings.Builder
	err := llm.StreamCompletion(
		context.Background(),
		"system",
		[]LLMMessage{{Role: "user", Content: "hi"}},
		func(text string) { got.WriteString(text) },
	)
	if err != nil {
		t.Fatalf("StreamCompletion returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected injected http client to be called")
	}
	if got.String() != "hello" {
		t.Fatalf("unexpected chunk content: %q", got.String())
	}
}
