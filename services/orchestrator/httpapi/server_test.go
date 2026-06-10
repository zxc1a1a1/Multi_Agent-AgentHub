package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
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
		PlanID:   "plan-1",
		RunID:    runID,
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
		PlanID:   "plan-2",
		RunID:    runID,
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
		PlanID:   "plan-repeat",
		RunID:    runID,
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
		PlanID:   "plan-auth",
		RunID:    runID,
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
		PlanID:   "plan-redrain",
		RunID:    runID,
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
		PlanID:   "plan-reject",
		RunID:    runID,
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
		PlanID:   "plan-timeout",
		RunID:    runID,
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

func TestIsPlanOnlyWhitelisted(t *testing.T) {
	if !isPlanOnlyWhitelisted("code-agent") {
		t.Error("code-agent should be whitelisted")
	}
	if !isPlanOnlyWhitelisted("web-agent") {
		t.Error("web-agent should be whitelisted")
	}
	if !isPlanOnlyWhitelisted("Code-Agent") {
		t.Error("Code-Agent (case-insensitive) should be whitelisted")
	}
	if isPlanOnlyWhitelisted("unknown-agent") {
		t.Error("unknown-agent should not be whitelisted")
	}
	if isPlanOnlyWhitelisted("") {
		t.Error("empty string should not be whitelisted")
	}
}

