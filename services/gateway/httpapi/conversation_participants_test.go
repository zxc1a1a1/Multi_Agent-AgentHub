package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func newParticipantsTestServer(t *testing.T) (*Server, *store.MemoryStore) {
	t.Helper()
	st := store.NewMemoryStore()
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}
	return srv, st
}

func TestCreateGroupConversationPersistsParticipants(t *testing.T) {
	srv, st := newParticipantsTestServer(t)
	body := `{"userId":"user-1","agentNames":["code-agent","review-agent","code-agent"]}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}
	var conv store.Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &conv); err != nil {
		t.Fatal(err)
	}
	if conv.Type != store.ConversationTypeGroup {
		t.Fatalf("type=%q, want group", conv.Type)
	}
	if len(conv.Participants) != 2 {
		t.Fatalf("duplicate names must be collapsed: %#v", conv.Participants)
	}
	if conv.Participants[0].AgentName != "code-agent" || conv.Participants[0].Role != "owner" || !conv.Participants[0].Selected {
		t.Fatalf("unexpected owner participant: %#v", conv.Participants[0])
	}
	if conv.Participants[1].AgentName != "review-agent" {
		t.Fatalf("unexpected second participant: %#v", conv.Participants[1])
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/conversations/"+conv.ID+"/participants", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list participants failed: %d %s", rec.Code, rec.Body.String())
	}
	var list []store.Participant
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("participants=%d, want 2", len(list))
	}

	loaded, err := st.GetConversation(t.Context(), conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Type != store.ConversationTypeGroup || len(loaded.Participants) != 2 {
		t.Fatalf("roster not persisted: %#v", loaded)
	}
}

func TestCreateSingleConversationPersistsOwner(t *testing.T) {
	srv, _ := newParticipantsTestServer(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/conversations",
		strings.NewReader(`{"userId":"user-1","agentName":"code-agent"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}
	var conv store.Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &conv); err != nil {
		t.Fatal(err)
	}
	if conv.Type != store.ConversationTypeSingle || len(conv.Participants) != 1 {
		t.Fatalf("single conversation must persist its owner: %#v", conv)
	}
}

func TestUpdateConversationParticipantsReplacesRoster(t *testing.T) {
	srv, _ := newParticipantsTestServer(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/conversations",
		strings.NewReader(`{"userId":"user-1","agentNames":["code-agent","review-agent"]}`)))
	var conv store.Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &conv); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/api/conversations/"+conv.ID,
		strings.NewReader(`{"agentNames":["test-agent"]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch failed: %d %s", rec.Code, rec.Body.String())
	}
	var updated store.Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Type != store.ConversationTypeSingle || len(updated.Participants) != 1 || updated.Participants[0].AgentName != "test-agent" {
		t.Fatalf("roster not replaced: %#v", updated)
	}
}

func TestConversationParticipantsRejectInvalidInput(t *testing.T) {
	srv, _ := newParticipantsTestServer(t)
	cases := []struct {
		name string
		body string
	}{
		{"missing user", `{"agentNames":["code-agent"]}`},
		{"missing agents", `{"userId":"user-1"}`},
		{"invalid type", `{"userId":"user-1","agentName":"code-agent","type":"swarm"}`},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(tc.body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d %s", tc.name, rec.Code, rec.Body.String())
		}
	}
}

func TestConversationParticipantsUnknownConversation(t *testing.T) {
	srv, _ := newParticipantsTestServer(t)
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(method, "/api/conversations/missing/participants",
			strings.NewReader(`{"agentNames":["code-agent"]}`)))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s unknown conversation: expected 404, got %d %s", method, rec.Code, rec.Body.String())
		}
	}
}
