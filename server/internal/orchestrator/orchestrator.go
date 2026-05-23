package orchestrator

import (
	"context"
	"log"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/google/uuid"
	a2aclient "github.com/your-org/multi-agent-framework/server/internal/a2a"
	"github.com/your-org/multi-agent-framework/server/internal/config"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

// Orchestrator routes requests to the appropriate agent and handles protocol conversion.
// Per gateway-orchestrator-contract: embedded in Gateway process for MVP.
type Orchestrator struct {
	a2aClient *a2aclient.Client
	agents    map[string]config.AgentConfig
}

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

	// 3. Build user message from history + current request
	userMessage := o.buildUserMessage(req, history)

	// 4. Extract frontend skills
	skills := o.extractSkills(req.Tools)

	// 5. Create protocol converter
	converter := NewConverter(skills)

	// 6. Call agent via official a2a-go/v2 client
	eventIter, err := o.a2aClient.SendStreamingMessage(ctx, agentCfg.URL, userMessage)
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

// buildUserMessage constructs the full user message from history + request
func (o *Orchestrator) buildUserMessage(req model.AGUIRunRequest, history []model.Message) string {
	var parts []string

	// Add history context
	for _, h := range history {
		role := "User"
		if h.SenderType == "agent" {
			role = "Assistant"
		}
		parts = append(parts, role+": "+h.Content)
	}

	// Add current message
	for _, m := range req.Messages {
		parts = append(parts, m.Content)
	}

	return strings.Join(parts, "\n\n")
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

// Helper to get text from a2a event artifacts/messages
func extractTextFromEvent(event a2a.Event) string {
	switch e := event.(type) {
	case *a2a.TaskArtifactUpdateEvent:
		if e.Artifact != nil {
			for _, part := range e.Artifact.Parts {
				if text, ok := part.Content.(a2a.Text); ok {
					return string(text)
				}
			}
		}
	case *a2a.Message:
		for _, part := range e.Parts {
			if text, ok := part.Content.(a2a.Text); ok {
				return string(text)
			}
		}
	}
	return ""
}
