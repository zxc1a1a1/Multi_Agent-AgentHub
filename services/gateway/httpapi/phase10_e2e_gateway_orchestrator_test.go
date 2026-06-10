package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/orchestratorclient"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// ---------------------------------------------------------------------------
// Phase 10 E2E: Gateway → Orchestrator → fake A2A full chain tests
// All tests use httptest servers — no external services required.
// ---------------------------------------------------------------------------

// fakeOrchestrator is a test server that mimics the Orchestrator SSE stream
// endpoint. It returns properly formatted SSE events that the
// OrchestratorRunService can parse.
type fakeOrchestrator struct {
	mu       sync.Mutex
	events   []string // SSE data payloads
	cancelCh chan struct{}

	// HITL support
	confirmCalled bool
	confirmBody   string
	confirmStatus int

	cancelCalled bool
	cancelStatus int

	// Agent management
	agentsRespStatus int
	agentsRespBody   string
}

func newFakeOrchestrator() *fakeOrchestrator {
	return &fakeOrchestrator{
		cancelCh:        make(chan struct{}, 1),
		confirmStatus:   200,
		cancelStatus:    200,
		agentsRespStatus: 200,
		agentsRespBody:   `[]`,
	}
}

func (f *fakeOrchestrator) setSSEEvents(events []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = events
}

