package httpapi

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

func newTestServer() *Server {
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:19999", Description: "code"},
		{Name: "web-agent", URL: "http://127.0.0.1:19998", Description: "web"},
	})
	if err != nil {
		panic(err)
	}
	return NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
	)
}

func TestRunStreamMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/runs/stream")
	if err != nil {
		t.Fatalf("GET run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestRunStreamUnauthorized(t *testing.T) {
	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:19999"},
	})
	srv := NewServer(
		WithInternalToken("secret-token"),
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_001","messages":[{"role":"user","text":"hello"}]}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/orchestrator/runs/stream", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	body2 := strings.NewReader(`{"runId":"run_001","messages":[{"role":"user","text":"hello"}]}`)
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/orchestrator/runs/stream", body2)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer secret-token")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("POST run/stream with token failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", resp2.StatusCode)
	}
}

func TestRunStreamBadRequest(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`not-json`)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRunStreamUnknownAgentFallsBackToDefault(t *testing.T) {
	// When an unknown agentName is sent, the RulePlanner ignores it and falls
	// back to the default (code-agent). Since no real agent is listening the
	// dispatch fails with ORCHESTRATOR_AGENT_FAILED.
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_001","agentName":"unknown-agent","messages":[{"role":"user","text":"hello"}]}`)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var events []OrchestratorStreamEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event OrchestratorStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			events = append(events, event)
		}
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Type != "run_started" {
		t.Errorf("expected run_started, got %q", events[0].Type)
	}

	// run_started must carry the planId.
	if events[0].State == nil || events[0].State["planId"] == nil || events[0].State["planId"] == "" {
		t.Error("expected planId in run_started state")
	}

	// Should fail because no real agent at the registry URL.
	hasError := false
	for _, e := range events {
		if e.Type == "run_error" {
			hasError = true
			if e.Error == nil || e.Error.Code != "ORCHESTRATOR_AGENT_FAILED" {
				t.Errorf("expected ORCHESTRATOR_AGENT_FAILED, got %+v", e.Error)
			}
		}
	}
	if !hasError {
		t.Error("expected run_error event because dispatch fails")
	}
}

func TestRunStreamDefaultAgentMissingMessage(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_001"}`)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var events []OrchestratorStreamEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event OrchestratorStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			events = append(events, event)
		}
	}

	hasError := false
	for _, e := range events {
		if e.Type == "run_error" {
			hasError = true
			if e.Error == nil || e.Error.Code != "ORCHESTRATOR_BAD_REQUEST" {
				t.Errorf("expected ORCHESTRATOR_BAD_REQUEST error, got %+v", e.Error)
			}
		}
	}
	if !hasError {
		t.Error("expected run_error event for missing message")
	}
}

func TestRunStreamWithMockAgent(t *testing.T) {
	// Create a mock A2A agent that returns a known response.
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"taskId": "task-1",
			"status": "completed",
			"events": []map[string]any{
				{
					"author":  "code-agent",
					"role":    "assistant",
					"final":   true,
					"partial": false,
					"parts": []map[string]any{
						{"type": "text", "text": "Hello from code-agent!"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockAgent.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: mockAgent.URL, Description: "code"},
	})
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_test_001","agentName":"code-agent","conversationId":"conv_001","messages":[{"role":"user","text":"hello"}]}`)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %q", ct)
	}

	scanner := bufio.NewScanner(resp.Body)
	var events []OrchestratorStreamEvent
	var currentType string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentType = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event OrchestratorStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			if currentType != "" && event.Type != currentType {
				t.Errorf("event type mismatch: data has %q, event line has %q", event.Type, currentType)
			}
			events = append(events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if len(events) < 5 {
		t.Fatalf("expected at least 5 events, got %d", len(events))
	}

	expectedTypes := []string{"run_started", "message_start", "message_delta", "message_end", "run_finished"}
	for i, expected := range expectedTypes {
		if events[i].Type != expected {
			t.Errorf("event[%d]: expected type %q, got %q", i, expected, events[i].Type)
		}
	}

	if events[0].RunID != "run_test_001" {
		t.Errorf("expected run_test_001, got %q", events[0].RunID)
	}
	if events[0].State == nil {
		t.Error("expected state in run_started")
	} else {
		if events[0].State["phase"] != "dispatching" {
			t.Errorf("expected phase=dispatching, got %v", events[0].State["phase"])
		}
		if events[0].State["planId"] == nil || events[0].State["planId"] == "" {
			t.Error("expected planId in run_started state")
		}
	}

	if events[2].Delta == "" {
		t.Error("expected non-empty delta in message_delta")
	}
	if events[2].Delta != "Hello from code-agent!" {
		t.Errorf("expected 'Hello from code-agent!', got %q", events[2].Delta)
	}

	if events[2].Sender == nil {
		t.Error("expected sender in message events")
	} else if events[2].Sender.Name != "code-agent" {
		t.Errorf("expected sender name code-agent, got %q", events[2].Sender.Name)
	}
}

func TestRunStreamOrderedParallelNotImplemented(t *testing.T) {
	// Mixed keywords should produce an ordered_parallel plan.
	// Phase 4 handler must return a safe NOT_IMPLEMENTED event and not dispatch.
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_op_001","messages":[{"role":"user","text":"帮我做一个登录页面和 Go 登录接口"}]}`)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var events []OrchestratorStreamEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event OrchestratorStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			events = append(events, event)
		}
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (run_started + run_error), got %d", len(events))
	}
	if events[0].Type != "run_started" {
		t.Errorf("expected run_started, got %q", events[0].Type)
	}
	if events[0].State == nil || events[0].State["planId"] == nil || events[0].State["planId"] == "" {
		t.Error("expected planId in run_started state")
	}
	if events[1].Type != "run_error" {
		t.Errorf("expected run_error, got %q", events[1].Type)
	}
	if events[1].Error == nil || events[1].Error.Code != "ORCHESTRATOR_NOT_IMPLEMENTED" {
		t.Errorf("expected ORCHESTRATOR_NOT_IMPLEMENTED, got %+v", events[1].Error)
	}
}

func TestRunStreamAgentNameFromSelectedAgentNames(t *testing.T) {
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"events": []map[string]any{
				{
					"author": "web-agent",
					"role":   "assistant",
					"parts":  []map[string]any{{"type": "text", "text": "web output"}},
				},
			},
		})
	}))
	defer mockAgent.Close()

	reg, _ := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "web-agent", URL: mockAgent.URL, Description: "web"},
	})
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
	)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_sel_001","selectedAgentNames":["web-agent"],"messages":[{"role":"user","text":"make a page"}]}`)
	resp, _ := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var events []OrchestratorStreamEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var event OrchestratorStreamEvent
			json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event)
			events = append(events, event)
		}
	}

	if len(events) < 5 {
		t.Fatalf("expected at least 5 events, got %d", len(events))
	}
	if events[1].Sender == nil || events[1].Sender.Name != "web-agent" {
		t.Errorf("expected web-agent sender, got %+v", events[1].Sender)
	}
}
