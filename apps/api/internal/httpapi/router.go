package httpapi

import (
	"net/http"

	"github.com/your-org/multi-agent-framework/apps/api/internal/config"
	"github.com/your-org/multi-agent-framework/apps/api/internal/service"
)

// Router wires HTTP routes to application services.
type Router struct {
	mux    *http.ServeMux
	cfg    config.Config
	agents *service.AgentService
}

// NewRouter creates the HTTP handler for the API server.
func NewRouter(cfg config.Config) http.Handler {
	r := &Router{
		mux:    http.NewServeMux(),
		cfg:    cfg,
		agents: service.NewAgentService(),
	}
	r.routes()
	return r.withCORS(r.mux)
}

func (r *Router) routes() {
	r.mux.HandleFunc("GET /healthz", r.handleHealthz)
	r.mux.HandleFunc("GET /.well-known/agent-card.json", r.handleGetAgentCard)
	r.mux.HandleFunc("POST /api/v1/agent-runs", r.handleCreateAgentRun)
	r.mux.HandleFunc("POST /api/v1/agent-runs:stream", r.handleStreamAgentRun)
}

func (r *Router) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if origin == r.cfg.WebOrigin || origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", r.cfg.WebOrigin)
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, req)
	})
}
