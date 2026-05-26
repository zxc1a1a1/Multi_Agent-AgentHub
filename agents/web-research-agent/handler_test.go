package main

import (
	"os"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestExtractWebResearchArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:web-research.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "web_research" {
		t.Fatalf("expected web_research, got %q", art.Type)
	}
	if art.Title != "web-research.json" {
		t.Fatalf("expected web-research.json, got %q", art.Title)
	}
}

func TestExtractWebResearchArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"k\":1}\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != webResearchDefaultFilename {
		t.Fatalf("expected %q, got %q", webResearchDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractWebResearchArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractWebResearchArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractWebResearchArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```markdown:note.md\n# title\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractWebResearchArtifacts_MetadataFields(t *testing.T) {
	input := "```json:web-research.json\n{\"x\":1}\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}

func TestSystemPrompt_NoActiveFetchConstraint(t *testing.T) {
	if !strings.Contains(systemPrompt, "can only analyze URL/title/body excerpts that the caller provided") {
		t.Fatalf("systemPrompt must constrain analysis source")
	}
	if !strings.Contains(systemPrompt, "MUST NOT fetch URLs") {
		t.Fatalf("systemPrompt must forbid active networking")
	}
}

func TestNoNetworkOrExecCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "http.Get") || strings.Contains(content, "http.Client") || strings.Contains(content, ".Do(") {
		t.Fatalf("handler.go must not perform network fetch")
	}
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not execute commands")
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
