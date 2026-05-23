package handler

import (
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/orchestrator"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/store"
)

type Handler struct {
	db  *store.MySQL
	orc *orchestrator.Orchestrator
}

func New(db *store.MySQL, orc *orchestrator.Orchestrator) *Handler {
	return &Handler{db: db, orc: orc}
}
