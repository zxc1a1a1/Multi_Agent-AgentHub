package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/db"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
)

// TimelineRepository is the Gateway persistence boundary. It exposes domain
// types only; SQL and sqlc types remain internal to this package.
type TimelineRepository interface {
	AppendMessage(context.Context, sqlite.Message) error
	CreateRun(context.Context, sqlite.Run) error
	CreateRunStep(context.Context, sqlite.RunStep) error
	UpdateRunStatus(context.Context, string, string, string, string, time.Time) error
	UpdateRunStepStatus(context.Context, string, string, string, string, time.Time) error
	UpdateMessageContentAndStatus(context.Context, string, string, string, string, string, time.Time) error
}

type Repository struct {
	db *sql.DB
	q  *db.Queries
}

func New(database *sql.DB) *Repository { return &Repository{db: database, q: db.New(database)} }

func (r *Repository) AppendMessage(ctx context.Context, m sqlite.Message) error {
	if m.ID == "" {
		m.ID = id()
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	if m.Status == "" {
		m.Status = "sent"
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := r.q.WithTx(tx)
	seq, err := q.NextMessageSequence(ctx, m.ConversationID)
	if err != nil {
		return err
	}
	_, err = q.CreateMessage(ctx, db.CreateMessageParams{ID: m.ID, ConversationID: m.ConversationID, RunID: null(m.RunID), MessageID: m.MessageID, Role: m.Role, SenderType: m.SenderType, Content: m.Content, ContentJson: null(m.ContentJSON), ReplyToMessageID: null(m.ReplyToMessageID), Status: m.Status, CreatedAt: stamp(m.CreatedAt), UpdatedAt: stamp(m.UpdatedAt), ClientMessageID: null(m.ClientMessageID), Sequence: seq})
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) CreateRun(ctx context.Context, v sqlite.Run) error {
	if v.ID == "" {
		v.ID = id()
	}
	if v.StartedAt.IsZero() {
		v.StartedAt = time.Now().UTC()
	}
	_, err := r.q.CreateRun(ctx, db.CreateRunParams{ID: v.ID, ConversationID: v.ConversationID, TriggerMessageID: null(v.TriggerMessageID), Mode: v.Mode, Status: v.Status, PlanID: null(v.PlanID), ConfirmedPlanVersion: null(v.ConfirmedPlanVersion), ContextSnapshotID: null(v.ContextSnapshotID), StartedAt: stamp(v.StartedAt), UpdatedAt: null(stamp(v.StartedAt))})
	return err
}
func (r *Repository) CreateRunStep(ctx context.Context, v sqlite.RunStep) error {
	if v.ID == "" {
		v.ID = id()
	}
	return r.q.CreateRunStep(ctx, db.CreateRunStepParams{ID: v.ID, RunID: v.RunID, ConversationID: v.ConversationID, TaskID: v.TaskID, StepIndex: int64(v.StepIndex), AgentName: v.AgentName, CapabilityID: v.CapabilityID, Status: v.Status, StartedAt: null(stamp(v.StartedAt))})
}
func (r *Repository) UpdateRunStatus(ctx context.Context, id, status, code, message string, finished time.Time) error {
	_, err := r.q.CompareAndSetRunStatus(ctx, db.CompareAndSetRunStatusParams{Status: status, FinishedAt: null(stamp(finished)), UpdatedAt: null(stamp(time.Now().UTC())), ID: id, Status_2: "pending"})
	return err
}
func (r *Repository) UpdateRunStepStatus(ctx context.Context, id, status, code, message string, finished time.Time) error {
	_, err := r.q.UpdateRunStepStatus(ctx, db.UpdateRunStepStatusParams{Status: status, ErrorCode: code, ErrorMessage: message, FinishedAt: null(stamp(finished)), ID: id})
	return err
}
func (r *Repository) UpdateMessageContentAndStatus(ctx context.Context, id, content, status, code, message string, updated time.Time) error {
	_, err := r.q.UpdateMessageContentStatus(ctx, db.UpdateMessageContentStatusParams{Content: content, Status: status, ErrorCode: code, ErrorMessage: message, UpdatedAt: stamp(updated), ID: id})
	return err
}
func null(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
func id() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e == nil {
		return hex.EncodeToString(b)
	}
	return stamp(time.Now())
}
