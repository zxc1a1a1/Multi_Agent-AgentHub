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

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
	if client.httpClient != http.DefaultClient {
		t.Fatalf("unexpected default http client: got=%p want=%p", client.httpClient, http.DefaultClient)
	}
}

func TestClient_WithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	client := NewClient(WithHTTPClient(custom))
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.httpClient != custom {
		t.Fatal("expected custom http client injection to work")
	}
}

func TestClient_Send_DirectRequest(t *testing.T) {
	agent := &mockServerAgent{
		name: "direct-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "hello from direct"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("direct-agent"), agent)
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
		t.Fatalf("send direct request failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.TaskID == "" {
		t.Fatal("expected task id in response")
	}
	if resp.Status != "completed" {
		t.Fatalf("unexpected status: got=%q want=%q", resp.Status, "completed")
	}
	if len(resp.Events) == 0 {
		t.Fatal("expected events in response")
	}
	if !hasPartTypeInDTO(resp.Events, "text") {
		t.Fatalf("expected text part, response=%+v", resp)
	}
}

func TestClient_Send_JSONRPCRequest(t *testing.T) {
	agent := &mockServerAgent{
		name: "jsonrpc-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "hello from jsonrpc"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("jsonrpc-agent"), agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()
	resp, err := client.SendJSONRPC(context.Background(), httpServer.URL+"/a2a/tasks/sendSubscribe", RunRequest{
		SessionID: sessionID,
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("send json-rpc request failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Events) == 0 {
		t.Fatal("expected events in response")
	}
	if !hasPartTypeInDTO(resp.Events, "text") {
		t.Fatalf("expected text part, response=%+v", resp)
	}
}

func TestClient_ParseTextPart(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-text",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "assistant",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{Type: "text", Text: "plain text"},
					},
				},
			},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	resp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "session-1",
		Message: Message{
			Role:    "user",
			Content: "hi",
		},
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	part, ok := findPartByType(resp.Events, "text")
	if !ok {
		t.Fatalf("expected text part, response=%+v", resp)
	}
	if part.Text != "plain text" {
		t.Fatalf("unexpected text: got=%q want=%q", part.Text, "plain text")
	}
}

func TestClient_ParseToolCallAndToolResultParts(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-tool",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "assistant",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{
							Type:      "tool_call",
							ID:        "call-1",
							Name:      "echo",
							Arguments: map[string]any{"input": "hello"},
						},
						{
							Type:    "tool_result",
							CallID:  "call-1",
							Name:    "echo",
							Content: "echo-result",
							IsError: false,
						},
					},
				},
			},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	resp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "session-tool",
		Message: Message{
			Role:    "user",
			Content: "use tool",
		},
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if !hasPartTypeInDTO(resp.Events, "tool_call") {
		t.Fatalf("expected tool_call part, response=%+v", resp)
	}
	if !hasPartTypeInDTO(resp.Events, "tool_result") {
		t.Fatalf("expected tool_result part, response=%+v", resp)
	}
}

func TestClient_DoesNotExposeThinkingContent(t *testing.T) {
	const secretThinking = "PRIVATE KEY should never leave thinking"
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-thinking",
			Status: "completed",
			Events: []EventDTO{
				{
					Author: "assistant",
					Role:   "assistant",
					Final:  true,
					Parts: []PartDTO{
						{Type: "thinking", Text: secretThinking},
						{Type: "text", Text: "safe output"},
					},
				},
			},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	resp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "session-thinking",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	for _, event := range resp.Events {
		for _, part := range event.Parts {
			if part.Type != "thinking" {
				continue
			}
			if part.Text != "" {
				t.Fatalf("thinking text should be redacted, got=%q", part.Text)
			}
			if part.Content != "" {
				t.Fatalf("thinking content should be empty, got=%q", part.Content)
			}
		}
	}
	if strings.Contains(toJSONString(t, resp), secretThinking) {
		t.Fatalf("thinking content leaked in response: %s", toJSONString(t, resp))
	}
}

