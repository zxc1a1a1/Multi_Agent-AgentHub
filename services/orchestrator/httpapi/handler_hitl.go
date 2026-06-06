package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// HITLConfirmRequest is the Frontend→Gateway→Orchestrator confirmation payload.
type HITLConfirmRequest struct {
	RunID        string `json:"runId"`
	ActionID     string `json:"actionId"`
	Confirmed    bool   `json:"confirmed"`
	RejectReason string `json:"rejectReason"`
}

// handleHITLConfirm receives HITL confirmation responses from the Gateway.
// POST /internal/orchestrator/hitl/confirm
func (s *Server) handleHITLConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req HITLConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if strings.TrimSpace(req.RunID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "runId is required",
		})
		return
	}

	if strings.TrimSpace(req.ActionID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "actionId is required",
		})
		return
	}

	// TODO: route the confirmation to the waiting execution goroutine.
	// In the full implementation, the executor maintains a map of
	// runId → chan HITLConfirmRequest and signals the waiting goroutine.
	// For now, acknowledge receipt so the frontend flow works.
	_ = req.Confirmed
	_ = req.RejectReason
	_ = plan.StrategySingle // keep plan import

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "acknowledged",
	})
}
