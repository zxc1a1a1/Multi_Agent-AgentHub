package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const maxEventPayloadBytes = 1 << 20

func (s *Store) AppendEvent(ctx context.Context, e Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if e.ID == "" {
		e.ID = newID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if e.Payload == "" {
		e.Payload = "{}"
	}
	if len(e.Payload) > maxEventPayloadBytes || !json.Valid([]byte(e.Payload)) {
		return ErrInvalidArgument
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO events(id,conversation_id,run_id,invocation_id,event_type,payload,sequence,created_at) VALUES(?,?,?,?,?,?,?,?)`, e.ID, e.ConversationID, e.RunID, e.InvocationID, e.EventType, e.Payload, e.Sequence, formatTime(e.CreatedAt))
	return mapDBError(err)
}

// AppendEventNext allocates the next per-run sequence and inserts the event in
// one write transaction. SQLite's single-writer boundary makes concurrent
// appenders serialize without an in-process mutex.
func (s *Store) AppendEventNext(ctx context.Context, e Event) (Event, error) {
	if err := ctx.Err(); err != nil {
		return Event{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, mapDBError(err)
	}
	defer tx.Rollback()
	var next int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM events WHERE run_id=?`, e.RunID).Scan(&next); err != nil {
		return Event{}, mapDBError(err)
	}
	e.Sequence = next
	if e.ID == "" {
		e.ID = newID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if e.Payload == "" {
		e.Payload = "{}"
	}
	if len(e.Payload) > maxEventPayloadBytes || !json.Valid([]byte(e.Payload)) {
		return Event{}, ErrInvalidArgument
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO events(id,conversation_id,run_id,invocation_id,event_type,payload,sequence,created_at) VALUES(?,?,?,?,?,?,?,?)`, e.ID, e.ConversationID, e.RunID, e.InvocationID, e.EventType, e.Payload, e.Sequence, formatTime(e.CreatedAt)); err != nil {
		return Event{}, mapDBError(err)
	}
	if err = tx.Commit(); err != nil {
		return Event{}, mapDBError(err)
	}
	return e, nil
}

func (s *Store) ListEventsAfter(ctx context.Context, runID string, after, limit int64) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,conversation_id,run_id,invocation_id,event_type,payload,sequence,created_at FROM events WHERE run_id=? AND sequence>? ORDER BY sequence ASC LIMIT ?`, runID, after, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var ts string
		if err := rows.Scan(&e.ID, &e.ConversationID, &e.RunID, &e.InvocationID, &e.EventType, &e.Payload, &e.Sequence, &ts); err != nil {
			return nil, ErrInternal
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, ts)
		out = append(out, e)
	}
	return out, mapDBError(rows.Err())
}

func (s *Store) CompareAndSetRunStatus(ctx context.Context, runID, from, to string) error {
	if !validRunStatus(to) || !validTransition(from, to) {
		return ErrInvalidArgument
	}
	var finished any
	if terminalRunStatus(to) {
		finished = formatTime(time.Now().UTC())
	}
	r, err := s.db.ExecContext(ctx, `UPDATE runs SET status=?, finished_at=?, updated_at=? WHERE id=? AND status=?`, to, finished, formatTime(time.Now().UTC()), runID, from)
	if err != nil {
		return mapDBError(err)
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	return nil
}

func validRunStatus(s string) bool {
	switch s {
	case "pending", "planning", "awaiting_confirmation", "executing", "synthesizing", "completed", "partial_failure", "failed", "canceled":
		return true
	}
	return false
}
func terminalRunStatus(s string) bool {
	return s == "completed" || s == "partial_failure" || s == "failed" || s == "canceled"
}
func validTransition(from, to string) bool {
	if terminalRunStatus(from) || !validRunStatus(from) {
		return false
	}
	if from == to {
		return true
	}
	m := map[string][]string{"pending": {"planning", "canceled"}, "planning": {"awaiting_confirmation", "executing", "failed", "canceled"}, "awaiting_confirmation": {"executing", "canceled"}, "executing": {"synthesizing", "completed", "partial_failure", "failed", "canceled"}, "synthesizing": {"completed", "partial_failure", "failed", "canceled"}}
	for _, v := range m[from] {
		if v == to {
			return true
		}
	}
	return false
}
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed") {
		return ErrConflict
	}
	if strings.Contains(msg, "database is locked") || strings.Contains(msg, "database is busy") {
		return ErrDatabaseBusy
	}
	return ErrInternal
}
