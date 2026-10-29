package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func addJournalEvent(t *testing.T, s *journalStore, runID string, seq int64, typ string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"type": typ, "runId": runID, "seq": seq})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AppendPublicEvent(context.Background(), store.PublicEvent{
		RunID: runID, Seq: seq, Type: typ, Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEventStreamDeliversToTwoSubscribers(t *testing.T) {
	st := newJournalStore()
	runID := "run-live"
	addJournalEvent(t, st, runID, 1, "RUN_STARTED")
	addJournalEvent(t, st, runID, 2, "TEXT_MESSAGE_CONTENT")
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	bodies := make([]string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/runs/"+runID+"/events/stream?after=0", nil)
			ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
			defer cancel()
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req.WithContext(ctx))
			bodies[idx] = rec.Body.String()
		}(i)
	}
	time.Sleep(100 * time.Millisecond)
	addJournalEvent(t, st, runID, 3, "RUN_FINISHED")
	wg.Wait()

	for i, body := range bodies {
		if !strings.Contains(body, `"seq":1`) || !strings.Contains(body, `"seq":2`) || !strings.Contains(body, `RUN_FINISHED`) {
			t.Fatalf("subscriber %d did not receive full stream: %s", i, body)
		}
	}
}

func TestEventStreamHonorsAfterCursor(t *testing.T) {
	st := newJournalStore()
	runID := "run-live-cursor"
	addJournalEvent(t, st, runID, 1, "RUN_STARTED")
	addJournalEvent(t, st, runID, 2, "RUN_FINISHED")
	srv, _ := NewServer(st, &mockRunService{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runs/"+runID+"/events/stream?after=1", nil)
	srv.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, `"seq":1`) || !strings.Contains(body, `"seq":2`) {
		t.Fatalf("cursor filtering failed: %s", body)
	}
}
