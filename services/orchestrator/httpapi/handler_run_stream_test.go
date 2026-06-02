package httpapi

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunStreamMock(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_test_001","conversationId":"conv_test"}`)
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
			if event.Type != currentType {
				t.Errorf("event type mismatch: data has %q, event line has %q", event.Type, currentType)
			}
			events = append(events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
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
	if events[2].Delta == "" {
		t.Error("expected non-empty delta in message_delta")
	}
}

func TestRunStreamMethodNotAllowed(t *testing.T) {
	srv := NewServer()
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
	srv := NewServer(WithInternalToken("secret-token"))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"runId":"run_001"}`)
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

	body2 := strings.NewReader(`{"runId":"run_001"}`)
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
	srv := NewServer()
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

func TestRunStreamGeneratesRunID(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{}`)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/runs/stream", "application/json", body)
	if err != nil {
		t.Fatalf("POST run/stream failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var firstEvent OrchestratorStreamEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if err := json.Unmarshal([]byte(data), &firstEvent); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			break
		}
	}

	if firstEvent.RunID == "" {
		t.Error("expected auto-generated runId, got empty")
	}
	if !strings.HasPrefix(firstEvent.RunID, "run_mock_") {
		t.Errorf("expected runId to start with run_mock_, got %q", firstEvent.RunID)
	}
}
