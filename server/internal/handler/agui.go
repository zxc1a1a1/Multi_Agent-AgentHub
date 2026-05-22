package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

func (h *Handler) HandleAGUIRun(c *gin.Context) {
	var req model.AGUIRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	history, _ := h.db.GetMessages(req.ThreadID, 20)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	eventChan := make(chan model.AGUIEvent, 100)
	go func() {
		defer close(eventChan)
		h.orc.Process(c.Request.Context(), req, history, eventChan)
	}()

	c.Stream(func(w io.Writer) bool {
		event, ok := <-eventChan
		if !ok {
			return false
		}
		data, _ := json.Marshal(event)
		c.SSEvent("message", string(data))
		return true
	})
}
