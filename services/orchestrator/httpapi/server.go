package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/synthesizer"
)

// Server is the minimal Orchestrator HTTP server.
type Server struct {
	mux         *http.ServeMux
	token       string
	registry    *registry.StaticAgentRegistry
	dispatcher  *dispatcher.A2ADispatcher
	planner     planner.Planner // nil means use default RulePlanner
	synthesizer synthesizer.Synthesizer
}

// Option customizes Server behavior.
type Option func(*Server)

// WithInternalToken sets the service-to-service auth token.
func WithInternalToken(token string) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.token = token
	}
}

// WithRegistry injects a static agent registry.
func WithRegistry(r *registry.StaticAgentRegistry) Option {
	return func(s *Server) {
		if s == nil || r == nil {
			return
		}
		s.registry = r
	}
}

// WithDispatcher injects an A2A dispatcher.
func WithDispatcher(d *dispatcher.A2ADispatcher) Option {
	return func(s *Server) {
		if s == nil || d == nil {
			return
		}
		s.dispatcher = d
	}
}

// WithPlanner injects a custom Planner. When nil or not called, the server
// defaults to RulePlanner at request time.
func WithPlanner(p planner.Planner) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.planner = p
	}
}

// WithSynthesizer injects a synthesizer for multi-agent result aggregation.
func WithSynthesizer(syn synthesizer.Synthesizer) Option {
	return func(s *Server) {
		if s == nil {
			return
		}
		s.synthesizer = syn
	}
}

// NewServer returns a Server with routes registered.
func NewServer(opts ...Option) *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}
	s.registerRoutes()
	return s
}

// Handler returns the http.Handler for this server.
func (s *Server) Handler() http.Handler {
	if s == nil || s.mux == nil {
		return http.NotFoundHandler()
	}
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/internal/orchestrator/runs/stream", s.handleRunStream)
	s.mux.HandleFunc("/internal/orchestrator/hitl/confirm", s.handleHITLConfirm)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "orchestrator",
	})
}

// validateReady checks that all required dependencies are wired.
func (s *Server) validateReady() error {
	if s == nil {
		return errors.New("server is nil")
	}
	if s.registry == nil {
		return errors.New("registry is not wired")
	}
	if s.dispatcher == nil {
		return errors.New("dispatcher is not wired")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// DefaultConfig exposes the config package default for callers.
func DefaultConfig() config.Config {
	return config.DefaultConfig()
}
