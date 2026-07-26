package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Batch1Repository is the domain-facing persistence boundary for the minimum
// conversation timeline. HTTP handlers must depend on this interface, not SQL.
type Batch1Repository interface {
	CreateConversationV2(context.Context, Conversation) (Conversation, error)
	GetConversationForUser(context.Context, string, string) (Conversation, error)
	UpdateConversationV2(context.Context, string, string, int64, string) (Conversation, error)
	ArchiveConversation(context.Context, string, string, bool) error
	SoftDeleteConversation(context.Context, string, string) error
	AppendMessageV2(context.Context, Message) (Message, error)
	CreateRunV2(context.Context, Run) (Run, error)
	CancelRun(context.Context, string) error
	AppendEventNext(context.Context, Event) (Event, error)
	ListEventsAfter(context.Context, string, int64, int64) ([]Event, error)
}

func (s *Store) ArchiveConversation(ctx context.Context, userID, id string, archive bool) error {
	now := time.Now().UTC()
	var archived any
	if archive {
		archived = formatTime(now)
	}
	r, err := s.db.ExecContext(ctx, `UPDATE conversations SET archived_at=?,updated_at=?,version=version+1 WHERE id=? AND user_id=? AND deleted_at IS NULL`, archived, formatTime(now), id, userID)
	if err != nil {
		return mapDBError(err)
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SoftDeleteConversation(ctx context.Context, userID, id string) error {
	now := time.Now().UTC()
	r, err := s.db.ExecContext(ctx, `UPDATE conversations SET deleted_at=?,updated_at=?,version=version+1 WHERE id=? AND user_id=? AND deleted_at IS NULL`, formatTime(now), formatTime(now), id, userID)
	if err != nil {
		return mapDBError(err)
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListConversationsForUser(ctx context.Context, userID string, limit, offset int64) ([]Conversation, error) {
	if userID == "" || limit < 1 || limit > 100 || offset < 0 {
		return nil, ErrInvalidArgument
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,status,pinned,pinned_at,created_at,updated_at,metadata_json,user_id,mode,response_mode,version,last_message_at,archived_at,deleted_at FROM conversations WHERE user_id=? AND deleted_at IS NULL ORDER BY updated_at DESC,id DESC LIMIT ? OFFSET ?`, userID, limit, offset)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	var result []Conversation
	for rows.Next() {
		var c Conversation
		var created, updated string
		var pa, lma, aa, da sql.NullString
		if err := rows.Scan(&c.ID, &c.Title, &c.Status, &c.Pinned, &pa, &created, &updated, &c.MetadataJSON, &c.UserID, &c.Mode, &c.ResponseMode, &c.Version, &lma, &aa, &da); err != nil {
			return nil, ErrInternal
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		c.PinnedAt = parseTimePtr(pa)
		c.LastMessageAt = parseTimePtr(lma)
		c.ArchivedAt = parseTimePtr(aa)
		c.DeletedAt = parseTimePtr(da)
		result = append(result, c)
	}
	return result, mapDBError(rows.Err())
}

func (s *Store) CreateConversationV2(ctx context.Context, c Conversation) (Conversation, error) {
	if c.ID == "" || c.UserID == "" || !validMode(c.Mode) || !validResponseMode(c.ResponseMode) {
		return Conversation{}, ErrInvalidArgument
	}
	if c.Status == "" {
		c.Status = "active"
	}
	if c.Version == 0 {
		c.Version = 1
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	if c.MetadataJSON == "" {
		c.MetadataJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO conversations(id,title,status,pinned,pinned_at,created_at,updated_at,metadata_json,user_id,mode,response_mode,version,last_message_at,archived_at,deleted_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.ID, c.Title, c.Status, c.Pinned, formatTimeOrNullPtr(c.PinnedAt), formatTime(c.CreatedAt), formatTime(c.UpdatedAt), c.MetadataJSON, c.UserID, c.Mode, c.ResponseMode, c.Version, formatTimeOrNullPtr(c.LastMessageAt), formatTimeOrNullPtr(c.ArchivedAt), nil)
	return c, mapDBError(err)
}

func (s *Store) GetConversationForUser(ctx context.Context, userID, id string) (Conversation, error) {
	var c Conversation
	var created, updated string
	var pa, lma, aa, da sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id,title,status,pinned,pinned_at,created_at,updated_at,metadata_json,user_id,mode,response_mode,version,last_message_at,archived_at,deleted_at FROM conversations WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID).Scan(&c.ID, &c.Title, &c.Status, &c.Pinned, &pa, &created, &updated, &c.MetadataJSON, &c.UserID, &c.Mode, &c.ResponseMode, &c.Version, &lma, &aa, &da)
	if err != nil {
		return Conversation{}, mapDBError(err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	c.PinnedAt = parseTimePtr(pa)
	c.LastMessageAt = parseTimePtr(lma)
	c.ArchivedAt = parseTimePtr(aa)
	c.DeletedAt = parseTimePtr(da)
	return c, nil
}

func (s *Store) UpdateConversationV2(ctx context.Context, userID, id string, version int64, title string) (Conversation, error) {
	if version < 1 {
		return Conversation{}, ErrInvalidArgument
	}
	now := time.Now().UTC()
	r, err := s.db.ExecContext(ctx, `UPDATE conversations SET title=?,version=version+1,updated_at=? WHERE id=? AND user_id=? AND version=? AND deleted_at IS NULL`, title, formatTime(now), id, userID, version)
	if err != nil {
		return Conversation{}, mapDBError(err)
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return Conversation{}, ErrConflict
	}
	return s.GetConversationForUser(ctx, userID, id)
}

func (s *Store) AppendMessageV2(ctx context.Context, m Message) (Message, error) {
	if m.ConversationID == "" || m.SenderType == "" || m.MessageID == "" {
		return Message{}, ErrInvalidArgument
	}
	if m.ContentJSON != "" && (len(m.ContentJSON) > maxEventPayloadBytes || !json.Valid([]byte(m.ContentJSON))) {
		return Message{}, ErrInvalidArgument
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, mapDBError(err)
	}
	defer tx.Rollback()
	var seq int64
	if m.ClientMessageID != "" {
		var existing string
		err = tx.QueryRowContext(ctx, `SELECT id FROM messages WHERE conversation_id=? AND client_message_id=? AND deleted_at IS NULL`, m.ConversationID, m.ClientMessageID).Scan(&existing)
		if err == nil {
			return m, tx.Rollback()
		}
		if err != sql.ErrNoRows {
			return Message{}, mapDBError(err)
		}
	}
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE conversation_id=?`, m.ConversationID).Scan(&seq); err != nil {
		return Message{}, mapDBError(err)
	}
	m.Sequence = seq
	if m.ID == "" {
		m.ID = newID()
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	if m.Status == "" {
		m.Status = "completed"
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO messages(id,conversation_id,run_id,step_id,message_id,role,sender_type,sender_name,agent_name,content,status,error_code,error_message,created_at,updated_at,metadata_json,client_message_id,reply_to_message_id,content_json,sequence,deleted_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, m.ID, m.ConversationID, strOrNull(m.RunID), strOrNull(m.StepID), m.MessageID, m.Role, m.SenderType, m.SenderName, m.AgentName, m.Content, m.Status, m.ErrorCode, m.ErrorMessage, formatTime(m.CreatedAt), formatTime(m.UpdatedAt), m.MetadataJSON, strOrNull(m.ClientMessageID), strOrNull(m.ReplyToMessageID), strOrNull(m.ContentJSON), m.Sequence, nil)
	if err != nil {
		return Message{}, mapDBError(err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE conversations SET last_message_at=?,updated_at=? WHERE id=? AND deleted_at IS NULL`, formatTime(now), formatTime(now), m.ConversationID)
	if err != nil {
		return Message{}, mapDBError(err)
	}
	if err = tx.Commit(); err != nil {
		return Message{}, mapDBError(err)
	}
	return m, nil
}

func (s *Store) CreateRunV2(ctx context.Context, r Run) (Run, error) {
	if r.ID == "" || r.ConversationID == "" || !validMode(r.Mode) {
		return Run{}, ErrInvalidArgument
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	if !validRunStatus(r.Status) {
		return Run{}, ErrInvalidArgument
	}
	if r.StartedAt.IsZero() {
		r.StartedAt = time.Now().UTC()
	}
	if r.MetadataJSON == "" {
		r.MetadataJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO runs(id,conversation_id,status,planning_mode,started_at,finished_at,error_code,error_message,metadata_json,trigger_message_id,mode,plan_id,confirmed_plan_version,context_snapshot_id,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.ID, r.ConversationID, r.Status, r.PlanningMode, formatTime(r.StartedAt), nil, r.ErrorCode, r.ErrorMessage, r.MetadataJSON, strOrNull(r.TriggerMessageID), r.Mode, strOrNull(r.PlanID), strOrNull(r.ConfirmedPlanVersion), strOrNull(r.ContextSnapshotID), formatTime(r.StartedAt))
	return r, mapDBError(err)
}
func (s *Store) CancelRun(ctx context.Context, id string) error {
	return s.CompareAndSetRunStatus(ctx, id, "pending", "canceled")
}
func validMode(v string) bool         { return v == "direct" || v == "manual_multi" || v == "auto" }
func validResponseMode(v string) bool { return v == "separate" || v == "synthesize" }
func formatTimeOrNullPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}
func parseTimePtr(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v.String)
	if err != nil {
		return nil
	}
	return &t
}
