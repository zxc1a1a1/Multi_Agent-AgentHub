package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// Store provides SQLite-backed persistence for AgentHub entities.
// It does NOT implement the gateway store.Store interface directly —
// that adapter will be introduced in Step 3-D when the runtime write
// path is connected. This Store exposes the full model with sender
// identity, run/step linkage, and artifact metadata.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store backed by an already-open *sql.DB.
// The caller is responsible for opening and closing the database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// DB exposes the underlying *sql.DB for callers that need direct access
// (e.g. for running migrations).
func (s *Store) DB() *sql.DB {
	return s.db
}

// ---------------------------------------------------------------------------
// Conversations
// ---------------------------------------------------------------------------

func (s *Store) CreateConversation(ctx context.Context, conv Conversation) error {
	if conv.ID == "" {
		conv.ID = newID()
	}
	now := time.Now().UTC()
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = now
	}
	if conv.UpdatedAt.IsZero() {
		conv.UpdatedAt = now
	}
	if conv.Status == "" {
		conv.Status = "active"
	}
	if conv.MetadataJSON == "" {
		conv.MetadataJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO conversations (id, title, status, created_at, updated_at, metadata_json)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		conv.ID, conv.Title, conv.Status,
		formatTime(conv.CreatedAt), formatTime(conv.UpdatedAt),
		conv.MetadataJSON,
	)
	return err
}

func (s *Store) GetConversation(ctx context.Context, id string) (Conversation, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, status, created_at, updated_at, metadata_json
		 FROM conversations WHERE id = ?`, id,
	)
	var c Conversation
	var ca, ua string
	if err := row.Scan(&c.ID, &c.Title, &c.Status, &ca, &ua, &c.MetadataJSON); err != nil {
		return Conversation{}, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return c, nil
}

func (s *Store) ListConversations(ctx context.Context) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, status, created_at, updated_at, metadata_json
		 FROM conversations ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Conversation
	for rows.Next() {
		var c Conversation
		var ca, ua string
		if err := rows.Scan(&c.ID, &c.Title, &c.Status, &ca, &ua, &c.MetadataJSON); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		out = append(out, c)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

func (s *Store) AppendMessage(ctx context.Context, msg Message) error {
	if msg.ID == "" {
		msg.ID = newID()
	}
	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	if msg.UpdatedAt.IsZero() {
		msg.UpdatedAt = now
	}
	if msg.MetadataJSON == "" {
		msg.MetadataJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (
			id, conversation_id, run_id, step_id, message_id,
			role, sender_type, sender_name, agent_name,
			content, status, error_code, error_message,
			created_at, updated_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ConversationID, strOrNull(msg.RunID), strOrNull(msg.StepID), msg.MessageID,
		msg.Role, msg.SenderType, msg.SenderName, msg.AgentName,
		msg.Content, msg.Status, msg.ErrorCode, msg.ErrorMessage,
		formatTime(msg.CreatedAt), formatTime(msg.UpdatedAt), msg.MetadataJSON,
	)
	return err
}

func (s *Store) ListMessages(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, run_id, step_id, message_id,
		        role, sender_type, sender_name, agent_name,
		        content, status, error_code, error_message,
		        created_at, updated_at, metadata_json
		 FROM messages
		 WHERE conversation_id = ?
		 ORDER BY created_at ASC`, conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var msg Message
		var runID, stepID sql.NullString
		var ca, ua string
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &runID, &stepID, &msg.MessageID,
			&msg.Role, &msg.SenderType, &msg.SenderName, &msg.AgentName,
			&msg.Content, &msg.Status, &msg.ErrorCode, &msg.ErrorMessage,
			&ca, &ua, &msg.MetadataJSON,
		); err != nil {
			return nil, err
		}
		msg.RunID = runID.String
		msg.StepID = stepID.String
		msg.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		msg.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		out = append(out, msg)
	}
	return out, rows.Err()
}

