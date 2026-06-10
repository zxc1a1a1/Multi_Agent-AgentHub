package a2a

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- NormalizeAgentURL tests ---

func TestNormalizeAgentURL_ValidHTTP(t *testing.T) {
	result, err := NormalizeAgentURL("http://localhost:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "http://localhost:8080" {
		t.Fatalf("unexpected result: got=%q want=%q", result, "http://localhost:8080")
	}
}

func TestNormalizeAgentURL_ValidHTTPS(t *testing.T) {
	result, err := NormalizeAgentURL("https://agent.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "https://agent.example.com" {
		t.Fatalf("unexpected result: got=%q want=%q", result, "https://agent.example.com")
	}
}

func TestNormalizeAgentURL_RejectsEmpty(t *testing.T) {
	_, err := NormalizeAgentURL("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected error to mention 'empty', got: %v", err)
	}
}

func TestNormalizeAgentURL_RejectsFile(t *testing.T) {
	_, err := NormalizeAgentURL("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected error for file:// URL")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected error to mention 'scheme', got: %v", err)
	}
}

func TestNormalizeAgentURL_RejectsJavascript(t *testing.T) {
	_, err := NormalizeAgentURL("javascript:alert(1)")
	if err == nil {
		t.Fatal("expected error for javascript: URL")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected error to mention 'scheme', got: %v", err)
	}
}

func TestNormalizeAgentURL_RejectsNoHost(t *testing.T) {
	_, err := NormalizeAgentURL("http://")
	if err == nil {
		t.Fatal("expected error for URL without host")
	}
	if !strings.Contains(err.Error(), "host") {
		t.Fatalf("expected error to mention 'host', got: %v", err)
	}
}

func TestNormalizeAgentURL_TrimsTrailingSlash(t *testing.T) {
	result, err := NormalizeAgentURL("http://localhost:8080/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "http://localhost:8080" {
		t.Fatalf("expected trailing slash to be trimmed, got=%q", result)
	}
}

func TestNormalizeAgentURL_TrimsWhitespace(t *testing.T) {
	result, err := NormalizeAgentURL("  http://localhost:8080  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "http://localhost:8080" {
		t.Fatalf("expected whitespace to be trimmed, got=%q", result)
	}
}

// --- FetchAgentCard tests ---

// validAgentCardJSON returns JSON that passes ValidateAgentCard.
const validAgentCardJSON = `{
	"name": "test-agent",
	"description": "A test agent for unit tests",
	"version": "v1.0.0",
	"streaming": true,
	"url": "http://localhost:8080",
	"skills": [
		{"id": "code", "name": "code"}
	],
	"inputModes": ["text/plain"],
	"outputModes": ["text/plain", "application/json"],
	"supportedInterfaces": [
		{"type": "JSONRPC", "url": "http://localhost:8080"}
	]
}`

func TestFetchAgentCard_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/agent.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validAgentCardJSON))
	}))
	defer srv.Close()

	card, err := FetchAgentCard(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatal("expected non-nil card")
	}
	if card.Name != "test-agent" {
		t.Fatalf("unexpected name: got=%q want=%q", card.Name, "test-agent")
	}
	if card.Version != "v1.0.0" {
		t.Fatalf("unexpected version: got=%q want=%q", card.Version, "v1.0.0")
	}
	if len(card.Skills) != 1 || card.Skills[0].ID != "code" {
		t.Fatalf("unexpected skills: %+v", card.Skills)
	}
}

func TestFetchAgentCard_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	_, err := FetchAgentCard(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected error to mention 'HTTP 500', got: %v", err)
	}
}

func TestFetchAgentCard_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := FetchAgentCard(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected error to mention 'decode', got: %v", err)
	}
}

func TestFetchAgentCard_ValidationFails(t *testing.T) {
	// Card with empty name — ValidateAgentCard rejects it.
	const badCardJSON = `{
		"name": "",
		"description": "no name",
		"version": "v1.0.0",
		"url": "http://localhost:8080",
		"skills": [],
		"inputModes": [],
		"outputModes": [],
		"supportedInterfaces": []
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(badCardJSON))
	}))
	defer srv.Close()

	_, err := FetchAgentCard(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid agent card")
	}
	if !strings.Contains(err.Error(), "validate") {
		t.Fatalf("expected error to mention 'validate', got: %v", err)
	}
}

func TestFetchAgentCard_BadBaseURL(t *testing.T) {
	_, err := FetchAgentCard(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
	if !strings.Contains(err.Error(), "normalize") {
		t.Fatalf("expected error to mention 'normalize', got: %v", err)
	}
}

func TestFetchAgentCard_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validAgentCardJSON))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := FetchAgentCard(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
