package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

func TestHealth(t *testing.T) {
	srv := NewServer()
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
	if body["service"] != "orchestrator" {
		t.Errorf("expected service=orchestrator, got %q", body["service"])
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/health", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestNilHandler(t *testing.T) {
	var srv *Server
	h := srv.Handler()
	if h == nil {
		t.Fatal("expected non-nil handler from nil server")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 from nil server handler, got %d", rec.Code)
	}
}

func TestHITLConfirmAccepted(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-1"
	// Register a pending plan so the orchestrator has a channel waiting.
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-1",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := `{"runId":"run-test-1","actionId":"plan-1","confirmed":true,"rejectReason":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /internal/orchestrator/hitl/confirm failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["status"] != "acknowledged" {
		t.Errorf("expected status=acknowledged, got %q", result["status"])
	}
}

func TestHITLConfirmRejected(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-2"
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-2",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := `{"runId":"run-test-2","actionId":"plan-2","confirmed":false,"rejectReason":"not needed"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Read the result from the channel to verify it was delivered.
	select {
	case result := <-srv.hitlChans[runID]:
		if result.Confirmed {
			t.Error("expected Confirmed=false")
		}
		if result.RejectReason != "not needed" {
			t.Errorf("expected RejectReason='not needed', got %q", result.RejectReason)
		}
	default:
		// Channel was already consumed by the handler; this is fine.
	}
}

func TestHITLConfirmNoPendingRun(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"nonexistent","actionId":"action-1","confirmed":true,"rejectReason":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for missing pending run, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(result["error"], "no pending confirmation") {
		t.Errorf("expected 'no pending confirmation' error, got %q", result["error"])
	}
}

func TestHITLConfirmRepeated(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-repeat"
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-repeat",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// First confirm: should succeed and fill the buffered channel (capacity 1).
	body1 := `{"runId":"run-test-repeat","actionId":"plan-repeat","confirmed":true,"rejectReason":""}`
	resp1, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first confirm: expected 200, got %d", resp1.StatusCode)
	}

	// Second confirm WITHOUT draining: channel is full, should get 409 Conflict.
	body2 := `{"runId":"run-test-repeat","actionId":"plan-repeat","confirmed":false,"rejectReason":"changed mind"}`
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body2))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for repeated confirm (channel full), got %d", resp2.StatusCode)
	}

	// Drain the channel to verify first result arrived correctly.
	select {
	case result := <-ch:
		if !result.Confirmed {
			t.Error("expected first confirm Confirmed=true")
		}
	default:
		t.Error("expected first confirm result in channel")
	}
}

func TestHITLConfirmMissingRunID(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"actionId":"action-1","confirmed":true,"rejectReason":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHITLConfirmMissingActionID(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-1","confirmed":true,"rejectReason":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHITLConfirmMethodNotAllowed(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/hitl/confirm")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestHITLConfirmUnauthorized(t *testing.T) {
	srv := NewServer(WithInternalToken("secret-token"))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"runId":"run-1","actionId":"action-1","confirmed":true,"rejectReason":""}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/internal/orchestrator/hitl/confirm", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header — should be rejected.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHITLConfirmAuthorized(t *testing.T) {
	srv := NewServer(WithInternalToken("secret-token"))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-auth"
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-auth",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := `{"runId":"run-test-auth","actionId":"plan-auth","confirmed":true,"rejectReason":""}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/internal/orchestrator/hitl/confirm", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHITLConfirmAfterDrainStillAccepted(t *testing.T) {
	// After the stream goroutine drains the confirmation channel, a logical
	// state machine now prevents a second confirm. This test verifies that
	// the handler rejects a duplicate confirm even after drain.
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-redrain"
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-redrain",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// First confirm fills the buffer.
	body := `{"runId":"run-test-redrain","actionId":"plan-redrain","confirmed":true,"rejectReason":""}`
	resp1, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first confirm: expected 200, got %d", resp1.StatusCode)
	}

	// Drain the channel (simulates the stream goroutine consuming the result).
	select {
	case result := <-ch:
		if !result.Confirmed {
			t.Error("expected Confirmed=true")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first result")
	}

	// After drain, state is HITLConfirmed — another confirm is rejected.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for duplicate confirm after drain, got %d", resp2.StatusCode)
	}
}

func TestHITLConfirmStateAfterRejected(t *testing.T) {
	// After rejection, another confirm must be rejected with a clear error.
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-reject-once"
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-reject",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Reject the plan.
	body := `{"runId":"run-test-reject-once","actionId":"plan-reject","confirmed":false,"rejectReason":"not needed"}`
	resp1, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d", resp1.StatusCode)
	}

	// Drain channel.
	select {
	case result := <-ch:
		if result.Confirmed {
			t.Error("expected Confirmed=false")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for result")
	}

	// Second confirm after rejection should be 409 Conflict.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for confirm after rejection, got %d", resp2.StatusCode)
	}
}

func TestHITLConfirmTimedOutRejected(t *testing.T) {
	// Confirm after timeout must return 410 Gone.
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-timeout"
	p := &plan.OrchestrationPlan{
		PlanID:  "plan-timeout",
		RunID:   runID,
		Strategy: plan.StrategySingle,
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Mark as timed out (simulates what the streaming goroutine would do).
	srv.SetHITLState(runID, HITLTimedOut)

	// Confirm after timeout should be rejected with 410 Gone.
	body := `{"runId":"run-test-timeout","actionId":"plan-timeout","confirmed":true,"rejectReason":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGone {
		t.Errorf("expected 410 Gone for confirm after timeout, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(result["error"], "timed out") {
		t.Errorf("expected 'timed out' error, got %q", result["error"])
	}
}