// UpdateMessageContentAndStatus updates a message's content and status atomically.
func (s *Store) UpdateMessageContentAndStatus(ctx context.Context, id, content, status, errorCode, errorMessage string, updatedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE messages SET content=?, status=?, error_code=?, error_message=?, updated_at=? WHERE id=?`,
		content, status, errorCode, errorMessage, formatTime(updatedAt), id,
	)
	return err
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

func (s *Store) CreateRun(ctx context.Context, run Run) error {
	if run.ID == "" {
		run.ID = newID()
	}
	now := time.Now().UTC()
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	if run.MetadataJSON == "" {
		run.MetadataJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO runs (
			id, conversation_id, status, planning_mode,
			started_at, finished_at, error_code, error_message, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.ConversationID, run.Status, run.PlanningMode,
		formatTime(run.StartedAt), formatTimeOrNull(run.FinishedAt),
		run.ErrorCode, run.ErrorMessage, run.MetadataJSON,
	)
	return err
}

func (s *Store) GetRun(ctx context.Context, id string) (Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, conversation_id, status, planning_mode,
		        started_at, finished_at, error_code, error_message, metadata_json
		 FROM runs WHERE id = ?`, id,
	)
	var r Run
	var sa string
	var fa sql.NullString
	if err := row.Scan(
		&r.ID, &r.ConversationID, &r.Status, &r.PlanningMode,
		&sa, &fa, &r.ErrorCode, &r.ErrorMessage, &r.MetadataJSON,
	); err != nil {
		return Run{}, err
	}
	r.StartedAt, _ = time.Parse(time.RFC3339, sa)
	if fa.Valid {
		r.FinishedAt, _ = time.Parse(time.RFC3339, fa.String)
	}
	return r, nil
}

// UpdateRunStatus updates the status of a run, including error fields and finished time.
func (s *Store) UpdateRunStatus(ctx context.Context, runID, status, errorCode, errorMessage string, finishedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status=?, error_code=?, error_message=?, finished_at=? WHERE id=?`,
		status, errorCode, errorMessage, formatTimeOrNull(finishedAt), runID,
	)
	return err
}

// ListRunsByConversation returns all runs for a conversation, ordered by started_at DESC.
func (s *Store) ListRunsByConversation(ctx context.Context, conversationID string) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, status, planning_mode,
		        started_at, finished_at, error_code, error_message, metadata_json
		 FROM runs
		 WHERE conversation_id = ?
		 ORDER BY started_at DESC`, conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		var r Run
		var sa string
		var fa sql.NullString
		if err := rows.Scan(
			&r.ID, &r.ConversationID, &r.Status, &r.PlanningMode,
			&sa, &fa, &r.ErrorCode, &r.ErrorMessage, &r.MetadataJSON,
		); err != nil {
			return nil, err
		}
		r.StartedAt, _ = time.Parse(time.RFC3339, sa)
		if fa.Valid {
			r.FinishedAt, _ = time.Parse(time.RFC3339, fa.String)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// RunSteps
// ---------------------------------------------------------------------------

func (s *Store) CreateRunStep(ctx context.Context, step RunStep) error {
	if step.ID == "" {
		step.ID = newID()
	}
	now := time.Now().UTC()
	if step.StartedAt.IsZero() {
		step.StartedAt = now
	}
	if step.MetadataJSON == "" {
		step.MetadataJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO run_steps (
			id, run_id, conversation_id, task_id, step_index,
			agent_name, capability_id, status,
			started_at, finished_at, error_code, error_message, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		step.ID, step.RunID, step.ConversationID, step.TaskID, step.StepIndex,
		step.AgentName, step.CapabilityID, step.Status,
		formatTimeOrNull(step.StartedAt), formatTimeOrNull(step.FinishedAt),
		step.ErrorCode, step.ErrorMessage, step.MetadataJSON,
	)
	return err
}

func (s *Store) ListRunSteps(ctx context.Context, runID string) ([]RunStep, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, conversation_id, task_id, step_index,
		        agent_name, capability_id, status,
		        started_at, finished_at, error_code, error_message, metadata_json
		 FROM run_steps
		 WHERE run_id = ?
		 ORDER BY step_index ASC`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RunStep
	for rows.Next() {
		var step RunStep
		var sa, fa sql.NullString
		if err := rows.Scan(
			&step.ID, &step.RunID, &step.ConversationID, &step.TaskID, &step.StepIndex,
			&step.AgentName, &step.CapabilityID, &step.Status,
			&sa, &fa, &step.ErrorCode, &step.ErrorMessage, &step.MetadataJSON,
		); err != nil {
			return nil, err
		}
		if sa.Valid {
			step.StartedAt, _ = time.Parse(time.RFC3339, sa.String)
		}
		if fa.Valid {
			step.FinishedAt, _ = time.Parse(time.RFC3339, fa.String)
		}
		out = append(out, step)
	}
	return out, rows.Err()
}

// UpdateRunStepStatus updates the status of a run step.
func (s *Store) UpdateRunStepStatus(ctx context.Context, stepID, status, errorCode, errorMessage string, finishedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE run_steps SET status=?, error_code=?, error_message=?, finished_at=? WHERE id=?`,
		status, errorCode, errorMessage, formatTimeOrNull(finishedAt), stepID,
	)
	return err
}

// ---------------------------------------------------------------------------
// Artifacts
// ---------------------------------------------------------------------------

func (s *Store) CreateArtifactMetadata(ctx context.Context, a Artifact) error {
	if a.ID == "" {
		a.ID = newID()
	}
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	if a.MetadataJSON == "" {
		a.MetadataJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts (
			id, conversation_id, run_id, step_id, message_id,
			artifact_type, title, mime_type, preview_type, content_ref, status,
			created_at, updated_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ConversationID, strOrNull(a.RunID), strOrNull(a.StepID), strOrNull(a.MessageID),
		a.ArtifactType, a.Title, a.MimeType, a.PreviewType, a.ContentRef, a.Status,
		formatTime(a.CreatedAt), formatTime(a.UpdatedAt), a.MetadataJSON,
	)
	return err
}

// ---------------------------------------------------------------------------
// Auditing (Step 3-F: Failure / Retry / Audit)
// ---------------------------------------------------------------------------

// ListFailedRunsByConversation returns runs with status='failed' for a conversation.
func (s *Store) ListFailedRunsByConversation(ctx context.Context, conversationID string) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, status, planning_mode,
		        started_at, finished_at, error_code, error_message, metadata_json
		 FROM runs
		 WHERE conversation_id = ? AND status = 'failed'
		 ORDER BY started_at DESC`, conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		var r Run
		var sa string
		var fa sql.NullString
		if err := rows.Scan(
			&r.ID, &r.ConversationID, &r.Status, &r.PlanningMode,
			&sa, &fa, &r.ErrorCode, &r.ErrorMessage, &r.MetadataJSON,
		); err != nil {
			return nil, err
		}
		r.StartedAt, _ = time.Parse(time.RFC3339, sa)
		if fa.Valid {
			r.FinishedAt, _ = time.Parse(time.RFC3339, fa.String)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListFailedMessagesByConversation returns messages with status='failed' for a conversation.
func (s *Store) ListFailedMessagesByConversation(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, run_id, step_id, message_id,
		        role, sender_type, sender_name, agent_name,
		        content, status, error_code, error_message,
		        created_at, updated_at, metadata_json
		 FROM messages
		 WHERE conversation_id = ? AND status = 'failed'
		 ORDER BY created_at ASC`, conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var msg Message
		var runID, stepID sql.NullString
		var ca, ua string
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &runID, &stepID, &msg.MessageID,
			&msg.Role, &msg.SenderType, &msg.SenderName, &msg.AgentName,
			&msg.Content, &msg.Status, &msg.ErrorCode, &msg.ErrorMessage,
			&ca, &ua, &msg.MetadataJSON,
		); err != nil {
			return nil, err
		}
		msg.RunID = runID.String
		msg.StepID = stepID.String
		msg.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		msg.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		out = append(out, msg)
	}
	return out, rows.Err()
}

