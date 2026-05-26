package adk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func TestBuildAgentCard_MapsConfigFields(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "web-agent",
		Description: "Handles web browsing tasks",
		Version:     "1.2.3",
		URL:         "http://localhost:8091",
		Skills:      []string{"web_search", "web_extract"},
		InputModes:  []string{"text", "markdown"},
		OutputModes: []string{"text", "markdown"},
		Streaming:   true,
	}

	card := BuildAgentCard(cfg)

	if card.Name != cfg.Name {
		t.Fatalf("name mismatch: got %q want %q", card.Name, cfg.Name)
	}
	if card.Description != cfg.Description {
		t.Fatalf("description mismatch: got %q want %q", card.Description, cfg.Description)
	}
	if card.Version != cfg.Version {
		t.Fatalf("version mismatch: got %q want %q", card.Version, cfg.Version)
	}
	if card.Capabilities.Streaming != cfg.Streaming {
		t.Fatalf("streaming mismatch: got %v want %v", card.Capabilities.Streaming, cfg.Streaming)
	}

	if len(card.SupportedInterfaces) != 1 {
		t.Fatalf("supported interfaces mismatch: got %d want 1", len(card.SupportedInterfaces))
	}
	if got := card.SupportedInterfaces[0].URL; got != cfg.URL {
		t.Fatalf("url mismatch: got %q want %q", got, cfg.URL)
	}
	if got := card.SupportedInterfaces[0].ProtocolBinding; got != a2a.TransportProtocolJSONRPC {
		t.Fatalf("protocol mismatch: got %q want %q", got, a2a.TransportProtocolJSONRPC)
	}

	if len(card.DefaultInputModes) != len(cfg.InputModes) {
		t.Fatalf("inputModes length mismatch: got %d want %d", len(card.DefaultInputModes), len(cfg.InputModes))
	}
	for i := range cfg.InputModes {
		if card.DefaultInputModes[i] != cfg.InputModes[i] {
			t.Fatalf("inputModes[%d] mismatch: got %q want %q", i, card.DefaultInputModes[i], cfg.InputModes[i])
		}
	}

	if len(card.DefaultOutputModes) != len(cfg.OutputModes) {
		t.Fatalf("outputModes length mismatch: got %d want %d", len(card.DefaultOutputModes), len(cfg.OutputModes))
	}
	for i := range cfg.OutputModes {
		if card.DefaultOutputModes[i] != cfg.OutputModes[i] {
			t.Fatalf("outputModes[%d] mismatch: got %q want %q", i, card.DefaultOutputModes[i], cfg.OutputModes[i])
		}
	}

	if len(card.Skills) != len(cfg.Skills) {
		t.Fatalf("skills length mismatch: got %d want %d", len(card.Skills), len(cfg.Skills))
	}
	for i := range cfg.Skills {
		if card.Skills[i].ID != cfg.Skills[i] {
			t.Fatalf("skills[%d].id mismatch: got %q want %q", i, card.Skills[i].ID, cfg.Skills[i])
		}
		if card.Skills[i].Name != cfg.Skills[i] {
			t.Fatalf("skills[%d].name mismatch: got %q want %q", i, card.Skills[i].Name, cfg.Skills[i])
		}
	}
}

func TestBuildAgentCard_DifferentConfigsProduceDifferentCards(t *testing.T) {
	cfgA := &AgentConfig{
		Name:        "agent-a",
		Description: "A",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"skill_a"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
		Streaming:   true,
	}
	cfgB := &AgentConfig{
		Name:        "agent-b",
		Description: "B",
		Version:     "0.2.0",
		URL:         "http://localhost:8082",
		Skills:      []string{"skill_b"},
		InputModes:  []string{"json"},
		OutputModes: []string{"markdown"},
		Streaming:   false,
	}

	cardA := BuildAgentCard(cfgA)
	cardB := BuildAgentCard(cfgB)

	if cardA.Name == cardB.Name {
		t.Fatalf("expected different names, got both %q", cardA.Name)
	}
	if cardA.SupportedInterfaces[0].URL == cardB.SupportedInterfaces[0].URL {
		t.Fatalf("expected different urls, got both %q", cardA.SupportedInterfaces[0].URL)
	}
	if cardA.Capabilities.Streaming == cardB.Capabilities.Streaming {
		t.Fatalf("expected different streaming flags, got both %v", cardA.Capabilities.Streaming)
	}
	if cardA.Skills[0].ID == cardB.Skills[0].ID {
		t.Fatalf("expected different skills, got both %q", cardA.Skills[0].ID)
	}
}

func TestBuildAgentCard_DoesNotInjectHardcodedSkill(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "no-skill-agent",
		Description: "No skills",
		Version:     "0.1.0",
		URL:         "http://localhost:8099",
		Skills:      nil,
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
		Streaming:   true,
	}

	card := BuildAgentCard(cfg)
	if len(card.Skills) != 0 {
		t.Fatalf("expected no skills, got %d", len(card.Skills))
	}
}

