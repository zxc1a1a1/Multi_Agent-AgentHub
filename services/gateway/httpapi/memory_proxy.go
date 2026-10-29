package httpapi

import (
	"context"
	"io"
	"net/http"
	"strings"
)

// MemoryProxy forwards Memory admin calls to the Orchestrator.
type MemoryProxy interface {
	DoMemory(ctx context.Context, method, path string, body []byte) (int, []byte, error)
}

func (s *Server) handleMemoryProxy(w http.ResponseWriter, r *http.Request) {
	if s.memoryProxy == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "memory proxy unavailable")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, POST, PUT")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	path := "/internal/memory/" + strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/memory/"), "/")
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	status, respBody, err := s.memoryProxy.DoMemory(r.Context(), r.Method, path, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "memory service unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(respBody)
}
