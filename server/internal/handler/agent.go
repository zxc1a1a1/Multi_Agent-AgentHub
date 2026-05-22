package handler

import (
	"github.com/your-org/multi-agent-framework/server/internal/orchestrator"
	"github.com/your-org/multi-agent-framework/server/internal/store"
)

type Handler struct {
	db  *store.MySQL
	orc *orchestrator.Orchestrator
}

func New(db *store.MySQL, orc *orchestrator.Orchestrator) *Handler {
	return &Handler{db: db, orc: orc}
}
