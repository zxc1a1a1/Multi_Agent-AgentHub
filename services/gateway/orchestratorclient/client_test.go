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
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
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

func TestNoOverallClientTimeout(t *testing.T) {
	// The streaming client must NOT set an overall http.Client.Timeout because
	// the SSE body may stay open for long periods (e.g. HITL awaiting_confirmation).
	// Only Transport-level timeouts (dial, TLS, response header) should be used.
	svc, err := NewOrchestratorRunService("http://localhost:12345", "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	if svc.httpClient.Timeout != 0 {
		t.Errorf("http.Client.Timeout must be 0 (no overall timeout) for SSE streaming, got %v", svc.httpClient.Timeout)
	}
	if svc.httpClient.Transport == nil {
		t.Error("http.Client.Transport must be set for connection-level timeouts")
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

func TestOrchestratorRunService_PlanningModeFieldsForwarded(t *testing.T) {
	// Verify that the orchestrator receives planningMode, agentName,
	// selectedAgentNames, and mentions in the JSON body.
	var capturedBody map[string]any

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture and decode the request body.
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

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

	// Build context with all four fields.
	ctx := context.Background()
	ctx = runservice.WithAgentName(ctx, "code-agent")
	ctx = runservice.WithSelectedAgentNames(ctx, []string{"code-agent", "web-agent"})
	ctx = runservice.WithMentions(ctx, []string{"document-agent"})
	ctx = runservice.WithPlanningMode(ctx, runservice.PlanningModeManual)

	seq := svc.Run(ctx, "conv_001", &adk.Content{
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
		return true
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	if capturedBody == nil {
		t.Fatal("expected request body to be captured")
	}

	// Verify planningMode.
	if pm, ok := capturedBody["planningMode"].(string); !ok || pm != "manual" {
		t.Errorf("expected planningMode='manual', got %v", capturedBody["planningMode"])
	}

	// Verify agentName.
	if an, ok := capturedBody["agentName"].(string); !ok || an != "code-agent" {
		t.Errorf("expected agentName='code-agent', got %v", capturedBody["agentName"])
	}

	// Verify selectedAgentNames.
	sanRaw, ok := capturedBody["selectedAgentNames"].([]interface{})
	if !ok {
		t.Errorf("expected selectedAgentNames to be an array, got %T", capturedBody["selectedAgentNames"])
	} else {
		if len(sanRaw) != 2 {
			t.Errorf("expected selectedAgentNames length 2, got %d", len(sanRaw))
		} else {
			if sanRaw[0].(string) != "code-agent" || sanRaw[1].(string) != "web-agent" {
				t.Errorf("unexpected selectedAgentNames: %v", sanRaw)
			}
		}
	}

	// Verify mentions.
	mentionsRaw, ok := capturedBody["mentions"].([]interface{})
	if !ok {
		t.Errorf("expected mentions to be an array, got %T", capturedBody["mentions"])
	} else {
		if len(mentionsRaw) != 1 || mentionsRaw[0].(string) != "document-agent" {
			t.Errorf("expected mentions=[document-agent], got %v", mentionsRaw)
		}
	}
}

func TestParseSSEStreamUsesInternalStreamEvent(t *testing.T) {
	// Verify that the SSE data deserializes to agui.InternalStreamEvent
	// and that Activity is typed (*agui.ActivitySnapshot), not json.RawMessage.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		fmt.Fprint(w, `event: run_started
data: {"type":"run_started","runId":"run_001","state":{"phase":"accepted"}}

event: activity_snapshot
data: {"type":"activity_snapshot","runId":"run_001","messageId":"msg_001","sender":{"type":"agent","name":"orchestrator"},"activity":{"activityId":"plan_001","activityType":"plan_approval","status":"awaiting_confirmation","executionPath":"single_chat","planId":"plan_001","revision":1,"planOwner":{"type":"main_agent","agentName":"main-agent","isMainAgent":true},"participants":[{"agentName":"code-agent","required":true,"selected":true}],"tasks":[{"taskId":"task_001","agentName":"code-agent","content":"Write code","priority":1,"riskLevel":"low"}],"allowedActions":["approve","revise","cancel"]}}

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

	// Verify activity_snapshot event was processed and carries StateDelta["activity"] as json.RawMessage
	foundActivity := false
	for _, e := range events {
		evtType, _ := e.Metadata["eventType"].(string)
		if evtType != "activity_snapshot" {
			continue
		}
		foundActivity = true
		if e.Actions == nil || e.Actions.StateDelta == nil {
			t.Fatal("activity_snapshot event should have Actions.StateDelta")
		}
		activityRaw, ok := e.Actions.StateDelta["activity"]
		if !ok {
			t.Fatal("activity_snapshot event should have StateDelta[\"activity\"]")
		}
		// Should be json.RawMessage (byte slice)
		switch activityRaw.(type) {
		case json.RawMessage:
			// expected — the translator expects this type
		default:
			t.Errorf("activity in StateDelta should be json.RawMessage, got %T", activityRaw)
		}
	}
	if !foundActivity {
		t.Error("expected activity_snapshot event in stream")
	}
}

func TestGatewayActivitySnapshotFieldPassthrough(t *testing.T) {
	// Verify that ActivitySnapshot fields survive parseSSEStream without loss.
	activityJSON := `{"activityId":"plan_001","activityType":"plan_approval","status":"awaiting_confirmation","executionPath":"single_chat","planId":"plan_001","revision":2,"planOwner":{"type":"main_agent","agentName":"main-agent","isMainAgent":true},"participants":[{"agentName":"code-agent","required":true,"selected":true},{"agentName":"web-agent","required":false,"selected":false}],"tasks":[{"taskId":"task_001","agentName":"code-agent","content":"Build API","priority":1,"riskLevel":"low"}],"allowedActions":["approve","revise","cancel"]}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		fmt.Fprintf(w, `event: activity_snapshot
data: {"type":"activity_snapshot","runId":"run_001","messageId":"msg_001","sender":{"type":"agent","name":"orchestrator"},"activity":%s}

event: run_finished
data: {"type":"run_finished","runId":"run_001","state":{"status":"completed"}}

`, activityJSON)
		flusher.Flush()
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

	for _, e := range events {
		evtType, _ := e.Metadata["eventType"].(string)
		if evtType != "activity_snapshot" {
			continue
		}
		if e.Actions == nil || e.Actions.StateDelta == nil {
			t.Fatal("activity_snapshot should have Actions.StateDelta")
		}
		raw, ok := e.Actions.StateDelta["activity"].(json.RawMessage)
		if !ok {
			t.Fatalf("activity should be json.RawMessage, got %T", e.Actions.StateDelta["activity"])
		}

		var parsed map[string]any
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("failed to unmarshal activity: %v", err)
		}

		// Verify key fields passthrough
		if parsed["activityId"] != "plan_001" {
			t.Errorf("expected activityId=plan_001, got %v", parsed["activityId"])
		}
		if parsed["revision"].(float64) != 2 {
			t.Errorf("expected revision=2, got %v", parsed["revision"])
		}
		if parsed["executionPath"] != "single_chat" {
			t.Errorf("expected executionPath=single_chat, got %v", parsed["executionPath"])
		}

		// PlanOwner
		po, ok := parsed["planOwner"].(map[string]any)
		if !ok {
			t.Fatal("planOwner should be an object")
		}
		if po["type"] != "main_agent" || po["agentName"] != "main-agent" {
			t.Errorf("planOwner fields incorrect: %v", po)
		}

		// Participants
		participants, ok := parsed["participants"].([]interface{})
		if !ok || len(participants) != 2 {
			t.Errorf("expected 2 participants, got %v", parsed["participants"])
		}

		// Tasks
		tasks, ok := parsed["tasks"].([]interface{})
		if !ok || len(tasks) != 1 {
			t.Errorf("expected 1 task, got %v", parsed["tasks"])
		}
	}
}

func TestGatewayAgentTurnFieldPassthrough(t *testing.T) {
	// Verify that AGENT_TURN fields (turnIndex, stepId, agentName, delta, status)
	// survive parseSSEStream and appear in adk.Event Metadata correctly.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// agent_turn_started
		fmt.Fprint(w, `event: agent_turn_started
data: {"type":"agent_turn_started","runId":"run_001","messageId":"msg_001","turnIndex":0,"stepId":"step-1","agentName":"code-agent","sender":{"type":"agent","name":"code-agent"}}

`)
		// agent_turn_content
		fmt.Fprint(w, `event: agent_turn_content
data: {"type":"agent_turn_content","runId":"run_001","messageId":"msg_001","turnIndex":0,"delta":"Here is some code"}

`)
		// agent_turn_finished
		fmt.Fprint(w, `event: agent_turn_finished
data: {"type":"agent_turn_finished","runId":"run_001","messageId":"msg_001","turnIndex":0,"agentName":"code-agent","status":"completed"}

`)
		// run_finished
		fmt.Fprint(w, `event: run_finished
data: {"type":"run_finished","runId":"run_001","state":{"status":"completed"}}

`)
		flusher.Flush()
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

	foundStarted := false
	foundContent := false
	foundFinished := false

	for _, e := range events {
		evtType, _ := e.Metadata["eventType"].(string)
		switch evtType {
		case "agent_turn_started":
			foundStarted = true
			// turnIndex stored as int in meta via ose.TurnIndex (int)
			ti, ok := e.Metadata["turnIndex"].(int)
			if !ok || ti != 0 {
				t.Errorf("agent_turn_started: expected meta turnIndex=0 (int), got %v (type=%T)", e.Metadata["turnIndex"], e.Metadata["turnIndex"])
			}
			stepID, _ := e.Metadata["stepId"].(string)
			if stepID != "step-1" {
				t.Errorf("agent_turn_started: expected stepId=step-1, got %q", stepID)
			}
			agentName, _ := e.Metadata["agentName"].(string)
			if agentName != "code-agent" {
				t.Errorf("agent_turn_started: expected agentName=code-agent, got %q", agentName)
			}

		case "agent_turn_content":
			foundContent = true
			ti, ok := e.Metadata["turnIndex"].(int)
			if !ok || ti != 0 {
				t.Errorf("agent_turn_content: expected meta turnIndex=0 (int), got %v (type=%T)", e.Metadata["turnIndex"], e.Metadata["turnIndex"])
			}
			if e.Content == nil || len(e.Content.Parts) == 0 {
				t.Error("agent_turn_content: expected Content with parts")
			} else {
				tp, ok := e.Content.Parts[0].(adk.TextPart)
				if !ok || tp.Text != "Here is some code" {
					t.Errorf("agent_turn_content: expected delta text, got %v", e.Content.Parts[0])
				}
			}

		case "agent_turn_finished":
			foundFinished = true
			ti, ok := e.Metadata["turnIndex"].(int)
			if !ok || ti != 0 {
				t.Errorf("agent_turn_finished: expected meta turnIndex=0 (int), got %v (type=%T)", e.Metadata["turnIndex"], e.Metadata["turnIndex"])
			}
			agentName, _ := e.Metadata["agentName"].(string)
			if agentName != "code-agent" {
				t.Errorf("agent_turn_finished: expected agentName=code-agent, got %q", agentName)
			}
			status, _ := e.Metadata["status"].(string)
			if status != "completed" {
				t.Errorf("agent_turn_finished: expected status=completed, got %q", status)
			}
			if !e.Final {
				t.Error("agent_turn_finished: expected Final=true")
			}
		}
	}

	if !foundStarted {
		t.Error("expected agent_turn_started event in stream")
	}
	if !foundContent {
		t.Error("expected agent_turn_content event in stream")
	}
	if !foundFinished {
		t.Error("expected agent_turn_finished event in stream")
	}
}

func TestOrchestratorRunService_PlanningModeDefaultsToAuto(t *testing.T) {
	// When no planningMode is set in context, the client should default to "auto".
	var capturedBody map[string]any

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

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

	// No planning mode set in context.
	ctx := context.Background()

	seq := svc.Run(ctx, "conv_001", &adk.Content{
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
		return true
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	if capturedBody == nil {
		t.Fatal("expected request body to be captured")
	}

	// planningMode should default to "auto".
	if pm, ok := capturedBody["planningMode"].(string); !ok || pm != "auto" {
		t.Errorf("expected planningMode='auto' as default, got %v", capturedBody["planningMode"])
	}

	// selectedAgentNames should be empty array (not null).
	sanRaw, ok := capturedBody["selectedAgentNames"].([]interface{})
	if !ok {
		t.Errorf("expected selectedAgentNames to be an array, got %T", capturedBody["selectedAgentNames"])
	} else if len(sanRaw) != 0 {
		t.Errorf("expected empty selectedAgentNames, got %v", sanRaw)
	}

	// mentions should be empty array (not null).
	mentionsRaw, ok := capturedBody["mentions"].([]interface{})
	if !ok {
		t.Errorf("expected mentions to be an array, got %T", capturedBody["mentions"])
	} else if len(mentionsRaw) != 0 {
		t.Errorf("expected empty mentions, got %v", mentionsRaw)
	}
}
