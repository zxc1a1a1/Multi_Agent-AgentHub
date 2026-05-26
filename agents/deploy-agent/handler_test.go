package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractDeploymentArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:deployment-plan.json\n{\"plan\":\"x\"}\n```"
	artifacts := extractDeploymentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "deployment_plan" {
		t.Fatalf("expected deployment_plan, got %q", art.Type)
	}
	if art.Title != "deployment-plan.json" {
		t.Fatalf("expected deployment-plan.json, got %q", art.Title)
	}
}

func TestExtractDeploymentArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"plan\":\"x\"}\n```"
	artifacts := extractDeploymentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != deploymentPlanDefaultFilename {
		t.Fatalf("expected %q, got %q", deploymentPlanDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractDeploymentArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractDeploymentArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDeploymentArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```markdown:note.md\n# text\n```"
	artifacts := extractDeploymentArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDeploymentArtifacts_MetadataFields(t *testing.T) {
	input := "```json:deployment-plan.json\n{\"risk\":\"low\"}\n```"
	artifacts := extractDeploymentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	meta := artifacts[0].Metadata
	if meta["format"] != "json" || meta["execution"] != "not_executed" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestSystemPrompt_PlanOnlyConstraint(t *testing.T) {
	if !strings.Contains(systemPrompt, "can only generate deployment plans") {
		t.Fatalf("systemPrompt must constrain to planning")
	}
	if !strings.Contains(systemPrompt, "MUST NOT claim that you already executed deployment actions") {
		t.Fatalf("systemPrompt must forbid fake execution claims")
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
	if strings.Contains(content, "exec.Command(\"docker\"") ||
		strings.Contains(content, "exec.Command(\"kubectl\"") ||
		strings.Contains(content, "exec.Command(\"ssh\"") ||
		strings.Contains(content, "exec.Command(\"systemctl\"") ||
		strings.Contains(content, "exec.Command(\"curl\"") {
		t.Fatalf("handler.go must not execute deployment commands")
	}
}
