// Package sqlidem implements the durable (consumerKey, eventId) idempotency
// record of R2-05. The business effect and the "done" row are written in one
// MySQL transaction, so a crash between them can only roll both back.
package sqlidem

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus"
)

// ErrIdempotencyConflict means the same eventId was seen with different bytes.
// Retrying cannot fix it; the caller should alert and dead-letter the event.
var ErrIdempotencyConflict = errors.New("sqlidem: eventId reused with different payload")

// Store owns the event_consumer_idempotency table.
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlidem: db is required")
	}
	return &Store{db: db}, nil
}

// Result reports whether the event had already completed.
type Result struct {
	Duplicate bool
	ResultRef string
}

// Process runs fn inside a transaction that also owns the idempotency row.
//   - a completed event returns Duplicate=true without calling fn;
//   - fn error rolls back the business effect and records status=failed;
//   - a concurrent duplicate blocks on the row lock, then sees the winner's
//     terminal status.
func (s *Store) Process(ctx context.Context, consumerKey, eventID, eventHash string,
	fn func(ctx context.Context, tx *sql.Tx) (resultRef string, err error)) (Result, error) {
	consumerKey = strings.TrimSpace(consumerKey)
	eventID = strings.TrimSpace(eventID)
	eventHash = strings.TrimSpace(eventHash)
	if consumerKey == "" || eventID == "" || eventHash == "" {
		return Result{}, fmt.Errorf("sqlidem: consumerKey, eventID and eventHash are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, fmt.Errorf("sqlidem: begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `INSERT INTO event_consumer_idempotency
		(consumer_key, event_id, event_hash, status, attempts)
		VALUES (?, ?, ?, 'processing', 1)
		ON DUPLICATE KEY UPDATE attempts = attempts + 1`, consumerKey, eventID, eventHash); err != nil {
		return Result{}, fmt.Errorf("sqlidem: claim: %w", err)
	}
	var status, storedHash, resultRef string
	if err := tx.QueryRowContext(ctx, `SELECT status, event_hash, result_ref
		FROM event_consumer_idempotency
		WHERE consumer_key = ? AND event_id = ? FOR UPDATE`, consumerKey, eventID).
		Scan(&status, &storedHash, &resultRef); err != nil {
		return Result{}, fmt.Errorf("sqlidem: read claim: %w", err)
	}
	if status == "done" {
		if storedHash != eventHash {
			return Result{}, ErrIdempotencyConflict
		}
		if err := tx.Commit(); err != nil {
			return Result{}, fmt.Errorf("sqlidem: commit duplicate: %w", err)
		}
		return Result{Duplicate: true, ResultRef: resultRef}, nil
	}

	resultRef, fnErr := fn(ctx, tx)
	if fnErr != nil {
		// Roll back the business write first; the failed attempt is then
		// recorded in a separate transaction so maxAttempts/DLQ accounting
		// survives redelivery without committing half a business effect.
		_ = tx.Rollback()
		if err := s.recordFailure(ctx, consumerKey, eventID, eventHash); err != nil {
			return Result{}, fmt.Errorf("sqlidem: record failure: %w", err)
		}
		return Result{}, fnErr
	}
	if _, err := tx.ExecContext(ctx, `UPDATE event_consumer_idempotency
		SET status = 'done', result_ref = ?, event_hash = ?
		WHERE consumer_key = ? AND event_id = ?`, resultRef, eventHash, consumerKey, eventID); err != nil {
		return Result{}, fmt.Errorf("sqlidem: complete: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("sqlidem: commit: %w", err)
	}
	return Result{ResultRef: resultRef}, nil
}

// recordFailure persists the failed attempt after the business transaction was
// rolled back, so the business effect is never committed on error.
func (s *Store) recordFailure(ctx context.Context, consumerKey, eventID, eventHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO event_consumer_idempotency
		(consumer_key, event_id, event_hash, status, attempts)
		VALUES (?, ?, ?, 'failed', 1)
		ON DUPLICATE KEY UPDATE status = 'failed', attempts = attempts + 1,
			event_hash = VALUES(event_hash)`, consumerKey, eventID, eventHash); err != nil {
		return err
	}
	return tx.Commit()
}

// Wrap adapts Process to an EventBus handler. The handler receives the same
// transaction, so its business writes and the idempotency row commit together.
func (s *Store) Wrap(consumerKey string,
	fn func(ctx context.Context, msg *eventbus.Message, tx *sql.Tx) (resultRef string, err error)) eventbus.Handler {
	return func(ctx context.Context, msg *eventbus.Message) error {
		if msg == nil {
			return fmt.Errorf("sqlidem: message is required")
		}
		frame, err := msg.Envelope.Encode()
		if err != nil {
			return fmt.Errorf("sqlidem: encode envelope: %w", err)
		}
		sum := sha256.Sum256(frame)
		_, err = s.Process(ctx, consumerKey, msg.Envelope.EventID, hex.EncodeToString(sum[:]),
			func(ctx context.Context, tx *sql.Tx) (string, error) {
				return fn(ctx, msg, tx)
			})
		return err
	}
}
