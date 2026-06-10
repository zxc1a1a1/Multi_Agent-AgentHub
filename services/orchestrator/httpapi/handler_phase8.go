package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
)

type diffRequest struct {
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
	Path          string `json:"path"`
	DiffText      string `json:"diffText"`
	DryRunID      string `json:"dryRunId,omitempty"`
	Confirmed     bool   `json:"confirmed,omitempty"`
}

type diffResult struct {
	DryRunID string   `json:"dryRunId,omitempty"`
	Status   string   `json:"status"`
	Files    []string `json:"files"`
	Warnings []string `json:"warnings,omitempty"`
	Message  string   `json:"message,omitempty"`
}

type regenerateRequest struct {
	RunID            string         `json:"runId"`
	ConversationID   string         `json:"conversationId"`
	MessageID        string         `json:"messageId"`
	PinnedMessageIDs []string       `json:"pinnedMessageIds,omitempty"`
	Context          []MessageInput `json:"context,omitempty"`
}

type regenerateResponse struct {
	MessageID        string         `json:"messageId"`
	Status           string         `json:"status"`
	PreserveOriginal bool           `json:"preserveOriginal"`
	PinnedMessageIDs []string       `json:"pinnedMessageIds,omitempty"`
	Context          []MessageInput `json:"context,omitempty"`
}

func (s *Server) handleRunRegenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req regenerateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.MessageID = strings.TrimSpace(req.MessageID)
	if req.MessageID == "" {
		writeJSONError(w, http.StatusBadRequest, "messageId is required")
		return
	}
	if len(req.Context) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "context is required for regenerate",
			"code":    "CONTEXT_REQUIRED",
			"message": "Provide the conversation context as a 'context' array of {id,role,text} messages.",
		})
		return
	}

	// Find the target message and validate its role.
	targetIdx := -1
	for i, m := range req.Context {
		if strings.TrimSpace(m.ID) == req.MessageID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":     "message not found in provided context",
			"code":      "MESSAGE_NOT_FOUND",
			"messageId": req.MessageID,
		})
		return
	}

	targetRole := strings.ToLower(strings.TrimSpace(req.Context[targetIdx].Role))
	if targetRole != "assistant" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":     fmt.Sprintf("only assistant messages can be regenerated, got %q", targetRole),
			"code":      "INVALID_TARGET_ROLE",
			"messageId": req.MessageID,
		})
		return
	}

	// Build truncated context: include all messages up to (but NOT including)
	// the target assistant message. The caller uses this context to start a
	// normal chat run, which remains cancellable via the run cancel path.
	truncated := make([]MessageInput, 0, targetIdx)
	for i := 0; i < targetIdx; i++ {
		truncated = append(truncated, req.Context[i])
	}

	writeJSON(w, http.StatusOK, regenerateResponse{
		MessageID:        req.MessageID,
		Status:           "ready",
		PreserveOriginal: true,
		PinnedMessageIDs: sanitizeStringSlice(req.PinnedMessageIDs),
		Context:          truncated,
	})
}

func (s *Server) handleDiffDryRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req diffRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	files, err := validateDiffRequest(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diffResult{
		DryRunID: buildDryRunID(req),
		Status:   "dry_run_ok",
		Files:    files,
		Message:  "diff validated; call apply with confirmed=true and dryRunId",
	})
}

func (s *Server) handleDiffApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req diffRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	files, err := validateDiffRequest(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !req.Confirmed {
		writeJSONError(w, http.StatusBadRequest, "confirmed=true is required before applying a diff")
		return
	}
	expected := buildDryRunID(req)
	if strings.TrimSpace(req.DryRunID) == "" || strings.TrimSpace(req.DryRunID) != expected {
		writeJSONError(w, http.StatusBadRequest, "valid dryRunId from dry-run is required")
		return
	}
	// Safety note: Phase 8 validates the patch and returns an apply contract, but
	// does not perform arbitrary workspace writes in the Orchestrator process.
	// This keeps Gateway/Orchestrator from becoming an unsafe file mutator while
	// preserving the frontend dry-run/apply handshake.
	writeJSON(w, http.StatusOK, diffResult{
		DryRunID: expected,
		Status:   "apply_recorded",
		Files:    files,
		Warnings: []string{"metadata-only apply: no workspace file was modified by orchestrator"},
		Message:  "apply request accepted after dry-run validation",
	})
}

func (s *Server) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		store := s.ensureArtifactStore()
		items, err := store.ListByRun(r.Context(), strings.TrimSpace(r.URL.Query().Get("runId")))
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list artifacts")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": items})
	case http.MethodPost:
		var rec artifacts.ArtifactRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if strings.TrimSpace(rec.ID) == "" {
			rec.ID = fmt.Sprintf("artifact_%d", time.Now().UnixNano())
		}
		if rec.CreatedAt.IsZero() {
			rec.CreatedAt = time.Now().UTC()
		}
		if err := s.ensureArtifactStore().Create(r.Context(), rec); err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to create artifact")
			return
		}
		writeJSON(w, http.StatusCreated, rec)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleArtifactByID(w http.ResponseWriter, r *http.Request) {
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/internal/orchestrator/artifacts/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		rec, ok, err := s.ensureArtifactStore().Get(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to get artifact")
			return
		}
		if !ok {
			writeJSONError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeJSON(w, http.StatusOK, rec)
	case http.MethodDelete:
		if err := s.ensureArtifactStore().Delete(r.Context(), id); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete artifact")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodDelete)
	}
}

func (s *Server) ensureArtifactStore() artifacts.Store {
	if s.artifactStore != nil {
		return s.artifactStore
	}
	path := filepath.Join(".tmp", "orchestrator-artifacts.json")
	store, err := artifacts.NewJSONStore(path)
	if err != nil {
		panic(err)
	}
	s.artifactStore = store
	return store
}

func validateDiffRequest(req diffRequest) ([]string, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, errors.New("path is required")
	}
	if filepath.IsAbs(path) || strings.Contains(path, "..") || strings.Contains(path, "\\") {
		return nil, errors.New("invalid path")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, ".git/") {
		return nil, errors.New("invalid path")
	}
	if !isAllowedDiffPath(clean) {
		return nil, errors.New("path is outside allowlist")
	}
	diffText := strings.TrimSpace(req.DiffText)
	if diffText == "" {
		return nil, errors.New("diffText is required")
	}
	if !strings.Contains(diffText, "@@") || (!strings.Contains(diffText, "---") && !strings.Contains(diffText, "+++")) {
		return nil, errors.New("invalid unified diff")
	}
	return []string{clean}, nil
}

func isAllowedDiffPath(path string) bool {
	allowed := []string{"frontend/src/", "src/", "pkg/", "services/gateway/", "services/orchestrator/", "docs/"}
	for _, prefix := range allowed {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func buildDryRunID(req diffRequest) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(req.Path) + "\n" + strings.TrimSpace(req.DiffText)))
	return "dry_" + hex.EncodeToString(h[:])[:24]
}

func sanitizeStringSlice(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, item := range in {
		v := strings.TrimSpace(item)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
