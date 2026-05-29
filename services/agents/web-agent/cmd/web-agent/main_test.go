package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"
)

func TestBuildHandler_Success(t *testing.T) {
	handler, err := buildHandler("http://web-agent.test")
	if err != nil {
		t.Fatalf("build handler failed: %v", err)
	}
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestBuildHandler_HealthEndpoint(t *testing.T) {
	handler, err := buildHandler("http://web-agent.test")
	if err != nil {
		t.Fatalf("build handler failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestBuildHandler_AgentCardEndpoint(t *testing.T) {
	const publicURL = "http://web-agent.test"

	handler, err := buildHandler(publicURL)
	if err != nil {
		t.Fatalf("build handler failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("decode card failed: %v", err)
	}
	if card.URL != publicURL {
		t.Fatalf("unexpected card url: got=%q want=%q", card.URL, publicURL)
	}
}

func TestBuildHandler_EmptyPublicURLUsesDefault(t *testing.T) {
	handler, err := buildHandler("")
	if err != nil {
		t.Fatalf("build handler failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("decode card failed: %v", err)
	}
	if card.URL != defaultPublicURL {
		t.Fatalf("unexpected default card url: got=%q want=%q", card.URL, defaultPublicURL)
	}
}