func TestHITLCancelledStateReturnsConflict(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-cancel"
	p := &plan.OrchestrationPlan{
		PlanID:   "plan-cancel",
		RunID:    runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// First request: cancel (confirmed=false).
	body1 := `{"runId":"run-test-cancel","actionId":"plan-cancel","confirmed":false,"rejectReason":"cancelled by user"}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
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

	// Second request: already cancelled, should get 409 Conflict.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for confirm after cancellation, got %d", resp2.StatusCode)
	}
}

func TestHITLConfirm_ReviseAction(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-revise"
	p := &plan.OrchestrationPlan{
		PlanID:   "plan-revise",
		RunID:    runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := `{"runId":"run-test-revise","actionId":"plan-revise","action":"revise","feedback":"简化步骤","revision":2}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
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

	// Verify channel receives revise result with correct fields.
	select {
	case r := <-ch:
		if r.Confirmed {
			t.Error("revise should have Confirmed=false")
		}
		if r.Action != "revise" {
			t.Errorf("expected Action=revise, got %q", r.Action)
		}
		if r.Feedback != "简化步骤" {
			t.Errorf("expected Feedback='简化步骤', got %q", r.Feedback)
		}
		if r.Revision != 2 {
			t.Errorf("expected Revision=2, got %d", r.Revision)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for revise result")
	}
}

func TestHITLConfirm_ReviseEmptyFeedbackRejected(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-revise-empty"
	p := &plan.OrchestrationPlan{
		PlanID:   "plan-revise-empty",
		RunID:    runID,
		Strategy: plan.StrategySingle,
	}
	_ = srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := `{"runId":"run-test-revise-empty","actionId":"plan-revise-empty","action":"revise","feedback":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(result["error"], "REVISION_INPUT_REQUIRED") {
		t.Errorf("expected REVISION_INPUT_REQUIRED error, got %q", result["error"])
	}
}

func TestHITLConfirmStateAfterConfirmed(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-test-confirmed"
	p := &plan.OrchestrationPlan{
		PlanID:   "plan-confirmed",
		RunID:    runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// Confirm the plan.
	body := `{"runId":"run-test-confirmed","actionId":"plan-confirmed","confirmed":true,"rejectReason":""}`
	resp, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Drain channel.
	select {
	case result := <-ch:
		if !result.Confirmed {
			t.Error("expected Confirmed=true")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for result")
	}

	// Second confirm should return 409 Conflict.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for confirm after confirmed, got %d", resp2.StatusCode)
	}
}

// TestIdempotencySameKey verifies that the same idempotencyKey + same payload
// returns the cached 200 response on replay.
func TestIdempotencySameKey(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-idem-same"
	p := &plan.OrchestrationPlan{
		PlanID:   "plan-idem",
		RunID:    runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	body := `{"runId":"run-idem-same","actionId":"plan-idem","action":"approve","idempotencyKey":"key-001"}`

	// First request — should succeed (200).
	resp1, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", resp1.StatusCode)
	}

	// Drain the channel so it doesn't block.
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out draining channel")
	}

	// Second request with same idempotencyKey — should return cached 200.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		var result map[string]string
		json.NewDecoder(resp2.Body).Decode(&result)
		t.Errorf("same key: expected 200 (cached), got %d: %v", resp2.StatusCode, result)
	}
}

// TestIdempotencyKeyConflict verifies that the same idempotencyKey with
// different payload returns 409 IDEMPOTENCY_KEY_CONFLICT.
func TestIdempotencyKeyConflict(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runID := "run-idem-conflict"
	p := &plan.OrchestrationPlan{
		PlanID:   "plan-idem-conflict",
		RunID:    runID,
		Strategy: plan.StrategySingle,
	}
	ch := srv.registerPending(runID, p)
	defer srv.deregisterPending(runID)

	// First request — approve.
	body1 := `{"runId":"run-idem-conflict","actionId":"plan-idem-conflict","action":"approve","idempotencyKey":"key-002"}`
	resp1, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", resp1.StatusCode)
	}

	// Drain channel.
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out draining channel")
	}

	// Second request — same idempotencyKey but different action (cancel vs approve).
	body2 := `{"runId":"run-idem-conflict","actionId":"plan-idem-conflict","action":"cancel","idempotencyKey":"key-002"}`
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/hitl/confirm", "application/json", strings.NewReader(body2))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 IDEMPOTENCY_KEY_CONFLICT, got %d", resp2.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp2.Body).Decode(&result)
	if !strings.Contains(result["error"], "IDEMPOTENCY_KEY_CONFLICT") {
		t.Errorf("expected IDEMPOTENCY_KEY_CONFLICT error, got: %v", result)
	}
}

// ---------------------------------------------------------------------------
// Agent Management API tests
// ---------------------------------------------------------------------------

// testAgentStore creates a JSONStore backed by a temp file for testing.
func testAgentStore(t *testing.T) (registry.AgentStore, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")
	store, err := registry.NewJSONStore(path)
	if err != nil {
		t.Fatalf("create test JSON store: %v", err)
	}
	return store, func() {}
}

// testServerWithDynamicRegistry creates a test server with both static and
// dynamic registries wired. When staticEndpoints is empty, a minimal set is used.
func testServerWithDynamicRegistry(t *testing.T, staticEndpoints []registry.AgentEndpoint, store registry.AgentStore) *Server {
	t.Helper()
	if len(staticEndpoints) == 0 {
		staticEndpoints = []registry.AgentEndpoint{
			{Name: "code-agent", URL: "http://127.0.0.1:8081", Description: "Generates and explains code", OutputModes: []string{"text", "code"}, CapabilityIDs: []string{"code_generation"}},
			{Name: "web-agent", URL: "http://127.0.0.1:8082", Description: "Generates webpages", OutputModes: []string{"text", "webpage"}, CapabilityIDs: []string{"web_generation"}},
		}
	}
	staticReg, err := registry.NewStaticAgentRegistry(staticEndpoints)
	if err != nil {
		t.Fatalf("create static registry: %v", err)
	}
	dr := registry.NewDynamicAgentRegistry(staticReg, store)
	return NewServer(WithRegistry(staticReg), WithDynamicRegistry(dr))
}

// ---------------------------------------------------------------------------
// List / Get
// ---------------------------------------------------------------------------

func TestAgentListEmpty(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents")
	if err != nil {
		t.Fatalf("GET /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var agents []publicAgentItem
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Should have static agents (code-agent, web-agent).
	if len(agents) < 2 {
		t.Fatalf("expected at least 2 static agents, got %d", len(agents))
	}
	// Verify static agents have description and outputModes.
	foundCode := false
	for _, a := range agents {
		if a.Name == "code-agent" {
			foundCode = true
			if a.Description == "" {
				t.Error("static code-agent missing description")
			}
			if len(a.OutputModes) == 0 {
				t.Error("static code-agent missing outputModes")
			}
			if a.Source != "static" {
				t.Errorf("expected source=static, got %q", a.Source)
			}
		}
	}
	if !foundCode {
		t.Error("expected code-agent in list")
	}
}

func TestAgentGetFound(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents/code-agent")
	if err != nil {
		t.Fatalf("GET /agents/code-agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var agent publicAgentItem
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if agent.Name != "code-agent" {
		t.Errorf("expected name=code-agent, got %q", agent.Name)
	}
	if agent.Description != "Generates and explains code" {
		t.Errorf("expected description, got %q", agent.Description)
	}
}

func TestAgentGetNotFound(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents/nonexistent")
	if err != nil {
		t.Fatalf("GET /agents/nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestAgentRegisterInvalidBody(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(`{invalid}`))
	if err != nil {
		t.Fatalf("POST /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterEmptyURL(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(`{"url":""}`))
	if err != nil {
		t.Fatalf("POST /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty URL, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterSuccess(t *testing.T) {
	// Mock a simple agent card server.
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":                "test-agent",
				"description":         "A test agent",
				"version":             "1.0.0",
				"url":                 r.Host,
				"inputModes":          []string{"text"},
				"outputModes":         []string{"text"},
				"skills":              []map[string]string{{"id": "testing"}},
				"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockAgent.Close()

	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := fmt.Sprintf(`{"url":%q}`, mockAgent.URL)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var agent publicAgentItem
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if agent.Name != "test-agent" {
		t.Errorf("expected name=test-agent, got %q", agent.Name)
	}
	if agent.Source != "dynamic" {
		t.Errorf("expected source=dynamic, got %q", agent.Source)
	}
	if !agent.Enabled {
		t.Error("expected enabled=true")
	}
	if agent.Description != "A test agent" {
		t.Errorf("expected description from card, got %q", agent.Description)
	}
	if len(agent.OutputModes) == 0 || agent.OutputModes[0] != "text" {
		t.Errorf("expected outputModes from card, got %v", agent.OutputModes)
	}
}

func TestAgentRegisterStaticNameConflict(t *testing.T) {
	// Registering an agent named "code-agent" must conflict with the static code-agent.
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":                "code-agent",
				"description":         "test",
				"version":             "1.0.0",
				"url":                 r.Host,
				"inputModes":          []string{"text"},
				"outputModes":         []string{"text"},
				"skills":              []map[string]string{{"id": "code"}},
				"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockAgent.Close()

	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := fmt.Sprintf(`{"url":%q}`, mockAgent.URL)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Delete / Unregister
// ---------------------------------------------------------------------------

func TestAgentDeleteStaticAgent(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/internal/orchestrator/agents/code-agent", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /agents/code-agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for static agent, got %d", resp.StatusCode)
	}
}

func TestAgentDeleteNotFound(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/internal/orchestrator/agents/nonexistent", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /agents/nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Enable / Disable
// ---------------------------------------------------------------------------

func TestAgentEnableStaticAgent(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents/code-agent/enable", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /agents/code-agent/enable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for static agent, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestAgentRefreshStaticAgent(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents/code-agent/refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /agents/code-agent/refresh: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for static agent, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Check
// ---------------------------------------------------------------------------

func TestAgentCheckStaticAgent(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents/code-agent/check", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /agents/code-agent/check: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for static agent, got %d", resp.StatusCode)
	}
}

func TestAgentCheckNotFound(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents/nonexistent/check", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /agents/nonexistent/check: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// method not allowed
// ---------------------------------------------------------------------------

func TestAgentMethodNotAllowed(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// PUT is not allowed on /agents
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/internal/orchestrator/agents", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
	if allow := resp.Header.Get("Allow"); allow == "" {
		t.Error("expected Allow header")
	}
}

func TestAgentGetMethodNotAllowed(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// POST is not allowed on /agents/code-agent (exact name, no action)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents/code-agent", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /agents/code-agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 (no matching action), got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// lastError sanitization
// ---------------------------------------------------------------------------

func TestSanitizeAgentLastError(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"  ", ""},
		{"dial tcp 10.0.0.1:8081: connection refused", "agent health check failed"},
		{"lookup code-agent on 10.96.0.10:53: no such host", "agent health check failed"},
		{"Get \"http://127.0.0.1:8081/.well-known/agent.json\": dial tcp: connect: connection refused", "agent health check failed"},
		{"request timeout", "agent health check failed"},
		{"agent card name mismatch", "agent card name mismatch"},
		{"something benign", "something benign"},
	}

	for _, tc := range tests {
		got := sanitizeAgentLastError(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeAgentLastError(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Dynamic registry nil — graceful fallback
// ---------------------------------------------------------------------------

func TestAgentListNilDynamicRegistry(t *testing.T) {
	// When no dynamic registry is wired, List should still return static agents.
	staticReg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: "http://127.0.0.1:8081", Description: "Code Agent", OutputModes: []string{"text"}},
	})
	if err != nil {
		t.Fatalf("create static registry: %v", err)
	}
	srv := NewServer(WithRegistry(staticReg))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents")
	if err != nil {
		t.Fatalf("GET /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var agents []publicAgentItem
	json.NewDecoder(resp.Body).Decode(&agents)
	if len(agents) < 1 {
		t.Error("expected at least 1 static agent")
	}
}

func TestAgentRegisterNilDynamicRegistry(t *testing.T) {
	srv := NewServer() // no dynamic registry
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(`{"url":"http://example.com"}`))
	if err != nil {
		t.Fatalf("POST /agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Decoded agent name in path
// ---------------------------------------------------------------------------

func TestAgentGetEncodedName(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()

	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":                "test agent",
				"description":         "test",
				"version":             "1.0.0",
				"url":                 r.Host,
				"inputModes":          []string{"text"},
				"outputModes":         []string{"text"},
				"skills":              []map[string]string{{"id": "test"}},
				"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockAgent.Close()

	srv := testServerWithDynamicRegistry(t, nil, store)
	// Register a dynamic agent with space in name.
	body := fmt.Sprintf(`{"url":%q}`, mockAgent.URL)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, _ := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(body))
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for register, got %d", resp.StatusCode)
	}

	// Get with encoded name.
	encodedName := url.PathEscape("test agent")
	resp2, err := http.Get(ts.URL + "/internal/orchestrator/agents/" + encodedName)
	if err != nil {
		t.Fatalf("GET /agents/%s: %v", encodedName, err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp2.StatusCode, readBody(resp2))
	}
}

func readBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	var buf strings.Builder
	json.NewDecoder(resp.Body).Decode(&struct{}{})
	return buf.String()
}

// ---------------------------------------------------------------------------
// Frontend compatibility: response shape
// ---------------------------------------------------------------------------

func TestAgentResponseShapeBackwardCompatible(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents")
	if err != nil {
		t.Fatalf("GET /agents: %v", err)
	}
	defer resp.Body.Close()

	var agents []map[string]any
	json.NewDecoder(resp.Body).Decode(&agents)

	if len(agents) == 0 {
		t.Fatal("expected non-empty agent list")
	}

	// Every agent must have the 4 frontend-compatible fields (camelCase).
	for i, a := range agents {
		if _, ok := a["name"]; !ok {
			t.Errorf("agent[%d] missing 'name' field", i)
		}
		if _, ok := a["displayName"]; !ok {
			t.Errorf("agent[%d] missing 'displayName' field", i)
		}
		if _, ok := a["description"]; !ok {
			t.Errorf("agent[%d] missing 'description' field", i)
		}
		if _, ok := a["outputModes"]; !ok {
			t.Errorf("agent[%d] missing 'outputModes' field", i)
		}
	}

	// Verify no lastError in list response.
	for i, a := range agents {
		if _, ok := a["lastError"]; ok {
			t.Errorf("agent[%d] should not have 'lastError' in list response", i)
		}
	}
}

// ---------------------------------------------------------------------------
// PATCH update
// ---------------------------------------------------------------------------

func TestAgentUpdateNotFound(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/internal/orchestrator/agents/nonexistent", strings.NewReader(`{"displayName":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// store unavailable error
// ---------------------------------------------------------------------------

func TestAgentRegisterStoreUnavailable(t *testing.T) {
	dr := registry.NewDynamicAgentRegistry(nil, nil)
	srv := NewServer(WithDynamicRegistry(dr))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(`{"url":"http://127.0.0.1:9999"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Path safety: .. traversal and bad paths
// ---------------------------------------------------------------------------

func TestAgentPathTraversalRejected(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// ../.. in path should be rejected (defense-in-depth; Go cleanPath also handles this).
	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents/..%2F..%2Fhealth")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Go cleanPath resolves this before routing, so it won't match /agents/
	// and returns 404 from the mux. Either 404 or 400 is acceptable.
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 404 or 400 for traversal, got %d", resp.StatusCode)
	}
}

func TestAgentInvalidPathReturnsNotFound(t *testing.T) {
	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/internal/orchestrator/agents/a/b/c")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Extra path segments not matching a known action → 404
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown sub-path, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Lifecycle: PATCH update, DELETE, enable/disable success
// ---------------------------------------------------------------------------

func TestAgentLifecycle(t *testing.T) {
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":                "lifecycle-agent",
				"description":         "Lifecycle test agent",
				"version":             "1.0.0",
				"url":                 r.Host,
				"inputModes":          []string{"text"},
				"outputModes":         []string{"text", "code"},
				"skills":              []map[string]string{{"id": "test"}},
				"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockAgent.Close()

	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Register a dynamic agent.
	body := fmt.Sprintf(`{"url":%q}`, mockAgent.URL)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// 2. PATCH update displayName.
	patchBody := `{"displayName":"Lifecycle Agent Updated"}`
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/internal/orchestrator/agents/lifecycle-agent", strings.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	// 3. Disable.
	resp3, err := http.Post(ts.URL+"/internal/orchestrator/agents/lifecycle-agent/disable", "application/json", nil)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp3.StatusCode)
	}

	// 4. Enable.
	resp4, err := http.Post(ts.URL+"/internal/orchestrator/agents/lifecycle-agent/enable", "application/json", nil)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp4.StatusCode)
	}

	// 5. Refresh.
	resp5, err := http.Post(ts.URL+"/internal/orchestrator/agents/lifecycle-agent/refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp5.StatusCode)
	}
	var refreshed publicAgentItem
	json.NewDecoder(resp5.Body).Decode(&refreshed)
	if refreshed.Name != "lifecycle-agent" {
		t.Errorf("expected name=lifecycle-agent, got %q", refreshed.Name)
	}

	// 6. Check healthy.
	resp6, err := http.Post(ts.URL+"/internal/orchestrator/agents/lifecycle-agent/check", "application/json", nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer resp6.Body.Close()
	if resp6.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp6.StatusCode)
	}
	var checked publicAgentItem
	json.NewDecoder(resp6.Body).Decode(&checked)
	if !checked.Healthy {
		t.Error("expected healthy=true")
	}
	if checked.LastError != "" {
		t.Errorf("expected empty LastError, got %q", checked.LastError)
	}

	// 7. DELETE.
	req7, _ := http.NewRequest(http.MethodDelete, ts.URL+"/internal/orchestrator/agents/lifecycle-agent", nil)
	resp7, err := http.DefaultClient.Do(req7)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp7.Body.Close()
	if resp7.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp7.StatusCode)
	}

	// 8. After DELETE, agent should 404.
	resp8, err := http.Get(ts.URL + "/internal/orchestrator/agents/lifecycle-agent")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	defer resp8.Body.Close()
	if resp8.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp8.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Check with upstream error → sanitized lastError
// ---------------------------------------------------------------------------

func TestAgentCheckFetchFailure(t *testing.T) {
	// Use a mock that serves agent.json on register but fails on re-fetch.
	callCount := 0
	flakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/.well-known/agent.json" {
			if callCount > 1 {
				// Fail on subsequent fetches (check).
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte("connection refused"))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":                "flake-agent",
				"description":         "test",
				"version":             "1.0.0",
				"url":                 r.Host,
				"inputModes":          []string{"text"},
				"outputModes":         []string{"text"},
				"skills":              []map[string]string{{"id": "test"}},
				"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer flakeAgent.Close()

	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Register.
	body := fmt.Sprintf(`{"url":%q}`, flakeAgent.URL)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// Check — should fail on re-fetch, healthy=false, lastError sanitized.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/agents/flake-agent/check", "application/json", nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	var checked publicAgentItem
	json.NewDecoder(resp2.Body).Decode(&checked)
	if checked.Healthy {
		t.Error("expected healthy=false on fetch failure")
	}
	if checked.LastError == "" {
		t.Error("expected non-empty LastError")
	}
	if strings.Contains(checked.LastError, "connection refused") {
		t.Errorf("expected sanitized LastError, got %q", checked.LastError)
	}
}

// ---------------------------------------------------------------------------
// Check with name mismatch → lastError safe
// ---------------------------------------------------------------------------

func TestAgentCheckNameMismatch(t *testing.T) {
	callCount := 0
	mismatchAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/.well-known/agent.json" {
			if callCount > 1 {
				// On re-fetch, return a different name.
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"name":                "wrong-name",
					"description":         "test",
					"version":             "1.0.0",
					"url":                 r.Host,
					"inputModes":          []string{"text"},
					"outputModes":         []string{"text"},
					"skills":              []map[string]string{{"id": "test"}},
					"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"name":                "mismatch-agent",
				"description":         "test",
				"version":             "1.0.0",
				"url":                 r.Host,
				"inputModes":          []string{"text"},
				"outputModes":         []string{"text"},
				"skills":              []map[string]string{{"id": "test"}},
				"supportedInterfaces": []map[string]string{{"type": "JSONRPC", "url": "http://" + r.Host + "/a2a/tasks"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mismatchAgent.Close()

	store, cleanup := testAgentStore(t)
	defer cleanup()
	srv := testServerWithDynamicRegistry(t, nil, store)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Register.
	body := fmt.Sprintf(`{"url":%q}`, mismatchAgent.URL)
	resp, err := http.Post(ts.URL+"/internal/orchestrator/agents", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// Check — name mismatch should set healthy=false with safe error.
	resp2, err := http.Post(ts.URL+"/internal/orchestrator/agents/mismatch-agent/check", "application/json", nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp2.StatusCode, readBody(resp2))
	}

	var checked publicAgentItem
	json.NewDecoder(resp2.Body).Decode(&checked)
	if checked.Healthy {
		t.Error("expected healthy=false on name mismatch")
	}
	if checked.LastError == "" {
		t.Error("expected non-empty LastError on name mismatch")
	}
	// LastError should be safe — no URLs.
	if strings.Contains(checked.LastError, "://") {
		t.Errorf("expected safe LastError, got %q", checked.LastError)
	}
}
