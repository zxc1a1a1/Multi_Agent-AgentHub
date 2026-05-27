// Package gateway defines the standalone Gateway module scaffold.
//
// Gateway is the frontend entry, protocol bridge, SSE output boundary, and
// conversation/message persistence boundary. It does not perform orchestration
// and does not implement Planner/Executor. Runtime execution is delegated to an
// injected RunService.
package gateway

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/auth"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/config"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/httpapi"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// RunService delegates chat execution to an injected runtime service.
type RunService = httpapi.RunService

// Gateway wires HTTP API + middleware without embedding orchestration logic.
type Gateway struct {
	cfg    config.Config
	api    *httpapi.Server
	auth   *auth.Middleware
	cors   *auth.CORS
	handle http.Handler
}

// New builds a Gateway scaffold with minimal auth/cors/httpapi capabilities.
func New(cfg config.Config, st store.Store, runner RunService, opts ...httpapi.Option) (*Gateway, error) {
	if strings.TrimSpace(cfg.Addr) == "" {
		cfg.Addr = config.DefaultConfig().Addr
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	api, err := httpapi.NewServer(st, runner, opts...)
	if err != nil {
		return nil, err
	}

	authMiddleware, err := auth.New(cfg.EnableAuth, cfg.AuthToken)
	if err != nil {
		return nil, err
	}
	corsMiddleware := auth.NewCORS(cfg.AllowedOrigins)

	handler := corsMiddleware.Wrap(authMiddleware.Wrap(api.Handler()))
	if handler == nil {
		return nil, fmt.Errorf("gateway handler is nil")
	}

	return &Gateway{
		cfg:    cfg,
		api:    api,
		auth:   authMiddleware,
		cors:   corsMiddleware,
		handle: handler,
	}, nil
}

// Handler returns the gateway HTTP handler chain.
func (g *Gateway) Handler() http.Handler {
	if g == nil {
		return http.NotFoundHandler()
	}
	return g.handle
}
