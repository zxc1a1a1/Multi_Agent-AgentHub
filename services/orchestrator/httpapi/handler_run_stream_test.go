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
		{Name: "code-agent", URL: "http://127.0.0.1:19999", Description: "code",
			CapabilityIDs: []string{"code_generation"}, OutputTypes: []string{"code", "text"}},
		{Name: "web-agent", URL: "http://127.0.0.1:19998", Description: "web",
			CapabilityIDs: []string{"web_generation"}, OutputTypes: []string{"webpage", "html", "text", "markdown"}},
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
		{Name: "code-agent", URL: mockAgent.URL, Description: "code",
				CapabilityIDs: []string{"code_generation"}, OutputTypes: []string{"code", "text"}},
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
		if events[0].State["phase"] != "executing" {
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

func TestRunStreamOrderedParallelWithMockAgents(t *testing.T) {
	// Phase 7: mixed keywords produce ordered_parallel plan that executes
	// web-agent and code-agent serially, then emits orchestrator summary.
	mockWeb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"events": []map[string]any{
				{
					"author": "web-agent",
					"role":   "assistant",
					"parts":  []map[string]any{{"type": "text", "text": "<html>login page</html>"}},
				},
			},
		})
	}))
	defer mockWeb.Close()

	mockCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"events": []map[string]any{
				{
					"author": "code-agent",
					"role":   "assistant",
					"parts":  []map[string]any{{"type": "text", "text": "package main\nfunc main() {}"}},
				},
			},
		})
	}))
	defer mockCode.Close()

	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: mockCode.URL, Description: "code",
			CapabilityIDs: []string{"code_generation"}, OutputTypes: []string{"code", "text"}},
		{Name: "web-agent", URL: mockWeb.URL, Description: "web",
			CapabilityIDs: []string{"web_generation"}, OutputTypes: []string{"webpage", "html", "text", "markdown"}},
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

	if len(events) < 10 {
		t.Fatalf("expected at least 10 events, got %d", len(events))
	}

	// 1. run_started
	if events[0].Type != "run_started" {
		t.Errorf("event[0]: expected run_started, got %q", events[0].Type)
	}
	if events[0].State == nil || events[0].State["planId"] == nil || events[0].State["planId"] == "" {
		t.Error("expected planId in run_started state")
	}
	if events[0].RunID != "run_op_001" {
		t.Errorf("expected runId=run_op_001, got %q", events[0].RunID)
	}

	// 2-4: web-agent task (priority 1)
	if events[1].Type != "message_start" {
		t.Errorf("event[1]: expected message_start, got %q", events[1].Type)
	}
	if events[1].Sender == nil || events[1].Sender.Name != "web-agent" {
		t.Errorf("event[1]: expected sender=web-agent, got %+v", events[1].Sender)
	}
	if events[1].TaskID == "" {
		t.Error("event[1]: expected taskId")
	}

	if events[2].Type != "message_delta" {
		t.Errorf("event[2]: expected message_delta, got %q", events[2].Type)
	}
	if events[2].Sender == nil || events[2].Sender.Name != "web-agent" {
		t.Errorf("event[2]: expected sender=web-agent, got %+v", events[2].Sender)
	}
	if events[2].Delta == "" {
		t.Error("event[2]: expected non-empty delta")
	}

	if events[3].Type != "message_end" {
		t.Errorf("event[3]: expected message_end, got %q", events[3].Type)
	}
	if events[3].Sender == nil || events[3].Sender.Name != "web-agent" {
		t.Errorf("event[3]: expected sender=web-agent, got %+v", events[3].Sender)
	}

	// 5-7: code-agent task (priority 2)
	if events[4].Type != "message_start" {
		t.Errorf("event[4]: expected message_start, got %q", events[4].Type)
	}
	if events[4].Sender == nil || events[4].Sender.Name != "code-agent" {
		t.Errorf("event[4]: expected sender=code-agent, got %+v", events[4].Sender)
	}

	if events[5].Type != "message_delta" {
		t.Errorf("event[5]: expected message_delta, got %q", events[5].Type)
	}
	if events[5].Sender == nil || events[5].Sender.Name != "code-agent" {
		t.Errorf("event[5]: expected sender=code-agent, got %+v", events[5].Sender)
	}

	if events[6].Type != "message_end" {
		t.Errorf("event[6]: expected message_end, got %q", events[6].Type)
	}
	if events[6].Sender == nil || events[6].Sender.Name != "code-agent" {
		t.Errorf("event[6]: expected sender=code-agent, got %+v", events[6].Sender)
	}

	// 8-10: orchestrator summary
	if events[7].Type != "message_start" {
		t.Errorf("event[7]: expected message_start, got %q", events[7].Type)
	}
	if events[7].Sender == nil || events[7].Sender.Name != "orchestrator" {
		t.Errorf("event[7]: expected sender=orchestrator, got %+v", events[7].Sender)
	}

	if events[8].Type != "message_delta" {
		t.Errorf("event[8]: expected message_delta, got %q", events[8].Type)
	}
	if events[8].Sender == nil || events[8].Sender.Name != "orchestrator" {
		t.Errorf("event[8]: expected sender=orchestrator, got %+v", events[8].Sender)
	}
	if events[8].Delta == "" {
		t.Error("event[8]: expected non-empty summary delta")
	}

	if events[9].Type != "message_end" {
		t.Errorf("event[9]: expected message_end, got %q", events[9].Type)
	}
	if events[9].Sender == nil || events[9].Sender.Name != "orchestrator" {
		t.Errorf("event[9]: expected sender=orchestrator, got %+v", events[9].Sender)
	}

	// run_finished
	if events[10].Type != "run_finished" {
		t.Errorf("event[10]: expected run_finished, got %q", events[10].Type)
	}
	if events[10].State == nil || events[10].State["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", events[10].State)
	}

	// Verify web-agent and code-agent have different senders (not merged).
	if events[1].Sender.Name == events[4].Sender.Name {
		t.Error("web-agent and code-agent must have different sender names")
	}

	// Verify taskIds differ.
	if events[1].TaskID == events[4].TaskID {
		t.Error("expected different taskIds for web-agent and code-agent")
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
		{Name: "web-agent", URL: mockAgent.URL, Description: "web",
			CapabilityIDs: []string{"web_generation"}, OutputTypes: []string{"webpage", "html", "text", "markdown"}},
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
