package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type stubResumeService struct {
	status int
	body   string
	runID  string
}

func (s *stubResumeService) ResumeRun(_ context.Context, runID string) (*http.Response, error) {
	s.runID = runID
	return &http.Response{
		StatusCode: s.status,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

func TestResumeProxyStreamsSSE(t *testing.T) {
	stub := &stubResumeService{status: 200, body: "event: run_started\ndata: {\"type\":\"RUN_STARTED\"}\n\n"}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithRunResumeService(stub))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs/run-1/resume", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "RUN_STARTED") {
		t.Fatalf("resume proxy failed: %d %s", rec.Code, rec.Body.String())
	}
	if stub.runID != "run-1" {
		t.Fatalf("runId not forwarded: %s", stub.runID)
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type lost: %s", rec.Header().Get("Content-Type"))
	}
}

func TestResumeProxyUnavailable(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs/run-1/resume", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
