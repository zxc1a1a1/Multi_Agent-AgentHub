package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
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
