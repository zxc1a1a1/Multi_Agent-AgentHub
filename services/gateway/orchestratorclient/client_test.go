package orchestratorclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func TestOrchestratorRunService_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("expected flusher")
			return
		}

		events := []string{
			`event: run_started
data: {"type":"run_started","runId":"run_001","state":{"phase":"accepted"}}

`,
			`event: message_start
data: {"type":"message_start","runId":"run_001","messageId":"msg_001","sender":{"type":"agent","name":"test-agent"}}

`,
			`event: message_delta
data: {"type":"message_delta","runId":"run_001","messageId":"msg_001","delta":"Hello from"}

`,
			`event: message_delta
data: {"type":"message_delta","runId":"run_001","messageId":"msg_001","delta":" orchestrator"}

`,
			`event: message_end
data: {"type":"message_end","runId":"run_001","messageId":"msg_001"}

`,
			`event: run_finished
data: {"type":"run_finished","runId":"run_001","state":{"status":"completed"}}

`,
		}

		for _, e := range events {
			fmt.Fprint(w, e)
			flusher.Flush()
		}
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	var events []adk.Event
	seq := svc.Run(context.Background(), "conv_001", &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: "hello"},
		},
	})

	var runErr error
	seq(func(event adk.Event, err error) bool {
		if err != nil {
			runErr = err
			return false
		}
		events = append(events, event)
		return true
	})

	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	if len(events) == 0 {
		t.Fatal("expected non-empty events")
	}

	// Verify we got delta events with correct author and metadata
	foundDelta := false
	foundMeta := false
	foundRunID := false
	for _, e := range events {
		if e.Author == "test-agent" {
			foundDelta = true
		}
		if e.Metadata != nil {
			if et, ok := e.Metadata["eventType"].(string); ok && et != "" {
				foundMeta = true
			}
			if rid, ok := e.Metadata["runId"].(string); ok && rid == "run_001" {
				foundRunID = true
			}
		}
		if e.Content != nil {
			for _, p := range e.Content.Parts {
				if tp, ok := p.(adk.TextPart); ok && tp.Text != "" {
					foundDelta = true
				}
			}
		}
	}
	if !foundDelta {
		t.Error("expected delta events with agent author")
	}
	if !foundMeta {
		t.Error("expected metadata with eventType on events")
	}
	if !foundRunID {
		t.Error("expected metadata with runId=run_001 on events")
	}

	// Verify we got a final event
	foundFinal := false
	for _, e := range events {
		if e.Final {
			foundFinal = true
		}
	}
	if !foundFinal {
		t.Error("expected at least one final event")
	}

	// Verify message_delta events carry messageId in metadata
	foundMsgID := false
	for _, e := range events {
		if e.Metadata != nil {
			if mid, ok := e.Metadata["messageId"].(string); ok && mid == "msg_001" {
				foundMsgID = true
			}
		}
	}
	if !foundMsgID {
		t.Error("expected messageId=msg_001 in metadata of delta events")
	}
}

func TestOrchestratorRunService_Error(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		fmt.Fprint(w, `event: run_started
data: {"type":"run_started","runId":"run_err","state":{"phase":"accepted"}}

event: run_error
data: {"type":"run_error","runId":"run_err","error":{"code":"ORCHESTRATOR_INTERNAL","message":"something went wrong"}}

`)
		flusher.Flush()
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	seq := svc.Run(context.Background(), "conv_001", &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: "hello"},
		},
	})

	var events []adk.Event
	var gotErr error
	seq(func(event adk.Event, err error) bool {
		if err != nil {
			gotErr = err
			return false
		}
		events = append(events, event)
		return true
	})

	// run_error is now delivered as an adk.Event (not an error yield) so the
	// Gateway/Translator can produce a RUN_ERROR AG-UI event.
	if gotErr != nil {
		t.Fatalf("unexpected error: run_error should be an event, not an error: %v", gotErr)
	}

	// Should have received at least the run_started and run_error events
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (run_started + run_error), got %d", len(events))
	}

	// The last event should be run_error with metadata and error info in Actions
	lastEvent := events[len(events)-1]
	if !lastEvent.Final {
		t.Error("run_error event should be marked Final=true")
	}
	evtType, _ := lastEvent.Metadata["eventType"].(string)
	if evtType != "run_error" {
		t.Errorf("expected metadata eventType=run_error, got %q", evtType)
	}
	runID, _ := lastEvent.Metadata["runId"].(string)
	if runID != "run_err" {
		t.Errorf("expected metadata runId=run_err, got %q", runID)
	}
	if lastEvent.Actions == nil || lastEvent.Actions.StateDelta == nil {
		t.Fatal("run_error event should carry error info in Actions.StateDelta")
	}
	code, _ := lastEvent.Actions.StateDelta["code"].(string)
	msg, _ := lastEvent.Actions.StateDelta["message"].(string)
	if code != "ORCHESTRATOR_INTERNAL" {
		t.Errorf("expected error code ORCHESTRATOR_INTERNAL, got %q", code)
	}
	if !strings.Contains(msg, "something went wrong") {
		t.Errorf("expected error message to contain 'something went wrong', got %q", msg)
	}
}

func TestOrchestratorRunService_HTTPError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "down"})
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	seq := svc.Run(context.Background(), "conv_001", &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: "hello"},
		},
	})

	var gotErr error
	seq(func(event adk.Event, err error) bool {
		if err != nil {
			gotErr = err
			return false
		}
		return true
	})

	if gotErr == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(gotErr.Error(), "503") {
		t.Errorf("expected 503 in error, got %q", gotErr.Error())
	}
}

func TestOrchestratorRunService_InvalidURL(t *testing.T) {
	_, err := NewOrchestratorRunService("", "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestOrchestratorRunService_NilContext(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, `event: run_started
data: {"type":"run_started","runId":"run_001","state":{"phase":"accepted"}}

event: run_finished
data: {"type":"run_finished","runId":"run_001","state":{"status":"completed"}}

`)
		flusher.Flush()
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	seq := svc.Run(context.Background(), "conv_001", &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: "hello"},
		},
	})

	var count int
	seq(func(event adk.Event, err error) bool {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return false
		}
		count++
		return true
	})

	if count < 1 {
		t.Error("expected at least 1 event")
	}
}
