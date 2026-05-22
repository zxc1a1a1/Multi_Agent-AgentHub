package orchestrator

import "github.com/your-org/multi-agent-framework/server/internal/model"

type ProtocolConverter struct {
	messageID      string
	artifactBuffer []model.AGUIEvent
	frontendSkills []string
}

func NewConverter(skills []string) *ProtocolConverter {
	return &ProtocolConverter{
		frontendSkills: skills,
	}
}

func (c *ProtocolConverter) Convert(a2aEvent model.AGUIEvent) []model.AGUIEvent {
	return nil
}
