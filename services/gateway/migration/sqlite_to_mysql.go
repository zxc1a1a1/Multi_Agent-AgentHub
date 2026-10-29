// Package migration imports a synthetic or authorized SQLite demo database
// into the target MySQL store. It never reads .env or credentials.
package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type ConversationRecord struct {
	Conversation store.Conversation
	Messages     []store.Message
	RequestIDs   []string
	RunIDs       []string
}

type Source struct {
	Conversations []ConversationRecord
}

type Destination interface {
	ImportConversation(ctx context.Context, conv store.Conversation) error
	ImportMessage(ctx context.Context, msg store.Message, requestID, runID string) error
}

type Stats struct {
	Conversations int
	Messages      int
}

func Load(ctx context.Context, sqlitePath string) (*Source, error) {
	if strings.TrimSpace(sqlitePath) == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	db, err := sql.Open("sqlite", sqlitePath+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := persistence.RunMigrations(db); err != nil {
		return nil, fmt.Errorf("sqlite migrations: %w", err)
	}
	src := sqlite.NewStore(db)
	conversations, err := src.ListConversations(ctx)
	if err != nil {
		return nil, err
	}
	out := &Source{}
	for _, conv := range conversations {
		userID, agentName := legacyIdentity(conv.MetadataJSON)
		messages, err := src.ListMessages(ctx, conv.ID)
		if err != nil {
			return nil, fmt.Errorf("list messages %s: %w", conv.ID, err)
		}
		record := ConversationRecord{
			Conversation: store.Conversation{
				ID: conv.ID, Title: conv.Title, UserID: userID, AgentName: agentName,
				Pinned: conv.Pinned, PinnedAt: conv.PinnedAt, CreatedAt: conv.CreatedAt, UpdatedAt: conv.UpdatedAt,
			},
		}
		for _, msg := range messages {
			record.Messages = append(record.Messages, store.Message{
				ID: msg.ID, ConversationID: msg.ConversationID,
				Author: msg.SenderName, Role: chooseRole(msg.Role, msg.SenderType),
				Text: msg.Content, CreatedAt: msg.CreatedAt,
			})
			record.RequestIDs = append(record.RequestIDs, "")
			record.RunIDs = append(record.RunIDs, msg.RunID)
		}
		out.Conversations = append(out.Conversations, record)
	}
	return out, nil
}

func Apply(ctx context.Context, source *Source, dst Destination) (Stats, error) {
	if source == nil || dst == nil {
		return Stats{}, fmt.Errorf("source and destination are required")
	}
	stats := Stats{}
	for _, record := range source.Conversations {
		if err := dst.ImportConversation(ctx, record.Conversation); err != nil {
			return stats, fmt.Errorf("import conversation %s: %w", record.Conversation.ID, err)
		}
		stats.Conversations++
		for i, msg := range record.Messages {
			requestID, runID := "", ""
			if i < len(record.RequestIDs) {
				requestID = record.RequestIDs[i]
			}
			if i < len(record.RunIDs) {
				runID = record.RunIDs[i]
			}
			if err := dst.ImportMessage(ctx, msg, requestID, runID); err != nil {
				return stats, fmt.Errorf("import message %s: %w", msg.ID, err)
			}
			stats.Messages++
		}
	}
	return stats, nil
}

func legacyIdentity(metadataJSON string) (userID, agentName string) {
	userID, agentName = "legacy-user", ""
	if strings.TrimSpace(metadataJSON) == "" {
		return
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return
	}
	if value, ok := metadata["userId"].(string); ok && strings.TrimSpace(value) != "" {
		userID = strings.TrimSpace(value)
	}
	if value, ok := metadata["agentName"].(string); ok {
		agentName = strings.TrimSpace(value)
	}
	return
}

func chooseRole(role, senderType string) string {
	if strings.TrimSpace(role) != "" {
		return role
	}
	return senderType
}
