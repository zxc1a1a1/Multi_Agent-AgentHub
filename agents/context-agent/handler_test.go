package main

import (
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestExtractContextArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:context-bundle.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "context_bundle" {
		t.Fatalf("expected context_bundle, got %q", art.Type)
	}
	if art.Title != "context-bundle.json" {
		t.Fatalf("expected context-bundle.json, got %q", art.Title)
	}
	if art.Content != "{\"summary\":\"ok\"}" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
	if art.Metadata["format"] != "json" {
		t.Fatalf("expected format json, got %q", art.Metadata["format"])
	}
}

func TestExtractContextArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"k\":1}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != contextBundleDefaultFilename {
		t.Fatalf("expected %q, got %q", contextBundleDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractContextArtifacts_MultipleJSONBlocks(t *testing.T) {
	input := "```json:a.json\n{\"a\":1}\n```\ntext\n```json:b.json\n{\"b\":2}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestExtractContextArtifacts_NoJSONBlock(t *testing.T) {
	input := "plain text without json block"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractContextArtifacts_NonJSONBlockIgnored(t *testing.T) {
	input := "```markdown:note.md\n# no json\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractContextArtifacts_MetadataFields(t *testing.T) {
	input := "```json:context-bundle.json\n{\"x\":true}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
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