// ListArtifactsByConversation returns all artifact metadata for a conversation.
func (s *Store) ListArtifactsByConversation(ctx context.Context, conversationID string) ([]Artifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, run_id, step_id, message_id,
		        artifact_type, title, mime_type, preview_type, content_ref, status,
		        created_at, updated_at, metadata_json
		 FROM artifacts
		 WHERE conversation_id = ?
		 ORDER BY created_at ASC`, conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Artifact
	for rows.Next() {
		var a Artifact
		var runID, stepID, messageID sql.NullString
		var ca, ua string
		if err := rows.Scan(
			&a.ID, &a.ConversationID, &runID, &stepID, &messageID,
			&a.ArtifactType, &a.Title, &a.MimeType, &a.PreviewType, &a.ContentRef, &a.Status,
			&ca, &ua, &a.MetadataJSON,
		); err != nil {
			return nil, err
		}
		a.RunID = runID.String
		a.StepID = stepID.String
		a.MessageID = messageID.String
		a.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListArtifactsByMessageID returns artifact metadata for a specific message.
func (s *Store) ListArtifactsByMessageID(ctx context.Context, messageID string) ([]Artifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, run_id, step_id, message_id,
		        artifact_type, title, mime_type, preview_type, content_ref, status,
		        created_at, updated_at, metadata_json
		 FROM artifacts
		 WHERE message_id = ?
		 ORDER BY created_at ASC`, messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Artifact
	for rows.Next() {
		var a Artifact
		var runID, stepID, msgID sql.NullString
		var ca, ua string
		if err := rows.Scan(
			&a.ID, &a.ConversationID, &runID, &stepID, &msgID,
			&a.ArtifactType, &a.Title, &a.MimeType, &a.PreviewType, &a.ContentRef, &a.Status,
			&ca, &ua, &a.MetadataJSON,
		); err != nil {
			return nil, err
		}
		a.RunID = runID.String
		a.StepID = stepID.String
		a.MessageID = msgID.String
		a.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatTimeOrNull(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func strOrNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}
