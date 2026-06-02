package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/config"
)

// Server is the minimal Orchestrator HTTP server.
type Server struct {
	mux   *http.ServeMux
	token string
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// DefaultConfig exposes the config package default for callers.
func DefaultConfig() config.Config {
	return config.DefaultConfig()
}
