package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi"
)

const (
	defaultAddr    = ":8080"
	defaultShutdown = 10 * time.Second
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

	cfg := config.Config{Addr: addr}
	if err := cfg.Validate(); err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: httpapi.NewServer().Handler(),
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
