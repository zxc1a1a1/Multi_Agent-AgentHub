package main

import (
	"strings"
	"testing"
)

func TestExtractTestArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:test-report.json\n{\"status\":\"failed\"}\n```"
	artifacts := extractTestArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "test_report" {
		t.Fatalf("expected test_report, got %q", art.Type)
	}
	if art.Title != "test-report.json" {
		t.Fatalf("expected test-report.json, got %q", art.Title)
	}
}

func TestExtractTestArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"status\":\"ok\"}\n```"
	artifacts := extractTestArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != testReportDefaultFilename {
		t.Fatalf("expected %q, got %q", testReportDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractTestArtifacts_NoJSONBlock(t *testing.T) {
	input := "log analysis text only"
	artifacts := extractTestArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractTestArtifacts_NonJSONBlockIgnored(t *testing.T) {
	input := "```markdown:report.md\n# no json\n```"
	artifacts := extractTestArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestSystemPrompt_OnlyAnalyzeLogsAndNoCommandExecutionClaim(t *testing.T) {
	if !strings.Contains(systemPrompt, "ONLY analyze logs provided in the user messages") {
		t.Fatalf("systemPrompt must state log-only analysis")
	}
	if !strings.Contains(systemPrompt, "MUST NOT claim that you executed") {
		t.Fatalf("systemPrompt must prohibit fake command execution claims")
	}
}

func TestExtractTestArtifacts_MetadataFields(t *testing.T) {
	input := "```json:test-report.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractTestArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}
