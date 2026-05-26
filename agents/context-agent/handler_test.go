package main

import "testing"

func TestExtractContextArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:context-bundle.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "context_bundle" {
		t.Fatalf("expected context_bundle, got %q", art.Type)
	}
	if art.Title != "context-bundle.json" {
		t.Fatalf("expected context-bundle.json, got %q", art.Title)
	}
	if art.Content != "{\"summary\":\"ok\"}" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
	if art.Metadata["format"] != "json" {
		t.Fatalf("expected format json, got %q", art.Metadata["format"])
	}
}

func TestExtractContextArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"k\":1}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != contextBundleDefaultFilename {
		t.Fatalf("expected %q, got %q", contextBundleDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractContextArtifacts_MultipleJSONBlocks(t *testing.T) {
	input := "```json:a.json\n{\"a\":1}\n```\ntext\n```json:b.json\n{\"b\":2}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestExtractContextArtifacts_NoJSONBlock(t *testing.T) {
	input := "plain text without json block"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractContextArtifacts_NonJSONBlockIgnored(t *testing.T) {
	input := "```markdown:note.md\n# no json\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractContextArtifacts_MetadataFields(t *testing.T) {
	input := "```json:context-bundle.json\n{\"x\":true}\n```"
	artifacts := extractContextArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}
