package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// MySQLStore is the authoritative Gateway conversation/message store.
type MySQLStore struct {
	db *sql.DB
}

func OpenMySQL(ctx context.Context, dsn string) (*MySQLStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("mysql dsn is required")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn: %w", err)
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	store := &MySQLStore{db: db}
	if err := store.RunMigrations(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *MySQLStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *MySQLStore) DB() *sql.DB { return s.db }

func (s *MySQLStore) RunMigrations() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return err
	}
	applied := map[string]bool{}
	rows, err := s.db.Query("SELECT name FROM schema_migrations")
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		if applied[name] {
			continue
		}
		data, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		for _, stmt := range splitSQLStatements(string(data)) {
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
		}
		if _, err := s.db.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name); err != nil {
			return err
		}
	}
	return nil
}

func splitSQLStatements(text string) []string {
	var out []string
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(b.String()), ";"))
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

func (s *MySQLStore) CreateConversation(ctx context.Context, userID, agentName string) (*Conversation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	conv := Conversation{
		ID:        newID(),
		Type:      ConversationTypeSingle,
		UserID:    userID,
		AgentName: agentName,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO conversations
		(id, title, conversation_type, agent_name, user_id, workspace_id, created_at, updated_at)
		VALUES (?, '', 'single', ?, ?, 'default', ?, ?)`,
		conv.ID, nullString(agentName), userID, now, now)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return &conv, nil
}

func (s *MySQLStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, conversation_type, agent_name, user_id, pinned, pinned_at, created_at, updated_at
		FROM conversations WHERE id = ?`, id)
	conv, err := scanConversation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return conv, nil
}

func (s *MySQLStore) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	query := `SELECT id, title, conversation_type, agent_name, user_id, pinned, pinned_at, created_at, updated_at
		FROM conversations`
	args := []any{}
	if userID != "" {
		query += " WHERE user_id = ?"
		args = append(args, userID)
	}
	query += " ORDER BY pinned DESC, updated_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Conversation, 0)
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *conv)
	}
	return out, rows.Err()
}

