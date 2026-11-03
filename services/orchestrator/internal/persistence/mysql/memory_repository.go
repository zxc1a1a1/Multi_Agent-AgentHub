package mysql

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

var (
	ErrMemoryNotFound         = errors.New("memory entry not found")
	ErrMemoryConflict         = errors.New("memory logical key already exists")
	ErrMemoryRevisionConflict = errors.New("memory revision conflict")
	ErrMemoryInvalid          = errors.New("invalid memory operation")
)

type MemoryRepository struct {
	db *sql.DB
}

func NewMemoryRepository(db *sql.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

func (r *MemoryRepository) DB() *sql.DB { return r.db }

// EnsureScope returns the scope id and current epoch, creating the scope on
// first use. Scope identity comes from trusted fields only.
func (r *MemoryRepository) EnsureScope(ctx context.Context, scope rtmemory.Scope, mode rtmemory.Mode, policyJSON []byte) (string, int64, error) {
	if err := scope.Validate(); err != nil {
		return "", 0, err
	}
	if !mode.Valid() {
		return "", 0, ErrMemoryInvalid
	}
	if len(policyJSON) == 0 {
		policyJSON = []byte(`{}`)
	}
	scopeID := memoryScopeID(scope)
	_, err := r.db.ExecContext(ctx, `INSERT INTO memory_scopes
		(scope_id, env, workspace_id, principal_id, session_id, agent_name, thread_id, mode, policy_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE mode = VALUES(mode), policy_json = VALUES(policy_json)`,
		scopeID, scope.Env, scope.WorkspaceID, scope.PrincipalID, scope.SessionID, scope.AgentName, scope.ThreadID, mode, policyJSON)
	if err != nil {
		return "", 0, err
	}
	var epoch int64
	if err := r.db.QueryRowContext(ctx, "SELECT epoch FROM memory_scopes WHERE scope_id = ?", scopeID).Scan(&epoch); err != nil {
		return "", 0, err
	}
	return scopeID, epoch, nil
}

func memoryScopeID(scope rtmemory.Scope) string {
	sum := sha256.Sum256([]byte(scope.Key()))
	return "ms_" + hex.EncodeToString(sum[:16])
}

// Recall returns only active, unexpired entries in the given scope. Candidate,
// revoked and expired entries never appear.
func (r *MemoryRepository) Recall(ctx context.Context, scopeID string, query string, limit int) ([]rtmemory.Entry, error) {
	if scopeID == "" {
		return nil, ErrMemoryInvalid
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	now := time.Now().UTC()
	sqlText := `SELECT e.memory_id, e.scope_id, e.kind, e.logical_key,
		r.content, e.current_revision, e.status, e.source_refs, e.created_at,
		COALESCE(e.verified_at, e.created_at), e.expires_at, e.core, e.recheck_after
		FROM memory_entries e
		JOIN memory_revisions r ON r.memory_id = e.memory_id AND r.revision = e.current_revision
		WHERE e.scope_id = ? AND e.status = 'active'
		  AND (e.expires_at IS NULL OR e.expires_at > ?)`
	args := []any{scopeID, now}
	if terms := tokenizeMemoryQuery(query); len(terms) > 0 {
		clauses := make([]string, 0, len(terms))
		for _, term := range terms {
			clauses = append(clauses, "(INSTR(LOWER(r.content), ?) > 0 OR INSTR(LOWER(e.logical_key), ?) > 0)")
			lowered := strings.ToLower(term)
			args = append(args, lowered, lowered)
		}
		sqlText += " AND ((" + strings.Join(clauses, " OR ") + ") OR e.core = 1)"
	}
	sqlText += " ORDER BY e.core DESC, e.verified_at DESC, e.memory_id ASC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryEntries(rows)
}

func (r *MemoryRepository) Get(ctx context.Context, scopeID, memoryID string) (*rtmemory.Entry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT e.memory_id, e.scope_id, e.kind, e.logical_key,
		r.content, e.current_revision, e.status, e.source_refs, e.created_at,
		COALESCE(e.verified_at, e.created_at), e.expires_at, e.core, e.recheck_after
		FROM memory_entries e
		JOIN memory_revisions r ON r.memory_id = e.memory_id AND r.revision = e.current_revision
		WHERE e.scope_id = ? AND e.memory_id = ?`, scopeID, memoryID)
	var entry rtmemory.Entry
	var refs []byte
	var expires sql.NullTime
	var recheck sql.NullTime
	var verified time.Time
	if err := row.Scan(&entry.MemoryID, &entry.ScopeKey, &entry.Kind, &entry.LogicalKey, &entry.Content,
		&entry.Revision, &entry.Status, &refs, &entry.CreatedAt, &verified, &expires, &entry.Core, &recheck); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	entry.VerifiedAt = verified
	if expires.Valid {
		entry.ExpiresAt = &expires.Time
	}
	if recheck.Valid {
		entry.RecheckAfter = &recheck.Time
	}
	_ = json.Unmarshal(refs, &entry.SourceRefs)
	return &entry, nil
}

// Propose creates a candidate entry and its first revision. It never activates
// memory; Confirm is a separate, revision-checked operation.
func (r *MemoryRepository) Propose(ctx context.Context, scopeID string, entry rtmemory.Entry, expectedRevision int) (*rtmemory.Entry, error) {
	if scopeID == "" || entry.MemoryID == "" || entry.Kind == "" || entry.LogicalKey == "" || entry.Content == "" {
		return nil, ErrMemoryInvalid
	}
	if expectedRevision != 0 {
		return nil, ErrMemoryInvalid
	}
	if len(entry.SourceRefs) == 0 {
		return nil, fmt.Errorf("%w: source refs required", ErrMemoryInvalid)
	}
	refs, err := json.Marshal(entry.SourceRefs)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(entry.Content))
	now := time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_entries
		(memory_id, scope_id, kind, logical_key, status, current_revision, content_hash, source_refs, created_at, expires_at, core, recheck_after)
		VALUES (?, ?, ?, ?, 'candidate', 1, ?, ?, ?, ?, ?, ?)`,
		entry.MemoryID, scopeID, entry.Kind, entry.LogicalKey, hex.EncodeToString(hash[:]), refs, now,
		nullIfNilTime(entry.ExpiresAt), entry.Core, nullIfNilTime(entry.RecheckAfter)); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrMemoryConflict
		}
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_revisions
		(memory_id, revision, content, source_refs) VALUES (?, 1, ?, ?)`,
		entry.MemoryID, entry.Content, refs); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_mutations
		(mutation_id, scope_id, memory_id, operation, expected_revision, status, purge_status, actor, idempotency_key, completed_at)
		VALUES (?, ?, ?, 'propose', 0, 'completed', 'completed', 'controller', ?, ?)`,
		newMemoryID(), scopeID, entry.MemoryID, "propose:"+entry.MemoryID, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	entry.ScopeKey = scopeID
	entry.Revision = 1
	entry.Status = "candidate"
	entry.CreatedAt = now
	return &entry, nil
}

