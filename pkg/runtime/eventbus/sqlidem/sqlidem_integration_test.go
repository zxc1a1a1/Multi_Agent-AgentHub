package sqlidem

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
)

// R2-05 integration tests. They need a real MySQL; the table DDL mirrors
// migration 008 so the package test has no cross-module import.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, ddl := range []string{
		`CREATE TABLE IF NOT EXISTS event_consumer_idempotency (
			consumer_key VARCHAR(128) NOT NULL, event_id VARCHAR(160) NOT NULL,
			event_hash CHAR(64) NOT NULL, status VARCHAR(16) NOT NULL,
			result_ref VARCHAR(255) NOT NULL DEFAULT '', attempts INT NOT NULL DEFAULT 1,
			created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (consumer_key, event_id), KEY idx_event_consumer_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS r205_effects (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			consumer_key VARCHAR(128) NOT NULL,
			event_id VARCHAR(160) NOT NULL,
			payload VARCHAR(128) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	if _, err := db.Exec("DELETE FROM r205_effects"); err != nil {
		t.Fatalf("clean effects: %v", err)
	}
	if _, err := db.Exec("DELETE FROM event_consumer_idempotency"); err != nil {
		t.Fatalf("clean idempotency: %v", err)
	}
	return db
}

func effectCount(t *testing.T, db *sql.DB, eventID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM r205_effects WHERE event_id = ?", eventID).Scan(&n); err != nil {
		t.Fatalf("count effects: %v", err)
	}
	return n
}

func processEffect(store *Store, db *sql.DB, consumerKey, eventID, hash string) (Result, error) {
	return store.Process(context.Background(), consumerKey, eventID, hash,
		func(ctx context.Context, tx *sql.Tx) (string, error) {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO r205_effects (consumer_key, event_id, payload) VALUES (?, ?, ?)",
				consumerKey, eventID, "ok"); err != nil {
				return "", err
			}
			return "result-" + eventID, nil
		})
}

func TestIntegrationIdempotentProcessHasOneEffect(t *testing.T) {
	db := openTestDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	eventID := fmt.Sprintf("evt_run_r205_once_%d_1_0", time.Now().UnixNano())

	first, err := processEffect(store, db, "assistant_message.v1", eventID, "hash-a")
	if err != nil || first.Duplicate {
		t.Fatalf("first process duplicate=%v err=%v", first.Duplicate, err)
	}
	second, err := processEffect(store, db, "assistant_message.v1", eventID, "hash-a")
	if err != nil || !second.Duplicate {
		t.Fatalf("second process duplicate=%v err=%v, want duplicate", second.Duplicate, err)
	}
	if second.ResultRef != first.ResultRef {
		t.Fatalf("duplicate resultRef=%q, want %q", second.ResultRef, first.ResultRef)
	}
	if got := effectCount(t, db, eventID); got != 1 {
		t.Fatalf("business effects=%d, want exactly 1", got)
	}
}

func TestIntegrationConcurrentDuplicateHasOneEffect(t *testing.T) {
	db := openTestDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	eventID := fmt.Sprintf("evt_run_r205_concurrent_%d_1_0", time.Now().UnixNano())
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan Result, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := processEffect(store, db, "projection.v1", eventID, "hash-b")
			results <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent process: %v", err)
		}
	}
	duplicates := 0
	for result := range results {
		if result.Duplicate {
			duplicates++
		}
	}
	if duplicates != 1 {
		t.Fatalf("duplicates=%d, want exactly 1", duplicates)
	}
	if got := effectCount(t, db, eventID); got != 1 {
		t.Fatalf("business effects=%d, want exactly 1", got)
	}
}

func TestIntegrationFailedHandlerRollsBackAndRetries(t *testing.T) {
	db := openTestDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	eventID := fmt.Sprintf("evt_run_r205_retry_%d_1_0", time.Now().UnixNano())
	boom := errors.New("business handler failed")
	_, err = store.Process(context.Background(), "audit.v1", eventID, "hash-c",
		func(ctx context.Context, tx *sql.Tx) (string, error) {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO r205_effects (consumer_key, event_id) VALUES (?, ?)", "audit.v1", eventID); err != nil {
				return "", err
			}
			return "", boom
		})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v, want handler error", err)
	}
	if got := effectCount(t, db, eventID); got != 0 {
		t.Fatalf("failed handler must roll back its effect, got %d rows", got)
	}
	result, err := processEffect(store, db, "audit.v1", eventID, "hash-c")
	if err != nil || result.Duplicate {
		t.Fatalf("retry duplicate=%v err=%v", result.Duplicate, err)
	}
	if got := effectCount(t, db, eventID); got != 1 {
		t.Fatalf("retry effects=%d, want 1", got)
	}
}

func TestIntegrationSameEventDifferentPayloadIsConflict(t *testing.T) {
	db := openTestDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	eventID := fmt.Sprintf("evt_run_r205_conflict_%d_1_0", time.Now().UnixNano())
	if _, err := processEffect(store, db, "signal.v1", eventID, "hash-d"); err != nil {
		t.Fatal(err)
	}
	_, err = processEffect(store, db, "signal.v1", eventID, "hash-e")
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("err=%v, want ErrIdempotencyConflict", err)
	}
	if got := effectCount(t, db, eventID); got != 1 {
		t.Fatalf("conflict must not add effects, got %d", got)
	}
}

func TestIntegrationWrapHandlesDuplicateEventBusMessage(t *testing.T) {
	db := openTestDB(t)
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := eventbus.NewEnvelope(eventbus.EnvelopeParams{
		Type: "state_update", RunID: "run_r205_wrap", ConversationID: "conv_r205",
		Seq: 1, OccurredAt: time.Now().UTC(), Payload: map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := &eventbus.Message{Envelope: envelope, TransportID: "1-0", Attempt: 1}
	handler := store.Wrap("assistant_message.v1",
		func(ctx context.Context, _ *eventbus.Message, tx *sql.Tx) (string, error) {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO r205_effects (consumer_key, event_id, payload) VALUES (?, ?, ?)",
				"assistant_message.v1", envelope.EventID, "wrapped")
			return "wrapped-result", err
		})
	for i := 0; i < 2; i++ {
		if err := handler(context.Background(), msg); err != nil {
			t.Fatalf("handler pass %d: %v", i, err)
		}
	}
	if got := effectCount(t, db, envelope.EventID); got != 1 {
		t.Fatalf("wrapped effects=%d, want 1", got)
	}
}
