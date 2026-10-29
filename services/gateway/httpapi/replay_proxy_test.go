package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type stubReplayService struct {
	status string
	events string
	after  int64
}

func (s *stubReplayService) GetRunStatus(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(s.status), nil
}

func (s *stubReplayService) ListRunEvents(_ context.Context, _ string, after int64, _ int) (json.RawMessage, error) {
	s.after = after
	return json.RawMessage(s.events), nil
}

func TestReplayProxyEndpoints(t *testing.T) {
	stub := &stubReplayService{
		status: `{"runId":"run-1","status":"completed","currentSeq":3}`,
		events: `{"runId":"run-1","events":[{"seq":3,"type":"run_finished"}]}`,
	}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithRunReplayService(stub))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs/run-1/status", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"completed"`) {
		t.Fatalf("unexpected status response: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs/run-1/events?after=2&limit=5", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"seq":3`) {
		t.Fatalf("unexpected events response: %d %s", rec.Code, rec.Body.String())
	}
	if stub.after != 2 {
		t.Fatalf("after cursor not forwarded: %d", stub.after)
	}
}

func TestReplayProxyUnavailable(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs/run-1/status", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
