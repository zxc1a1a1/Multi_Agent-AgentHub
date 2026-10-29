package mysql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// The integration suite requires a real MySQL instance. It never reads .env;
// set AGENTHUB_MYSQL_TEST_DSN explicitly in a controlled environment.
func openIntegrationRepo(t *testing.T) *Repository {
	t.Helper()
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP real MySQL integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	db, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	cleanup := func() {
		for _, table := range []string{
			"outbox", "execution_events", "tool_calls", "approvals",
			"checkpoints", "plan_revisions", "task_executions", "executions",
		} {
			if _, err := db.Exec("DELETE FROM " + table + " WHERE run_id LIKE 'run-itest-%'"); err != nil {
				t.Fatalf("cleanup %s: %v", table, err)
			}
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	return NewRepository(db)
}

func acceptIntegrationRun(t *testing.T, repo *Repository, suffix string) *Execution {
	t.Helper()
	exec, _, err := repo.AcceptRun(context.Background(), AcceptRunParams{
		RunID:          "run-itest-" + suffix,
		RequestID:      "req-" + suffix,
		ConversationID: "conv-" + suffix,
		WorkspaceID:    "ws-test",
		PrincipalID:    "principal-test",
		ExecutionPath:  "main_agent_orchestration",
	})
	if err != nil {
		t.Fatalf("accept run: %v", err)
	}
	return exec
}

func TestIntegrationDuplicateRequestIsIdempotent(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	first := acceptIntegrationRun(t, repo, suffix)
	second, dedup, err := repo.AcceptRun(context.Background(), AcceptRunParams{
		RunID: first.RunID, RequestID: first.RequestID, ConversationID: first.ConversationID,
		WorkspaceID: first.WorkspaceID, PrincipalID: first.PrincipalID, ExecutionPath: first.ExecutionPath,
	})
	if err != nil {
		t.Fatalf("dedup accept: %v", err)
	}
	if !dedup || second.RunID != first.RunID {
		t.Fatalf("expected deduplicated same run, got dedup=%v run=%s", dedup, second.RunID)
	}

	_, _, err = repo.AcceptRun(context.Background(), AcceptRunParams{
		RunID: "run-other-" + suffix, RequestID: first.RequestID, ConversationID: first.ConversationID,
		WorkspaceID: first.WorkspaceID, PrincipalID: first.PrincipalID, ExecutionPath: first.ExecutionPath,
	})
	if !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("expected ErrRequestConflict, got %v", err)
	}
}

func TestIntegrationConcurrentSeqAndStaleFencing(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	run := acceptIntegrationRun(t, repo, suffix)

	fencing, err := repo.AcquireLease(context.Background(), run.RunID, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}

	const writers = 2
	seqs := make(chan int64, writers)
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seq, err := repo.AppendStateChange(context.Background(), StateChange{
				RunID: run.RunID, FencingVersion: fencing, Status: "executing",
				EventType: "state_update", Payload: []byte(fmt.Sprintf(`{"writer":%d}`, i)),
			})
			if err != nil {
				errs <- err
				return
			}
			seqs <- seq
		}(i)
	}
	wg.Wait()
	close(seqs)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent append failed: %v", err)
	}
	seen := map[int64]bool{}
	for seq := range seqs {
		if seen[seq] {
			t.Fatalf("duplicate seq %d", seq)
		}
		seen[seq] = true
	}
	if len(seen) != writers {
		t.Fatalf("expected %d unique seqs, got %d", writers, len(seen))
	}

	_, err = repo.AppendStateChange(context.Background(), StateChange{
		RunID: run.RunID, FencingVersion: fencing - 1, Status: "executing", EventType: "state_update",
	})
	if !errors.Is(err, ErrStaleFencing) {
		t.Fatalf("expected ErrStaleFencing for old version, got %v", err)
	}

	if _, err := repo.AppendStateChange(context.Background(), StateChange{
		RunID: run.RunID, FencingVersion: fencing, Status: "completed", EventType: "run_finished",
		TerminalReason: "completed", Payload: []byte(`{"status":"completed"}`),
	}); err != nil {
		t.Fatalf("complete run: %v", err)
	}
	if _, err := repo.AppendStateChange(context.Background(), StateChange{
		RunID: run.RunID, FencingVersion: fencing, Status: "executing", EventType: "state_update",
	}); !errors.Is(err, ErrTerminal) {
		t.Fatalf("expected ErrTerminal after completion, got %v", err)
	}

	events, err := repo.ListEventsAfter(context.Background(), run.RunID, 0, 100)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != writers+1 {
		t.Fatalf("expected %d events, got %d", writers+1, len(events))
	}
	for i, e := range events {
		if e.Seq != int64(i+1) {
			t.Fatalf("event %d has seq %d, expected %d", i, e.Seq, i+1)
		}
	}
	items, err := repo.PendingOutboxForRun(context.Background(), run.RunID, 100)
	if err != nil {
		t.Fatalf("pending outbox: %v", err)
	}
	if len(items) != writers+1 {
		t.Fatalf("expected outbox rows in same tx, got %d", len(items))
	}
}

