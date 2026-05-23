package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/handler"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/middleware"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/orchestrator"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := store.NewMySQL(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	a2aClient := a2a.NewClient()
	orc := orchestrator.New(a2aClient, cfg.Agents)

	r := gin.Default()
	r.Use(cors.Default())

	// Health endpoint — no auth, per docker-compose-delivery healthcheck policy
	r.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "unhealthy", "error": "db unreachable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	h := handler.New(db, orc)
	api := r.Group("/api")
	api.Use(middleware.TokenAuth(cfg.APIToken))
	{
		api.GET("/conversations", h.ListConversations)
		api.POST("/conversations", h.CreateConversation)
		api.GET("/conversations/:id/messages", h.ListMessages)
		api.GET("/agents", h.ListAgents)
		api.POST("/agui/run", h.HandleAGUIRun)
	}

	log.Printf("server starting on :%s", cfg.Port)
	r.Run(":" + cfg.Port)
}