// ——— A2A Protocol Integration Tests ———

func testHandler(ctx *Context, messages []a2a.Message) error {
	return nil
}

func TestA2AServer_HealthEndpoint(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent-a",
		Description: "A test agent for A2A protocol verification",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"test_skill"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
		Streaming:   true,
	}

	server := NewA2AServer(cfg, testHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("GET /health: expected Content-Type application/json, got %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	var health struct {
		Status string `json:"status"`
		Agent  string `json:"agent"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("GET /health: invalid JSON response: %v", err)
	}
	if health.Status != "ok" {
		t.Fatalf("GET /health: expected status ok, got %q", health.Status)
	}
	if health.Agent != cfg.Name {
		t.Fatalf("GET /health: expected agent %q, got %q", cfg.Name, health.Agent)
	}
}

func TestA2AServer_AgentCardEndpoint(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent-a",
		Description: "A test agent for A2A protocol verification",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"skill_a", "skill_b"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text", "code"},
		Streaming:   true,
	}

	server := NewA2AServer(cfg, testHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("GET /.well-known/agent.json failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /.well-known/agent.json: expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("GET /.well-known/agent.json: expected Content-Type application/json, got %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	var card map[string]any
	if err := json.Unmarshal(body, &card); err != nil {
		t.Fatalf("GET /.well-known/agent.json: invalid JSON: %v", err)
	}

	// Required A2A AgentCard fields
	required := []string{"name", "description", "version", "capabilities", "skills", "defaultInputModes", "defaultOutputModes", "supportedInterfaces"}
	for _, field := range required {
		if _, ok := card[field]; !ok {
			t.Errorf("AgentCard missing required field: %q", field)
		}
	}

	// Verify key values
	if name, _ := card["name"].(string); name != cfg.Name {
		t.Errorf("AgentCard name: expected %q, got %q", cfg.Name, name)
	}
	if ver, _ := card["version"].(string); ver != cfg.Version {
		t.Errorf("AgentCard version: expected %q, got %q", cfg.Version, ver)
	}

	// Verify skills match
	skills, ok := card["skills"].([]any)
	if !ok || len(skills) != 2 {
		t.Errorf("AgentCard skills: expected 2 skills, got %v", card["skills"])
	}

	// Verify capabilities
	caps, ok := card["capabilities"].(map[string]any)
	if !ok {
		t.Errorf("AgentCard capabilities: missing or not an object")
	} else if streaming, _ := caps["streaming"].(bool); streaming != true {
		t.Errorf("AgentCard capabilities.streaming: expected true, got %v", streaming)
	}

	// Verify supportedInterfaces
	ifaces, ok := card["supportedInterfaces"].([]any)
	if !ok || len(ifaces) != 1 {
		t.Errorf("AgentCard supportedInterfaces: expected 1 interface, got %v", card["supportedInterfaces"])
	}
}

func TestA2AServer_AgentCardNoSecrets(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent-a",
		Description: "A clean agent for code generation and review tasks",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"code_generation", "code_review"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text", "code"},
		Streaming:   true,
	}

	server := NewA2AServer(cfg, testHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("GET /.well-known/agent.json failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	raw := string(body)

	// Security: AgentCard JSON must not contain secrets, internal paths, or system prompts
	forbidden := []string{
		"API_KEY", "api_key", "APIKEY",
		"sk-", "sk-ant-",
		"BEGIN RSA", "PRIVATE KEY",
		"DB_PASSWORD", "DATABASE_URL",
		"/internal/", "/admin/", "/debug/",
		"system prompt", "system_prompt",
	}
	for _, kw := range forbidden {
		if contains(raw, kw) {
			t.Errorf("AgentCard contains forbidden keyword: %q", kw)
		}
	}
}

func TestA2AServer_EndpointsReturnCorrectStatus(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent-a",
		Description: "Test agent",
		Version:     "0.1.0",
		URL:         "http://localhost:8081",
		Skills:      []string{"s"},
		InputModes:  []string{"text"},
		OutputModes: []string{"text"},
	}

	server := NewA2AServer(cfg, testHandler)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	tests := []struct {
		method       string
		path         string
		expectedCode int
	}{
		{"GET", "/health", 200},
		{"GET", "/.well-known/agent.json", 200},
		{"POST", "/", 200},                         // A2A JSON-RPC (accepts even empty body for protocol compliance)
		{"POST", "/a2a/tasks/sendSubscribe", 200},   // MVP compat
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, ts.URL+tt.path, nil)
			if tt.method == "POST" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, resp.StatusCode)
			}
		})
	}
}
