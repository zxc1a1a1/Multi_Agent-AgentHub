package main

import (
	"log"
	"os"

	"github.com/agenthub/agents/adk"
)

func main() {
	// Per adk-runtime-contract section 12: load config from config.yaml
	config, err := adk.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load agent config: %v", err)
	}

	// Determine port
	port := os.Getenv("CODE_AGENT_PORT")
	if port == "" {
		port = "8081"
	}

	// Create A2A server using a2a-go/v2 for protocol compliance
	server := adk.NewA2AServer(config, handleTask)

	log.Printf("Code-Agent [%s] starting on port %s", config.Name, port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
