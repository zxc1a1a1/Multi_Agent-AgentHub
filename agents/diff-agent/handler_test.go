package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractDiffArtifacts_WithFilename(t *testing.T) {
	input := "```diff:patch.diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "diff" {
		t.Fatalf("expected diff, got %q", art.Type)
	}
	if art.Title != "patch.diff" {
		t.Fatalf("expected patch.diff, got %q", art.Title)
	}
	if art.Metadata["format"] != "unified_diff" {
		t.Fatalf("expected unified_diff, got %q", art.Metadata["format"])
	}
	if art.Metadata["applyStatus"] != "not_applied" {
		t.Fatalf("expected not_applied, got %q", art.Metadata["applyStatus"])
	}
}

func TestExtractDiffArtifacts_WithoutFilenameDefaults(t *testing.T) {
	input := "```diff\n--- a/a.txt\n+++ b/a.txt\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != diffDefaultFilename {
		t.Fatalf("expected %q, got %q", diffDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractDiffArtifacts_MultipleBlocks(t *testing.T) {
	input := "```diff:a.diff\n--- a\n+++ b\n```\ntext\n```diff:b.diff\n--- c\n+++ d\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDiffArtifacts_NoDiffBlock(t *testing.T) {
	artifacts := extractDiffArtifacts("no diff block")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractDiffArtifacts_NonDiffIgnored(t *testing.T) {
	input := "```json:patch.json\n{\"k\":1}\n```"
	artifacts := extractDiffArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestNoExecOrGitApplyCalls(t *testing.T) {
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