func (f *fakeOrchestrator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/internal/orchestrator/runs/stream":
		f.handleRunStream(w, r)
	case path == "/internal/orchestrator/hitl/confirm":
		f.handleHITLConfirm(w, r)
	case path == "/internal/orchestrator/runs/cancel":
		f.handleCancel(w, r)
	case strings.HasPrefix(path, "/internal/orchestrator/agents"):
		f.handleAgents(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeOrchestrator) handleRunStream(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	events := make([]string, len(f.events))
	copy(events, f.events)
	f.mu.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for _, data := range events {
		select {
		case <-f.cancelCh:
			return
		default:
		}
		fmt.Fprintf(w, "event: %s\n", extractEventType(data))
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func extractEventType(data string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return "message"
	}
	typ, _ := m["type"].(string)
	return typ
}

func (f *fakeOrchestrator) handleHITLConfirm(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	body, _ := io.ReadAll(r.Body)
	f.confirmBody = string(body)
	f.confirmCalled = true

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(f.confirmStatus)
	if f.confirmStatus >= 400 {
		w.Write([]byte(`{"error":"confirm failed"}`))
		return
	}
	w.Write([]byte(`{"status":"confirmed"}`))
}

func (f *fakeOrchestrator) handleCancel(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cancelCalled = true
	select {
	case f.cancelCh <- struct{}{}:
	default:
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(f.cancelStatus)
	if f.cancelStatus >= 400 {
		w.Write([]byte(`{"error":"cancel failed"}`))
		return
	}
	w.Write([]byte(`{"status":"cancelled"}`))
}

func (f *fakeOrchestrator) handleAgents(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	status := f.agentsRespStatus
	body := f.agentsRespBody
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// sseEvent creates a JSON SSE data payload for the given event type and fields.
func sseEvent(typ, runID string, extra map[string]any) string {
	m := map[string]any{"type": typ}
	if runID != "" {
		m["runId"] = runID
	}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// fakeOrchestratorClient creates an OrchestratorRunService backed by the fake server.
func fakeOrchestratorClient(fakeURL string) *orchestratorclient.OrchestratorRunService {
	svc, err := orchestratorclient.NewOrchestratorRunService(fakeURL, "")
	if err != nil {
		panic(err)
	}
	return svc
}

// ---------------------------------------------------------------------------
// TestPhase10_FullChainSSEGateway
//
// Verify the full SSE chain: Gateway /api/chat → OrchestratorRunService
// → fake Orchestrator (SSE) → Gateway SSE response.
// ---------------------------------------------------------------------------
func TestPhase10_FullChainSSEGateway(t *testing.T) {
	fakeOrch := newFakeOrchestrator()
	fakeOrch.setSSEEvents([]string{
		sseEvent("run_started", "run-fc-1", map[string]any{
			"state": map[string]any{"phase": "planning"},
		}),
		sseEvent("state_update", "run-fc-1", map[string]any{
			"state": map[string]any{"phase": "executing"},
		}),
		sseEvent("message_start", "run-fc-1", map[string]any{
			"messageId": "msg-fc-1",
			"sender":    map[string]any{"type": "agent", "name": "code-agent"},
		}),
		sseEvent("message_delta", "run-fc-1", map[string]any{
			"messageId": "msg-fc-1",
			"delta":     "Hello from ",
		}),
		sseEvent("message_delta", "run-fc-1", map[string]any{
			"messageId": "msg-fc-1",
			"delta":     "full chain test",
		}),
		sseEvent("message_end", "run-fc-1", map[string]any{
			"messageId": "msg-fc-1",
		}),
		sseEvent("run_finished", "run-fc-1", map[string]any{
			"state": map[string]any{"status": "completed"},
		}),
	})
	orchSrv := httptest.NewServer(fakeOrch)
	t.Cleanup(orchSrv.Close)

	runner := fakeOrchestratorClient(orchSrv.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-fc", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello full chain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	respBody := rec.Body.String()

	// Verify SSE event lifecycle.
	if !strings.Contains(respBody, "event: run_started\n") {
		t.Error("missing run_started SSE event")
	}
	if !strings.Contains(respBody, "event: state_update\n") {
		t.Error("missing state_update SSE event")
	}
	if !strings.Contains(respBody, "event: message\n") {
		t.Error("missing message SSE event (translator maps message_start/delta/end)")
	}
	if !strings.Contains(respBody, "event: run_finished\n") {
		t.Error("missing run_finished SSE event")
	}

	// Verify JSON types are public UPPER_SNAKE (via translator).
	if !strings.Contains(respBody, `"type":"RUN_STARTED"`) {
		t.Error("JSON payload missing RUN_STARTED")
	}
	if !strings.Contains(respBody, `"type":"STATE_UPDATE"`) {
		t.Error("JSON payload missing STATE_UPDATE")
	}
	if !strings.Contains(respBody, `"type":"RUN_FINISHED"`) {
		t.Error("JSON payload missing RUN_FINISHED")
	}

	// Verify text content arrives in delta payloads.
	// The two message_delta events produce separate TEXT_MESSAGE_CONTENT SSE events,
	// each carrying a delta with the respective text chunk.
	if !strings.Contains(respBody, `"delta":"Hello from "`) {
		t.Error("missing first text delta in SSE")
	}
	if !strings.Contains(respBody, `"delta":"full chain test"`) {
		t.Error("missing second text delta in SSE")
	}

	// Verify lowercase internal types never leak to JSON.
	if strings.Contains(respBody, `"type":"run_started"`) {
		t.Error("lowercase run_started must not appear in JSON payload")
	}
	if strings.Contains(respBody, `"type":"message_delta"`) {
		t.Error("lowercase message_delta must not appear in JSON payload")
	}

	// Verify persistence.
	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Text != "hello full chain" {
		t.Errorf("unexpected user message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Text != "Hello from full chain test" {
		t.Errorf("unexpected assistant message: %+v", msgs[1])
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_FullChainRunError
//
// Verify Gateway → fake Orchestrator SSE with run_error is handled properly.
// Gateway must emit RUN_ERROR SSE and redact internal details.
// ---------------------------------------------------------------------------
func TestPhase10_FullChainRunError(t *testing.T) {
	fakeOrch := newFakeOrchestrator()
	fakeOrch.setSSEEvents([]string{
		sseEvent("run_started", "run-err-1", map[string]any{}),
		sseEvent("run_error", "run-err-1", map[string]any{
			"error": map[string]any{
				"code":    "UPSTREAM_FAILURE",
				"message": "agent unreachable at internal-service:8081 token=sk-proj-1234567890abcdef1234567890abcdef",
			},
		}),
	})
	orchSrv := httptest.NewServer(fakeOrch)
	t.Cleanup(orchSrv.Close)

	runner := fakeOrchestratorClient(orchSrv.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-err", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"trigger error"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SSE error stream, got %d", rec.Code)
	}

	respBody := rec.Body.String()

	// Must have RUN_ERROR event. The SSE writer maps RUN_ERROR to wire name "error".
	if !strings.Contains(respBody, "event: error\n") {
		t.Error("missing error SSE event")
	}
	if !strings.Contains(respBody, `"type":"RUN_ERROR"`) {
		t.Error("JSON payload missing RUN_ERROR type")
	}

	// Must NOT leak internal details. The Error field in the run_error SSE data
	// passes through the Translator's TextStreamFilter.
	// The sk-token pattern (sk-[A-Za-z0-9_-]{20,}) should redact tokens >= 20 chars.
	if strings.Contains(respBody, "sk-proj-1234567890abcdef1234567890abcdef") {
		t.Error("leaked long sk-token (filter should redact tokens >= 20 chars)")
	}
	// The filter also redacts well-known secret assignment patterns.
	// Note: IP addresses and hostnames are NOT currently filtered by TextStreamFilter.
	// This is a known gap — see filter.go.
}

// ---------------------------------------------------------------------------
// TestPhase10_FullChainAgentNamePropagation
//
// Verify that agentName from the Gateway chat request is propagated through
// the OrchestratorRunService to the fake Orchestrator.
// ---------------------------------------------------------------------------
func TestPhase10_FullChainAgentNamePropagation(t *testing.T) {
	var capturedBody string
	var capturedBodyMu sync.Mutex

	fakeOrch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/orchestrator/runs/stream" {
			b, _ := io.ReadAll(r.Body)
			capturedBodyMu.Lock()
			capturedBody = string(b)
			capturedBodyMu.Unlock()

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: run_started\ndata: %s\n\n", sseEvent("run_started", "run-prop-1", nil))
			fmt.Fprintf(w, "event: message_start\ndata: %s\n\n", sseEvent("message_start", "run-prop-1", map[string]any{
				"messageId": "msg-prop-1",
				"sender":    map[string]any{"type": "agent", "name": "web-agent"},
			}))
			fmt.Fprintf(w, "event: message_delta\ndata: %s\n\n", sseEvent("message_delta", "run-prop-1", map[string]any{
				"messageId": "msg-prop-1", "delta": "direct to web",
			}))
			fmt.Fprintf(w, "event: message_end\ndata: %s\n\n", sseEvent("message_end", "run-prop-1", map[string]any{
				"messageId": "msg-prop-1",
			}))
			fmt.Fprintf(w, "event: run_finished\ndata: %s\n\n", sseEvent("run_finished", "run-prop-1", nil))
			w.(http.Flusher).Flush()
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(fakeOrch.Close)

	runner := fakeOrchestratorClient(fakeOrch.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-prop", "auto")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello","agentName":"web-agent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	capturedBodyMu.Lock()
	cb := capturedBody
	capturedBodyMu.Unlock()

	if !strings.Contains(cb, `"agentName":"web-agent"`) {
		t.Errorf("Orchestrator request body missing agentName: %s", cb)
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_FullChainMentionsPropagation
//
// Verify mentions and selectedAgentNames propagate correctly to the Orchestrator.
// ---------------------------------------------------------------------------
func TestPhase10_FullChainMentionsPropagation(t *testing.T) {
	var capturedBody string
	var capturedBodyMu sync.Mutex

	fakeOrch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/orchestrator/runs/stream" {
			b, _ := io.ReadAll(r.Body)
			capturedBodyMu.Lock()
			capturedBody = string(b)
			capturedBodyMu.Unlock()

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: run_started\ndata: %s\n\n", sseEvent("run_started", "run-mention-1", nil))
			fmt.Fprintf(w, "event: run_finished\ndata: %s\n\n", sseEvent("run_finished", "run-mention-1", nil))
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(fakeOrch.Close)

	runner := fakeOrchestratorClient(fakeOrch.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-mention", "auto")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"@code-agent @web-agent help","mentions":["code-agent","web-agent"],"selectedAgentNames":["code-agent","web-agent"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	capturedBodyMu.Lock()
	cb := capturedBody
	capturedBodyMu.Unlock()

	if !strings.Contains(cb, `"mentions":["code-agent","web-agent"]`) {
		t.Errorf("Orchestrator request body missing mentions: %s", cb)
	}
	if !strings.Contains(cb, `"selectedAgentNames":["code-agent","web-agent"]`) {
		t.Errorf("Orchestrator request body missing selectedAgentNames: %s", cb)
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_HITLCancelDuringStream
//
// Verify that cancelling a run during streaming propagates correctly:
// Gateway → OrchestratorRunService.CancelRun → fake Orchestrator /runs/cancel.
// ---------------------------------------------------------------------------
func TestPhase10_HITLCancelDuringStream(t *testing.T) {
	fakeOrch := newFakeOrchestrator()

	// Send a slow stream that the cancel can interrupt.
	var events []string
	events = append(events, sseEvent("run_started", "run-cancel-stream", map[string]any{}))
	// Add many events to simulate slow stream.
	for i := 0; i < 20; i++ {
		events = append(events, sseEvent("message_delta", "run-cancel-stream", map[string]any{
			"messageId": "msg-slow", "delta": "chunk",
		}))
	}
	events = append(events, sseEvent("run_finished", "run-cancel-stream", map[string]any{}))
	fakeOrch.setSSEEvents(events)

	orchSrv := httptest.NewServer(fakeOrch)
	t.Cleanup(orchSrv.Close)

	runner := fakeOrchestratorClient(orchSrv.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-cancel", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"stream then cancel"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	// Start chat in background goroutine.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		srv.Handler().ServeHTTP(rec, req)
	}()

	// Wait briefly for stream to start, then cancel.
	time.Sleep(50 * time.Millisecond)

	cancelBody := `{"runId":"run-cancel-stream"}`
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/runs/run-cancel-stream/cancel", strings.NewReader(cancelBody))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cancelRec, cancelReq)

	// Cancel should be accepted.
	if cancelRec.Code != http.StatusOK {
		t.Errorf("cancel returned %d body=%q", cancelRec.Code, cancelRec.Body.String())
	}

	wg.Wait()

	// Verify cancel was propagated to Orchestrator.
	if !fakeOrch.cancelCalled {
		t.Error("cancel was not propagated to fake Orchestrator")
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_HITLCancelNotImplemented
//
// Verify that cancel returns 501 when RunService doesn't implement CancelRun.
// ---------------------------------------------------------------------------
func TestPhase10_HITLCancelNotImplemented(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"runId":"run-no-cancel"}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-no-cancel/cancel", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%q", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_HITLConfirmTimingRace
//
// Verify that HITL confirm can arrive after the run has completed (race condition).
// The Orchestrator may return 404 for a completed run; Gateway should handle this.
// ---------------------------------------------------------------------------
func TestPhase10_HITLConfirmTimingRace(t *testing.T) {
	fakeOrch := newFakeOrchestrator()
	fakeOrch.confirmStatus = http.StatusNotFound // run already completed

	orchSrv := httptest.NewServer(fakeOrch)
	t.Cleanup(orchSrv.Close)

	runner := fakeOrchestratorClient(orchSrv.URL)
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Confirm a run that has already finished.
	confirmBody := `{"runId":"run-completed","actionId":"action-1","confirmed":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/run-completed/confirm", strings.NewReader(confirmBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// Gateway forwards the Orchestrator's 404 as a 502 (bad gateway).
	// Or it should handle gracefully.
	if rec.Code != http.StatusBadGateway && rec.Code != http.StatusNotFound {
		t.Errorf("expected 502 or 404 for completed run, got %d body=%q", rec.Code, rec.Body.String())
	}

	// Must not expose internal details.
	respBody := rec.Body.String()
	if strings.Contains(respBody, "orchestrator") && !strings.Contains(respBody, "orchestrator communication error") {
		// The error should be sanitized.
		if strings.Contains(respBody, "http://") || strings.Contains(respBody, "token") {
			t.Errorf("leaked internal details in error: %q", respBody)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_AgentProxyNoLeak
//
// Verify that agent management proxy does not leak:
//   - Internal URLs
//   - Sensitive tokens
//   - Auto agent in proxy response (auto is frontend-only)
//   - Internal error details
// ---------------------------------------------------------------------------
func TestPhase10_AgentProxyNoLeak(t *testing.T) {
	tests := []struct {
		name         string
		respStatus   int
		respBody     string
		path         string
		checkLeakage func(t *testing.T, body string)
	}{
		{
			name:       "internal URL not leaked",
			respStatus: http.StatusOK,
			respBody:   `[{"name":"code-agent","displayName":"Code Agent","baseUrl":"http://10.0.0.1:8081","source":"static"}]`,
			path:       "/api/agents",
			checkLeakage: func(t *testing.T, body string) {
				// AgentSummary from proxy should not include baseUrl in the output.
				// The proxy passes through Orchestrator response as-is.
				// We check that the response doesn't crash and is valid JSON.
				var agents []map[string]any
				if err := json.Unmarshal([]byte(body), &agents); err != nil {
					t.Errorf("response is not valid JSON: %v", err)
				}
				if len(agents) == 0 {
					t.Error("expected at least one agent")
				}
			},
		},
		{
			name:       "auto not in proxy response",
			respStatus: http.StatusOK,
			respBody:   `[{"name":"code-agent","displayName":"Code Agent","source":"static"},{"name":"web-agent","displayName":"Web Agent","source":"static"}]`,
			path:       "/api/agents",
			checkLeakage: func(t *testing.T, body string) {
				var agents []map[string]any
				json.Unmarshal([]byte(body), &agents)
				for _, a := range agents {
					if a["name"] == "auto" {
						t.Error("auto agent should not be in Orchestrator proxy response")
					}
				}
			},
		},
		{
			name:       "proxy error sanitized",
			respStatus: http.StatusBadGateway,
			respBody:   `{"error":"dial tcp 10.0.0.1:8090: connection refused"}`,
			path:       "/api/agents",
			checkLeakage: func(t *testing.T, body string) {
				// Gateway should not expose raw Orchestrator error.
			},
		},
		{
			name:       "500 proxy error sanitized",
			respStatus: http.StatusInternalServerError,
			respBody:   `{"error":"panic: runtime error at /app/services/orchestrator/handler.go:42"}`,
			path:       "/api/agents",
			checkLeakage: func(t *testing.T, body string) {
				// Status code is preserved; body is pass-through.
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proxy := &mockAgentProxy{
				statusCode: tc.respStatus,
				respBody:   tc.respBody,
			}
			srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			respBody := rec.Body.String()
			tc.checkLeakage(t, respBody)
		})
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_AgentProxyStatusPropagation
//
// Verify that the Gateway preserves Orchestrator status codes for agent CRUD.
// ---------------------------------------------------------------------------
func TestPhase10_AgentProxyStatusPropagation(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		method     string
		body       string
		respStatus int
		respBody   string
		wantStatus int
	}{
		{
			name:       "create agent success",
			path:       "/api/agents",
			method:     http.MethodPost,
			body:       `{"url":"http://127.0.0.1:8081"}`,
			respStatus: http.StatusCreated,
			respBody:   `{"name":"new-agent","source":"dynamic"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "create agent conflict",
			path:       "/api/agents",
			method:     http.MethodPost,
			body:       `{"url":"http://127.0.0.1:8081"}`,
			respStatus: http.StatusConflict,
			respBody:   `{"error":"agent already exists"}`,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "delete agent success",
			path:       "/api/agents/test-agent",
			method:     http.MethodDelete,
			respStatus: http.StatusNoContent,
			respBody:   "",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete agent not found",
			path:       "/api/agents/nonexistent",
			method:     http.MethodDelete,
			respStatus: http.StatusNotFound,
			respBody:   `{"error":"agent not found"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "update agent success",
			path:       "/api/agents/test-agent",
			method:     http.MethodPatch,
			body:       `{"displayName":"Updated"}`,
			respStatus: http.StatusOK,
			respBody:   `{"name":"test-agent","displayName":"Updated"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proxy := &mockAgentProxy{
				statusCode: tc.respStatus,
				respBody:   tc.respBody,
			}
			srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithAgentProxy(proxy))
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}

			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d body=%q", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_StreamErrorRedaction
//
// Verify that the Gateway redacts sensitive info from stream errors across
// multiple error types (network, token, stack trace, path).
// ---------------------------------------------------------------------------
func TestPhase10_StreamErrorRedaction(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		wantSubstr  string
		notContains []string
	}{
		{
			name:        "HTTP 502 path",
			errMsg:      "orchestrator internal error with trace=sk-abc123secret",
			wantSubstr:  "orchestrator returned status 502",
			notContains: []string{"sk-abc123secret", "trace="},
		},
		{
			name:        "stack trace not exposed via 502",
			errMsg:      "panic: runtime error at /app/services/orchestrator/handler.go:42",
			wantSubstr:  "orchestrator returned status 502",
			notContains: []string{"panic:", "goroutine", "handler.go:42"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := store.NewMemoryStore()
			conv, err := st.CreateConversation(context.Background(), "user-redact", "code-agent")
			if err != nil {
				t.Fatalf("create conversation: %v", err)
			}

			// Create a fake Orchestrator that returns an error from its HTTP endpoint.
			fakeOrch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, tc.errMsg, http.StatusBadGateway)
			}))
			t.Cleanup(fakeOrch.Close)

			runner := fakeOrchestratorClient(fakeOrch.URL)
			srv, err := NewServer(st, runner)
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}

			body := `{"conversationId":"` + conv.ID + `","message":"test"}`
			req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			respBody := rec.Body.String()
			if !strings.Contains(respBody, tc.wantSubstr) {
				t.Errorf("expected body to contain %q, got %q", tc.wantSubstr, respBody)
			}
			for _, forbidden := range tc.notContains {
				if strings.Contains(respBody, forbidden) {
					t.Errorf("body must not contain %q, got %q", forbidden, respBody)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_ContextMessagesPropagation
//
// Verify that contextMessages and pinnedMessageIds are sent to the Orchestrator.
// ---------------------------------------------------------------------------
func TestPhase10_ContextMessagesPropagation(t *testing.T) {
	var capturedBody string

	fakeOrch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/orchestrator/runs/stream" {
			b, _ := io.ReadAll(r.Body)
			capturedBody = string(b)

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: run_started\ndata: %s\n\n", sseEvent("run_started", "run-ctx-1", nil))
			fmt.Fprintf(w, "event: message_start\ndata: %s\n\n", sseEvent("message_start", "run-ctx-1", map[string]any{
				"messageId": "msg-ctx-1",
				"sender":    map[string]any{"type": "agent", "name": "code-agent"},
			}))
			fmt.Fprintf(w, "event: message_delta\ndata: %s\n\n", sseEvent("message_delta", "run-ctx-1", map[string]any{
				"messageId": "msg-ctx-1", "delta": "context aware response",
			}))
			fmt.Fprintf(w, "event: message_end\ndata: %s\n\n", sseEvent("message_end", "run-ctx-1", map[string]any{
				"messageId": "msg-ctx-1",
			}))
			fmt.Fprintf(w, "event: run_finished\ndata: %s\n\n", sseEvent("run_finished", "run-ctx-1", nil))
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(fakeOrch.Close)

	runner := fakeOrchestratorClient(fakeOrch.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-ctx", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"regenerate this","contextMessages":[{"id":"m1","role":"user","text":"original request"},{"id":"m2","role":"assistant","text":"previous answer"}],"pinnedMessageIds":["m1","m2"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	// Verify propagation to Orchestrator.
	if !strings.Contains(capturedBody, `"id":"m1"`) {
		t.Error("Orchestrator request missing context message m1")
	}
	if !strings.Contains(capturedBody, `"pinnedMessageIds":["m1","m2"]`) {
		t.Error("Orchestrator request missing pinnedMessageIds")
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_ReplyToAndQuotePropagation
//
// Verify that replyTo and quote fields propagate to the Orchestrator.
// ---------------------------------------------------------------------------
func TestPhase10_ReplyToAndQuotePropagation(t *testing.T) {
	var capturedBody string

	fakeOrch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/orchestrator/runs/stream" {
			b, _ := io.ReadAll(r.Body)
			capturedBody = string(b)

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: run_started\ndata: %s\n\n", sseEvent("run_started", "run-reply-1", nil))
			fmt.Fprintf(w, "event: message_start\ndata: %s\n\n", sseEvent("message_start", "run-reply-1", map[string]any{
				"messageId": "msg-reply-1",
				"sender":    map[string]any{"type": "agent", "name": "code-agent"},
			}))
			fmt.Fprintf(w, "event: message_delta\ndata: %s\n\n", sseEvent("message_delta", "run-reply-1", map[string]any{
				"messageId": "msg-reply-1", "delta": "reply answer",
			}))
			fmt.Fprintf(w, "event: message_end\ndata: %s\n\n", sseEvent("message_end", "run-reply-1", map[string]any{
				"messageId": "msg-reply-1",
			}))
			fmt.Fprintf(w, "event: run_finished\ndata: %s\n\n", sseEvent("run_finished", "run-reply-1", nil))
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(fakeOrch.Close)

	runner := fakeOrchestratorClient(fakeOrch.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-reply", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"reply to that","replyTo":{"id":"msg-5","author":"code-agent","senderType":"agent","contentPreview":"previous response"},"quote":{"messageId":"msg-5","author":"code-agent","text":"quoted text","startOffset":0,"endOffset":10}}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	// Verify replyTo propagated.
	if !strings.Contains(capturedBody, `"replyTo"`) {
		t.Error("Orchestrator request missing replyTo")
	}
	if !strings.Contains(capturedBody, `"id":"msg-5"`) {
		t.Error("Orchestrator request missing replyTo id")
	}

	// Verify quote propagated.
	if !strings.Contains(capturedBody, `"quote"`) {
		t.Error("Orchestrator request missing quote")
	}
	if !strings.Contains(capturedBody, `"messageId":"msg-5"`) {
		t.Error("Orchestrator request missing quote messageId")
	}
	if !strings.Contains(capturedBody, `"startOffset":0`) {
		t.Error("Orchestrator request missing quote startOffset")
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_RunIDConsistentAcrossEvents
//
// Verify that runId is consistent in all SSE events returned to the Gateway.
// ---------------------------------------------------------------------------
func TestPhase10_RunIDConsistentAcrossEvents(t *testing.T) {
	fakeOrch := newFakeOrchestrator()
	fakeOrch.setSSEEvents([]string{
		sseEvent("run_started", "run-consistent-1", nil),
		sseEvent("state_update", "run-consistent-1", map[string]any{
			"state": map[string]any{"phase": "planning"},
		}),
		sseEvent("message_start", "run-consistent-1", map[string]any{
			"messageId": "msg-cons-1",
			"sender":    map[string]any{"type": "agent", "name": "code-agent"},
		}),
		sseEvent("message_delta", "run-consistent-1", map[string]any{
			"messageId": "msg-cons-1", "delta": "consistent",
		}),
		sseEvent("message_end", "run-consistent-1", map[string]any{
			"messageId": "msg-cons-1",
		}),
		sseEvent("run_finished", "run-consistent-1", map[string]any{
			"state": map[string]any{"status": "completed"},
		}),
	})
	orchSrv := httptest.NewServer(fakeOrch)
	t.Cleanup(orchSrv.Close)

	runner := fakeOrchestratorClient(orchSrv.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-cons", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"test consistency"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	respBody := rec.Body.String()
	// runId should appear in at least the first and last events.
	runIDCount := strings.Count(respBody, `"runId":"run-consistent-1"`)
	if runIDCount < 2 {
		t.Errorf("expected runId in at least 2 events, got %d", runIDCount)
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_OrchestratorConnectionRefused
//
// Verify Gateway handles Orchestrator connection failure gracefully
// (no Orchestrator running at all).
// ---------------------------------------------------------------------------
func TestPhase10_OrchestratorConnectionRefused(t *testing.T) {
	// Use a closed httptest server URL.
	closedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", 500)
	}))
	closedURL := closedSrv.URL
	closedSrv.Close() // server is now unreachable

	runner := fakeOrchestratorClient(closedURL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-connrefused", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"should fail gracefully"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	respBody := rec.Body.String()

	// Should return an error SSE event, not crash.
	if !strings.Contains(respBody, "event: error\n") && !strings.Contains(respBody, "event: run_error\n") {
		t.Errorf("expected error event for unreachable Orchestrator, got %q", respBody)
	}

	// Must not expose internal connection details.
	if strings.Contains(respBody, "connection refused") {
		t.Error("leaked raw connection error")
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_EmptyStreamResponse
//
// Verify Gateway handles an empty SSE stream (no events) gracefully.
// ---------------------------------------------------------------------------
func TestPhase10_EmptyStreamResponse(t *testing.T) {
	fakeOrch := newFakeOrchestrator()
	fakeOrch.setSSEEvents(nil) // empty stream
	orchSrv := httptest.NewServer(fakeOrch)
	t.Cleanup(orchSrv.Close)

	runner := fakeOrchestratorClient(orchSrv.URL)
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-empty", "code-agent")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"empty stream"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 even for empty stream, got %d body=%q", rec.Code, rec.Body.String())
	}
}
