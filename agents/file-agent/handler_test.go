package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractFileSummaryArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:file-summary.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractFileSummaryArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "file_summary" {
		t.Fatalf("expected file_summary, got %q", art.Type)
	}
	if art.Title != "file-summary.json" {
		t.Fatalf("expected file-summary.json, got %q", art.Title)
	}
}

func TestExtractFileSummaryArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"k\":1}\n```"
	artifacts := extractFileSummaryArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != fileSummaryDefaultFilename {
		t.Fatalf("expected %q, got %q", fileSummaryDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractFileSummaryArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractFileSummaryArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractFileSummaryArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```markdown:note.md\n# title\n```"
	artifacts := extractFileSummaryArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractFileSummaryArtifacts_MetadataFields(t *testing.T) {
	input := "```json:file-summary.json\n{\"x\":1}\n```"
	artifacts := extractFileSummaryArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}

func TestSystemPrompt_ContentOnlyConstraint(t *testing.T) {
	if !strings.Contains(systemPrompt, "ONLY analyze file text provided by the caller") {
		t.Fatalf("systemPrompt must constrain analysis to provided text")
	}
	if !strings.Contains(systemPrompt, "MUST NOT claim that you read local files") {
		t.Fatalf("systemPrompt must forbid local file read claims")
	}
}

func TestNoFileReadOrExecCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "os.ReadFile") || strings.Contains(content, "os.Open") {
		t.Fatalf("handler.go must not read local files")
	}
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not execute commands")
	}
}
