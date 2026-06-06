package dispatcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestDispatch_RetriesOn5xxThenSucceeds verifies transient 5xx failures are retried.
func TestDispatch_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":{"code":"internal","message":"temporary"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"taskId":"t","status":"completed","events":[{"author":"a","parts":[{"type":"text","text":"ok"}]}]}`)
	}))
	defer srv.Close()

	d := NewA2ADispatcher(WithRetry(3, 1*time.Millisecond))
	res, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "a",
		ConversationID: "c1",
		Message:        "hi",
	})
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if res == nil || res.Text != "ok" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

// TestDispatch_NoRetryOn4xx verifies client errors are not retried.
func TestDispatch_NoRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":"bad_request","message":"nope"}}`)
	}))
	defer srv.Close()

	d := NewA2ADispatcher(WithRetry(3, 1*time.Millisecond))
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "a",
		ConversationID: "c1",
		Message:        "hi",
	})
	if err == nil {
		t.Fatal("expected error for 4xx")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 attempt (no retry on 4xx), got %d", got)
	}
}

// TestDispatch_RetryExhausted verifies retries are capped by maxAttempts.
func TestDispatch_RetryExhausted(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"code":"internal","message":"down"}}`)
	}))
	defer srv.Close()

	d := NewA2ADispatcher(WithRetry(2, 1*time.Millisecond))
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "a",
		ConversationID: "c1",
		Message:        "hi",
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected exactly 2 attempts, got %d", got)
	}
}

// TestDispatch_PerCallTimeout verifies a slow agent is cut off by the timeout.
func TestDispatch_PerCallTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"events":[]}`)
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	d := NewA2ADispatcher(WithPerCallTimeout(20 * time.Millisecond))
	start := time.Now()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "a",
		ConversationID: "c1",
		Message:        "hi",
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Fatalf("timeout not enforced; took %v", elapsed)
	}
}

// TestDispatch_CircuitBreakerOpens verifies the breaker short-circuits after
// consecutive failures without making further HTTP calls.
func TestDispatch_CircuitBreakerOpens(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"code":"internal","message":"down"}}`)
	}))
	defer srv.Close()

	d := NewA2ADispatcher(WithCircuitBreaker(2, time.Minute))
	in := DispatchInput{AgentURL: srv.URL, AgentName: "a", ConversationID: "c1", Message: "hi"}

	// First 2 calls hit the server and fail, opening the breaker.
	for i := 0; i < 2; i++ {
		if _, err := d.Dispatch(context.Background(), in); err == nil {
			t.Fatalf("call %d expected error", i)
		}
	}
	callsAfterOpen := atomic.LoadInt32(&calls)

	// Third call should short-circuit (no new HTTP call).
	if _, err := d.Dispatch(context.Background(), in); err == nil {
		t.Fatal("expected breaker-open error")
	}
	if got := atomic.LoadInt32(&calls); got != callsAfterOpen {
		t.Fatalf("breaker did not short-circuit: calls went from %d to %d", callsAfterOpen, got)
	}
}

// TestDispatch_DefaultNoRetry verifies the zero-config dispatcher makes exactly
// one attempt (deterministic, no behavior change for CI).
func TestDispatch_DefaultNoRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"code":"internal","message":"down"}}`)
	}))
	defer srv.Close()

	d := NewA2ADispatcher()
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		AgentURL: srv.URL, AgentName: "a", ConversationID: "c1", Message: "hi",
	})
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("default dispatcher should make exactly 1 attempt, got %d", got)
	}
}
