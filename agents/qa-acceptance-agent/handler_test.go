package main

import "testing"

func TestExtractAcceptanceArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:acceptance-report.json\n{\"decision\":\"pass\"}\n```"
	artifacts := extractAcceptanceArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "acceptance_report" {
		t.Fatalf("expected acceptance_report, got %q", art.Type)
	}
	if art.Title != "acceptance-report.json" {
		t.Fatalf("expected acceptance-report.json, got %q", art.Title)
	}
	if art.Content != "{\"decision\":\"pass\"}" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
}

func TestExtractAcceptanceArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"decision\":\"manual\"}\n```"
	artifacts := extractAcceptanceArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != acceptanceReportDefaultFilename {
		t.Fatalf("expected %q, got %q", acceptanceReportDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractAcceptanceArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractAcceptanceArtifacts("plain acceptance text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractAcceptanceArtifacts_NonJSONBlockIgnored(t *testing.T) {
	input := "```markdown:note.md\n# no json\n```"
	artifacts := extractAcceptanceArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractAcceptanceArtifacts_MetadataFields(t *testing.T) {
	input := "```json:acceptance-report.json\n{\"items\":[]}\n```"
	artifacts := extractAcceptanceArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}
