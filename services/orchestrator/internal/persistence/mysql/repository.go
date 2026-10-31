package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound          = errors.New("execution not found")
	ErrRequestConflict   = errors.New("request id already bound to a different run")
	ErrStaleFencing      = errors.New("stale fencing version")
	ErrLeaseHeld         = errors.New("run lease is held by another owner")
	ErrTerminal          = errors.New("run is already terminal")
	ErrApprovalConsumed  = errors.New("approval already consumed")
	ErrApprovalExpired   = errors.New("approval expired")
	ErrApprovalMismatch  = errors.New("approval does not match tool call or arguments")
	ErrDuplicateToolCall = errors.New("tool call already dispatched")
	ErrApprovalClosed    = errors.New("approval is not pending")
)

// Repository is the Orchestrator execution ledger backed by MySQL.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB exposes the underlying handle for migrations and tests.
func (r *Repository) DB() *sql.DB { return r.db }

var terminalStatuses = map[string]bool{
	"completed": true,
	"failed":    true,
	"cancelled": true,
}

func IsTerminal(status string) bool { return terminalStatuses[status] }

func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

type Execution struct {
	RunID          string
	RequestID      string
	ConversationID string
	WorkspaceID    string
	PrincipalID    string
	ExecutionPath  string
	Status         string
	TerminalReason string
	CurrentSeq     int64
	PlanID         string
	PlanRevision   int
	FencingVersion int64
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	BudgetJSON     []byte
	DeadlineAt     *time.Time
}

type AcceptRunParams struct {
	RunID          string
	RequestID      string
	ConversationID string
	WorkspaceID    string
	PrincipalID    string
	ExecutionPath  string
	BudgetJSON     []byte
	DeadlineAt     *time.Time
}

