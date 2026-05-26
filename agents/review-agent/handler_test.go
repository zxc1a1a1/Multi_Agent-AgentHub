package main

import "testing"

func TestExtractReviewArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:review-report.json\n{\"verdict\":\"ok\"}\n```"
	artifacts := extractReviewArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "review_report" {
		t.Fatalf("expected review_report, got %q", art.Type)
	}
	if art.Title != "review-report.json" {
		t.Fatalf("expected review-report.json, got %q", art.Title)
	}
	if art.Content != "{\"verdict\":\"ok\"}" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
}

func TestExtractReviewArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"score\":90}\n```"
	artifacts := extractReviewArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != reviewReportDefaultFilename {
		t.Fatalf("expected %q, got %q", reviewReportDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractReviewArtifacts_NoJSONBlock(t *testing.T) {
	input := "plain review text"
	artifacts := extractReviewArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractReviewArtifacts_NonJSONBlockIgnored(t *testing.T) {
	input := "```markdown:report.md\n# no json\n```"
	artifacts := extractReviewArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractReviewArtifacts_MetadataFields(t *testing.T) {
	input := "```json:review-report.json\n{\"risk\":\"low\"}\n```"
	artifacts := extractReviewArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}
