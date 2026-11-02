package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type handoffTestServer struct {
	mu       sync.Mutex
	requests []handoffRequest
	status   int
	response string
}

func (s *handoffTestServer) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req handoffRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		s.mu.Lock()
		s.requests = append(s.requests, req)
		status, response := s.status, s.response
		s.mu.Unlock()
		if status == 0 {
			status = http.StatusOK
		}
		if response == "" {
			response = `{"status":"completed","taskId":"remote-1","text":"review done"}`
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
}

func handoffIdentity(depth int) context.Context {
	return adk.WithRunIdentity(context.Background(), adk.RunIdentity{
		RunID: "run-1", TaskID: "task-1", PrincipalID: "user-1",
		FencingVersion: 3, TraceID: "trace-1", DelegateDepth: depth,
	})
}

func handoffArgsJSON(target, message, reason string) json.RawMessage {
	raw, _ := json.Marshal(map[string]string{"targetAgent": target, "message": message, "reason": reason})
	return raw
}

func TestHandoffToolRequestsAllowlistedTarget(t *testing.T) {
	remote := &handoffTestServer{}
	server := remote.server()
	defer server.Close()
	tool, err := NewHandoffTool(HandoffConfig{Endpoint: server.URL, AllowedAgents: []string{"review-agent"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Execute(handoffIdentity(0), handoffArgsJSON("REVIEW-AGENT", "check this patch", "needs review"))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.IsError || result.Content != "[handoff:review-agent] review done" {
		t.Fatalf("unexpected result: %#v", result)
	}
	remote.mu.Lock()
	defer remote.mu.Unlock()
	if len(remote.requests) != 1 {
		t.Fatalf("requests=%d, want 1", len(remote.requests))
	}
	req := remote.requests[0]
	if req.TargetAgent != "review-agent" || req.Message != "check this patch" || req.Reason != "needs review" {
		t.Fatalf("unexpected request: %#v", req)
	}
	if req.RunID != "run-1" || req.TaskID != "task-1" || req.FencingVersion != 3 || req.DelegateDepth != 0 {
		t.Fatalf("identity not forwarded: %#v", req)
	}
}

func TestHandoffToolRejectsOutsideAllowlistWithoutCall(t *testing.T) {
	remote := &handoffTestServer{}
	server := remote.server()
	defer server.Close()
	tool, _ := NewHandoffTool(HandoffConfig{Endpoint: server.URL, AllowedAgents: []string{"review-agent"}})
	result, err := tool.Execute(handoffIdentity(0), handoffArgsJSON("web-agent", "search", "needs web"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(result.Content, "allowlist") {
		t.Fatalf("unexpected result: %#v", result)
	}
	remote.mu.Lock()
	defer remote.mu.Unlock()
	if len(remote.requests) != 0 {
		t.Fatalf("non-allowlisted target must not reach the platform: %d", len(remote.requests))
	}
}

func TestHandoffToolRejectsDepthAndContextOverflow(t *testing.T) {
	remote := &handoffTestServer{}
	server := remote.server()
	defer server.Close()
	tool, _ := NewHandoffTool(HandoffConfig{Endpoint: server.URL, AllowedAgents: []string{"review-agent"}, MaxMessageBytes: 8})

	result, _ := tool.Execute(handoffIdentity(1), handoffArgsJSON("review-agent", "x", "y"))
	if !result.IsError || !strings.Contains(result.Content, "depth") {
		t.Fatalf("depth limit not enforced: %#v", result)
	}
	result, _ = tool.Execute(handoffIdentity(0), handoffArgsJSON("review-agent", "123456789", "y"))
	if !result.IsError || !strings.Contains(result.Content, "exceeds") {
		t.Fatalf("message limit not enforced: %#v", result)
	}
	result, _ = tool.Execute(handoffIdentity(0), handoffArgsJSON("review-agent", "x", strings.Repeat("r", 513)))
	if !result.IsError || !strings.Contains(result.Content, "reason") {
		t.Fatalf("reason limit not enforced: %#v", result)
	}
	remote.mu.Lock()
	defer remote.mu.Unlock()
	if len(remote.requests) != 0 {
		t.Fatalf("rejected calls must not reach the platform: %d", len(remote.requests))
	}
}

func TestHandoffToolRequiresIdentity(t *testing.T) {
	remote := &handoffTestServer{}
	server := remote.server()
	defer server.Close()
	tool, _ := NewHandoffTool(HandoffConfig{Endpoint: server.URL, AllowedAgents: []string{"review-agent"}})
	result, err := tool.Execute(context.Background(), handoffArgsJSON("review-agent", "x", "y"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(result.Content, "identity") {
		t.Fatalf("identity requirement not enforced: %#v", result)
	}
}

func TestHandoffToolMapsDenialAndTruncatesOutput(t *testing.T) {
	remote := &handoffTestServer{status: http.StatusForbidden, response: `{"error":"denied"}`}
	server := remote.server()
	defer server.Close()
	tool, _ := NewHandoffTool(HandoffConfig{Endpoint: server.URL, AllowedAgents: []string{"review-agent"}})
	result, _ := tool.Execute(handoffIdentity(0), handoffArgsJSON("review-agent", "x", "y"))
	if !result.IsError || !strings.Contains(result.Content, "denied") {
		t.Fatalf("denial not mapped: %#v", result)
	}
	server.Close()
	remote = &handoffTestServer{response: `{"status":"completed","taskId":"t","text":"` + strings.Repeat("é", 1000) + `"}`}
	server = remote.server()
	defer server.Close()
	tool, _ = NewHandoffTool(HandoffConfig{Endpoint: server.URL, AllowedAgents: []string{"review-agent"}, MaxOutputBytes: 101})
	result, _ = tool.Execute(handoffIdentity(0), handoffArgsJSON("review-agent", "x", "y"))
	if result.IsError {
		t.Fatalf("unexpected error: %#v", result)
	}
	text := strings.TrimPrefix(result.Content, "[handoff:review-agent] ")
	if len(text) > 101 || !utf8.ValidString(text) {
		t.Fatalf("output not safely truncated: bytes=%d valid=%v", len(text), utf8.ValidString(text))
	}
}

func TestHandoffDescriptorIsValid(t *testing.T) {
	tool, _ := NewHandoffTool(HandoffConfig{Endpoint: "http://127.0.0.1:1/handoff", AllowedAgents: []string{"review-agent"}})
	if err := tool.Descriptor().Validate(); err != nil {
		t.Fatalf("descriptor invalid: %v", err)
	}
	if tool.SideEffect() {
		t.Fatal("handoff is a routing request; target tools keep their own approvals")
	}
}