func TestClient_ServerBadRequest(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, RunResponse{
			Error: &ResponseError{
				Code:    "bad_request",
				Message: "message.content is required",
			},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	_, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "session-err",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err == nil {
		t.Fatal("expected error for bad request")
	}
}

func TestClient_ErrorResponseSanitized(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, RunResponse{
			Error: &ResponseError{
				Code:    "internal_error",
				Message: "panic at C:\\secret\\stack.go with sk-demo-token",
			},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	_, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: "session-err",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	errText := err.Error()
	if strings.Contains(errText, "C:\\secret\\stack.go") || strings.Contains(errText, "sk-") {
		t.Fatalf("error should be sanitized, got=%q", errText)
	}
}

func TestClient_EmptyURL(t *testing.T) {
	client := NewClient()
	_, err := client.Send(context.Background(), "", RunRequest{
		SessionID: "session-1",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestClient_EmptySessionID(t *testing.T) {
	client := NewClient()
	_, err := client.Send(context.Background(), "http://localhost:8080", RunRequest{
		SessionID: "",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err == nil {
		t.Fatal("expected error for empty sessionId")
	}
}

func TestClient_EmptyContent(t *testing.T) {
	client := NewClient()
	_, err := client.Send(context.Background(), "http://localhost:8080", RunRequest{
		SessionID: "session-1",
		Message: Message{
			Role:    "user",
			Content: "   ",
		},
	})
	if err == nil {
		t.Fatal("expected error for empty message content")
	}
}

func TestClient_ContextCanceled(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, RunResponse{
			TaskID: "task-canceled",
			Status: "completed",
		})
	}))
	defer httpServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient()
	_, err := client.Send(ctx, httpServer.URL, RunRequest{
		SessionID: "session-1",
		Message: Message{
			Role:    "user",
			Content: "hello",
		},
	})
	if err == nil {
		t.Fatal("expected context canceled error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "context") && !strings.Contains(strings.ToLower(err.Error()), "canceled") {
		t.Fatalf("unexpected canceled error: %v", err)
	}
}

func TestClient_GetTask_Success(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a2a/tasks/get" {
			t.Fatalf("expected /a2a/tasks/get, got %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, Task{
			TaskID:    "task-get-1",
			Status:    TaskStatusRunning,
			SessionID: "session-1",
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	task, err := client.GetTask(context.Background(), httpServer.URL, "task-get-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.TaskID != "task-get-1" {
		t.Errorf("expected TaskID=task-get-1, got %q", task.TaskID)
	}
	if task.Status != TaskStatusRunning {
		t.Errorf("expected Status=running, got %q", task.Status)
	}
}

func TestClient_GetTask_NotFound(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, runResponse{
			Error: &responseError{Code: "not_found", Message: "task not found"},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	_, err := client.GetTask(context.Background(), httpServer.URL, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestClient_CancelTask_Running(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a2a/tasks/cancel" {
			t.Fatalf("expected /a2a/tasks/cancel, got %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, Task{
			TaskID: "task-cancel-1",
			Status: TaskStatusCancelled,
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	task, err := client.CancelTask(context.Background(), httpServer.URL, "task-cancel-1")
	if err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}
	if task.Status != TaskStatusCancelled {
		t.Errorf("expected Status=cancelled, got %q", task.Status)
	}
}

func TestClient_CancelTask_CompletedIdempotent(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, Task{
			TaskID: "task-completed-1",
			Status: TaskStatusCompleted,
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	task, err := client.CancelTask(context.Background(), httpServer.URL, "task-completed-1")
	if err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected Status=completed (unchanged), got %q", task.Status)
	}
}

func TestClient_CancelTask_NotFound(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, runResponse{
			Error: &responseError{Code: "not_found", Message: "task not found"},
		})
	}))
	defer httpServer.Close()

	client := NewClient()
	_, err := client.CancelTask(context.Background(), httpServer.URL, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestClient_TaskEndpoints_AcceptExplicitEndpointURL(t *testing.T) {
	var paths []string
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/a2a/tasks/get":
			writeJSON(w, http.StatusOK, Task{TaskID: "task-1", Status: TaskStatusRunning})
		case "/a2a/tasks/cancel":
			writeJSON(w, http.StatusOK, Task{TaskID: "task-1", Status: TaskStatusCancelled})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer httpServer.Close()

	client := NewClient()
	if _, err := client.GetTask(context.Background(), httpServer.URL+"/a2a/tasks/get", "task-1"); err != nil {
		t.Fatalf("GetTask explicit endpoint failed: %v", err)
	}
	if _, err := client.CancelTask(context.Background(), httpServer.URL+"/a2a/tasks/cancel", "task-1"); err != nil {
		t.Fatalf("CancelTask explicit endpoint failed: %v", err)
	}
	if got := strings.Join(paths, ","); got != "/a2a/tasks/get,/a2a/tasks/cancel" {
		t.Fatalf("unexpected paths: %s", got)
	}
}

func TestClient_GetTask_EmptyTaskID(t *testing.T) {
	client := NewClient()
	_, err := client.GetTask(context.Background(), "http://localhost:8080", "  ")
	if err == nil {
		t.Fatal("expected error for empty taskId")
	}
}

func TestClient_CancelTask_EmptyTaskID(t *testing.T) {
	client := NewClient()
	_, err := client.CancelTask(context.Background(), "http://localhost:8080", "  ")
	if err == nil {
		t.Fatal("expected error for empty taskId")
	}
}

func TestClient_ExistingSignaturesStillWork(t *testing.T) {
	// Verify Send, SendJSONRPC, SendJSONRPCStream signatures compile unchanged.
	agent := &mockServerAgent{
		name: "sig-agent",
		generate: func(ctx context.Context, req *adk.GenerateRequest) (*adk.GenerateResponse, error) {
			return &adk.GenerateResponse{
				Parts:        []adk.Part{adk.TextPart{Text: "ok"}},
				FinishReason: adk.FinishStop,
			}, nil
		},
	}
	server, sessionID := newTestServer(t, defaultConfig("sig-agent"), agent)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient()

	// Send (direct)
	resp, err := client.Send(context.Background(), httpServer.URL, RunRequest{
		SessionID: sessionID,
		Message:   Message{Role: "user", Content: "hello"},
	})
	if err != nil || resp == nil {
		t.Fatalf("Send signature test failed: err=%v", err)
	}

	// SendJSONRPC
	resp2, err := client.SendJSONRPC(context.Background(), httpServer.URL+"/a2a/tasks/sendSubscribe", RunRequest{
		SessionID: sessionID,
		Message:   Message{Role: "user", Content: "hello"},
	})
	if err != nil || resp2 == nil {
		t.Fatalf("SendJSONRPC signature test failed: err=%v", err)
	}

	// SendJSONRPCStream
	streamCalled := false
	for chunk := range client.SendJSONRPCStream(context.Background(), httpServer.URL+"/a2a/tasks/sendSubscribe", RunRequest{
		SessionID: sessionID,
		Message:   Message{Role: "user", Content: "hello"},
	}) {
		if chunk.Err != nil {
			t.Fatalf("SendJSONRPCStream signature test failed: err=%v", chunk.Err)
		}
		streamCalled = true
	}
	if !streamCalled {
		t.Fatal("expected at least one stream chunk")
	}
}

func hasPartTypeInDTO(events []EventDTO, partType string) bool {
	_, ok := findPartByType(events, partType)
	return ok
}

func findPartByType(events []EventDTO, partType string) (PartDTO, bool) {
	for _, event := range events {
		for _, part := range event.Parts {
			if part.Type == partType {
				return part, true
			}
		}
	}
	return PartDTO{}, false
}

func toJSONString(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json failed: %v", err)
	}
	return string(raw)
}
