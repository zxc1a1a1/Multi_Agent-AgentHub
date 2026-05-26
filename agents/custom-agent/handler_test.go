package main

import (
	"strings"
	"testing"
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
