package adk

import (
	"context"
	"fmt"
	"iter"
	"log"
	"net/http"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

// A2AServer exposes the agent as a standard A2A-compatible HTTP server.
//
// Per adk-runtime-contract section 14 (A2A Server):
//   - Must expose: /health, /.well-known/agent.json, and A2A message endpoints
//   - Per a2a-compatibility rules: MVP uses project compat endpoints
//
// This implementation uses the official a2a-go/v2 library for protocol compliance.
type A2AServer struct {
	config  *AgentConfig
	handler TaskHandler
	mux     *http.ServeMux
}

// codeAgentExecutor implements a2asrv.AgentExecutor
type codeAgentExecutor struct {
	handler TaskHandler
}

// Execute implements a2asrv.AgentExecutor.
// It delegates to our ADK TaskHandler via ExecuteHandler.
func (e *codeAgentExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return ExecuteHandler(ctx, execCtx, e.handler)
}

// Cancel implements a2asrv.AgentExecutor.
func (e *codeAgentExecutor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		cancelEvent := a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil)
		yield(cancelEvent, nil)
	}
}

func buildAgentSkills(skills []string) []a2a.AgentSkill {
	mapped := make([]a2a.AgentSkill, 0, len(skills))
	for _, skill := range skills {
		mapped = append(mapped, a2a.AgentSkill{
			ID:          skill,
			Name:        skill,
			Description: skill,
		})
	}
	return mapped
}

// BuildAgentCard creates an A2A AgentCard from config.
// Per a2a-agent-contract: AgentCard must not expose secrets.
func BuildAgentCard(config *AgentConfig) *a2a.AgentCard {
	card := &a2a.AgentCard{
		Name:               config.Name,
		Description:        config.Description,
		Version:            config.Version,
		Capabilities:       a2a.AgentCapabilities{Streaming: config.Streaming},
		DefaultInputModes:  append([]string(nil), config.InputModes...),
		DefaultOutputModes: append([]string(nil), config.OutputModes...),
		Skills:             buildAgentSkills(config.Skills),
	}

	if config.URL != "" {
		card.SupportedInterfaces = []*a2a.AgentInterface{
			a2a.NewAgentInterface(config.URL, a2a.TransportProtocolJSONRPC),
		}
	}

	return card
}

// NewA2AServer creates a new A2A server using the official a2a-go/v2 library.
//
// Per adk-runtime-contract section 13 (AgentCard):
//   - Must accurately declare capabilities
//   - Must NOT contain: API key, tokens, internal paths, DB connection strings
func NewA2AServer(config *AgentConfig, handler TaskHandler) *A2AServer {
	executor := &codeAgentExecutor{handler: handler}

	// Create the A2A request handler from a2a-go/v2
	a2aHandler := a2asrv.NewHandler(executor)

	// Build AgentCard per a2a-agent-contract
	// Per adk-runtime-contract section 13: AgentCard must not expose secrets
	card := BuildAgentCard(config)

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","agent":"%s"}`, config.Name)
	})

	// AgentCard endpoint per a2a-compatibility rules
	// MVP: /.well-known/agent.json
	mux.Handle("GET /.well-known/agent.json", a2asrv.NewStaticAgentCardHandler(card))

	// A2A JSON-RPC endpoint (the official a2a-go/v2 way)
	mux.Handle("POST /", a2asrv.NewJSONRPCHandler(a2aHandler))

	// Also register the MVP compat endpoint that our Gateway expects
	// Per a2a-compatibility: MVP endpoints are the current landing compat layer
	mux.Handle("POST /a2a/tasks/sendSubscribe", a2asrv.NewJSONRPCHandler(a2aHandler))

	return &A2AServer{
		config:  config,
		handler: handler,
		mux:     mux,
	}
}

// Run starts the A2A server.
func (s *A2AServer) Run(addr string) error {
	log.Printf("A2A server [%s] starting on %s", s.config.Name, addr)
	return http.ListenAndServe(addr, s.mux)
}

// Handler returns the http.Handler for testing.
// Use with httptest.NewServer to test endpoints without binding a real port.
func (s *A2AServer) Handler() http.Handler {
	return s.mux
}
