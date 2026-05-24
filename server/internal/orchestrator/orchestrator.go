package orchestrator

import (
	"context"
	"log"

	"github.com/google/uuid"
	a2aclient "github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/model"
)

// Orchestrator routes requests to the appropriate agent and handles protocol conversion.
// Per gateway-orchestrator-contract: embedded in Gateway process for MVP.
type Orchestrator struct {
	a2aClient *a2aclient.Client
	agents    map[string]config.AgentConfig
}

const (
	roleUser      = "user"
	roleAssistant = "assistant"
	roleSystem    = "system"
)

// New creates a new Orchestrator instance.
func New(client *a2aclient.Client, agents map[string]config.AgentConfig) *Orchestrator {
	return &Orchestrator{
		a2aClient: client,
		agents:    agents,
	}
}

// Process handles a complete AG-UI run request.
// Per agui-event-contract: emits RUN_STARTED → TEXT_MESSAGE_* → TOOL_CALL_* → RUN_FINISHED
// Per gateway-orchestrator-contract: converts A2A events to AG-UI events
func (o *Orchestrator) Process(
	ctx context.Context,
	req model.AGUIRunRequest,
	history []model.Message,
	events chan<- model.AGUIEvent,
) {
	// 1. Emit RUN_STARTED
	runID := req.RunID
	if runID == "" {
		runID = "run-" + uuid.New().String()[:8]
	}
	events <- model.AGUIEvent{Type: "RUN_STARTED", RunID: runID}

	// 2. Determine target agent (MVP: direct routing to code-agent)
	agentCfg, ok := o.agents["code-agent"]
	if !ok {
		events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "no agent configured"}
		return
	}

	// 3. Build structured role-aware message context from history + current request
	messages := o.buildStructuredMessages(req, history)

	// 4. Extract frontend skills
	skills := o.extractSkills(req.Tools)

	// 5. Create protocol converter
	converter := NewConverter(skills)

	// 6. Call agent via official a2a-go/v2 client
	eventIter, err := o.a2aClient.SendStreamingMessages(ctx, agentCfg.URL, messages)
	if err != nil {
		log.Printf("A2A client error: %v", err)
		events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "agent communication error"}
		return
	}

	// 7. Convert A2A events → AG-UI events
	streamCompleted := false
	for event, err := range eventIter {
		if err != nil {
			if !streamCompleted {
				events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "agent streaming error"}
			}
			return
		}

		aguiEvents := converter.ConvertA2AEvent(event)
		for _, e := range aguiEvents {
			if e.Type == "RUN_FINISHED" || e.Type == "RUN_ERROR" {
				streamCompleted = true
			}
			select {
			case <-ctx.Done():
				events <- model.AGUIEvent{Type: "RUN_ERROR", Error: "request cancelled"}
				return
			case events <- e:
			}
		}
	}

	// If stream ended without completion signal, emit RUN_FINISHED
	if !streamCompleted {
		events <- model.AGUIEvent{Type: "RUN_FINISHED"}
	}
}

// buildStructuredMessages keeps role semantics for history + current messages.
func (o *Orchestrator) buildStructuredMessages(req model.AGUIRunRequest, history []model.Message) []a2aclient.StructuredMessage {
	total := len(history) + len(req.Messages)
	messages := make([]a2aclient.StructuredMessage, 0, total)

	for _, h := range history {
		messages = append(messages, a2aclient.StructuredMessage{
			Role:    mapHistorySenderToRole(h.SenderType),
			Content: h.Content,
		})
	}

	for _, m := range req.Messages {
		messages = append(messages, a2aclient.StructuredMessage{
			Role:    normalizeAGUIRole(m.Role),
			Content: m.Content,
		})
	}

	return messages
}

func mapHistorySenderToRole(senderType string) string {
	switch senderType {
	case "agent":
		return roleAssistant
	default:
		return roleUser
	}
}

func normalizeAGUIRole(role string) string {
	switch role {
	case "agent":
		return roleAssistant
	case roleAssistant:
		return roleAssistant
	case roleSystem:
		return roleSystem
	default:
		return roleUser
	}
}

// extractSkills extracts skill names from the frontend's declared tools
func (o *Orchestrator) extractSkills(tools []model.AGUITool) []string {
	skills := make([]string, 0, len(tools))
	for _, t := range tools {
		skills = append(skills, t.Name)
	}
	if len(skills) == 0 {
		skills = []string{"code_preview"}
	}
	return skills
}
