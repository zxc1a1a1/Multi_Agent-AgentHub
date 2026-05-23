package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenAuth validates Authorization: Bearer <token>.
// If expectedToken is empty, auth is disabled for local MVP development.
func TokenAuth(expectedToken string) gin.HandlerFunc {
	if expectedToken == "" {
		log.Printf("warning: AGENTHUB_API_TOKEN is empty, /api auth is disabled for local development")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		if token == "" || token != expectedToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}
