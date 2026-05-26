package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractVersionArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:version-snapshot.json\n{\"version\":\"v1.0.1\"}\n```"
	artifacts := extractVersionArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "version_snapshot" {
		t.Fatalf("expected version_snapshot, got %q", art.Type)
	}
	if art.Title != "version-snapshot.json" {
		t.Fatalf("expected version-snapshot.json, got %q", art.Title)
	}
	if art.Content != "{\"version\":\"v1.0.1\"}" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
}

func TestExtractVersionArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"version\":\"v1.0.2\"}\n```"
	artifacts := extractVersionArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != versionSnapshotDefaultFilename {
		t.Fatalf("expected %q, got %q", versionSnapshotDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractVersionArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractVersionArtifacts("plain summary text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractVersionArtifacts_NonJSONBlockIgnored(t *testing.T) {
	input := "```diff:patch.diff\n--- a\n+++ b\n```"
	artifacts := extractVersionArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractVersionArtifacts_MetadataFields(t *testing.T) {
	input := "```json:version-snapshot.json\n{\"rollback\":\"manual\"}\n```"
	artifacts := extractVersionArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" || artifacts[0].Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}

func TestNoExecOrGitWriteCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not use os/exec")
	}
	if strings.Contains(content, "exec.Command(\"git\"") || strings.Contains(content, "exec.CommandContext(\"git\"") {
		t.Fatalf("handler.go must not execute git commands")
	}
}