// AcceptRun creates the run or returns the existing run for the same requestId.
// A requestId bound to a different run/workspace/principal is a conflict.
func (r *Repository) AcceptRun(ctx context.Context, p AcceptRunParams) (*Execution, bool, error) {
	if p.RunID == "" || p.RequestID == "" {
		return nil, false, fmt.Errorf("runId and requestId are required")
	}
	budget := p.BudgetJSON
	if len(budget) == 0 {
		budget = []byte(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO executions
		(run_id, request_id, conversation_id, workspace_id, principal_id, execution_path, status, budget_json, deadline_at)
		VALUES (?, ?, ?, ?, ?, ?, 'accepted', ?, ?)`,
		p.RunID, p.RequestID, p.ConversationID, p.WorkspaceID, p.PrincipalID, p.ExecutionPath, budget, p.DeadlineAt)
	if err == nil {
		return &Execution{
			RunID: p.RunID, RequestID: p.RequestID, ConversationID: p.ConversationID,
			WorkspaceID: p.WorkspaceID, PrincipalID: p.PrincipalID, ExecutionPath: p.ExecutionPath,
			Status: "accepted", BudgetJSON: budget, DeadlineAt: p.DeadlineAt,
		}, false, nil
	}
	if !isDuplicateKey(err) {
		return nil, false, fmt.Errorf("insert execution: %w", err)
	}

	existing, getErr := r.GetExecutionByRequestID(ctx, p.RequestID)
	if getErr != nil {
		return nil, false, fmt.Errorf("load existing execution: %w", getErr)
	}
	if existing.RunID != p.RunID || existing.WorkspaceID != p.WorkspaceID || existing.PrincipalID != p.PrincipalID {
		return nil, false, ErrRequestConflict
	}
	return existing, true, nil
}

func (r *Repository) GetExecution(ctx context.Context, runID string) (*Execution, error) {
	return r.scanExecution(r.db.QueryRowContext(ctx, executionSelect+" WHERE run_id = ?", runID))
}

func (r *Repository) GetExecutionByRequestID(ctx context.Context, requestID string) (*Execution, error) {
	return r.scanExecution(r.db.QueryRowContext(ctx, executionSelect+" WHERE request_id = ?", requestID))
}

const executionSelect = `SELECT run_id, request_id, conversation_id, workspace_id, principal_id,
	execution_path, status, COALESCE(terminal_reason,''), current_seq, COALESCE(plan_id,''),
	plan_revision, fencing_version, COALESCE(lease_owner,''), lease_expires_at, budget_json, deadline_at
	FROM executions`

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanExecution(row rowScanner) (*Execution, error) {
	var e Execution
	var budget []byte
	var leaseExpires, deadline sql.NullTime
	if err := row.Scan(&e.RunID, &e.RequestID, &e.ConversationID, &e.WorkspaceID, &e.PrincipalID,
		&e.ExecutionPath, &e.Status, &e.TerminalReason, &e.CurrentSeq, &e.PlanID,
		&e.PlanRevision, &e.FencingVersion, &e.LeaseOwner, &leaseExpires, &budget, &deadline); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if leaseExpires.Valid {
		e.LeaseExpiresAt = &leaseExpires.Time
	}
	if deadline.Valid {
		e.DeadlineAt = &deadline.Time
	}
	e.BudgetJSON = budget
	return &e, nil
}

type StateChange struct {
	RunID          string
	FencingVersion int64
	Status         string
	TerminalReason string
	EventType      string
	Payload        []byte
	BudgetJSON     []byte
}

// AppendStateChange writes the execution state, event and outbox row in one
// transaction and allocates the next per-run seq under a row lock.
func (r *Repository) AppendStateChange(ctx context.Context, c StateChange) (int64, error) {
	if c.RunID == "" || c.EventType == "" || c.Status == "" {
		return 0, fmt.Errorf("runId, status and eventType are required")
	}
	payload := c.Payload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var status string
	var seq, fencing int64
	if err := tx.QueryRowContext(ctx,
		"SELECT status, current_seq, fencing_version FROM executions WHERE run_id = ? FOR UPDATE", c.RunID,
	).Scan(&status, &seq, &fencing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if IsTerminal(status) {
		return 0, ErrTerminal
	}
	if fencing != c.FencingVersion {
		return 0, ErrStaleFencing
	}

	next := seq + 1
	res, err := tx.ExecContext(ctx, `UPDATE executions
		SET current_seq = ?, status = ?, terminal_reason = ?, budget_json = COALESCE(?, budget_json)
		WHERE run_id = ? AND fencing_version = ?`,
		next, c.Status, nullIfEmpty(c.TerminalReason), nullIfEmptyBytes(c.BudgetJSON), c.RunID, c.FencingVersion)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, ErrStaleFencing
	}

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO execution_events (run_id, seq, event_type, payload_json) VALUES (?, ?, ?, ?)",
		c.RunID, next, c.EventType, payload); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO outbox (run_id, seq, event_type) VALUES (?, ?, ?)",
		c.RunID, next, c.EventType); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return next, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullIfEmptyBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullIfNilTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// AcquireLease increments fencingVersion and grants the lease when no live
// holder exists. Expired owners cannot commit afterwards because every write
// checks the fencing version.
func (r *Repository) AcquireLease(ctx context.Context, runID, owner string, ttl time.Duration) (int64, error) {
	if owner == "" || ttl <= 0 {
		return 0, fmt.Errorf("owner and positive ttl are required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var status string
	var fencing int64
	var currentOwner sql.NullString
	var expires sql.NullTime
	if err := tx.QueryRowContext(ctx,
		"SELECT status, fencing_version, lease_owner, lease_expires_at FROM executions WHERE run_id = ? FOR UPDATE", runID,
	).Scan(&status, &fencing, &currentOwner, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if IsTerminal(status) {
		return 0, ErrTerminal
	}
	now := time.Now().UTC()
	if currentOwner.Valid && currentOwner.String != owner && expires.Valid && expires.Time.After(now) {
		return 0, ErrLeaseHeld
	}

	nextFencing := fencing + 1
	res, err := tx.ExecContext(ctx,
		"UPDATE executions SET lease_owner = ?, lease_expires_at = ?, fencing_version = ? WHERE run_id = ? AND fencing_version = ?",
		owner, now.Add(ttl), nextFencing, runID, fencing)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, ErrStaleFencing
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return nextFencing, nil
}

func (r *Repository) RenewLease(ctx context.Context, runID, owner string, fencing int64, ttl time.Duration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentOwner, status string
	var currentFencing int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(lease_owner,''), fencing_version, status FROM executions WHERE run_id = ? FOR UPDATE",
		runID).Scan(&currentOwner, &currentFencing, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if currentOwner != owner || currentFencing != fencing || IsTerminal(status) {
		return ErrLeaseHeld
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE executions SET lease_expires_at = ? WHERE run_id = ?", time.Now().UTC().Add(ttl), runID); err != nil {
		return err
	}
	return tx.Commit()
}

type CheckpointRow struct {
	Version        int64
	StateJSON      []byte
	BudgetJSON     []byte
	FencingVersion int64
	CreatedAt      time.Time
}

func (r *Repository) LatestCheckpoint(ctx context.Context, runID string) (*CheckpointRow, error) {
	var c CheckpointRow
	if err := r.db.QueryRowContext(ctx, `SELECT version, state_json, budget_json, fencing_version, created_at
		FROM checkpoints WHERE run_id = ? ORDER BY version DESC LIMIT 1`, runID).Scan(
		&c.Version, &c.StateJSON, &c.BudgetJSON, &c.FencingVersion, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

type PlanRevisionRow struct {
	PlanID    string
	Revision  int
	PlanJSON  []byte
	PlanHash  string
	CreatedAt time.Time
}

func (r *Repository) LatestPlanRevision(ctx context.Context, runID string) (*PlanRevisionRow, error) {
	var p PlanRevisionRow
	if err := r.db.QueryRowContext(ctx, `SELECT plan_id, revision, plan_json, plan_hash, created_at
		FROM plan_revisions WHERE run_id = ? ORDER BY revision DESC LIMIT 1`, runID).Scan(
		&p.PlanID, &p.Revision, &p.PlanJSON, &p.PlanHash, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

type CheckpointParams struct {
	RunID             string
	FencingVersion    int64
	CheckpointVersion int
	StateJSON         []byte
	BudgetJSON        []byte
}

// SaveCheckpoint appends a new versioned checkpoint after verifying fencing.
func (r *Repository) SaveCheckpoint(ctx context.Context, p CheckpointParams) (int64, error) {
	if p.RunID == "" || len(p.StateJSON) == 0 || len(p.BudgetJSON) == 0 {
		return 0, fmt.Errorf("runId, state and budget are required")
	}
	if p.CheckpointVersion <= 0 {
		p.CheckpointVersion = 1
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var fencing int64
	if err := tx.QueryRowContext(ctx,
		"SELECT fencing_version FROM executions WHERE run_id = ? FOR UPDATE", p.RunID).Scan(&fencing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if fencing != p.FencingVersion {
		return 0, ErrStaleFencing
	}
	var maxVersion int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version),0) FROM checkpoints WHERE run_id = ?", p.RunID).Scan(&maxVersion); err != nil {
		return 0, err
	}
	next := maxVersion + 1
	if _, err := tx.ExecContext(ctx, `INSERT INTO checkpoints
		(run_id, version, checkpoint_version, state_json, budget_json, fencing_version)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.RunID, next, p.CheckpointVersion, p.StateJSON, p.BudgetJSON, p.FencingVersion); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return next, nil
}

type Approval struct {
	ApprovalID     string
	RunID          string
	PlanID         string
	PlanRevision   int
	ToolCallID     string
	ToolVersion    string
	ArgsHash       string
	WorkspaceID    string
	PrincipalID    string
	Status         string
	ExpiresAt      time.Time
	IdempotencyKey string
}

func (r *Repository) CreateApproval(ctx context.Context, a Approval) (*Approval, bool, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO approvals
		(approval_id, run_id, plan_id, plan_revision, tool_call_id, tool_version, args_hash,
		 workspace_id, principal_id, status, expires_at, idempotency_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)`,
		a.ApprovalID, a.RunID, a.PlanID, a.PlanRevision, nullIfEmpty(a.ToolCallID), nullIfEmpty(a.ToolVersion),
		nullIfEmpty(a.ArgsHash), a.WorkspaceID, a.PrincipalID, a.ExpiresAt, a.IdempotencyKey)
	if err == nil {
		a.Status = "pending"
		return &a, false, nil
	}
	if !isDuplicateKey(err) {
		return nil, false, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT approval_id, run_id, plan_id, plan_revision,
		COALESCE(tool_call_id,''), COALESCE(tool_version,''), COALESCE(args_hash,''),
		workspace_id, principal_id, status, expires_at, idempotency_key
		FROM approvals WHERE idempotency_key = ?`, a.IdempotencyKey).Scan(
		&a.ApprovalID, &a.RunID, &a.PlanID, &a.PlanRevision, &a.ToolCallID, &a.ToolVersion, &a.ArgsHash,
		&a.WorkspaceID, &a.PrincipalID, &a.Status, &a.ExpiresAt, &a.IdempotencyKey); err != nil {
		return nil, false, err
	}
	return &a, true, nil
}