func (s *MySQLStore) UpdateConversation(ctx context.Context, id string, patch ConversationPatch) (*Conversation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var current Conversation
	row := tx.QueryRowContext(ctx, `SELECT id, title, conversation_type, agent_name, user_id, pinned, pinned_at, created_at, updated_at
		FROM conversations WHERE id = ? FOR UPDATE`, id)
	conv, err := scanConversation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	current = *conv
	now := time.Now().UTC()
	if patch.Pinned != nil {
		current.Pinned = *patch.Pinned
		if *patch.Pinned {
			current.PinnedAt = &now
		} else {
			current.PinnedAt = nil
		}
	}
	if patch.Title != nil {
		current.Title = *patch.Title
	}
	current.UpdatedAt = now
	if _, err := tx.ExecContext(ctx, `UPDATE conversations SET title = ?, pinned = ?, pinned_at = ?, updated_at = ?
		WHERE id = ?`, current.Title, current.Pinned, nullTime(current.PinnedAt), now, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &current, nil
}

func (s *MySQLStore) UpdateConversationPin(ctx context.Context, id string, pinned bool) (*Conversation, error) {
	return s.UpdateConversation(ctx, id, ConversationPatch{Pinned: &pinned})
}

// SetConversationParticipants replaces the conversation roster and its type in
// one transaction. Replaying the same request is idempotent.
func (s *MySQLStore) SetConversationParticipants(ctx context.Context, id, conversationType string, participants []Participant) (*Conversation, error) {
	if conversationType == "" {
		conversationType = ConversationTypeGroup
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var current Conversation
	row := tx.QueryRowContext(ctx, `SELECT id, title, conversation_type, agent_name, user_id, pinned, pinned_at, created_at, updated_at
		FROM conversations WHERE id = ? FOR UPDATE`, id)
	conv, err := scanConversation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	current = *conv
	now := time.Now().UTC()
	agentName := current.AgentName
	if len(participants) > 0 {
		agentName = participants[0].AgentName
	}
	if _, err := tx.ExecContext(ctx, `UPDATE conversations
		SET conversation_type = ?, agent_name = ?, updated_at = ? WHERE id = ?`,
		conversationType, nullString(agentName), now, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_participants WHERE conversation_id = ?`, id); err != nil {
		return nil, err
	}
	for i, p := range participants {
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_participants
			(conversation_id, agent_name, role, selected, required, sort_order)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, p.AgentName, participantRole(p.Role), p.Selected, p.Required, i); err != nil {
			return nil, fmt.Errorf("insert participant %s: %w", p.AgentName, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	current.Type = conversationType
	current.AgentName = agentName
	current.Participants = cloneParticipants(participants)
	current.UpdatedAt = now
	return &current, nil
}

func (s *MySQLStore) ListConversationParticipants(ctx context.Context, id string) ([]Participant, error) {
	if _, err := s.GetConversation(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT agent_name, role, selected, required
		FROM conversation_participants WHERE conversation_id = ? ORDER BY sort_order, agent_name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Participant, 0)
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.AgentName, &p.Role, &p.Selected, &p.Required); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func participantRole(role string) string {
	if role == "" {
		return "member"
	}
	return role
}

func (s *MySQLStore) AppendMessage(ctx context.Context, msg Message) (*Message, error) {
	if msg.ConversationID == "" {
		return nil, ErrConversationNotFound
	}
	if strings.TrimSpace(msg.Text) == "" {
		return nil, ErrEmptyMessage
	}
	if msg.ID == "" {
		msg.ID = newID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	senderType := msg.Role
	if senderType == "" {
		senderType = msg.Author
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingID string
	if err := tx.QueryRowContext(ctx, "SELECT id FROM conversations WHERE id = ? FOR UPDATE", msg.ConversationID).Scan(&existingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = ? WHERE id = ?`, msg.CreatedAt, msg.ConversationID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO messages
		(id, conversation_id, sender_type, sender_name, content, metadata, request_id, run_id, created_at)
		VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, ?)`,
		msg.ID, msg.ConversationID, senderType, nullString(msg.Author), msg.Text, msg.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *MySQLStore) ProjectAssistantMessage(ctx context.Context, msg Message, runID string, seq int64, subIndex int) (bool, error) {
	if msg.ConversationID == "" {
		return false, ErrConversationNotFound
	}
	if strings.TrimSpace(msg.Text) == "" {
		return false, ErrEmptyMessage
	}
	if msg.ID == "" {
		msg.ID = newID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	if msg.RunID == "" {
		msg.RunID = runID
	}
	senderType := msg.Role
	if senderType == "" {
		senderType = msg.Author
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var existingID string
	if err := tx.QueryRowContext(ctx, "SELECT id FROM conversations WHERE id = ? FOR UPDATE", msg.ConversationID).Scan(&existingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrConversationNotFound
		}
		return false, err
	}

	var lastSeq int64
	var lastSubIndex int
	err = tx.QueryRowContext(ctx, `SELECT last_seq, last_sub_index FROM event_projection_offsets
		WHERE run_id = ? AND consumer_key = ? FOR UPDATE`,
		runID, ProjectionConsumerAssistantMessage).Scan(&lastSeq, &lastSubIndex)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if err == nil && (lastSeq > seq || (lastSeq == seq && lastSubIndex >= subIndex)) {
		return false, nil
	}

	if _, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = ? WHERE id = ?`, msg.CreatedAt, msg.ConversationID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO messages
		(id, conversation_id, sender_type, sender_name, content, metadata, request_id, run_id, created_at)
		VALUES (?, ?, ?, ?, ?, NULL, NULL, ?, ?)`,
		msg.ID, msg.ConversationID, senderType, nullString(msg.Author), msg.Text, msg.RunID, msg.CreatedAt); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO event_projection_offsets
		(run_id, consumer_key, last_seq, last_sub_index) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE last_seq = VALUES(last_seq), last_sub_index = VALUES(last_sub_index)`,
		runID, ProjectionConsumerAssistantMessage, seq, subIndex); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *MySQLStore) ListMessages(ctx context.Context, conversationID string) ([]Message, error) {
	if _, err := s.GetConversation(ctx, conversationID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, conversation_id, COALESCE(sender_name,''), sender_type,
		COALESCE(content,''), created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC, id ASC`,
		conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Author, &m.Role, &m.Text, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *MySQLStore) DeleteConversation(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrConversationNotFound
	}
	return nil
}

type ChatRequest struct {
	RequestID      string
	ConversationID string
	UserMessageID  string
	RunID          string
	Status         string
}

// RecordChatRequest stores a user message and its request record in one
// transaction. A duplicate requestId returns the existing record without
// appending another message or creating another run.
func (s *MySQLStore) RecordChatRequest(ctx context.Context, req ChatRequest, msg Message) (*ChatRequest, bool, error) {
	if req.RequestID == "" || req.ConversationID == "" {
		return nil, false, fmt.Errorf("requestId and conversationId are required")
	}
	if strings.TrimSpace(msg.Text) == "" {
		return nil, false, ErrEmptyMessage
	}
	if msg.ID == "" {
		msg.ID = newID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	senderType := msg.Role
	if senderType == "" {
		senderType = "user"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO chat_requests
		(request_id, conversation_id, user_message_id, run_id, status) VALUES (?, ?, ?, ?, 'accepted')`,
		req.RequestID, req.ConversationID, msg.ID, nullString(req.RunID))
	if err != nil {
		if isDuplicateKey(err) {
			var existing ChatRequest
			if err := tx.QueryRowContext(ctx, `SELECT request_id, conversation_id, user_message_id,
				COALESCE(run_id,''), status FROM chat_requests WHERE request_id = ?`, req.RequestID).Scan(
				&existing.RequestID, &existing.ConversationID, &existing.UserMessageID, &existing.RunID, &existing.Status); err != nil {
				return nil, false, err
			}
			return &existing, true, tx.Commit()
		}
		return nil, false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = ? WHERE id = ?`, msg.CreatedAt, req.ConversationID); err != nil {
		return nil, false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO messages
		(id, conversation_id, sender_type, sender_name, content, request_id, run_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, req.ConversationID, senderType, nullString(msg.Author), msg.Text, req.RequestID, nullString(req.RunID), msg.CreatedAt); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	req.UserMessageID = msg.ID
	req.Status = "accepted"
	return &req, false, nil
}

func (s *MySQLStore) GetChatRequest(ctx context.Context, requestID string) (*ChatRequest, error) {
	var req ChatRequest
	err := s.db.QueryRowContext(ctx, `SELECT request_id, conversation_id, user_message_id,
		COALESCE(run_id,''), status FROM chat_requests WHERE request_id = ?`, requestID).Scan(
		&req.RequestID, &req.ConversationID, &req.UserMessageID, &req.RunID, &req.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &req, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanConversation(row scanner) (*Conversation, error) {
	var conv Conversation
	var agentName sql.NullString
	var pinnedAt sql.NullTime
	if err := row.Scan(&conv.ID, &conv.Title, &conv.Type, &agentName, &conv.UserID, &conv.Pinned, &pinnedAt,
		&conv.CreatedAt, &conv.UpdatedAt); err != nil {
		return nil, err
	}
	conv.AgentName = agentName.String
	if pinnedAt.Valid {
		conv.PinnedAt = &pinnedAt.Time
	}
	return &conv, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// ImportConversation inserts a conversation with its original id and
// timestamps. It is intended for synthetic/offline migration and is
// idempotent on the primary key.
func (s *MySQLStore) ImportConversation(ctx context.Context, conv Conversation) error {
	if conv.ID == "" {
		return fmt.Errorf("conversation id is required")
	}
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = time.Now().UTC()
	}
	if conv.UpdatedAt.IsZero() {
		conv.UpdatedAt = conv.CreatedAt
	}
	userID := conv.UserID
	conversationType := conv.Type
	if conversationType == "" {
		conversationType = ConversationTypeSingle
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO conversations
		(id, title, conversation_type, agent_name, user_id, workspace_id, pinned, pinned_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'default', ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE title = VALUES(title), conversation_type = VALUES(conversation_type),
			agent_name = VALUES(agent_name), pinned = VALUES(pinned), pinned_at = VALUES(pinned_at),
			updated_at = VALUES(updated_at)`,
		conv.ID, conv.Title, conversationType, nullString(conv.AgentName), userID, conv.Pinned, nullTime(conv.PinnedAt), conv.CreatedAt, conv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("import conversation: %w", err)
	}
	return nil
}

// ImportMessage inserts a message with its original id/timestamp for migration.
func (s *MySQLStore) ImportMessage(ctx context.Context, msg Message, requestID, runID string) error {
	if msg.ID == "" || msg.ConversationID == "" {
		return fmt.Errorf("message id and conversation id are required")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	role := msg.Role
	if role == "" {
		role = msg.Author
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO messages
		(id, conversation_id, sender_type, sender_name, content, request_id, run_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE content = VALUES(content), sender_type = VALUES(sender_type),
			sender_name = VALUES(sender_name)`,
		msg.ID, msg.ConversationID, role, nullString(msg.Author), msg.Text,
		nullString(requestID), nullString(runID), msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("import message: %w", err)
	}
	return nil
}

// PublicEvent is one replayable browser event.
type PublicEvent struct {
	RunID     string          `json:"runId"`
	Seq       int64           `json:"seq"`
	SubIndex  int             `json:"subIndex"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"-"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (s *MySQLStore) AppendPublicEvent(ctx context.Context, event PublicEvent) error {
	if event.RunID == "" || event.Seq < 1 || event.Type == "" || len(event.Payload) == 0 {
		return fmt.Errorf("invalid public event")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO run_public_events
		(run_id, seq, sub_index, event_type, payload_json) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE payload_json = VALUES(payload_json), event_type = VALUES(event_type)`,
		event.RunID, event.Seq, event.SubIndex, event.Type, event.Payload)
	return err
}

func (s *MySQLStore) ListPublicEvents(ctx context.Context, runID string, afterSeq int64, limit int) ([]PublicEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT run_id, seq, sub_index, event_type, payload_json, created_at
		FROM run_public_events WHERE run_id = ? AND seq > ? ORDER BY seq ASC, sub_index ASC LIMIT ?`,
		runID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublicEvent
	for rows.Next() {
		var event PublicEvent
		if err := rows.Scan(&event.RunID, &event.Seq, &event.SubIndex, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (s *MySQLStore) ListPublicEventsAfter(ctx context.Context, runID string, afterSeq int64, afterSubIndex int, limit int) ([]PublicEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT run_id, seq, sub_index, event_type, payload_json, created_at
		FROM run_public_events
		WHERE run_id = ? AND (seq > ? OR (seq = ? AND sub_index > ?))
		ORDER BY seq ASC, sub_index ASC LIMIT ?`,
		runID, afterSeq, afterSeq, afterSubIndex, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublicEvent
	for rows.Next() {
		var event PublicEvent
		if err := rows.Scan(&event.RunID, &event.Seq, &event.SubIndex, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

// UpdateChatRequestStatus records the terminal state of a chat request.
func (s *MySQLStore) UpdateChatRequestStatus(ctx context.Context, requestID, status string) error {
	if requestID == "" || status == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE chat_requests SET status = ? WHERE request_id = ?", status, requestID)
	return err
}
