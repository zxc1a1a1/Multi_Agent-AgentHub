package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	webagent "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/agents/web-agent"
)

const (
	defaultAddr      = ":8080"
	defaultPublicURL = "http://localhost:8080"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("web-agent server failed: %v", err)
	}
}

func run() error {
	addr := strings.TrimSpace(os.Getenv("WEB_AGENT_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	publicURL := strings.TrimSpace(os.Getenv("WEB_AGENT_PUBLIC_URL"))
	handler, err := buildHandler(publicURL)
	if err != nil {
		return err
	}

	return http.ListenAndServe(addr, handler)
}

func buildHandler(publicURL string) (http.Handler, error) {
	resolvedURL := strings.TrimSpace(publicURL)
	if resolvedURL == "" {
		resolvedURL = defaultPublicURL
	}

	handler, _, err := webagent.NewHandler(webagent.ServerConfig{
		URL: resolvedURL,
	})
	if err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("web-agent handler is nil")
	}
	return handler, nil
}
