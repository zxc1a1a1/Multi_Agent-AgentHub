package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/your-org/multi-agent-framework/apps/api/pkg/protocol/a2a"
	"github.com/your-org/multi-agent-framework/apps/api/pkg/protocol/a2ui"
	"github.com/your-org/multi-agent-framework/apps/api/pkg/protocol/agui"
)

type createAgentRunRequest struct {
	Message string `json:"message"`
}

type createAgentRunResponse struct {
	Answer  string       `json:"answer"`
	Surface a2ui.Surface `json:"surface"`
}

func (r *Router) handleHealthz(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) handleCreateAgentRun(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeCreateAgentRunRequest(w, req)
	if !ok {
		return
	}

	result := r.agents.Run(req.Context(), body.Message)
	writeJSON(w, http.StatusOK, createAgentRunResponse{
		Answer:  result.Text,
		Surface: result.Surface,
	})
}

func (r *Router) handleGetAgentCard(w http.ResponseWriter, req *http.Request) {
	card := a2a.AgentCard{
		Name:        "go-react-root-agent",
		Description: "Root agent for React + Go multi-agent framework starter.",
		URL:         "http://localhost:8080/api/v1/a2a",
		Version:     "0.1.0",
		Skills: []a2a.Skill{
			{ID: "chat", Name: "Chat", Description: "Answer user questions and orchestrate local sub-agents."},
			{ID: "ui", Name: "A2UI", Description: "Return declarative UI surfaces for frontend rendering."},
		},
	}
	writeJSON(w, http.StatusOK, card)
}

func (r *Router) handleStreamAgentRun(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeCreateAgentRunRequest(w, req)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "streaming is not supported by this server", nil)
		return
	}

	send := func(event agui.Event) {
		payload, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}

	runID := fmt.Sprintf("run_%d", time.Now().UnixNano())
	messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	send(agui.Event{Type: agui.EventRunStarted, RunID: runID})
	send(agui.Event{Type: agui.EventTextMessageStart, RunID: runID, MessageID: messageID, Role: "assistant"})

	result := r.agents.Run(req.Context(), body.Message)
	for _, token := range chunkText(result.Text, 14) {
		select {
		case <-req.Context().Done():
			return
		default:
		}
		send(agui.Event{Type: agui.EventTextMessageContent, RunID: runID, MessageID: messageID, Delta: token})
		time.Sleep(80 * time.Millisecond)
	}

	send(agui.Event{Type: agui.EventTextMessageEnd, RunID: runID, MessageID: messageID})
	send(agui.Event{Type: agui.EventA2UISurface, RunID: runID, Surface: result.Surface})
	send(agui.Event{Type: agui.EventRunFinished, RunID: runID})
}

func decodeCreateAgentRunRequest(w http.ResponseWriter, req *http.Request) (createAgentRunRequest, bool) {
	var body createAgentRunRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", "request body must be valid JSON", map[string]any{"field": "body"})
		return body, false
	}

	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		writeError(w, http.StatusBadRequest, "invalid_argument", "message is required", map[string]any{"field": "message"})
		return body, false
	}

	return body, true
}

func chunkText(s string, size int) []string {
	if size <= 0 || len(s) <= size {
		return []string{s}
	}
	chunks := make([]string, 0, len(s)/size+1)
	runes := []rune(s)
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