// ResolveApproval closes a pending plan approval exactly once. Repeating the
// same target status is idempotent.
func (r *Repository) ResolveApproval(ctx context.Context, approvalID, status string) error {
	switch status {
	case "approved", "cancelled", "expired", "superseded":
	default:
		return fmt.Errorf("invalid approval status %q", status)
	}
	res, err := r.db.ExecContext(ctx, `UPDATE approvals
		SET status = ?, consumed_at = IF(? = 'approved', ?, consumed_at)
		WHERE approval_id = ? AND status = 'pending'
		  AND (expires_at > ? OR ? <> 'approved')`,
		status, status, time.Now().UTC(), approvalID, time.Now().UTC(), status)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var current string
	var expires time.Time
	if err := r.db.QueryRowContext(ctx, "SELECT status, expires_at FROM approvals WHERE approval_id = ?", approvalID).Scan(&current, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if current == status {
		return nil
	}
	if current == "pending" && !expires.After(time.Now().UTC()) {
		return ErrApprovalExpired
	}
	return ErrApprovalClosed
}

type DispatchParams struct {
	RunID          string
	FencingVersion int64
	ApprovalID     string
	ToolCallID     string
	TaskID         string
	ToolName       string
	ToolVersion    string
	ArgsHash       string
	IdempotencyKey string
	BudgetJSON     []byte
}

// ConsumeApprovalAndDispatch consumes the approval, reserves budget and records
// the dispatch intent in one transaction. The tool must not run before commit.
func (r *Repository) ConsumeApprovalAndDispatch(ctx context.Context, p DispatchParams) error {
	if p.RunID == "" || p.ApprovalID == "" || p.ToolCallID == "" {
		return fmt.Errorf("runId, approvalId and toolCallId are required")
	}
	budget := p.BudgetJSON
	if len(budget) == 0 {
		budget = []byte(`{}`)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status, runID string
	var toolCallID, argsHash string
	var expires time.Time
	if err := tx.QueryRowContext(ctx, `SELECT status, run_id, COALESCE(tool_call_id,''), COALESCE(args_hash,''), expires_at
		FROM approvals WHERE approval_id = ? FOR UPDATE`, p.ApprovalID).Scan(
		&status, &runID, &toolCallID, &argsHash, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if status == "consumed" {
		return ErrApprovalConsumed
	}
	if status != "pending" {
		return ErrApprovalMismatch
	}
	if !expires.After(time.Now().UTC()) {
		return ErrApprovalExpired
	}
	if runID != p.RunID || toolCallID != p.ToolCallID || argsHash != p.ArgsHash {
		return ErrApprovalMismatch
	}

	res, err := tx.ExecContext(ctx,
		"UPDATE approvals SET status = 'consumed', consumed_at = ? WHERE approval_id = ? AND status = 'pending'",
		time.Now().UTC(), p.ApprovalID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrApprovalConsumed
	}

	var currentFencing int64
	if err := tx.QueryRowContext(ctx,
		"SELECT fencing_version FROM executions WHERE run_id = ? FOR UPDATE", p.RunID).Scan(&currentFencing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if currentFencing != p.FencingVersion {
		return ErrStaleFencing
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE executions SET budget_json = ? WHERE run_id = ?", budget, p.RunID); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO tool_calls
		(run_id, tool_call_id, task_id, tool_name, tool_version, args_hash, idempotency_key,
		 status, approval_id, fencing_version, dispatched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'dispatched', ?, ?, ?)`,
		p.RunID, p.ToolCallID, nullIfEmpty(p.TaskID), p.ToolName, p.ToolVersion, p.ArgsHash,
		nullIfEmpty(p.IdempotencyKey), p.ApprovalID, p.FencingVersion, time.Now().UTC())
	if err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicateToolCall
		}
		return err
	}
	return tx.Commit()
}

type ToolResult struct {
	RunID      string
	ToolCallID string
	Status     string
	ResultRef  string
	ResultJSON []byte
}

type ToolCallState struct {
	Status     string
	ResultJSON string
}

// GetToolCall returns the current ledger state for a dispatched tool call.
func (r *Repository) GetToolCall(ctx context.Context, runID, toolCallID string) (*ToolCallState, error) {
	var state ToolCallState
	err := r.db.QueryRowContext(ctx, `SELECT status, COALESCE(result_json, '')
		FROM tool_calls WHERE run_id = ? AND tool_call_id = ?`, runID, toolCallID).Scan(&state.Status, &state.ResultJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &state, nil
}

// RecordToolResult stores a terminal tool outcome. A result is never applied to
// a call that is already terminal.
func (r *Repository) RecordToolResult(ctx context.Context, res ToolResult) error {
	if res.Status != "succeeded" && res.Status != "failed" && res.Status != "unknown" {
		return fmt.Errorf("invalid tool status %q", res.Status)
	}
	if len(res.ResultJSON) == 0 {
		res.ResultJSON = []byte(`{}`)
	}
	out, err := r.db.ExecContext(ctx, `UPDATE tool_calls
		SET status = ?, result_ref = ?, result_json = ?
		WHERE run_id = ? AND tool_call_id = ? AND status IN ('dispatched','unknown')`,
		res.Status, nullIfEmpty(res.ResultRef), res.ResultJSON, res.RunID, res.ToolCallID)
	if err != nil {
		return err
	}
	if n, _ := out.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return nil
}

type Event struct {
	RunID     string
	Seq       int64
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

func (r *Repository) ListEventsAfter(ctx context.Context, runID string, afterSeq int64, limit int) ([]Event, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id, seq, event_type, payload_json, created_at
		FROM execution_events WHERE run_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?`,
		runID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.RunID, &e.Seq, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type OutboxItem struct {
	ID          int64
	RunID       string
	Seq         int64
	EventType   string
	Attempts    int
	WorkspaceID string
}

func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]OutboxItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT o.id, o.run_id, o.seq, o.event_type, o.attempts,
		COALESCE(e.workspace_id, 'default')
		FROM outbox o LEFT JOIN executions e ON e.run_id = o.run_id
		WHERE o.status = 'pending' AND o.available_at <= NOW(3)
		ORDER BY o.id ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OutboxItem
	for rows.Next() {
		var item OutboxItem
		if err := rows.Scan(&item.ID, &item.RunID, &item.Seq, &item.EventType, &item.Attempts, &item.WorkspaceID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) PendingOutboxForRun(ctx context.Context, runID string, limit int) ([]OutboxItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT o.id, o.run_id, o.seq, o.event_type, o.attempts,
		COALESCE(e.workspace_id, 'default')
		FROM outbox o LEFT JOIN executions e ON e.run_id = o.run_id
		WHERE o.status = 'pending' AND o.available_at <= NOW(3) AND o.run_id = ?
		ORDER BY o.id ASC LIMIT ?`, runID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OutboxItem
	for rows.Next() {
		var item OutboxItem
		if err := rows.Scan(&item.ID, &item.RunID, &item.Seq, &item.EventType, &item.Attempts, &item.WorkspaceID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) GetEvent(ctx context.Context, runID string, seq int64) (*Event, error) {
	var e Event
	if err := r.db.QueryRowContext(ctx, `SELECT run_id, seq, event_type, payload_json, created_at
		FROM execution_events WHERE run_id = ? AND seq = ?`, runID, seq).Scan(
		&e.RunID, &e.Seq, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

type PublicEvent struct {
	RunID     string
	Seq       int64
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

func (r *Repository) AppendPublicEvent(ctx context.Context, event PublicEvent) error {
	if event.RunID == "" || event.Seq < 1 || event.EventType == "" || len(event.Payload) == 0 {
		return fmt.Errorf("invalid public event")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO orchestrator_public_events
		(run_id, seq, event_type, payload_json) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE event_type = VALUES(event_type), payload_json = VALUES(payload_json)`,
		event.RunID, event.Seq, event.EventType, event.Payload)
	return err
}

func (r *Repository) ListPublicEvents(ctx context.Context, runID string, afterSeq int64, limit int) ([]PublicEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id, seq, event_type, payload_json, created_at
		FROM orchestrator_public_events WHERE run_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?`,
		runID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublicEvent
	for rows.Next() {
		var e PublicEvent
		if err := rows.Scan(&e.RunID, &e.Seq, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE outbox SET status = 'published', published_at = ? WHERE id = ?", time.Now().UTC(), id)
	return err
}

func (r *Repository) MarkOutboxAttempt(ctx context.Context, id int64, retryAfter time.Duration) error {
	_, err := r.db.ExecContext(ctx, `UPDATE outbox
		SET attempts = attempts + 1, available_at = ? WHERE id = ?`, time.Now().UTC().Add(retryAfter), id)
	return err
}

func (r *Repository) SavePlanRevision(ctx context.Context, runID, planID string, revision int, planJSON []byte, planHash string) error {
	if runID == "" || planID == "" || revision < 1 || len(planJSON) == 0 || planHash == "" {
		return fmt.Errorf("invalid plan revision")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO plan_revisions
		(run_id, plan_id, revision, plan_json, plan_hash) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE plan_json = VALUES(plan_json), plan_hash = VALUES(plan_hash)`,
		runID, planID, revision, planJSON, planHash)
	return err
}

func (r *Repository) UpsertAgentCatalog(ctx context.Context, name, version string, descriptor []byte, enabled bool) error {
	if len(descriptor) == 0 {
		descriptor = json.RawMessage(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_catalog (agent_name, version, descriptor_json, enabled)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE version = VALUES(version), descriptor_json = VALUES(descriptor_json), enabled = VALUES(enabled)`,
		name, version, descriptor, enabled)
	return err
}

// SignalParams is a durable HITL decision appended by a confirm handler.
type SignalParams struct {
	RunID                string
	ActionID             string
	PlanID               string
	PlanRevision         int
	Action               string
	Feedback             string
	RejectReason         string
	SelectedParticipants []string
	IdempotencyKey       string
}

// SignalRow is a pending or consumed HITL signal.
type SignalRow struct {
	ID                   int64
	RunID                string
	ActionID             string
	PlanID               string
	PlanRevision         int
	Action               string
	Feedback             string
	RejectReason         string
	SelectedParticipants []string
	IdempotencyKey       string
	Status               string
	CreatedAt            time.Time
}

// AppendRunSignal records a HITL decision. Replaying the same idempotency key
// returns duplicate=true without creating a second signal.
func (r *Repository) AppendRunSignal(ctx context.Context, p SignalParams) (int64, bool, error) {
	if p.RunID == "" || p.ActionID == "" || p.PlanID == "" || p.Action == "" {
		return 0, false, fmt.Errorf("runId, actionId, planId and action are required")
	}
	var participants any
	if len(p.SelectedParticipants) > 0 {
		data, err := json.Marshal(p.SelectedParticipants)
		if err != nil {
			return 0, false, err
		}
		participants = data
	}
	var idempotency any
	if p.IdempotencyKey != "" {
		idempotency = p.IdempotencyKey
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO run_signals
		(run_id, action_id, plan_id, plan_revision, action, feedback, reject_reason,
		 selected_participants, idempotency_key, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		p.RunID, p.ActionID, p.PlanID, p.PlanRevision, p.Action,
		nullIfEmpty(p.Feedback), nullIfEmpty(p.RejectReason), participants, idempotency)
	if err != nil {
		if isDuplicateKey(err) {
			return 0, true, nil
		}
		return 0, false, err
	}
	id, err := res.LastInsertId()
	return id, false, err
}

// ClaimRunSignal atomically takes the oldest pending signal for a run. Multiple
// pollers cannot claim the same row because of FOR UPDATE SKIP LOCKED.
func (r *Repository) ClaimRunSignal(ctx context.Context, runID string) (*SignalRow, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var sig SignalRow
	var feedback, rejectReason, idempotency sql.NullString
	var participants []byte
	if err := tx.QueryRowContext(ctx, `SELECT id, run_id, action_id, plan_id, plan_revision, action,
		feedback, reject_reason, selected_participants, idempotency_key, status, created_at
		FROM run_signals WHERE run_id = ? AND status = 'pending'
		ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, runID).Scan(
		&sig.ID, &sig.RunID, &sig.ActionID, &sig.PlanID, &sig.PlanRevision, &sig.Action,
		&feedback, &rejectReason, &participants, &idempotency, &sig.Status, &sig.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if feedback.Valid {
		sig.Feedback = feedback.String
	}
	if rejectReason.Valid {
		sig.RejectReason = rejectReason.String
	}
	if idempotency.Valid {
		sig.IdempotencyKey = idempotency.String
	}
	if len(participants) > 0 {
		if err := json.Unmarshal(participants, &sig.SelectedParticipants); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE run_signals SET status = 'consumed', consumed_at = ?
		WHERE id = ? AND status = 'pending'`, time.Now().UTC(), sig.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	sig.Status = "consumed"
	return &sig, nil
}
