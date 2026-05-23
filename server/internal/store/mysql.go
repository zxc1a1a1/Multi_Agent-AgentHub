package store

import (
	"database/sql"
	"encoding/json"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/your-org/multi-agent-framework/server/internal/model"
)

// MySQL handles all database operations
type MySQL struct {
	db *sql.DB
}

// NewMySQL creates a new MySQL connection pool
func NewMySQL(dsn string) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &MySQL{db: db}, nil
}

// ListConversations returns all conversations ordered by most recently updated
func (s *MySQL) ListConversations() ([]model.Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, title, agent_name, created_at, updated_at
		 FROM conversations ORDER BY updated_at DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []model.Conversation
	for rows.Next() {
		var c model.Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.AgentName, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

// CreateConversation inserts a new conversation and returns it
func (s *MySQL) CreateConversation(title, agentName string) (*model.Conversation, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(
		`INSERT INTO conversations (id, title, agent_name) VALUES (?, ?, ?)`,
		id, title, agentName)
	if err != nil {
		return nil, err
	}

	var c model.Conversation
	err = s.db.QueryRow(
		`SELECT id, title, agent_name, created_at, updated_at FROM conversations WHERE id = ?`, id,
	).Scan(&c.ID, &c.Title, &c.AgentName, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetMessages returns messages for a conversation in chronological order
func (s *MySQL) GetMessages(conversationID string, limit int) ([]model.Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, sender_type, sender_name, content, artifacts, agui_run_id, created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC LIMIT ?`,
		conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		var senderName, artifacts, runID sql.NullString
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderType, &senderName,
			&m.Content, &artifacts, &runID, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.SenderName = senderName.String
		m.Artifacts = artifacts.String
		m.AGUIRunID = runID.String
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// SaveMessage inserts a new message into the database
func (s *MySQL) SaveMessage(conversationID, senderType, senderName, content string, artifacts []model.ArtifactData) error {
	id := uuid.New().String()
	var artifactsJSON sql.NullString
	if len(artifacts) > 0 {
		b, _ := json.Marshal(artifacts)
		artifactsJSON = sql.NullString{String: string(b), Valid: true}
	}
	_, err := s.db.Exec(
		`INSERT INTO messages (id, conversation_id, sender_type, sender_name, content, artifacts) VALUES (?, ?, ?, ?, ?, ?)`,
		id, conversationID, senderType, senderName, content, artifactsJSON)

	// Also update conversation's updated_at timestamp
	if err == nil {
		_, _ = s.db.Exec(`UPDATE conversations SET updated_at = NOW() WHERE id = ?`, conversationID)
	}
	return err
}

// Ping checks database connectivity.
func (s *MySQL) Ping() error {
	return s.db.Ping()
}

// UpdateConversationTitle updates the conversation title
func (s *MySQL) UpdateConversationTitle(id, title string) error {
	_, err := s.db.Exec(`UPDATE conversations SET title = ? WHERE id = ?`, title, id)
	return err
}
