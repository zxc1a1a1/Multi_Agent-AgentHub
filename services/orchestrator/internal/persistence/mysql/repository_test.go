package mysql

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	drivermysql "github.com/go-sql-driver/mysql"
)

func newMockRepo(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRepository(db), mock
}

func TestAcceptRunDeduplicatesRequestID(t *testing.T) {
	repo, mock := newMockRepo(t)
	dup := &drivermysql.MySQLError{Number: 1062, Message: "duplicate"}
	mock.ExpectExec("INSERT INTO executions").WillReturnError(dup)
	mock.ExpectQuery("SELECT run_id, request_id").WithArgs("req-1").WillReturnRows(
		sqlmock.NewRows([]string{
			"run_id", "request_id", "conversation_id", "workspace_id", "principal_id",
			"execution_path", "status", "terminal_reason", "current_seq", "plan_id",
			"plan_revision", "fencing_version", "lease_owner", "lease_expires_at", "budget_json", "deadline_at",
		}).AddRow("run-1", "req-1", "conv-1", "ws", "user", "main_agent_orchestration", "accepted", "", 0, "", 0, 0, "", nil, []byte(`{}`), nil),
	)

	exec, dedup, err := repo.AcceptRun(context.Background(), AcceptRunParams{
		RunID: "run-1", RequestID: "req-1", ConversationID: "conv-1", WorkspaceID: "ws", PrincipalID: "user",
		ExecutionPath: "main_agent_orchestration",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dedup {
		t.Fatal("expected deduplicated=true for a repeated requestId")
	}
	if exec.RunID != "run-1" {
		t.Fatalf("unexpected run: %s", exec.RunID)
	}
}

func TestAcceptRunRequestConflict(t *testing.T) {
	repo, mock := newMockRepo(t)
	dup := &drivermysql.MySQLError{Number: 1062, Message: "duplicate"}
	mock.ExpectExec("INSERT INTO executions").WillReturnError(dup)
	mock.ExpectQuery("SELECT run_id, request_id").WithArgs("req-1").WillReturnRows(
		sqlmock.NewRows([]string{
			"run_id", "request_id", "conversation_id", "workspace_id", "principal_id",
			"execution_path", "status", "terminal_reason", "current_seq", "plan_id",
			"plan_revision", "fencing_version", "lease_owner", "lease_expires_at", "budget_json", "deadline_at",
		}).AddRow("run-other", "req-1", "conv-1", "ws", "user", "main_agent_orchestration", "accepted", "", 0, "", 0, 0, "", nil, []byte(`{}`), nil),
	)

	_, _, err := repo.AcceptRun(context.Background(), AcceptRunParams{
		RunID: "run-1", RequestID: "req-1", ConversationID: "conv-1", WorkspaceID: "ws", PrincipalID: "user",
		ExecutionPath: "main_agent_orchestration",
	})
	if !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("expected ErrRequestConflict, got %v", err)
	}
}

func TestAppendStateChangeAllocatesNextSeq(t *testing.T) {
	repo, mock := newMockRepo(t)
	payload := []byte(`{"phase":"executing"}`)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, current_seq, fencing_version FROM executions WHERE run_id = ? FOR UPDATE")).
		WithArgs("run-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "current_seq", "fencing_version"}).AddRow("planning", 5, 2))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE executions")).
		WithArgs(int64(6), "executing", nil, nil, "run-1", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO execution_events").
		WithArgs("run-1", int64(6), "state_update", payload).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs("run-1", int64(6), "state_update").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	seq, err := repo.AppendStateChange(context.Background(), StateChange{
		RunID: "run-1", FencingVersion: 2, Status: "executing", EventType: "state_update", Payload: payload,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seq != 6 {
		t.Fatalf("expected seq 6, got %d", seq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppendStateChangeRejectsStaleFencing(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, current_seq, fencing_version FROM executions").
		WithArgs("run-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "current_seq", "fencing_version"}).AddRow("executing", 5, 2))
	mock.ExpectRollback()

	_, err := repo.AppendStateChange(context.Background(), StateChange{
		RunID: "run-1", FencingVersion: 1, Status: "executing", EventType: "state_update",
	})
	if !errors.Is(err, ErrStaleFencing) {
		t.Fatalf("expected ErrStaleFencing, got %v", err)
	}
}

func TestAppendStateChangeRejectsTerminalRun(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, current_seq, fencing_version FROM executions").
		WithArgs("run-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "current_seq", "fencing_version"}).AddRow("completed", 9, 3))
	mock.ExpectRollback()

	_, err := repo.AppendStateChange(context.Background(), StateChange{
		RunID: "run-1", FencingVersion: 3, Status: "executing", EventType: "state_update",
	})
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("expected ErrTerminal, got %v", err)
	}
}

func TestAcquireLeaseRejectsLiveHolder(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, fencing_version, lease_owner, lease_expires_at FROM executions").
		WithArgs("run-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "fencing_version", "lease_owner", "lease_expires_at"}).
			AddRow("executing", 3, "worker-a", time.Now().UTC().Add(time.Minute)))
	mock.ExpectRollback()

	_, err := repo.AcquireLease(context.Background(), "run-1", "worker-b", time.Minute)
	if !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("expected ErrLeaseHeld, got %v", err)
	}
}

func TestConsumeApprovalAndDispatchRejectsExpired(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, run_id, COALESCE(tool_call_id")).
		WithArgs("appr-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "run_id", "tool_call_id", "args_hash", "expires_at"}).
			AddRow("pending", "run-1", "tc-1", "hash", time.Now().UTC().Add(-time.Minute)))
	mock.ExpectRollback()

	err := repo.ConsumeApprovalAndDispatch(context.Background(), DispatchParams{
		RunID: "run-1", FencingVersion: 1, ApprovalID: "appr-1", ToolCallID: "tc-1",
		ToolName: "workspace.read", ToolVersion: "1", ArgsHash: "hash",
	})
	if !errors.Is(err, ErrApprovalExpired) {
		t.Fatalf("expected ErrApprovalExpired, got %v", err)
	}
}

func TestConsumeApprovalAndDispatchRejectsArgumentMismatch(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, run_id, COALESCE(tool_call_id")).
		WithArgs("appr-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "run_id", "tool_call_id", "args_hash", "expires_at"}).
			AddRow("pending", "run-1", "tc-1", "old-hash", time.Now().UTC().Add(time.Minute)))
	mock.ExpectRollback()

	err := repo.ConsumeApprovalAndDispatch(context.Background(), DispatchParams{
		RunID: "run-1", FencingVersion: 1, ApprovalID: "appr-1", ToolCallID: "tc-1",
		ToolName: "workspace.read", ToolVersion: "1", ArgsHash: "new-hash",
	})
	if !errors.Is(err, ErrApprovalMismatch) {
		t.Fatalf("expected ErrApprovalMismatch, got %v", err)
	}
}

func TestConsumeApprovalAndDispatchCommitsIntentAndBudget(t *testing.T) {
	repo, mock := newMockRepo(t)
	budget := []byte(`{"toolCallsRemaining":31}`)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, run_id, COALESCE(tool_call_id")).
		WithArgs("appr-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "run_id", "tool_call_id", "args_hash", "expires_at"}).
			AddRow("pending", "run-1", "tc-1", "hash", time.Now().UTC().Add(time.Minute)))
	mock.ExpectExec("UPDATE approvals SET status = 'consumed'").
		WithArgs(sqlmock.AnyArg(), "appr-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT fencing_version FROM executions WHERE run_id = ? FOR UPDATE")).
		WithArgs("run-1").
		WillReturnRows(sqlmock.NewRows([]string{"fencing_version"}).AddRow(int64(2)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE executions SET budget_json = ? WHERE run_id = ?")).
		WithArgs(budget, "run-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO tool_calls").
		WithArgs("run-1", "tc-1", nil, "workspace.read", "1", "hash", "idem-1", "appr-1", int64(2), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.ConsumeApprovalAndDispatch(context.Background(), DispatchParams{
		RunID: "run-1", FencingVersion: 2, ApprovalID: "appr-1", ToolCallID: "tc-1",
		ToolName: "workspace.read", ToolVersion: "1", ArgsHash: "hash",
		IdempotencyKey: "idem-1", BudgetJSON: budget,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
