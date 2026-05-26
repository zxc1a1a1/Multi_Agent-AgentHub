package main

import (
	"os"
	"strings"
	"testing"
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
