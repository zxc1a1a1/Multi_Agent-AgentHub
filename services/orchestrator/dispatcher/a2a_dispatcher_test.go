package dispatcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDispatchMissingURL(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestDispatchMissingConversationID(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for missing conversationID")
	}
}

func TestDispatchMissingMessage(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "",
	})
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestDispatchNilDispatcher(t *testing.T) {
	var d *A2ADispatcher
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for nil dispatcher")
	}
}

func TestDispatcherBadRequest(t *testing.T) {
	d := NewA2ADispatcher()
	// Use a non-routable IP to simulate connection failure.
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://10.255.255.1:1",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for bad agent URL")
	}
}

func TestNewA2ADispatcher(t *testing.T) {
	d := NewA2ADispatcher()
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	if d.client == nil {
		t.Error("expected non-nil client in dispatcher")
	}
}

// mockAgentServer returns an httptest server that mimics a minimal A2A agent.
// It handles JSON-RPC tasks/sendSubscribe requests at "/" and returns mock
// text events.
func mockAgentServer(t *testing.T, agentName, responseText string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			JSONRPC string          `json:"jsonrpc"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "bad_request", "message": err.Error()},
			})
			return
		}

		if body.Method != "tasks/sendSubscribe" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "bad_request", "message": "unsupported method"},
			})
			return
		}

		// Return mock text events.
		resp := map[string]any{
			"taskId": "task-mock-001",
			"status": "completed",
			"events": []map[string]any{
				{
					"author": agentName,
					"role":   "assistant",
					"parts": []map[string]any{
						{"type": "text", "text": responseText},
					},
					"final":   true,
					"partial": false,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestDispatchMockCodeAgent(t *testing.T) {
	srv := mockAgentServer(t, "code-agent", "code-agent mock: Go HTTP server code")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "code-agent",
		ConversationID: "conv-mock-code",
		RunID:          "run-mock-code",
		Message:        "用 Go 写一个 HTTP API 接口",
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil dispatch result")
	}
	if !strings.Contains(result.Text, "code-agent mock") {
		t.Fatalf("expected code-agent response text, got: %q", result.Text)
	}
}

func TestDispatchMockWebAgent(t *testing.T) {
	srv := mockAgentServer(t, "web-agent", "web-agent mock: HTML login page")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "web-agent",
		ConversationID: "conv-mock-web",
		RunID:          "run-mock-web",
		Message:        "写一个 HTML 登录页面",
	})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil dispatch result")
	}
	if !strings.Contains(result.Text, "web-agent mock") {
		t.Fatalf("expected web-agent response text, got: %q", result.Text)
	}
}

func TestDispatchMockAgentJSONRPCFormat(t *testing.T) {
	// Verify the dispatcher sends valid JSON-RPC that a real A2A server can decode.
	srv := mockAgentServer(t, "test-agent", "JSON-RPC roundtrip OK")
	defer srv.Close()

	d := NewA2ADispatcher()
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "test-agent",
		ConversationID: "conv-jsonrpc",
		RunID:          "run-jsonrpc",
		Message:        "test message",
	})
	if err != nil {
		t.Fatalf("JSON-RPC dispatch failed: %v", err)
	}
	if result == nil || result.Text != "JSON-RPC roundtrip OK" {
		t.Fatalf("unexpected dispatch result: %+v", result)
	}
}

func TestDispatchMockAgentSessionAutoCreate(t *testing.T) {
	// Verify that the dispatcher works with a new session ID (no pre-create needed).
	srv := mockAgentServer(t, "test-agent", "auto-created session works")
	defer srv.Close()

	d := NewA2ADispatcher()
	// Use a brand-new, never-seen session ID.
	result, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "test-agent",
		ConversationID: "brand-new-session-12345",
		RunID:          "run-session-test",
		Message:        "hello from new session",
	})
	if err != nil {
		t.Fatalf("dispatch with new session failed: %v", err)
	}
	if result == nil || result.Text == "" {
		t.Fatal("expected non-empty dispatch result for new session")
	}
}
