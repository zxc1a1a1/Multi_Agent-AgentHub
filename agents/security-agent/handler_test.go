package main

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestScanSecurityRisks_DetectDotEnv(t *testing.T) {
	findings := scanSecurityRisks("please commit .env file")
	if len(findings) == 0 {
		t.Fatalf("expected findings for .env")
	}
}

func TestScanSecurityRisks_DetectSKTokenPattern(t *testing.T) {
	fakeToken := "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
	findings := scanSecurityRisks("token: " + fakeToken)
	if len(findings) == 0 {
		t.Fatalf("expected findings for sk-token pattern")
	}
}

func TestScanSecurityRisks_DetectPrivateKey(t *testing.T) {
	findings := scanSecurityRisks("-----BEGIN PRIVATE KEY-----")
	if len(findings) == 0 {
		t.Fatalf("expected findings for private key")
	}
}

func TestScanSecurityRisks_DetectDangerousCommandRmRF(t *testing.T) {
	findings := scanSecurityRisks("run: rm -rf /tmp/demo")
	if len(findings) == 0 {
		t.Fatalf("expected findings for rm -rf")
	}
}

func TestScanSecurityRisks_AllowsPlaceholderToken(t *testing.T) {
	findings := scanSecurityRisks("use ${AGENTHUB_API_TOKEN} in template")
	if len(findings) != 0 {
		t.Fatalf("expected no findings for allowed placeholder, got %d", len(findings))
	}
}

func TestScanSecurityRisks_AllowsTestToken(t *testing.T) {
	findings := scanSecurityRisks("this is test-token for docs")
	if len(findings) != 0 {
		t.Fatalf("expected no findings for test-token, got %d", len(findings))
	}
}

func TestBuildFallbackSecurityArtifactWhenNoLLMJSON(t *testing.T) {
	findings := scanSecurityRisks("danger: rm -rf /")
	if !hasHighRiskFindings(findings) {
		t.Fatalf("expected high-risk findings")
	}

	art := buildFallbackSecurityArtifact(findings)
	if art.Type != "security_report" {
		t.Fatalf("expected security_report, got %q", art.Type)
	}
	if art.Title != securityReportFilename {
		t.Fatalf("expected %q, got %q", securityReportFilename, art.Title)
	}
	if art.Metadata["format"] != "json" || art.Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", art.Metadata)
	}
	if art.Content == "" {
		t.Fatalf("expected non-empty fallback content")
	}
}

func TestExtractSecurityArtifacts_MetadataAndType(t *testing.T) {
	input := "```json:security-report.json\n{\"risk\":\"high\"}\n```"
	artifacts := extractSecurityArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "security_report" {
		t.Fatalf("expected security_report, got %q", art.Type)
	}
	if art.Title != "security-report.json" {
		t.Fatalf("expected security-report.json, got %q", art.Title)
	}
	if art.Metadata["format"] != "json" {
		t.Fatalf("expected format json, got %q", art.Metadata["format"])
	}
}

func TestConfigLoadable(t *testing.T) {
	cfg, err := adk.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("failed to load config.yaml: %v", err)
	}
	errs := adk.ValidateConfig(cfg)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("config validation error: %v", e)
		}
	}
}
func TestAgentCardSecure(t *testing.T) {
	cfg, err := adk.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("failed to load config.yaml: %v", err)
	}
	card := adk.BuildAgentCard(cfg)
	errs := adk.ValidateAgentCard(card)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("agentcard security violation: %v", e)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  adk.AgentConfig
	}{
		{
			name: "missing name",
			cfg: adk.AgentConfig{
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "missing description",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "missing version",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "missing url",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "empty skills (nil)",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      nil,
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "empty inputModes (nil)",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  nil,
				OutputModes: []string{"text"},
			},
		},
		{
			name: "empty outputModes (nil)",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: nil,
			},
		},
		{
			name: "all fields empty",
			cfg:  adk.AgentConfig{},
		},
		{
			name: "secret in description",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "Uses API_KEY=sk-abc123 for auth",
				Version:     "0.1.0",
				URL:         "http://localhost:8081",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
		{
			name: "token in url",
			cfg: adk.AgentConfig{
				Name:        "test-agent",
				Description: "test description",
				Version:     "0.1.0",
				URL:         "http://localhost:8081?token=secret123",
				Skills:      []string{"s"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := adk.ValidateConfig(&tt.cfg)
			if len(errs) == 0 {
				t.Fatal("expected validation errors, got none")
			}
		})
	}
}

func TestA2AProtocol(t *testing.T) {
	cfg, err := adk.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("failed to load config.yaml: %v", err)
	}
	adk.TestA2AEndpoints(t, cfg)
}