func TestIntegrationCheckpointFencing(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	run := acceptIntegrationRun(t, repo, suffix)
	fencing, err := repo.AcquireLease(context.Background(), run.RunID, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}

	version, err := repo.SaveCheckpoint(context.Background(), CheckpointParams{
		RunID: run.RunID, FencingVersion: fencing,
		StateJSON: []byte(`{"cursor":"task-1"}`), BudgetJSON: []byte(`{"modelCalls":3}`),
	})
	if err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected checkpoint version 1, got %d", version)
	}
	if _, err := repo.SaveCheckpoint(context.Background(), CheckpointParams{
		RunID: run.RunID, FencingVersion: fencing - 1,
		StateJSON: []byte(`{"cursor":"task-2"}`), BudgetJSON: []byte(`{"modelCalls":4}`),
	}); !errors.Is(err, ErrStaleFencing) {
		t.Fatalf("expected ErrStaleFencing, got %v", err)
	}
}

func TestIntegrationApprovalConsumedOnce(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	run := acceptIntegrationRun(t, repo, suffix)
	fencing, err := repo.AcquireLease(context.Background(), run.RunID, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}

	approval, _, err := repo.CreateApproval(context.Background(), Approval{
		ApprovalID: "appr-" + suffix, RunID: run.RunID, PlanID: "plan-" + suffix, PlanRevision: 1,
		ToolCallID: "tc-" + suffix, ToolVersion: "1", ArgsHash: "hash",
		WorkspaceID: run.WorkspaceID, PrincipalID: run.PrincipalID,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute), IdempotencyKey: "idem-" + suffix,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	params := DispatchParams{
		RunID: run.RunID, FencingVersion: fencing, ApprovalID: approval.ApprovalID,
		ToolCallID: approval.ToolCallID, ToolName: "workspace.read", ToolVersion: "1",
		ArgsHash: "hash", IdempotencyKey: "tc-idem-" + suffix, BudgetJSON: []byte(`{"toolCalls":31}`),
	}
	if err := repo.ConsumeApprovalAndDispatch(context.Background(), params); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := repo.ConsumeApprovalAndDispatch(context.Background(), params); !errors.Is(err, ErrApprovalConsumed) {
		t.Fatalf("expected ErrApprovalConsumed, got %v", err)
	}

	var count int
	if err := repo.DB().QueryRow("SELECT COUNT(*) FROM tool_calls WHERE run_id = ?", run.RunID).Scan(&count); err != nil {
		t.Fatalf("count tool calls: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 tool call, got %d", count)
	}
}

func TestIntegrationToolResultUnknown(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	run := acceptIntegrationRun(t, repo, suffix)
	fencing, err := repo.AcquireLease(context.Background(), run.RunID, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	approval, _, err := repo.CreateApproval(context.Background(), Approval{
		ApprovalID: "appr-" + suffix, RunID: run.RunID, PlanID: "plan-" + suffix, PlanRevision: 1,
		ToolCallID: "tc-" + suffix, ToolVersion: "1", ArgsHash: "hash",
		WorkspaceID: run.WorkspaceID, PrincipalID: run.PrincipalID,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute), IdempotencyKey: "idem-" + suffix,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if err := repo.ConsumeApprovalAndDispatch(context.Background(), DispatchParams{
		RunID: run.RunID, FencingVersion: fencing, ApprovalID: approval.ApprovalID,
		ToolCallID: approval.ToolCallID, ToolName: "sandbox.run_tests", ToolVersion: "1",
		ArgsHash: "hash", IdempotencyKey: "tc-idem-" + suffix,
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := repo.RecordToolResult(context.Background(), ToolResult{
		RunID: run.RunID, ToolCallID: approval.ToolCallID, Status: "unknown",
	}); err != nil {
		t.Fatalf("record unknown: %v", err)
	}
	var status string
	if err := repo.DB().QueryRow("SELECT status FROM tool_calls WHERE run_id=? AND tool_call_id=?",
		run.RunID, approval.ToolCallID).Scan(&status); err != nil {
		t.Fatalf("query tool call: %v", err)
	}
	if status != "unknown" {
		t.Fatalf("expected unknown status, got %s", status)
	}
}

func TestIntegrationFailedStateChangeLeavesNoHalfResult(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	run := acceptIntegrationRun(t, repo, suffix)

	// Pre-occupy the outbox unique key so the outbox insert fails after the
	// execution update and event insert have run.
	if _, err := repo.DB().Exec("INSERT INTO outbox (run_id, seq, event_type) VALUES (?, 1, 'state_update')", run.RunID); err != nil {
		t.Fatalf("seed outbox conflict: %v", err)
	}
	if _, err := repo.AppendStateChange(context.Background(), StateChange{
		RunID: run.RunID, FencingVersion: 0, Status: "executing", EventType: "state_update",
	}); err == nil {
		t.Fatal("expected transaction failure on outbox conflict")
	}

	exec, err := repo.GetExecution(context.Background(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if exec.CurrentSeq != 0 || exec.Status != "accepted" {
		t.Fatalf("failed transaction left partial state: %#v", exec)
	}
	var events int
	if err := repo.DB().QueryRow("SELECT COUNT(*) FROM execution_events WHERE run_id = ?", run.RunID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 0 {
		t.Fatalf("failed transaction left events: %d", events)
	}
}

func TestIntegrationPlanApprovalLifecycle(t *testing.T) {
	repo := openIntegrationRepo(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	run := acceptIntegrationRun(t, repo, suffix)
	approvalID := "appr-itest-" + suffix
	if _, _, err := repo.CreateApproval(context.Background(), Approval{
		ApprovalID: approvalID, RunID: run.RunID, PlanID: "plan-" + suffix, PlanRevision: 1,
		WorkspaceID: run.WorkspaceID, PrincipalID: run.PrincipalID,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute), IdempotencyKey: "plan-" + suffix,
	}); err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if err := repo.ResolveApproval(context.Background(), approvalID, "approved"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := repo.ResolveApproval(context.Background(), approvalID, "approved"); err != nil {
		t.Fatalf("repeat approve must be idempotent: %v", err)
	}
	if err := repo.ResolveApproval(context.Background(), approvalID, "cancelled"); !errors.Is(err, ErrApprovalClosed) {
		t.Fatalf("closed approval must reject a different status, got %v", err)
	}
	var status string
	if err := repo.DB().QueryRow("SELECT status FROM approvals WHERE approval_id = ?", approvalID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "approved" {
		t.Fatalf("unexpected approval status: %s", status)
	}

	expiredID := "appr-expired-" + suffix
	if _, _, err := repo.CreateApproval(context.Background(), Approval{
		ApprovalID: expiredID, RunID: run.RunID, PlanID: "plan-expired-" + suffix, PlanRevision: 1,
		WorkspaceID: run.WorkspaceID, PrincipalID: run.PrincipalID,
		ExpiresAt: time.Now().UTC().Add(-time.Minute), IdempotencyKey: "plan-expired-" + suffix,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.ResolveApproval(context.Background(), expiredID, "approved"); !errors.Is(err, ErrApprovalExpired) {
		t.Fatalf("expired approval must not be approved, got %v", err)
	}
	if err := repo.ResolveApproval(context.Background(), expiredID, "cancelled"); err != nil {
		t.Fatalf("expired approval can be cancelled: %v", err)
	}
}
