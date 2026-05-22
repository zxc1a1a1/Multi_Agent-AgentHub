package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/your-org/multi-agent-framework/server/internal/config"
	"github.com/your-org/multi-agent-framework/server/internal/handler"
	"github.com/your-org/multi-agent-framework/server/internal/orchestrator"
	"github.com/your-org/multi-agent-framework/server/internal/a2a"
	"github.com/your-org/multi-agent-framework/server/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := store.NewMySQL(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	a2aClient := a2a.NewClient()
	orc := orchestrator.New(a2aClient, cfg.Agents)

	r := gin.Default()
	r.Use(cors.Default())

	h := handler.New(db, orc)
	api := r.Group("/api")
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
