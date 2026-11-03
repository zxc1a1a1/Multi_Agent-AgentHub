package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	memctl "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory"
)

type memoryScopeDTO struct {
	Env         string `json:"env"`
	WorkspaceID string `json:"workspaceId"`
	PrincipalID string `json:"principalId"`
	SessionID   string `json:"sessionId"`
	AgentName   string `json:"agentName"`
	ThreadID    string `json:"threadId"`
}

func (d memoryScopeDTO) Scope() rtmemory.Scope {
	return rtmemory.Scope{Env: d.Env, WorkspaceID: d.WorkspaceID, PrincipalID: d.PrincipalID,
		SessionID: d.SessionID, AgentName: d.AgentName, ThreadID: d.ThreadID}
}

func memoryScopeFromQuery(r *http.Request) rtmemory.Scope {
	q := r.URL.Query()
	return rtmemory.Scope{Env: q.Get("env"), WorkspaceID: q.Get("workspaceId"),
		PrincipalID: q.Get("principalId"), SessionID: q.Get("sessionId"),
		AgentName: q.Get("agentName"), ThreadID: q.Get("threadId")}
}

// handleMemory exposes the trusted MemoryController admin surface. Gateway is
// the only caller; scope fields come from the authenticated request context.
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	s.memoryMu.RLock()
	controller := s.memoryController
	s.memoryMu.RUnlock()
	if controller == nil || s.memoryRepo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "memory unavailable"})
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/internal/memory/"), "/")

	switch {
	case path == "policy" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, controller.Policy())
	case path == "policy" && r.Method == http.MethodPut:
		var policy rtmemory.Policy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid policy"})
			return
		}
		if err := policy.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		next, err := memctl.New(s.memoryBackend, policy)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.memoryMu.Lock()
		s.memoryController = next
		s.memoryMu.Unlock()
		writeJSON(w, http.StatusOK, policy)
	case path == "entries" && r.Method == http.MethodGet:
		scopeID, err := controller.ScopeID(r.Context(), memoryScopeFromQuery(r))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		entries, err := s.memoryRepo.ListEntries(r.Context(), scopeID, r.URL.Query().Get("status"), limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "memory read failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
	case strings.HasPrefix(path, "mutations/") && r.Method == http.MethodGet:
		mutation, err := s.memoryRepo.GetMutation(r.Context(), strings.TrimPrefix(path, "mutations/"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "mutation not found"})
			return
		}
		writeJSON(w, http.StatusOK, mutation)
	case strings.HasPrefix(path, "mutations/") && strings.HasSuffix(path, "/retry") && r.Method == http.MethodPost:
		mutationID := strings.TrimSuffix(strings.TrimPrefix(path, "mutations/"), "/retry")
		if mutationID == "" || strings.Contains(mutationID, "/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "mutation not found"})
			return
		}
		if err := s.memoryRepo.RetryMutation(r.Context(), mutationID); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "mutation not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mutationId": mutationID, "purgeStatus": "pending"})
	case r.Method == http.MethodPost:
		s.handleMemoryMutation(w, r, path, controller)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleMemoryMutation(w http.ResponseWriter, r *http.Request, path string, controller *memctl.Controller) {
	switch path {
	case "propose":
		var req struct {
			Scope memoryScopeDTO `json:"scope"`
			Entry struct {
				Kind         string     `json:"kind"`
				LogicalKey   string     `json:"logicalKey"`
				Content      string     `json:"content"`
				SourceRefs   []string   `json:"sourceRefs"`
				ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
				Core         bool       `json:"core,omitempty"`
				RecheckAfter *time.Time `json:"recheckAfter,omitempty"`
			} `json:"entry"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		entry, err := controller.Propose(r.Context(), req.Scope.Scope(), rtmemory.Entry{
			Kind: req.Entry.Kind, LogicalKey: req.Entry.LogicalKey, Content: req.Entry.Content,
			SourceRefs: req.Entry.SourceRefs, ExpiresAt: req.Entry.ExpiresAt,
			Core: req.Entry.Core, RecheckAfter: req.Entry.RecheckAfter,
		})
		if err != nil {
			writeMemoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, entry)
	case "forget-scope":
		var req struct {
			Scope memoryScopeDTO `json:"scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		count, err := controller.ForgetScope(r.Context(), req.Scope.Scope())
		if err != nil {
			writeMemoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"revokedCount": count, "recallDisabled": true, "purgeStatus": "pending"})
	case "confirm", "update", "forget":
		var req struct {
			Scope            memoryScopeDTO `json:"scope"`
			MemoryID         string         `json:"memoryId"`
			ExpectedRevision int            `json:"expectedRevision"`
			Content          string         `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		scope := req.Scope.Scope()
		switch path {
		case "confirm":
			entry, err := controller.Confirm(r.Context(), scope, req.MemoryID, req.ExpectedRevision)
			if err != nil {
				writeMemoryError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, entry)
		case "update":
			entry, err := controller.Update(r.Context(), scope, req.MemoryID, req.Content, req.ExpectedRevision)
			if err != nil {
				writeMemoryError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, entry)
		case "forget":
			if err := controller.Forget(r.Context(), scope, req.MemoryID); err != nil {
				writeMemoryError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"memoryId": req.MemoryID, "recallDisabled": true, "purgeStatus": "pending"})
		}
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func writeMemoryError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if strings.Contains(err.Error(), "denied") {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]string{"error": "memory operation failed"})
}
