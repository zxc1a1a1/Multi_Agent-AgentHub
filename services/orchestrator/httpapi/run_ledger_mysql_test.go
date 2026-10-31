package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/relay"
)

func TestIntegrationRunLedgerLifecycleMySQL(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	runID := "run-ledger-" + suffix
	requestID := "req-ledger-" + suffix
	t.Cleanup(func() {
		for _, q := range []string{
			"DELETE FROM outbox WHERE run_id = ?",
			"DELETE FROM execution_events WHERE run_id = ?",
			"DELETE FROM checkpoints WHERE run_id = ?",
			"DELETE FROM plan_revisions WHERE run_id = ?",
			"DELETE FROM task_executions WHERE run_id = ?",
			"DELETE FROM executions WHERE run_id = ?",
		} {
			_, _ = db.Exec(q, runID)
		}
	})

	repo := ledger.NewRepository(db)
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{{Name: "code-agent", URL: "http://127.0.0.1:1"}})
	if err != nil {
		t.Fatal(err)
	}
	answer := makeTestPlan(runID, plan.StrategyConversational, nil)
	answer.Answer = "persisted answer"
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: answer}),
		WithExecutionStore(execution.NewMySQLStore(repo)),
		WithInstanceID("test-integration"),
	)
	body := fmt.Sprintf(`{"runId":%q,"requestId":%q,"userId":"user-int","conversationId":"conv-int","messages":[{"role":"user","text":"hi"}],"requestedPath":"main_agent_orchestration"}`, runID, requestID)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "persisted answer") {
		t.Fatalf("answer not streamed: %s", rec.Body.String())
	}

	var status string
	var currentSeq int64
	if err := db.QueryRow("SELECT status, current_seq FROM executions WHERE run_id = ?", runID).Scan(&status, &currentSeq); err != nil {
		t.Fatalf("load execution: %v", err)
	}
	if status != "completed" || currentSeq != 3 {
		t.Fatalf("unexpected execution state: status=%s seq=%d", status, currentSeq)
	}
	var planCount, checkpointCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM plan_revisions WHERE run_id = ?", runID).Scan(&planCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM checkpoints WHERE run_id = ?", runID).Scan(&checkpointCount); err != nil {
		t.Fatal(err)
	}
	if planCount != 1 || checkpointCount < 2 {
		t.Fatalf("plan/checkpoint not persisted: plans=%d checkpoints=%d", planCount, checkpointCount)
	}
	rows, err := db.Query("SELECT seq, event_type FROM execution_events WHERE run_id = ? ORDER BY seq", runID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var seqs []int64
	var types []string
	for rows.Next() {
		var seq int64
		var typ string
		if err := rows.Scan(&seq, &typ); err != nil {
			t.Fatal(err)
		}
		seqs = append(seqs, seq)
		types = append(types, typ)
	}
	if len(seqs) != 3 || seqs[0] != 1 || seqs[2] != 3 || types[2] != "run_finished" {
		t.Fatalf("unexpected events: %v %v", seqs, types)
	}
	var outboxCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM outbox WHERE run_id = ?", runID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 3 {
		t.Fatalf("outbox must mirror events, got %d", outboxCount)
	}
}

func TestIntegrationOutboxRelayMySQLRedis(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	redisAddr := os.Getenv("AGENTHUB_REDIS_TEST_ADDR")
	if dsn == "" || redisAddr == "" {
		t.Skip("SKIP real MySQL+Redis integration: set AGENTHUB_MYSQL_TEST_DSN and AGENTHUB_REDIS_TEST_ADDR")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	repo := ledger.NewRepository(db)
	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	runID := "run-relay-it-" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM outbox WHERE run_id = ?", runID)
		_, _ = db.Exec("DELETE FROM execution_events WHERE run_id = ?", runID)
		_, _ = db.Exec("DELETE FROM executions WHERE run_id = ?", runID)
		_, _ = client.Del(context.Background(), fmt.Sprintf("agenthub:test:ws:run:%s:events", runID)).Result()
	})
	if _, _, err := repo.AcceptRun(ctx, ledger.AcceptRunParams{
		RunID: runID, RequestID: "req-" + suffix, ConversationID: "conv", WorkspaceID: "ws",
		PrincipalID: "p", ExecutionPath: "main_agent_orchestration",
	}); err != nil {
		t.Fatal(err)
	}
	for i, status := range []string{"planning", "completed"} {
		if _, err := repo.AppendStateChange(ctx, ledger.StateChange{
			RunID: runID, FencingVersion: 0, Status: status, EventType: "state_update",
			Payload: []byte(fmt.Sprintf(`{"step":%d}`, i+1)),
		}); err != nil {
			t.Fatalf("append %s: %v", status, err)
		}
	}
	rel, err := relay.New(&runScopedRepository{Repository: repo, runID: runID}, client, "test")
	if err != nil {
		t.Fatal(err)
	}
	published, err := rel.RunOnce(ctx, 10)
	if err != nil || published != 2 {
		t.Fatalf("relay run once: published=%d err=%v", published, err)
	}
	entries, err := client.XRange(ctx, fmt.Sprintf("agenthub:test:ws:run:%s:events", runID), "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 redis stream entries, got %d", len(entries))
	}
	var publishedCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM outbox WHERE run_id = ? AND status = 'published'", runID).Scan(&publishedCount); err != nil {
		t.Fatal(err)
	}
	if publishedCount != 2 {
		t.Fatalf("expected 2 published outbox rows, got %d", publishedCount)
	}
}

