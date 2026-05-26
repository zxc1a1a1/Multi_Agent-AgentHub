package main

import (
	"log"
	"os"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func main() {
	config, err := adk.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load agent config: %v", err)
	}

	port := os.Getenv("SECURITY_AGENT_PORT")
	if port == "" {
		port = "8097"
	}

	llm := adk.NewLLMClient()
	taskHandler := handleTaskWithLLM(llm)
	server := adk.NewA2AServer(config, taskHandler)

	log.Printf("Security-Agent [%s] starting on port %s", config.Name, port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
