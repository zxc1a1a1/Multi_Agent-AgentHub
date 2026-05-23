package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

// HandleAGUIRun is the core SSE endpoint for AG-UI protocol
// It receives a run request, calls the orchestrator, and streams events back
func (h *Handler) HandleAGUIRun(c *gin.Context) {
	var req model.AGUIRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("invalid agui run request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Save the user's latest message to DB
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			if err := h.db.SaveMessage(req.ThreadID, "user", "", lastMsg.Content, nil); err != nil {
				log.Printf("failed to save user message: %v", err)
			}
		}
	}

	// Load conversation history for context
	history, err := h.db.GetMessages(req.ThreadID, 20)
	if err != nil {
		log.Printf("failed to load conversation history: %v", err)
		history = nil
	}

	// Set SSE response headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create event channel for orchestrator to push AG-UI events
	eventChan := make(chan model.AGUIEvent, 100)
	go func() {
		defer close(eventChan)
		h.orc.Process(c.Request.Context(), req, history, eventChan)
	}()

	// Accumulate agent response during stream for DB persistence
	var agentText string
	var agentArtifacts []model.ArtifactData

	// Stream events as SSE (flush after each event for real-time delivery)
	c.Stream(func(w io.Writer) bool {
		event, ok := <-eventChan
		if !ok {
			return false
		}

		// Accumulate agent text from TEXT_MESSAGE_CONTENT events
		if event.Type == "TEXT_MESSAGE_CONTENT" {
			agentText += event.Content
		}

		// Parse TOOL_CALL_ARGS to collect code artifact data
		if event.Type == "TOOL_CALL_ARGS" {
			var args map[string]string
			if err := json.Unmarshal([]byte(event.Content), &args); err == nil {
				metadata := make(map[string]string)
				if lang, ok := args["language"]; ok && lang != "" {
					metadata["language"] = lang
				}
				if fn, ok := args["filename"]; ok && fn != "" {
					metadata["filename"] = fn
				}
				agentArtifacts = append(agentArtifacts, model.ArtifactData{
					Type:     "code",
					Title:    args["filename"],
					Content:  args["code"],
					Metadata: metadata,
				})
			}
		}

		data, _ := json.Marshal(event)
		// Write SSE format compatible with frontend parser
		fmt.Fprintf(w, "data: %s\n\n", data)
		c.Writer.Flush()
		return true
	})

	// Persist agent message after SSE stream completes
	if agentText != "" || len(agentArtifacts) > 0 {
		if err := h.db.SaveMessage(req.ThreadID, "agent", "", agentText, agentArtifacts); err != nil {
			log.Printf("failed to save agent message: %v", err)
		}
	}
}
