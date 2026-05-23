package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/multi-agent-framework/server/internal/model"
)

// ListConversations returns all conversations
func (h *Handler) ListConversations(c *gin.Context) {
	convs, err := h.db.ListConversations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if convs == nil {
		convs = []model.Conversation{}
	}
	c.JSON(http.StatusOK, convs)
}

type createConvRequest struct {
	Title     string `json:"title"`
	AgentName string `json:"agentName"`
}

// CreateConversation creates a new conversation
func (h *Handler) CreateConversation(c *gin.Context) {
	var req createConvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AgentName == "" {
		req.AgentName = "code-agent"
	}
	if req.Title == "" {
		req.Title = "New Conversation"
	}

	conv, err := h.db.CreateConversation(req.Title, req.AgentName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusCreated, conv)
}

// ListMessages returns messages for a conversation
func (h *Handler) ListMessages(c *gin.Context) {
	convID := c.Param("id")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	msgs, err := h.db.GetMessages(convID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if msgs == nil {
		msgs = []model.Message{}
	}
	c.JSON(http.StatusOK, msgs)
}

// ListAgents returns available agents (MVP: hardcoded code-agent)
func (h *Handler) ListAgents(c *gin.Context) {
	agents := []gin.H{
		{
			"name":        "code-agent",
			"description": "Generates, refactors, and reviews code.",
		},
	}
	c.JSON(http.StatusOK, agents)
}
