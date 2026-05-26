package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractAgentProfileArtifacts_YAMLWithFilename(t *testing.T) {
	input := "```yaml:agent-profile.yaml\nname: demo\nskills:\n  - summarize\n```"
	artifacts := extractAgentProfileArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "agent_profile" {
		t.Fatalf("expected agent_profile, got %q", art.Type)
	}
	if art.Title != "agent-profile.yaml" {
		t.Fatalf("expected agent-profile.yaml, got %q", art.Title)
	}
}

func TestExtractAgentProfileArtifacts_YMLWithFilename(t *testing.T) {
	input := "```yml:agent-profile.yaml\nname: x\n```"
	artifacts := extractAgentProfileArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != "agent-profile.yaml" {
		t.Fatalf("expected agent-profile.yaml, got %q", artifacts[0].Title)
	}
}

func TestExtractAgentProfileArtifacts_YAMLWithoutFilenameDefaults(t *testing.T) {
	input := "```yaml\nname: x\n```"
	artifacts := extractAgentProfileArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != agentProfileDefaultFilename {
		t.Fatalf("expected %q, got %q", agentProfileDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractAgentProfileArtifacts_NoYAMLBlock(t *testing.T) {
	artifacts := extractAgentProfileArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractAgentProfileArtifacts_NonYAMLIgnored(t *testing.T) {
	input := "```json:profile.json\n{\"name\":\"x\"}\n```"
	artifacts := extractAgentProfileArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractAgentProfileArtifacts_MetadataFields(t *testing.T) {
	input := "```yaml:agent-profile.yaml\nname: demo\n```"
	artifacts := extractAgentProfileArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "yaml" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}

func TestNoWriteOrExecCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "os.WriteFile") || strings.Contains(content, "os.Create") {
		t.Fatalf("handler.go must not create files")
	}
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not execute commands")
	}
}
