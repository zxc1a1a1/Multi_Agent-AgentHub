package a2a

import (
	"strings"
	"testing"
)

func TestBuildAgentCard(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent",
		Description: "example description",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}, {ID: "search", Name: "search"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"text/plain", "application/json"},
		Streaming:   true,
	}

	card := BuildAgentCard(cfg)
	if card == nil {
		t.Fatal("expected non-nil agent card")
	}
	if card.Name != cfg.Name {
		t.Fatalf("unexpected name: got=%q want=%q", card.Name, cfg.Name)
	}
	if card.Description != cfg.Description {
		t.Fatalf("unexpected description: got=%q want=%q", card.Description, cfg.Description)
	}
	if card.Version != cfg.Version {
		t.Fatalf("unexpected version: got=%q want=%q", card.Version, cfg.Version)
	}
	if card.Streaming != cfg.Streaming {
		t.Fatalf("unexpected streaming: got=%v want=%v", card.Streaming, cfg.Streaming)
	}
	if card.URL != cfg.URL {
		t.Fatalf("unexpected url: got=%q want=%q", card.URL, cfg.URL)
	}
	if len(card.Skills) != 2 {
		t.Fatalf("unexpected skills count: got=%d want=%d", len(card.Skills), 2)
	}
	if card.Skills[0].ID != "code" || card.Skills[0].Name != "code" {
		t.Fatalf("unexpected first skill: %+v", card.Skills[0])
	}
	if card.Skills[1].ID != "search" || card.Skills[1].Name != "search" {
		t.Fatalf("unexpected second skill: %+v", card.Skills[1])
	}
}

func TestBuildAgentCard_CopiesSlices(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent",
		Description: "example description",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}

	card := BuildAgentCard(cfg)
	if card == nil {
		t.Fatal("expected non-nil agent card")
	}

	cfg.InputModes[0] = "mutated-input"
	cfg.OutputModes[0] = "mutated-output"
	cfg.Skills[0] = AgentSkill{ID: "mutated-skill", Name: "mutated-skill"}

	if card.InputModes[0] != "text/plain" {
		t.Fatalf("input modes should be copied, got=%q", card.InputModes[0])
	}
	if card.OutputModes[0] != "application/json" {
		t.Fatalf("output modes should be copied, got=%q", card.OutputModes[0])
	}
	if card.Skills[0].ID != "code" || card.Skills[0].Name != "code" {
		t.Fatalf("skills should be copied from original config value: %+v", card.Skills[0])
	}
}

func TestBuildAgentCard_SupportedInterface(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent",
		Description: "example description",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}

	card := BuildAgentCard(cfg)
	if card == nil {
		t.Fatal("expected non-nil agent card")
	}
	if len(card.SupportedInterfaces) == 0 {
		t.Fatal("expected at least one supported interface")
	}

	hasJSONRPC := false
	for _, item := range card.SupportedInterfaces {
		if item.Type == "JSONRPC" && item.URL == cfg.URL {
			hasJSONRPC = true
			break
		}
	}
	if !hasJSONRPC {
		t.Fatalf("expected JSONRPC interface with URL=%q, got=%+v", cfg.URL, card.SupportedInterfaces)
	}
}

func TestValidateAgentConfig_Clean(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent",
		Description: "这是一个普通中文描述",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
		Streaming:   true,
	}

	errs := ValidateAgentConfig(cfg)
	if len(errs) != 0 {
		t.Fatalf("expected clean config, got errors: %v", errs)
	}
}

func TestValidateAgentConfig_RequiredFields(t *testing.T) {
	base := AgentConfig{
		Name:        "example-agent",
		Description: "example description",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}

	tests := []struct {
		name string
		cfg  AgentConfig
		want string
	}{
		{name: "missing name", cfg: withConfig(base, func(c *AgentConfig) { c.Name = "" }), want: "name"},
		{name: "missing description", cfg: withConfig(base, func(c *AgentConfig) { c.Description = "" }), want: "description"},
		{name: "missing version", cfg: withConfig(base, func(c *AgentConfig) { c.Version = "" }), want: "version"},
		{name: "missing url", cfg: withConfig(base, func(c *AgentConfig) { c.URL = "" }), want: "url"},
		{name: "empty skills", cfg: withConfig(base, func(c *AgentConfig) { c.Skills = nil }), want: "skills"},
		{name: "empty input modes", cfg: withConfig(base, func(c *AgentConfig) { c.InputModes = nil }), want: "input"},
		{name: "empty output modes", cfg: withConfig(base, func(c *AgentConfig) { c.OutputModes = nil }), want: "output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateAgentConfig(&tt.cfg)
			if len(errs) == 0 {
				t.Fatalf("expected validation errors for %s", tt.name)
			}
			if !containsError(errs, tt.want) {
				t.Fatalf("expected error mentioning %q, got %v", tt.want, errs)
			}
		})
	}
}

