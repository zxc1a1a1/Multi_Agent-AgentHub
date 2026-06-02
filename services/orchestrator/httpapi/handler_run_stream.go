package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
)

// OrchestratorRequest is the Gateway→Orchestrator run request.
type OrchestratorRequest struct {
	RunID              string         `json:"runId"`
	ConversationID     string         `json:"conversationId"`
	UserID             string         `json:"userId"`
	ConversationType   string         `json:"conversationType"`
	Messages           []MessageInput `json:"messages"`
	AgentName          string         `json:"agentName,omitempty"`
	SelectedAgentNames []string       `json:"selectedAgentNames"`
	Mentions           []string       `json:"mentions"`
	PlanningMode       string         `json:"planningMode"`
	TraceID            string         `json:"traceId"`
	RequestID          string         `json:"requestId"`
	DeadlineMs         int64          `json:"deadlineMs"`
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
		runID = fmt.Sprintf("run_%d", time.Now().UnixMilli())
	}
	convID := req.ConversationID
	if convID == "" {
		convID = fmt.Sprintf("conv_%d", time.Now().UnixMilli())
	}

	// Select target agent: use agentName from request, default to code-agent.
	targetName := strings.TrimSpace(req.AgentName)
	if targetName == "" && len(req.SelectedAgentNames) > 0 {
		targetName = strings.TrimSpace(req.SelectedAgentNames[0])
	}
	if targetName == "" {
		targetName = "code-agent"
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErrorEvent(w, runID, "ORCHESTRATOR_INTERNAL", "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	msgID := fmt.Sprintf("msg_%d", time.Now().UnixMilli())

	// Emit run_started.
	s.emitEvent(w, flusher, OrchestratorStreamEvent{
		Type:  "run_started",
		RunID: runID,
		State: map[string]any{"phase": "dispatching"},
	})

	// Resolve agent URL from registry.
	endpoint, ok := s.registry.Get(targetName)
	if !ok {
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_error",
			RunID: runID,
			Error: &SafeError{
				Code:    "ORCHESTRATOR_AGENT_UNAVAILABLE",
				Message: "Requested agent is not available: " + sanitizeForError(targetName),
			},
		})
		return
	}

	// Extract user text from messages.
	userText := extractUserText(req.Messages)
	if userText == "" {
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_error",
			RunID: runID,
			Error: &SafeError{
				Code:    "ORCHESTRATOR_BAD_REQUEST",
				Message: "Message content is required",
			},
		})
		return
	}

	// Emit message_start.
	s.emitEvent(w, flusher, OrchestratorStreamEvent{
		Type:      "message_start",
		RunID:     runID,
		MessageID: msgID,
		Sender:    &EventSender{Type: "agent", Name: targetName},
	})

	// Dispatch to the remote agent via A2A.
	input := dispatcher.DispatchInput{
		AgentURL:       endpoint.URL,
		AgentName:      targetName,
		ConversationID: convID,
		RunID:          runID,
		Message:        userText,
	}

	result, err := s.dispatcher.Dispatch(r.Context(), input)
	if err != nil {
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:  "run_error",
			RunID: runID,
			Error: &SafeError{
				Code:    "ORCHESTRATOR_AGENT_FAILED",
				Message: "Agent execution failed",
			},
		})
		return
	}

	// Emit message_delta with the response text.
	if result != nil && result.Text != "" {
		s.emitEvent(w, flusher, OrchestratorStreamEvent{
			Type:      "message_delta",
			RunID:     runID,
			MessageID: msgID,
			Sender:    &EventSender{Type: "agent", Name: targetName},
			Delta:     result.Text,
		})
	}

	// Emit message_end.
	s.emitEvent(w, flusher, OrchestratorStreamEvent{
		Type:      "message_end",
		RunID:     runID,
		MessageID: msgID,
		Sender:    &EventSender{Type: "agent", Name: targetName},
	})

	// Emit run_finished.
	s.emitEvent(w, flusher, OrchestratorStreamEvent{
		Type:  "run_finished",
		RunID: runID,
		State: map[string]any{"status": "completed"},
	})
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

func (s *Server) emitEvent(w http.ResponseWriter, flusher http.Flusher, event OrchestratorStreamEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}

func extractUserText(messages []MessageInput) string {
	var texts []string
	for _, m := range messages {
		if strings.EqualFold(strings.TrimSpace(m.Role), "user") {
			t := strings.TrimSpace(m.Text)
			if t != "" {
				texts = append(texts, t)
			}
		}
	}
	return strings.Join(texts, "\n")
}

func writeErrorEvent(w http.ResponseWriter, runID string, code, message string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(OrchestratorStreamEvent{
		Type:  "run_error",
		RunID: runID,
		Error: &SafeError{Code: code, Message: message},
	})
	fmt.Fprintf(w, "event: run_error\ndata: %s\n\n", payload)
}

func sanitizeForError(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}
