package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

// HandleAGUIRun is the core SSE endpoint for AG-UI protocol
// It receives a run request, calls the orchestrator, and streams events back
func (h *Handler) HandleAGUIRun(c *gin.Context) {
	var req model.AGUIRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save the user's latest message to DB
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			_ = h.db.SaveMessage(req.ThreadID, "user", "", lastMsg.Content, nil)
		}
	}

	// Load conversation history for context
	history, _ := h.db.GetMessages(req.ThreadID, 20)

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

	// Stream events as SSE (flush after each event for real-time delivery)
	c.Stream(func(w io.Writer) bool {
		event, ok := <-eventChan
		if !ok {
			return false
		}
		data, _ := json.Marshal(event)
		// Write SSE format compatible with frontend parser
		fmt.Fprintf(w, "data: %s\n\n", data)
		c.Writer.Flush()
		return true
	})
}
