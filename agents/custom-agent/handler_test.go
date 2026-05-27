package main

import (
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestExtractCustomArtifacts_MarkdownWithFilename(t *testing.T) {
	input := "```markdown:custom-output.md\n# Title\ncontent\n```"
	artifacts := extractCustomArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "document" {
		t.Fatalf("expected document, got %q", art.Type)
	}
	if art.Title != "custom-output.md" {
		t.Fatalf("expected custom-output.md, got %q", art.Title)
	}
}

func TestExtractCustomArtifacts_MDWithoutFilenameDefaults(t *testing.T) {
	input := "```md\n# A\n```"
	artifacts := extractCustomArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != customOutputDefaultFilename {
		t.Fatalf("expected %q, got %q", customOutputDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractCustomArtifacts_NoMarkdownBlock(t *testing.T) {
	artifacts := extractCustomArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractCustomArtifacts_NonMarkdownIgnored(t *testing.T) {
	input := "```json:report.json\n{\"k\":1}\n```"
	artifacts := extractCustomArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractCustomArtifacts_MetadataFields(t *testing.T) {
	input := "```markdown:custom-output.md\nhello\n```"
	artifacts := extractCustomArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	metadata := artifacts[0].Metadata
	if metadata["format"] != "markdown" || metadata["source"] != "custom-agent" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestSystemPrompt_SecurityBoundaries(t *testing.T) {
	if !strings.Contains(systemPrompt, "MUST NOT request or claim executing shell commands") {
		t.Fatalf("systemPrompt must forbid command execution claims")
	}
	if !strings.Contains(systemPrompt, ".env") {
		t.Fatalf("systemPrompt must mention .env boundary")
	}
	if !strings.Contains(systemPrompt, "MUST NOT output real API keys, tokens, passwords, private keys") {
		t.Fatalf("systemPrompt must forbid real secret output")
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