type runScopedRepository struct {
	*ledger.Repository
	runID string
}

func (r *runScopedRepository) PendingOutbox(ctx context.Context, limit int) ([]ledger.OutboxItem, error) {
	return r.Repository.PendingOutboxForRun(ctx, r.runID, limit)
}

func TestIntegrationLeaseRenewalAndStaleFencing(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	runID := "run-lease-" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM executions WHERE run_id = ?", runID)
	})
	repo := ledger.NewRepository(db)
	if _, _, err := repo.AcceptRun(ctx, ledger.AcceptRunParams{
		RunID: runID, RequestID: "req-" + suffix, ConversationID: "conv-" + suffix,
		WorkspaceID: "default", PrincipalID: "user-lease", ExecutionPath: "single_chat",
	}); err != nil {
		t.Fatalf("accept run: %v", err)
	}
	fencing, err := repo.AcquireLease(ctx, runID, "owner-a", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	if err := repo.RenewLease(ctx, runID, "owner-a", fencing, time.Minute); err != nil {
		t.Fatalf("renew lease: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	takeover, err := repo.AcquireLease(ctx, runID, "owner-b", time.Minute)
	if err != nil {
		t.Fatalf("takeover expired lease: %v", err)
	}
	if takeover <= fencing {
		t.Fatalf("fencing must increase on takeover: old=%d new=%d", fencing, takeover)
	}
	if err := repo.RenewLease(ctx, runID, "owner-a", fencing, time.Minute); !errors.Is(err, ledger.ErrLeaseHeld) {
		t.Fatalf("stale owner must not renew: %v", err)
	}
}

func TestIntegrationRunSignals(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	runID := "run-signal-" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM run_signals WHERE run_id = ?", runID)
		_, _ = db.Exec("DELETE FROM executions WHERE run_id = ?", runID)
	})
	repo := ledger.NewRepository(db)
	if _, _, err := repo.AcceptRun(ctx, ledger.AcceptRunParams{
		RunID: runID, RequestID: "req-signal-" + suffix, ConversationID: "conv-signal-" + suffix,
		WorkspaceID: "default", PrincipalID: "user-signal", ExecutionPath: "main_agent_orchestration",
	}); err != nil {
		t.Fatalf("accept run: %v", err)
	}
	params := ledger.SignalParams{
		RunID: runID, ActionID: "plan-1", PlanID: "plan-1", PlanRevision: 1,
		Action: "approve", SelectedParticipants: []string{"code-agent"}, IdempotencyKey: "idem-" + suffix,
	}
	id, duplicate, err := repo.AppendRunSignal(ctx, params)
	if err != nil || duplicate || id == 0 {
		t.Fatalf("append signal: id=%d dup=%v err=%v", id, duplicate, err)
	}
	again, duplicate, err := repo.AppendRunSignal(ctx, params)
	if err != nil || !duplicate || again != id {
		t.Fatalf("duplicate signal: id=%d dup=%v err=%v", again, duplicate, err)
	}
	claimed, err := repo.ClaimRunSignal(ctx, runID)
	if err != nil || claimed == nil {
		t.Fatalf("claim signal: %#v err=%v", claimed, err)
	}
	if claimed.Action != "approve" || len(claimed.SelectedParticipants) != 1 || claimed.SelectedParticipants[0] != "code-agent" {
		t.Fatalf("unexpected claimed signal: %#v", claimed)
	}
	second, err := repo.ClaimRunSignal(ctx, runID)
	if err != nil || second != nil {
		t.Fatalf("signal must be claimed once: %#v err=%v", second, err)
	}
}
