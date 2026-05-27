package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Middleware struct {
	enabled bool
	token   string
}

func New(enabled bool, token string) (*Middleware, error) {
	if enabled && strings.TrimSpace(token) == "" {
		return nil, errors.New("auth token is required when auth is enabled")
	}
	return &Middleware{
		enabled: enabled,
		token:   token,
	}, nil
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		header := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			writeUnauthorized(w)
			return
		}
		incoming := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if incoming == "" {
			writeUnauthorized(w)
			return
		}

		if subtle.ConstantTimeCompare([]byte(incoming), []byte(m.token)) != 1 {
			writeUnauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type CORS struct {
	allowedOrigins []string
}

func NewCORS(origins []string) *CORS {
	copied := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		copied = append(copied, origin)
	}
	return &CORS{allowedOrigins: copied}
}

func (c *CORS) Wrap(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		allowedOrigin := c.matchOrigin(origin)
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (c *CORS) matchOrigin(origin string) string {
	if c == nil || origin == "" || len(c.allowedOrigins) == 0 {
		return ""
	}
	for _, allowed := range c.allowedOrigins {
		if allowed == "*" {
			return "*"
		}
		if origin == allowed {
			return origin
		}
	}
	return ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "unauthorized",
	})
}
