package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrSnapshotNotFound = errors.New("context snapshot not found")

// ContextSnapshot is a rebuildable short-term summary with source lineage.
type ContextSnapshot struct {
	SnapshotID      string    `json:"snapshotId"`
	SessionID       string    `json:"sessionId"`
	AgentName       string    `json:"agentName"`
	Version         int64     `json:"version"`
	SourceEventFrom int64     `json:"sourceEventFrom"`
	SourceEventTo   int64     `json:"sourceEventTo"`
	Summary         string    `json:"summary"`
	MemoryRefs      []string  `json:"memoryRefs,omitempty"`
	Model           string    `json:"model,omitempty"`
	TemplateVersion string    `json:"templateVersion"`
	CreatedAt       time.Time `json:"createdAt"`
}

type SnapshotStore struct {
	db *sql.DB
}

func NewSnapshotStore(db *sql.DB) (*SnapshotStore, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &SnapshotStore{db: db}, nil
}

// Append allocates the next version for (session, agent) under a row lock.
func (s *SnapshotStore) Append(ctx context.Context, snap ContextSnapshot) (*ContextSnapshot, error) {
	if snap.SessionID == "" || snap.TemplateVersion == "" || snap.Summary == "" {
		return nil, fmt.Errorf("sessionId, templateVersion and summary are required")
	}
	if snap.SourceEventTo < snap.SourceEventFrom || snap.SourceEventFrom < 0 {
		return nil, fmt.Errorf("invalid source event range")
	}
	if snap.AgentName == "" {
		snap.AgentName = ""
	}
	refs, err := json.Marshal(snap.MemoryRefs)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var next int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0)+1
		FROM session_context_snapshots WHERE session_id = ? AND agent_name = ? FOR UPDATE`,
		snap.SessionID, snap.AgentName).Scan(&next); err != nil {
		return nil, err
	}
	snap.SnapshotID = newSnapshotID()
	snap.Version = next
	snap.CreatedAt = time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO session_context_snapshots
		(snapshot_id, session_id, agent_name, version, source_event_from, source_event_to,
		 summary, memory_refs, model, template_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.SnapshotID, snap.SessionID, snap.AgentName, snap.Version, snap.SourceEventFrom, snap.SourceEventTo,
		snap.Summary, refs, nullString(snap.Model), snap.TemplateVersion, snap.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &snap, nil
}

func (s *SnapshotStore) Latest(ctx context.Context, sessionID, agentName string) (*ContextSnapshot, error) {
	row := s.db.QueryRowContext(ctx, `SELECT snapshot_id, session_id, agent_name, version,
		source_event_from, source_event_to, summary, memory_refs, COALESCE(model,''), template_version, created_at
		FROM session_context_snapshots WHERE session_id = ? AND agent_name = ?
		ORDER BY version DESC LIMIT 1`, sessionID, agentName)
	return scanSnapshot(row)
}

func (s *SnapshotStore) List(ctx context.Context, sessionID, agentName string, limit int) ([]ContextSnapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT snapshot_id, session_id, agent_name, version,
		source_event_from, source_event_to, summary, memory_refs, COALESCE(model,''), template_version, created_at
		FROM session_context_snapshots WHERE session_id = ? AND agent_name = ?
		ORDER BY version DESC LIMIT ?`, sessionID, agentName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContextSnapshot
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *snap)
	}
	return out, rows.Err()
}

type snapshotScanner interface {
	Scan(dest ...any) error
}

func scanSnapshot(row snapshotScanner) (*ContextSnapshot, error) {
	var snap ContextSnapshot
	var refs []byte
	if err := row.Scan(&snap.SnapshotID, &snap.SessionID, &snap.AgentName, &snap.Version,
		&snap.SourceEventFrom, &snap.SourceEventTo, &snap.Summary, &refs, &snap.Model,
		&snap.TemplateVersion, &snap.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSnapshotNotFound
		}
		return nil, err
	}
	_ = json.Unmarshal(refs, &snap.MemoryRefs)
	return &snap, nil
}

func newSnapshotID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("snap_%d", time.Now().UnixNano())
	}
	return "snap_" + hex.EncodeToString(buf[:])
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
