package main

import (
	"os"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

func TestExtractReleaseArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:release-report.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractReleaseArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "release_report" {
		t.Fatalf("expected release_report, got %q", art.Type)
	}
	if art.Title != "release-report.json" {
		t.Fatalf("expected release-report.json, got %q", art.Title)
	}
}

func TestExtractReleaseArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractReleaseArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != releaseReportDefaultFilename {
		t.Fatalf("expected %q, got %q", releaseReportDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractReleaseArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractReleaseArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractReleaseArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```markdown:note.md\n# text\n```"
	artifacts := extractReleaseArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractReleaseArtifacts_MetadataFields(t *testing.T) {
	input := "```json:release-report.json\n{\"risk\":\"low\"}\n```"
	artifacts := extractReleaseArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	meta := artifacts[0].Metadata
	if meta["format"] != "json" || meta["published"] != "false" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestSystemPrompt_ReportOnlyConstraint(t *testing.T) {
	if !strings.Contains(systemPrompt, "MUST only generate reports from provided text") {
		t.Fatalf("systemPrompt must constrain to report generation")
	}
	if !strings.Contains(systemPrompt, "MUST NOT claim that a release is already published") {
		t.Fatalf("systemPrompt must forbid fake publication claims")
	}
}

func TestNoExecCommandCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not use os/exec")
	}
	if strings.Contains(content, "exec.Command(\"git\"") ||
		strings.Contains(content, "exec.Command(\"gh\"") ||
		strings.Contains(content, "exec.Command(\"npm\"") ||
		strings.Contains(content, "exec.Command(\"docker\"") {
		t.Fatalf("handler.go must not execute release commands")
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
