package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractSlideDeckArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:slide-deck.json\n{\"title\":\"Demo\"}\n```"
	artifacts := extractSlideDeckArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "slide_deck" {
		t.Fatalf("expected slide_deck, got %q", art.Type)
	}
	if art.Title != "slide-deck.json" {
		t.Fatalf("expected slide-deck.json, got %q", art.Title)
	}
}

func TestExtractSlideDeckArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"title\":\"Demo\"}\n```"
	artifacts := extractSlideDeckArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != slideDeckDefaultFilename {
		t.Fatalf("expected %q, got %q", slideDeckDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractSlideDeckArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractSlideDeckArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractSlideDeckArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```markdown:deck.md\n# x\n```"
	artifacts := extractSlideDeckArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractSlideDeckArtifacts_MetadataFields(t *testing.T) {
	input := "```json:slide-deck.json\n{\"slides\":[]}\n```"
	artifacts := extractSlideDeckArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	meta := artifacts[0].Metadata
	if meta["format"] != "json" || meta["fileGenerated"] != "false" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestSystemPrompt_DraftOnlyConstraint(t *testing.T) {
	if !strings.Contains(systemPrompt, "MUST only output draft content") {
		t.Fatalf("systemPrompt must constrain to draft output")
	}
	if !strings.Contains(systemPrompt, "MUST NOT claim that a real pptx file was generated") {
		t.Fatalf("systemPrompt must forbid fake pptx generation claims")
	}
}

func TestNoWriteOrExecCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "os.WriteFile") || strings.Contains(content, "os.Create") {
		t.Fatalf("handler.go must not write files")
	}
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not execute commands")
	}
	if strings.Contains(content, "exec.Command(\"python\"") || strings.Contains(content, "exec.Command(\"libreoffice\"") {
		t.Fatalf("handler.go must not run python/libreoffice commands")
	}
}
