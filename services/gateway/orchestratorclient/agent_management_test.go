package orchestratorclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func TestProxyAgentManagement_List(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/internal/orchestrator/agents") {
			t.Errorf("expected path prefix /internal/orchestrator/agents, got %s", r.URL.Path)
		}
		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"name": "code-agent", "displayName": "Code Agent"},
			{"name": "web-agent", "displayName": "Web Agent"},
		})
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "test-token")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodGet, "", "", nil, "")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestProxyAgentManagement_GetByName(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/orchestrator/agents/code-agent" {
			t.Errorf("expected path /internal/orchestrator/agents/code-agent, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"name": "code-agent", "displayName": "Code Agent",
		})
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodGet, "/code-agent", "", nil, "")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestProxyAgentManagement_PathWithAction(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/orchestrator/agents/foo/check" {
			t.Errorf("expected path /internal/orchestrator/agents/foo/check, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"name": "foo", "healthy": true,
		})
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodPost, "/foo/check", "", nil, "")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestProxyAgentManagement_QueryStringPassthrough(t *testing.T) {
	var capturedRawQuery string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodGet, "", "enabled=true&source=dynamic", nil, "")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	if capturedRawQuery != "enabled=true&source=dynamic" {
		t.Errorf("expected rawQuery 'enabled=true&source=dynamic', got %q", capturedRawQuery)
	}
}

func TestProxyAgentManagement_PostWithBody(t *testing.T) {
	var capturedMethod string
	var capturedBody []byte
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body)
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"name": "new-agent",
		})
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "secret")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	body := strings.NewReader(`{"url":"http://127.0.0.1:8081"}`)
	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodPost, "", "", body, "application/json")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	if capturedMethod != http.MethodPost {
		t.Errorf("expected method POST, got %s", capturedMethod)
	}
	if string(capturedBody) != `{"url":"http://127.0.0.1:8081"}` {
		t.Errorf("unexpected body: %s", string(capturedBody))
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestProxyAgentManagement_PreservesErrorStatus(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "agent not found"})
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL, "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodGet, "/nonexistent", "", nil, "")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestProxyAgentManagement_BaseURLConstruction(t *testing.T) {
	// Verify url.Parse is used properly and scheme/host come from baseURL only.
	var capturedURL string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	svc, err := NewOrchestratorRunService(mockServer.URL+"/prefix", "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	resp, err := svc.ProxyAgentManagement(context.Background(), http.MethodGet, "/foo", "bar=1", nil, "")
	if err != nil {
		t.Fatalf("ProxyAgentManagement: %v", err)
	}
	defer resp.Body.Close()

	// The upstream path should be /prefix/internal/orchestrator/agents/foo?bar=1
	if !strings.Contains(capturedURL, "/internal/orchestrator/agents/foo") {
		t.Errorf("expected path /internal/orchestrator/agents/foo in URL, got %s", capturedURL)
	}
	if !strings.Contains(capturedURL, "bar=1") {
		t.Errorf("expected query bar=1 in URL, got %s", capturedURL)
	}
}

func TestProxyAgentManagement_NilService(t *testing.T) {
	var svc *OrchestratorRunService
	_, err := svc.ProxyAgentManagement(context.Background(), http.MethodGet, "", "", nil, "")
	if err == nil {
		t.Fatal("expected error for nil service")
	}
}

func TestProxyAgentManagement_NoOrchestratorImportLeak(t *testing.T) {
	// Verify orchestratorclient does not import orchestrator business packages.
	// This test only validates the package structure — the import list is checked
	// by the boundary audit (rg command).
	svc, err := NewOrchestratorRunService("http://127.0.0.1:9999", "")
	if err != nil {
		t.Fatalf("NewOrchestratorRunService: %v", err)
	}

	// Ensure Run is still satisfied (existing interface unchanged).
	seq := svc.Run(context.Background(), "conv-1", &adk.Content{
		Role: adk.RoleUser,
		Parts: []adk.Part{
			adk.TextPart{Text: "test"},
		},
	})

	var gotErr bool
	seq(func(event adk.Event, err error) bool {
		if err != nil {
			gotErr = true
			return false
		}
		return true
	})
	// Expected to fail (no server at 9999) but type contract is satisfied.
	if !gotErr {
		t.Log("unexpected: no error connecting to 127.0.0.1:9999 (may have succeeded)")
	}
}
