package orchestrator

import (
	"context"

	"github.com/your-org/multi-agent-framework/server/internal/a2a"
	"github.com/your-org/multi-agent-framework/server/internal/config"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

type Orchestrator struct {
	a2aClient *a2a.Client
	agents    map[string]config.AgentConfig
}

func New(a2aClient *a2a.Client, agents map[string]config.AgentConfig) *Orchestrator {
	return &Orchestrator{
		a2aClient: a2aClient,
		agents:    agents,
	}
}

func (o *Orchestrator) Process(ctx context.Context, req model.AGUIRunRequest, history []model.Message, events chan<- model.AGUIEvent) {
	events <- model.AGUIEvent{Type: "RUN_STARTED", RunID: req.RunID}
	events <- model.AGUIEvent{Type: "RUN_FINISHED"}
}
