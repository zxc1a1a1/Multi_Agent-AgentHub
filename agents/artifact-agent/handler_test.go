package main

import "testing"

func TestExtractArtifactManifestArtifacts_WithFilename(t *testing.T) {
	input := "```json:artifact-manifest.json\n{\"type\":\"code\"}\n```"
	artifacts := extractArtifactManifestArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "artifact_manifest" {
		t.Fatalf("expected artifact_manifest, got %q", art.Type)
	}
	if art.Title != "artifact-manifest.json" {
		t.Fatalf("expected artifact-manifest.json, got %q", art.Title)
	}
	if art.Metadata["format"] != "json" {
		t.Fatalf("expected json format, got %q", art.Metadata["format"])
	}
}

func TestExtractArtifactManifestArtifacts_WithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"k\":1}\n```"
	artifacts := extractArtifactManifestArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != artifactManifestDefaultFilename {
		t.Fatalf("expected %q, got %q", artifactManifestDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractArtifactManifestArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractArtifactManifestArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractArtifactManifestArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```diff:patch.diff\n--- a\n+++ b\n```"
	artifacts := extractArtifactManifestArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractArtifactManifestArtifacts_MetadataFields(t *testing.T) {
	input := "```json:artifact-manifest.json\n{\"id\":\"x\"}\n```"
	artifacts := extractArtifactManifestArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}
