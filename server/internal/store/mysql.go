package store

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/your-org/multi-agent-framework/server/internal/model"
)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(dsn string) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &MySQL{db: db}, nil
}

func (s *MySQL) ListConversations() ([]model.Conversation, error) {
	return nil, nil
}

func (s *MySQL) CreateConversation(title, agentName string) (*model.Conversation, error) {
	return nil, nil
}

func (s *MySQL) GetMessages(conversationID string, limit int) ([]model.Message, error) {
	return nil, nil
}

func (s *MySQL) SaveMessage(conversationID, senderType, senderName, content string, artifacts string) error {
	return nil
}
