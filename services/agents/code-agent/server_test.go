package codeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestCodeAgentA2AServer_Health(t *testing.T) {
	handler, _, err := NewHandler(ServerConfig{URL: "http://code-agent.test"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d", rec.Code, http.StatusOK)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected health status: got=%q", payload["status"])
	}
	if payload["agent"] != "code-agent" {
		t.Fatalf("unexpected health agent: got=%q want=%q", payload["agent"], "code-agent")
	}
}

func TestCodeAgentA2AServer_AgentCard(t *testing.T) {
	handler, _, err := NewHandler(ServerConfig{URL: "http://code-agent.test"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("decode card failed: %v", err)
	}
	if card.Name != "code-agent" {
		t.Fatalf("unexpected card name: got=%q", card.Name)
	}
	if !contains(card.InputModes, "text") || !contains(card.InputModes, "code") {
		t.Fatalf("unexpected input modes: %+v", card.InputModes)
	}
	if !contains(card.OutputModes, "text") || !contains(card.OutputModes, "code") || !contains(card.OutputModes, "artifact_ref") {
		t.Fatalf("unexpected output modes: %+v", card.OutputModes)
	}
	if !containsSkill(card.Skills, "code_generation") || !containsSkill(card.Skills, "code_explanation") {
		t.Fatalf("missing expected skills: %+v", card.Skills)
	}
}

func TestCodeAgentA2AServer_PostRoot(t *testing.T) {
	handler, session, err := NewHandler(ServerConfig{URL: "http://code-agent.test"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}
	sessionID := mustCreateSession(t, session)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"hello"}}`
	rec := doJSONRequest(t, handler, http.MethodPost, "/", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response a2a.RunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode run response failed: %v", err)
	}
	if response.Status != "completed" {
		t.Fatalf("unexpected run status: got=%q", response.Status)
	}
	if !hasPartType(response.Events, "text") {
		t.Fatalf("expected text part in response: %+v", response.Events)
	}
}

func TestCodeAgentA2AServer_SendSubscribe(t *testing.T) {
	handler, session, err := NewHandler(ServerConfig{URL: "http://code-agent.test"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}
	sessionID := mustCreateSession(t, session)

	body := `{"jsonrpc":"2.0","id":"req-1","method":"tasks/sendSubscribe","params":{"sessionId":"` + sessionID + `","message":{"role":"user","content":"请生成 go 代码"}}}`
	rec := doJSONRequest(t, handler, http.MethodPost, "/a2a/tasks/sendSubscribe", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response a2a.RunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode run response failed: %v", err)
	}
	if !hasPartType(response.Events, "text") {
		t.Fatalf("expected text part in response: %+v", response.Events)
	}
}

func TestCodeAgentA2AServer_MissingSessionID(t *testing.T) {
	handler, _, err := NewHandler(ServerConfig{URL: "http://code-agent.test"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}

	body := `{"message":{"role":"user","content":"hello"}}`
	rec := doJSONRequest(t, handler, http.MethodPost, "/", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var response a2a.RunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode run response failed: %v", err)
	}
	if response.Error == nil || strings.TrimSpace(response.Error.Message) == "" {
		t.Fatalf("expected structured error response, got=%+v", response)
	}
}

func TestCodeAgentA2AServer_DoesNotExposeThinking(t *testing.T) {
	handler, session, err := NewHandler(ServerConfig{URL: "http://code-agent.test"})
	if err != nil {
		t.Fatalf("new handler failed: %v", err)
	}
	sessionID := mustCreateSession(t, session)

	body := `{"sessionId":"` + sessionID + `","message":{"role":"user","content":"explain this"}}`
	rec := doJSONRequest(t, handler, http.MethodPost, "/", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response a2a.RunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode run response failed: %v", err)
	}
	if hasPartType(response.Events, "thinking") {
		t.Fatalf("thinking part should not be exposed: %+v", response.Events)
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "thinking") {
		t.Fatalf("response should not leak thinking payload: %s", rec.Body.String())
	}
}

func mustCreateSession(t *testing.T, session adk.SessionService) string {
	t.Helper()
	s, err := session.Create(context.Background(), "user-1", nil)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	return s.ID
}

func doJSONRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsSkill(skills []a2a.AgentSkill, skillID string) bool {
	for _, skill := range skills {
		if skill.ID == skillID {
			return true
		}
	}
	return false
}

func hasPartType(events []a2a.EventDTO, partType string) bool {
	for _, event := range events {
		for _, part := range event.Parts {
			if part.Type == partType {
				return true
			}
		}
	}
	return false
}
