package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

const (
	defaultAddr           = ":8080"
	defaultCodeAgentURL   = "http://127.0.0.1:8081"
	defaultWebAgentURL    = "http://127.0.0.1:8082"
	defaultShutdown       = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("orchestrator startup failed: %v", err)
	}
}

func run() error {
	addr := strings.TrimSpace(os.Getenv("ORCHESTRATOR_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	internalToken := strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_TOKEN"))

	codeAgentURL := strings.TrimSpace(os.Getenv("CODE_AGENT_URL"))
	if codeAgentURL == "" {
		codeAgentURL = defaultCodeAgentURL
	}

	webAgentURL := strings.TrimSpace(os.Getenv("WEB_AGENT_URL"))
	if webAgentURL == "" {
		webAgentURL = defaultWebAgentURL
	}

	cfg := config.Config{
		Addr:          addr,
		InternalToken: internalToken,
		CodeAgentURL:  codeAgentURL,
		WebAgentURL:   webAgentURL,
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Build static agent registry from env-configured agent URLs.
	agentRegistry, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{
			Name:        "code-agent",
			URL:         codeAgentURL,
			Description: "Generates and explains code",
			OutputModes: []string{"text", "code", "artifact_ref"},
		},
		{
			Name:        "web-agent",
			URL:         webAgentURL,
			Description: "Generates webpages and HTML previews",
			OutputModes: []string{"text", "webpage", "html", "artifact_ref"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create agent registry: %w", err)
	}

	a2aDispatcher := dispatcher.NewA2ADispatcher()

	server := &http.Server{
		Addr: cfg.Addr,
		Handler: httpapi.NewServer(
			httpapi.WithInternalToken(internalToken),
			httpapi.WithRegistry(agentRegistry),
			httpapi.WithDispatcher(a2aDispatcher),
		).Handler(),
	}

	go func() {
		log.Printf("orchestrator listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("orchestrator http server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %s, shutting down orchestrator", sig)

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdown)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	log.Printf("orchestrator shutdown complete")
	return nil
}