// Confirm activates a candidate using expectedRevision CAS and increments the
// scope epoch in the same transaction.
func (r *MemoryRepository) Confirm(ctx context.Context, scopeID, memoryID string, expectedRevision int) (*rtmemory.Entry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var status string
	var current int
	if err := tx.QueryRowContext(ctx, `SELECT status, current_revision FROM memory_entries
		WHERE scope_id = ? AND memory_id = ? FOR UPDATE`, scopeID, memoryID).Scan(&status, &current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	if status != "candidate" || current != expectedRevision {
		return nil, ErrMemoryRevisionConflict
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE memory_entries SET status = 'active', verified_at = ? WHERE scope_id = ? AND memory_id = ?`,
		now, scopeID, memoryID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_scopes SET epoch = epoch + 1 WHERE scope_id = ?`, scopeID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_mutations
		(mutation_id, scope_id, memory_id, operation, expected_revision, status, purge_status, actor, idempotency_key, completed_at)
		VALUES (?, ?, ?, 'confirm', ?, 'completed', 'completed', 'controller', ?, ?)`,
		newMemoryID(), scopeID, memoryID, expectedRevision, "confirm:"+memoryID+":"+fmt.Sprint(expectedRevision), now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, scopeID, memoryID)
}

// Update appends a new active revision after checking the expected revision.
func (r *MemoryRepository) Update(ctx context.Context, scopeID, memoryID, content string, expectedRevision int) (*rtmemory.Entry, error) {
	if content == "" {
		return nil, ErrMemoryInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var status string
	var current int
	var refs []byte
	if err := tx.QueryRowContext(ctx, `SELECT status, current_revision, source_refs FROM memory_entries
		WHERE scope_id = ? AND memory_id = ? FOR UPDATE`, scopeID, memoryID).Scan(&status, &current, &refs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	if status != "active" {
		return nil, ErrMemoryRevisionConflict
	}
	if current != expectedRevision {
		return nil, ErrMemoryRevisionConflict
	}
	next := current + 1
	hash := sha256.Sum256([]byte(content))
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_revisions
		(memory_id, revision, content, source_refs) VALUES (?, ?, ?, ?)`, memoryID, next, content, refs); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_entries
		SET current_revision = ?, content_hash = ?, verified_at = ? WHERE scope_id = ? AND memory_id = ?`,
		next, hex.EncodeToString(hash[:]), now, scopeID, memoryID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_scopes SET epoch = epoch + 1 WHERE scope_id = ?`, scopeID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_mutations
		(mutation_id, scope_id, memory_id, operation, expected_revision, status, purge_status, actor, idempotency_key, completed_at)
		VALUES (?, ?, ?, 'update', ?, 'completed', 'completed', 'controller', ?, ?)`,
		newMemoryID(), scopeID, memoryID, expectedRevision, "update:"+memoryID+":"+fmt.Sprint(expectedRevision), now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, scopeID, memoryID)
}

// Forget tombstones the entry, increments the scope epoch and records a purge
// mutation. It is idempotent for an already revoked entry.
func (r *MemoryRepository) Forget(ctx context.Context, scopeID, memoryID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM memory_entries
		WHERE scope_id = ? AND memory_id = ? FOR UPDATE`, scopeID, memoryID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMemoryNotFound
		}
		return err
	}
	if status == "revoked" {
		return tx.Commit()
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE memory_entries SET status = 'revoked' WHERE scope_id = ? AND memory_id = ?`,
		scopeID, memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_scopes SET epoch = epoch + 1 WHERE scope_id = ?`, scopeID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_mutations
		(mutation_id, scope_id, memory_id, operation, status, purge_status, actor, idempotency_key, completed_at)
		VALUES (?, ?, ?, 'forget', 'completed', 'pending', 'user', ?, ?)`,
		newMemoryID(), scopeID, memoryID, "forget:"+memoryID, now); err != nil {
		return err
	}
	return tx.Commit()
}

// ForgetScope tombstones all active/candidate entries in a scope, bumps epoch
// once, and records a pending purge mutation.
func (r *MemoryRepository) ForgetScope(ctx context.Context, scopeID, actor string) (int, error) {
	if scopeID == "" {
		return 0, ErrMemoryInvalid
	}
	if actor == "" {
		actor = "user"
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE memory_entries SET status = 'revoked'
		WHERE scope_id = ? AND status IN ('active','candidate')`, scopeID)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return 0, tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_scopes SET epoch = epoch + 1 WHERE scope_id = ?`, scopeID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memory_mutations
		(mutation_id, scope_id, memory_id, operation, status, purge_status, actor, idempotency_key, completed_at)
		VALUES (?, ?, NULL, 'forget_scope', 'completed', 'pending', ?, ?, ?)`,
		newMemoryID(), scopeID, actor, "forget_scope:"+scopeID, time.Now().UTC()); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (r *MemoryRepository) ListEntries(ctx context.Context, scopeID, status string, limit int) ([]rtmemory.Entry, error) {
	if scopeID == "" {
		return nil, ErrMemoryInvalid
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT e.memory_id, e.scope_id, e.kind, e.logical_key,
		r.content, e.current_revision, e.status, e.source_refs, e.created_at,
		COALESCE(e.verified_at, e.created_at), e.expires_at, e.core, e.recheck_after
		FROM memory_entries e
		JOIN memory_revisions r ON r.memory_id = e.memory_id AND r.revision = e.current_revision
		WHERE e.scope_id = ? AND (? = '' OR e.status = ?)
		ORDER BY e.created_at DESC, e.memory_id ASC LIMIT ?`, scopeID, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryEntries(rows)
}

type Mutation struct {
	MutationID  string
	MemoryID    string
	Operation   string
	Status      string
	PurgeStatus string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (r *MemoryRepository) GetMutation(ctx context.Context, mutationID string) (*Mutation, error) {
	var m Mutation
	var memoryID sql.NullString
	var completed sql.NullTime
	if err := r.db.QueryRowContext(ctx, `SELECT mutation_id, memory_id, operation, status, purge_status, created_at, completed_at
		FROM memory_mutations WHERE mutation_id = ?`, mutationID).Scan(
		&m.MutationID, &memoryID, &m.Operation, &m.Status, &m.PurgeStatus, &m.CreatedAt, &completed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	m.MemoryID = memoryID.String
	if completed.Valid {
		m.CompletedAt = &completed.Time
	}
	return &m, nil
}

func (r *MemoryRepository) CountActive(ctx context.Context, scopeID string) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM memory_entries WHERE scope_id = ? AND status = 'active'", scopeID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func scanMemoryEntries(rows *sql.Rows) ([]rtmemory.Entry, error) {
	var out []rtmemory.Entry
	for rows.Next() {
		var e rtmemory.Entry
		var refs []byte
		var expires sql.NullTime
		var recheck sql.NullTime
		var verified time.Time
		if err := rows.Scan(&e.MemoryID, &e.ScopeKey, &e.Kind, &e.LogicalKey, &e.Content, &e.Revision,
			&e.Status, &refs, &e.CreatedAt, &verified, &expires, &e.Core, &recheck); err != nil {
			return nil, err
		}
		e.VerifiedAt = verified
		if expires.Valid {
			e.ExpiresAt = &expires.Time
		}
		if recheck.Valid {
			e.RecheckAfter = &recheck.Time
		}
		_ = json.Unmarshal(refs, &e.SourceRefs)
		out = append(out, e)
	}
	return out, rows.Err()
}

func newMemoryID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("mem_%d", time.Now().UnixNano())
	}
	return "mem_" + hex.EncodeToString(buf[:])
}

func tokenizeMemoryQuery(query string) []string {
	fields := strings.FieldsFunc(strings.TrimSpace(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, term := range fields {
		if len([]rune(term)) < 2 || seen[term] {
			continue
		}
		seen[term] = true
		out = append(out, term)
		if len(out) >= 8 {
			break
		}
	}
	return out
}
