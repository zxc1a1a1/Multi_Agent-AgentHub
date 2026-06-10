package httpapi

import (
	"io"
	"net/http"
	"strings"
)

func (s *Server) handleDiffDryRun(w http.ResponseWriter, r *http.Request) {
	s.proxyPhase8(w, r, "/internal/orchestrator/diffs/dry-run", http.MethodPost)
}

func (s *Server) handleDiffApply(w http.ResponseWriter, r *http.Request) {
	s.proxyPhase8(w, r, "/internal/orchestrator/diffs/apply", http.MethodPost)
}

func (s *Server) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	path := "/internal/orchestrator/artifacts"
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	s.proxyPhase8(w, r, path, r.Method)
}

func (s *Server) handleArtifactByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		writeMethodNotAllowed(w, http.MethodGet, http.MethodDelete)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/artifacts/"), "/")
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	s.proxyPhase8(w, r, "/internal/orchestrator/artifacts/"+id, r.Method)
}

func (s *Server) handleRunsRegenerate(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	path := "/internal/orchestrator/runs/regenerate"
	// Body contains runId too for idempotent server-side validation; keep URL runID
	// checked by Gateway so malformed /api/runs paths do not reach Orchestrator.
	if strings.TrimSpace(runID) == "" {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}
	s.proxyPhase8(w, r, path, http.MethodPost)
}

func (s *Server) proxyPhase8(w http.ResponseWriter, r *http.Request, internalPath string, method string) {
	proxy, ok := s.runner.(InternalOrchestratorProxy)
	if !ok {
		writeJSONError(w, http.StatusNotImplemented, "orchestrator proxy not configured")
		return
	}
	if r.Method != method {
		writeMethodNotAllowed(w, method)
		return
	}
	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}
	resp, err := proxy.ProxyInternal(r.Context(), method, internalPath, body, r.Header.Get("Content-Type"))
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "orchestrator proxy failed")
		return
	}
	defer resp.Body.Close()
	for k, values := range resp.Header {
		if strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Transfer-Encoding") {
			continue
		}
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
