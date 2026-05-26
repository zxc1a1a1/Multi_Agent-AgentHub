package main

import "testing"

func TestExtractDocumentArtifacts_MarkdownWithFilename(t *testing.T) {
	input := "```markdown:report.md\n# Report\nContent\n```"

	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}

	art := artifacts[0]
	if art.Type != "document" {
		t.Fatalf("expected type document, got %q", art.Type)
	}
	if art.Title != "report.md" {
		t.Fatalf("expected title report.md, got %q", art.Title)
	}
	if art.Content != "# Report\nContent" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
	if art.Metadata["format"] != "markdown" {
		t.Fatalf("expected format markdown, got %q", art.Metadata["format"])
	}
}

func TestExtractDocumentArtifacts_MDWithFilename(t *testing.T) {
	input := "```md:README.md\n# Readme\n```"

	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != "README.md" {
		t.Fatalf("expected title README.md, got %q", artifacts[0].Title)
	}
}

func TestExtractDocumentArtifacts_MarkdownWithoutFilenameDefaults(t *testing.T) {
	input := "```markdown\n# Untitled\n```"

	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != "untitled.md" {
		t.Fatalf("expected untitled.md, got %q", artifacts[0].Title)
	}
}

func TestExtractDocumentArtifacts_MultipleMarkdownBlocks(t *testing.T) {
	input := "```markdown:report.md\n# R1\n```\ntext\n```md:handoff.md\n# R2\n```"

	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].Title != "report.md" || artifacts[1].Title != "handoff.md" {
		t.Fatalf("unexpected titles: %q, %q", artifacts[0].Title, artifacts[1].Title)
	}
}

func TestExtractDocumentArtifacts_NoMarkdownBlocks(t *testing.T) {
	input := "plain text without markdown fence"
	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDocumentArtifacts_NonMarkdownFenceIgnored(t *testing.T) {
	input := "```html:index.html\n<html></html>\n```"
	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts for non-markdown fences, got %d", len(artifacts))
	}
}

func TestExtractDocumentArtifacts_MetadataFields(t *testing.T) {
	input := "```markdown:spec.md\n# Spec\n```"
	artifacts := extractDocumentArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "markdown" {
		t.Fatalf("expected format markdown, got %q", artifacts[0].Metadata["format"])
	}
	if artifacts[0].Metadata["language"] != "markdown" {
		t.Fatalf("expected language markdown, got %q", artifacts[0].Metadata["language"])
	}
}
