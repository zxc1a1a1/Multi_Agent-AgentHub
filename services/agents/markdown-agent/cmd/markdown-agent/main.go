package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/observability"
	markdownagent "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/agents/markdown-agent"
)

func main() {
	shutdownTracing, err := observability.Setup(context.Background(), "markdown-agent")
	if err != nil {
		log.Fatalf("markdown-agent tracing setup failed: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracing(shutdownCtx)
	}()

	addr := strings.TrimSpace(os.Getenv("MARKDOWN_AGENT_ADDR"))
	if addr == "" {
		addr = ":8092"
	}
	publicURL := strings.TrimSpace(os.Getenv("MARKDOWN_AGENT_PUBLIC_URL"))
	handler, _, err := markdownagent.NewHandler(markdownagent.ServerConfig{URL: publicURL})
	if err != nil {
		log.Fatalf("markdown-agent startup failed: %v", err)
	}
	log.Printf("markdown-agent listening on %s (not enabled in compose by default)", addr)
	if err := http.ListenAndServe(addr, observability.Handler("markdown-agent", handler)); err != nil {
		log.Fatalf("markdown-agent server failed: %v", err)
	}
}
