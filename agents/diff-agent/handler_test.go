package main

import (
	"os"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestExtractDiffArtifacts_WithFilename(t *testing.T) {
	input := "```diff:patch.diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "diff" {
		t.Fatalf("expected diff, got %q", art.Type)
	}
	if art.Title != "patch.diff" {
		t.Fatalf("expected patch.diff, got %q", art.Title)
	}
	if art.Metadata["format"] != "unified_diff" {
		t.Fatalf("expected unified_diff, got %q", art.Metadata["format"])
	}
	if art.Metadata["applyStatus"] != "not_applied" {
		t.Fatalf("expected not_applied, got %q", art.Metadata["applyStatus"])
	}
}

func TestExtractDiffArtifacts_WithoutFilenameDefaults(t *testing.T) {
	input := "```diff\n--- a/a.txt\n+++ b/a.txt\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != diffDefaultFilename {
		t.Fatalf("expected %q, got %q", diffDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractDiffArtifacts_MultipleBlocks(t *testing.T) {
	input := "```diff:a.diff\n--- a\n+++ b\n```\ntext\n```diff:b.diff\n--- c\n+++ d\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDiffArtifacts_NoDiffBlock(t *testing.T) {
	artifacts := extractDiffArtifacts("no diff block")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDiffArtifacts_NonDiffIgnored(t *testing.T) {
	input := "```json:patch.json\n{\"k\":1}\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestNoExecOrGitApplyCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not use os/exec")
	}
	if strings.Contains(content, "exec.Command(\"git\"") || strings.Contains(content, "exec.CommandContext(\"git\"") {
		t.Fatalf("handler.go must not execute git commands")
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
