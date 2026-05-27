package adk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestA2AEndpoints validates that an agent's A2A server exposes compliant
// /health and /.well-known/agent.json endpoints.
//
// Per a2a-agent-contract:
//   - /health must return {"status":"ok","agent":"<name>"}
//   - /.well-known/agent.json must return a valid AgentCard with all required fields
//   - AgentCard must not contain secrets, internal paths, or system prompts
//
// Each agent calls this from its TestA2AProtocol test.
func TestA2AEndpoints(t *testing.T, cfg *AgentConfig) {
	t.Helper()

	server := NewA2AServer(cfg, NoopHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// — /health —
	t.Run("health", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("GET /health failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}

		body, _ := io.ReadAll(resp.Body)
		var health struct {
			Status string `json:"status"`
			Agent  string `json:"agent"`
		}
		if err := json.Unmarshal(body, &health); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if health.Status != "ok" {
			t.Fatalf("expected status ok, got %q", health.Status)
		}
		if health.Agent != cfg.Name {
			t.Fatalf("expected agent %q, got %q", cfg.Name, health.Agent)
		}
	})

	// — /.well-known/agent.json —
	t.Run("agentcard", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/.well-known/agent.json")
		if err != nil {
			t.Fatalf("GET /.well-known/agent.json failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}

		body, _ := io.ReadAll(resp.Body)
		var card map[string]any
		if err := json.Unmarshal(body, &card); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		// Required A2A AgentCard fields per a2a-agent-contract
		required := []string{"name", "description", "version", "capabilities", "skills", "defaultInputModes", "defaultOutputModes", "supportedInterfaces"}
		for _, field := range required {
			if _, ok := card[field]; !ok {
				t.Errorf("AgentCard missing required field: %q", field)
			}
		}

		if name, _ := card["name"].(string); name != cfg.Name {
			t.Errorf("AgentCard name: expected %q, got %q", cfg.Name, name)
		}
		if ver, _ := card["version"].(string); ver != cfg.Version {
			t.Errorf("AgentCard version: expected %q, got %q", cfg.Version, ver)
		}

		caps, ok := card["capabilities"].(map[string]any)
		if !ok {
			t.Errorf("capabilities missing or not an object")
		} else if streaming, _ := caps["streaming"].(bool); streaming != cfg.Streaming {
			t.Errorf("capabilities.streaming: expected %v, got %v", cfg.Streaming, streaming)
		}
	})

	// — security: no secrets in AgentCard —
	t.Run("nosecrets", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/.well-known/agent.json")
		if err != nil {
			t.Fatalf("GET /.well-known/agent.json failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		raw := string(body)

		forbidden := []string{
			"API_KEY", "api_key", "APIKEY",
			"sk-", "sk-ant-",
			"BEGIN RSA", "PRIVATE KEY",
			"DB_PASSWORD", "DATABASE_URL",
			"/internal/", "/admin/", "/debug/",
			"system prompt", "system_prompt",
		}
		for _, kw := range forbidden {
			if containsStr(raw, kw) {
				t.Errorf("AgentCard contains forbidden keyword: %q", kw)
			}
		}
	})
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
