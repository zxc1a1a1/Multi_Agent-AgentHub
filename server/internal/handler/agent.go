package handler

import (
	"context"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/model"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/orchestrator"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/store"
)

type dbStore interface {
	ListConversations() ([]model.Conversation, error)
	CreateConversation(title, agentName string) (*model.Conversation, error)
	GetMessages(conversationID string, limit int) ([]model.Message, error)
	SaveMessage(conversationID, senderType, senderName, content string, artifacts []model.ArtifactData) error
	UpdateConversationTitle(id, title string) error
	ConversationExists(conversationID string) (bool, error)
}

type orchestratorRunner interface {
	Process(ctx context.Context, req model.AGUIRunRequest, history []model.Message, events chan<- model.AGUIEvent)
}

type Handler struct {
	db  dbStore
	orc orchestratorRunner
}

func New(db *store.MySQL, orc *orchestrator.Orchestrator) *Handler {
	return &Handler{db: db, orc: orc}
}
