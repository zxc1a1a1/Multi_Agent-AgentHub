package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListConversations(c *gin.Context) {
	c.JSON(http.StatusOK, []interface{}{})
}

func (h *Handler) CreateConversation(c *gin.Context) {
	c.JSON(http.StatusCreated, nil)
}

func (h *Handler) ListMessages(c *gin.Context) {
	c.JSON(http.StatusOK, []interface{}{})
}

func (h *Handler) ListAgents(c *gin.Context) {
	c.JSON(http.StatusOK, []interface{}{})
}
