package main

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/runservice"
)

func TestParseAllowedOrigins(t *testing.T) {
	got := parseAllowedOrigins(" http://localhost:3000, ,http://127.0.0.1:3000 ")
	if len(got) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(got))
	}
	if got[0] != "http://localhost:3000" || got[1] != "http://127.0.0.1:3000" {
		t.Fatalf("unexpected origins: %#v", got)
	}
}

func TestParseBool(t *testing.T) {
	trueCases := []string{"1", "true", "TRUE", "yes", "on"}
	for _, raw := range trueCases {
		if !parseBool(raw) {
			t.Fatalf("expected true for %q", raw)
		}
	}
	falseCases := []string{"", "0", "false", "off", "no", "unknown"}
	for _, raw := range falseCases {
		if parseBool(raw) {
			t.Fatalf("expected false for %q", raw)
		}
	}
}

func TestToAgentSummaries(t *testing.T) {
	endpoints := []runservice.AgentEndpoint{
		{
			Name:        "web-agent",
			Description: "web",
			OutputModes: []string{"text", "html"},
		},
	}

	summaries := toAgentSummaries(endpoints)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Name != "web-agent" {
		t.Fatalf("unexpected name: %q", summaries[0].Name)
	}
	if summaries[0].DisplayName != "Web Agent" {
		t.Fatalf("unexpected display name: %q", summaries[0].DisplayName)
	}
	if len(summaries[0].OutputModes) != 2 {
		t.Fatalf("unexpected output modes: %#v", summaries[0].OutputModes)
	}
}

func TestLoadRuntimeConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("AGENTHUB_API_TOKEN", "")
	t.Setenv("GATEWAY_ENABLE_AUTH", "")
	t.Setenv("GATEWAY_ADDR", "")
	t.Setenv("GATEWAY_ALLOWED_ORIGINS", "")
	t.Setenv("AGENT_CODE_URL", "")
	t.Setenv("AGENT_WEB_URL", "")
	t.Setenv("GATEWAY_DEFAULT_AGENT_NAME", "")

	cfg, err := loadRuntimeConfigFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Gateway.Addr != defaultGatewayAddr {
		t.Fatalf("expected default addr %q, got %q", defaultGatewayAddr, cfg.Gateway.Addr)
	}
	if cfg.Gateway.EnableAuth {
		t.Fatalf("expected auth disabled by default")
	}
	if cfg.DefaultAgentName != defaultAgentName {
		t.Fatalf("expected default agent %q, got %q", defaultAgentName, cfg.DefaultAgentName)
	}
	if len(cfg.AgentEndpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(cfg.AgentEndpoints))
	}
}

func TestLoadRuntimeConfigFromEnvEnableAuthWithoutToken(t *testing.T) {
	t.Setenv("AGENTHUB_API_TOKEN", "")
	t.Setenv("GATEWAY_ENABLE_AUTH", "true")

	if _, err := loadRuntimeConfigFromEnv(); err == nil {
		t.Fatalf("expected validation error when auth enabled without token")
	}
}
