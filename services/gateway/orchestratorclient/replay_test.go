package orchestratorclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReplayMethods(t *testing.T) {
	var statusPath, eventsQuery, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		switch {
		case strings.HasSuffix(r.URL.Path, "/status"):
			statusPath = r.URL.Path
			_, _ = w.Write([]byte(`{"runId":"run-1","status":"completed","currentSeq":3}`))
		case strings.HasSuffix(r.URL.Path, "/events"):
			eventsQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"runId":"run-1","after":2,"events":[{"seq":3,"type":"run_finished"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc, err := NewOrchestratorRunService(srv.URL, "internal-token")
	if err != nil {
		t.Fatal(err)
	}
	status, err := svc.GetRunStatus(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var statusBody map[string]any
	if err := json.Unmarshal(status, &statusBody); err != nil || statusBody["status"] != "completed" {
		t.Fatalf("unexpected status body: %s", status)
	}
	if statusPath != "/internal/orchestrator/runs/run-1/status" {
		t.Fatalf("unexpected status path: %s", statusPath)
	}
	if auth != "Bearer internal-token" {
		t.Fatalf("missing internal auth: %q", auth)
	}

	events, err := svc.ListRunEvents(context.Background(), "run-1", 2, 5)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if !strings.Contains(string(events), `"seq":3`) {
		t.Fatalf("unexpected events body: %s", events)
	}
	if eventsQuery != "after=2&limit=5" {
		t.Fatalf("unexpected events query: %s", eventsQuery)
	}
}

func TestReplayMethodsRejectErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer srv.Close()
	svc, _ := NewOrchestratorRunService(srv.URL, "")
	if _, err := svc.GetRunStatus(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for 404")
	}
}