func TestValidateAgentConfig_RejectsSecrets(t *testing.T) {
	base := AgentConfig{
		Name:        "example-agent",
		Description: "example description",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}

	tests := []struct {
		name        string
		description string
	}{
		{name: "OPENAI_API_KEY", description: "contains OPENAI_API_KEY placeholder"},
		{name: "DATABASE_URL", description: "contains DATABASE_URL placeholder"},
		{name: "PRIVATE KEY", description: "-----BEGIN RSA PRIVATE KEY-----"},
		{name: "BEGIN OPENSSH", description: "-----BEGIN OPENSSH PRIVATE KEY-----"},
		{name: "sk token", description: "prefix " + "sk-" + "demo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := withConfig(base, func(c *AgentConfig) {
				c.Description = tt.description
			})
			errs := ValidateAgentConfig(&cfg)
			if len(errs) == 0 {
				t.Fatalf("expected secret validation error for %q", tt.name)
			}
			if !containsError(errs, "sensitive") {
				t.Fatalf("expected sensitive info error, got %v", errs)
			}
		})
	}
}

func TestValidateAgentCard_Clean(t *testing.T) {
	cfg := &AgentConfig{
		Name:        "example-agent",
		Description: "普通中文描述",
		Version:     "v1.0.0",
		URL:         "http://localhost:8080",
		Skills:      []AgentSkill{{ID: "code", Name: "code"}},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
	}

	card := BuildAgentCard(cfg)
	errs := ValidateAgentCard(card)
	if len(errs) != 0 {
		t.Fatalf("expected clean card, got errors: %v", errs)
	}
}

func TestValidateAgentCard_RejectsSecrets(t *testing.T) {
	card := &AgentCard{
		Name:        "example-agent",
		Description: "contains AGENTHUB_API_TOKEN placeholder",
		Version:     "v1.0.0",
		Streaming:   false,
		Skills: []AgentSkill{
			{ID: "code", Name: "code"},
		},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
		URL:         "http://localhost:8080",
		SupportedInterfaces: []AgentInterface{
			{Type: "JSONRPC", URL: "http://localhost:8080"},
		},
	}

	errs := ValidateAgentCard(card)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for sensitive content")
	}
	if !containsError(errs, "sensitive") {
		t.Fatalf("expected sensitive info error, got %v", errs)
	}
}

func TestValidateAgentCard_AllowsNormalChineseDescription(t *testing.T) {
	card := &AgentCard{
		Name:        "example-agent",
		Description: "这是一个普通中文描述，不包含敏感信息",
		Version:     "v1.0.0",
		Streaming:   false,
		Skills: []AgentSkill{
			{ID: "code", Name: "code"},
		},
		InputModes:  []string{"text/plain"},
		OutputModes: []string{"application/json"},
		URL:         "http://localhost:8080",
		SupportedInterfaces: []AgentInterface{
			{Type: "JSONRPC", URL: "http://localhost:8080"},
		},
	}

	errs := ValidateAgentCard(card)
	if len(errs) != 0 {
		t.Fatalf("expected normal chinese description to pass, got %v", errs)
	}
}

func withConfig(base AgentConfig, mutate func(*AgentConfig)) AgentConfig {
	copied := base
	copied.Skills = append([]AgentSkill{}, base.Skills...)
	copied.InputModes = append([]string(nil), base.InputModes...)
	copied.OutputModes = append([]string(nil), base.OutputModes...)
	mutate(&copied)
	return copied
}

func containsError(errs []error, want string) bool {
	want = strings.ToLower(want)
	for _, err := range errs {
		if err == nil {
			continue
		}
		if strings.Contains(strings.ToLower(err.Error()), want) {
			return true
		}
	}
	return false
}
