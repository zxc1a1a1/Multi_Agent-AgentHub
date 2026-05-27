package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/sse"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// Store aliases the gateway persistence interface.
type Store = store.Store

// RunService is the injected runtime execution contract.
type RunService interface {
	Run(ctx context.Context, conversationID string, userContent *adk.Content) iter.Seq2[adk.Event, error]
}

type Option func(*Server)

// WithTranslator overrides the default AG-UI translator.
func WithTranslator(t *agui.Translator) Option {
	return func(s *Server) {
		if s == nil || t == nil {
			return
		}
		s.translator = t
	}
}

type Server struct {
	store      Store
	runner     RunService
	translator *agui.Translator
	mux        *http.ServeMux
}

func NewServer(st Store, runner RunService, opts ...Option) (*Server, error) {
	if st == nil {
		return nil, errors.New("store is required")
	}
	if runner == nil {
		return nil, errors.New("runner is required")
	}

	s := &Server{
		store:      st,
		runner:     runner,
		translator: agui.NewTranslator(),
		mux:        http.NewServeMux(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}
	if s.translator == nil {
		s.translator = agui.NewTranslator()
	}

	s.registerRoutes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	if s == nil || s.mux == nil {
		return http.NotFoundHandler()
	}
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/conversations", s.handleConversations)
	s.mux.HandleFunc("/api/conversations/", s.handleConversationMessages)
	s.mux.HandleFunc("/api/chat", s.handleChat)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "gateway",
	})
}

func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListConversations(w, r)
	case http.MethodPost:
		s.handleCreateConversation(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string `json:"userId"`
		AgentName string `json:"agentName"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.AgentName = strings.TrimSpace(req.AgentName)
	if req.UserID == "" || req.AgentName == "" {
		writeJSONError(w, http.StatusBadRequest, "userId and agentName are required")
		return
	}

	conv, err := s.store.CreateConversation(r.Context(), req.UserID, req.AgentName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	conversations, err := s.store.ListConversations(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

func (s *Server) handleConversationMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	conversationID, ok := extractConversationID(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	messages, err := s.store.ListMessages(r.Context(), conversationID)
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
		Message        string `json:"message"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Message = strings.TrimSpace(req.Message)
	if req.ConversationID == "" {
		writeJSONError(w, http.StatusBadRequest, "conversationId is required")
		return
	}
	if req.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	if _, err := s.store.GetConversation(r.Context(), req.ConversationID); err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			writeJSONError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load conversation")
		return
	}

	if _, err := s.store.AppendMessage(r.Context(), store.Message{
		ConversationID: req.ConversationID,
		Author:         "user",
		Role:           string(adk.RoleUser),
		Text:           req.Message,
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to append user message")
		return
	}

	sse.SetHeaders(w)
	writer := sse.NewWriter(w)
	ctx := r.Context()
	assistantText := strings.Builder{}

	seq := s.runner.Run(ctx, req.ConversationID, &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: req.Message},
		},
	})

	var streamErr error
	seq(func(event adk.Event, eventErr error) bool {
		if eventErr != nil {
			streamErr = eventErr
			return false
		}
		mapped := s.translator.Translate(event)
		for _, item := range mapped {
			if item.Role == string(adk.RoleAssistant) && (item.Type == "message" || item.Type == "message.delta") && item.Text != "" {
				assistantText.WriteString(item.Text)
			}
			if err := writer.WriteEvent(ctx, item); err != nil {
				streamErr = err
				return false
			}
		}
		return true
	})

	if streamErr != nil {
		_ = writer.WriteError(ctx, "runner_error", "assistant run failed")
		return
	}

	text := strings.TrimSpace(assistantText.String())
	if text == "" {
		return
	}
	_, _ = s.store.AppendMessage(ctx, store.Message{
		ConversationID: req.ConversationID,
		Author:         "assistant",
		Role:           string(adk.RoleAssistant),
		Text:           text,
	})
}

func extractConversationID(path string) (string, bool) {
	const prefix = "/api/conversations/"
	const suffix = "/messages"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}

	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, suffix)
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func decodeJSON(r *http.Request, out any) error {
	if r == nil || r.Body == nil {
		return fmt.Errorf("empty request")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("unexpected trailing json")
	}
	return nil
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	filter := agui.NewTextStreamFilter()
	safe := filter.FilterError(errors.New(message))
	writeJSON(w, status, map[string]string{
		"error": safe,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
