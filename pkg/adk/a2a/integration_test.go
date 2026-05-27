package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func TestIntegration_ClientServer_TextRoundTrip(t *testing.T) {
	agent := &mockServerAgent{
		name: "roundtrip-text-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "round-trip text"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("roundtrip-text-agent"), agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()
	resp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: sessionID,
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("client-server round trip failed: %v", err)
	}
	if resp.Status != "completed" {
		t.Fatalf("unexpected status: got=%q want=%q", resp.Status, "completed")
	}
	if !hasPartTypeInDTO(resp.Events, "text") {
		t.Fatalf("expected text part in round-trip response, got=%+v", resp)
	}
}

func TestIntegration_RemoteAgentServer_GenerateRoundTrip(t *testing.T) {
	agent := &mockServerAgent{
		name: "roundtrip-remote-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "hello from child agent"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("roundtrip-remote-agent"), agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	remote := NewRemoteAgent("remote-proxy", httpServer.URL, WithSessionID(sessionID))
	resp, err := remote.Generate(context.Background(), buildTextRequest("generate through remote"))
	if err != nil {
		t.Fatalf("remote-agent generate round trip failed: %v", err)
	}
	textPart, ok := firstTextPart(resp.Parts)
	if !ok {
		t.Fatalf("expected text part in generate response, got=%+v", resp.Parts)
	}
	if textPart.Text != "hello from child agent" {
		t.Fatalf("unexpected generated text: got=%q", textPart.Text)
	}
}

func TestIntegration_ClientServer_ToolCallRoundTrip(t *testing.T) {
	agent := &sequenceServerAgent{
		name: "tool-roundtrip-agent",
		responses: []*adk.GenerateResponse{
			{
				Parts: []adk.Part{
					adk.ToolCallPart{
						ID:        "call-1",
						Name:      "echo",
						Arguments: json.RawMessage(`{"input":"hello"}`),
					},
				},
				FinishReason: adk.FinishToolUse,
			},
			{
				Parts:        []adk.Part{adk.TextPart{Text: "tool done"}},
				FinishReason: adk.FinishStop,
			},
		},
	}
	tool := mockServerTool{name: "echo", content: "echo-result"}
	server, sessionID := newTestServer(t, defaultConfig("tool-roundtrip-agent"), agent, tool)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()
	resp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: sessionID,
		Message: Message{
			Role:    "user",
			Content: "run tool",
		},
	})
	if err != nil {
		t.Fatalf("tool round-trip failed: %v", err)
	}
	if !hasPartTypeInDTO(resp.Events, "tool_call") {
		t.Fatalf("expected tool_call part, got=%+v", resp.Events)
	}
	if !hasPartTypeInDTO(resp.Events, "tool_result") {
		t.Fatalf("expected tool_result part, got=%+v", resp.Events)
	}
	if !hasPartTypeInDTO(resp.Events, "text") {
		t.Fatalf("expected final text part, got=%+v", resp.Events)
	}
}

func TestIntegration_AgentCardStillAvailable(t *testing.T) {
	cfg := defaultConfig("agentcard-roundtrip-agent")
	agent := &mockServerAgent{
		name: cfg.Name,
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "normal response"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, cfg, agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()
	_, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: sessionID,
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("run endpoint should work before card check: %v", err)
	}

	resp, err := http.Get(httpServer.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("fetch agent card failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected card status: got=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("decode agent card failed: %v", err)
	}
	if card.Name != cfg.Name {
		t.Fatalf("unexpected card name: got=%q want=%q", card.Name, cfg.Name)
	}
}

func TestIntegration_ErrorRoundTrip(t *testing.T) {
	agent := &mockServerAgent{
		name: "roundtrip-error-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, _ := newTestServer(t, defaultConfig("roundtrip-error-agent"), agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()
	_, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "missing-session-id",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err == nil {
		t.Fatal("expected error when session does not exist")
	}

	_, err = client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "session-a",
		Message: Message{
			Role:    "user",
			Content: "",
		},
	})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestIntegration_ThinkingPartRedacted(t *testing.T) {
	const secretThinking = "OPENAI_API_KEY=should-not-leak"
	agent := &mockServerAgent{
		name: "thinking-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts: []adk.Part{
					adk.ThinkingPart{Thinking: secretThinking},
					adk.TextPart{Text: "safe text"},
				},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("thinking-agent"), agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()
	clientResp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: sessionID,
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("client round-trip failed: %v", err)
	}
	if strings.Contains(string(mustJSONMarshal(t, clientResp)), secretThinking) {
		t.Fatalf("client response leaked thinking content: %s", string(mustJSONMarshal(t, clientResp)))
	}
	for _, event := range clientResp.Events {
		for _, part := range event.Parts {
			if part.Type != "thinking" {
				continue
			}
			if part.Text != "" || part.Content != "" {
				t.Fatalf("thinking fields should be empty: %+v", part)
			}
		}
	}

	remote := NewRemoteAgent("thinking-proxy", httpServer.URL, WithSessionID(sessionID))
	remoteResp, err := remote.Generate(context.Background(), buildTextRequest("hello"))
	if err != nil {
		t.Fatalf("remote agent round-trip failed: %v", err)
	}
	for _, part := range remoteResp.Parts {
		if _, ok := part.(adk.ThinkingPart); ok {
			t.Fatalf("thinking part should not be exposed in remote response: %+v", part)
		}
	}
	if strings.Contains(string(mustJSONMarshal(t, remoteResp)), secretThinking) {
		t.Fatalf("remote response leaked thinking content: %s", string(mustJSONMarshal(t, remoteResp)))
	}
}
