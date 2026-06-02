package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

)

// OrchestratorRequest is the Gateway→Orchestrator run request.
type OrchestratorRequest struct {
	RunID               string           `json:"runId"`
	ConversationID      string           `json:"conversationId"`
	UserID              string           `json:"userId"`
	ConversationType    string           `json:"conversationType"`
	Messages            []MessageInput   `json:"messages"`
	SelectedAgentNames  []string         `json:"selectedAgentNames"`
	Mentions            []string         `json:"mentions"`
	PlanningMode        string           `json:"planningMode"`
	TraceID             string           `json:"traceId"`
	RequestID           string           `json:"requestId"`
	DeadlineMs          int64            `json:"deadlineMs"`
}

type MessageInput struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// OrchestratorStreamEvent is the internal event sent over SSE.
type OrchestratorStreamEvent struct {
	Type      string         `json:"type"`
	RunID     string         `json:"runId"`
	MessageID string         `json:"messageId,omitempty"`
	Sender    *EventSender   `json:"sender,omitempty"`
	Delta     string         `json:"delta,omitempty"`
	State     map[string]any `json:"state,omitempty"`
	Error     *SafeError     `json:"error,omitempty"`
}

type EventSender struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// SafeError is a sanitized error.
type SafeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
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

	var req OrchestratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	runID := req.RunID
	if runID == "" {
		runID = fmt.Sprintf("run_mock_%d", time.Now().UnixMilli())
	}

	s.writeMockStream(w, runID)
}

func (s *Server) checkServiceAuth(r *http.Request) bool {
	if s == nil {
		return false
	}
	if s.token == "" {
		return true
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return strings.TrimSpace(token) == s.token
}

func (s *Server) writeMockStream(w http.ResponseWriter, runID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	msgID := fmt.Sprintf("msg_mock_%d", time.Now().UnixMilli())

	events := []OrchestratorStreamEvent{
		{
			Type:  "run_started",
			RunID: runID,
			State: map[string]any{"phase": "accepted"},
		},
		{
			Type:      "message_start",
			RunID:     runID,
			MessageID: msgID,
			Sender:    &EventSender{Type: "agent", Name: "orchestrator"},
		},
		{
			Type:      "message_delta",
			RunID:     runID,
			MessageID: msgID,
			Delta:     "This is a mock orchestrator response. Your request has been received.",
		},
		{
			Type:      "message_end",
			RunID:     runID,
			MessageID: msgID,
		},
		{
			Type:  "run_finished",
			RunID: runID,
			State: map[string]any{"status": "completed"},
		},
	}

	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}
}
