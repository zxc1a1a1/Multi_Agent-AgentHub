package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractWebResearchArtifacts_JSONWithFilename(t *testing.T) {
	input := "```json:web-research.json\n{\"summary\":\"ok\"}\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "web_research" {
		t.Fatalf("expected web_research, got %q", art.Type)
	}
	if art.Title != "web-research.json" {
		t.Fatalf("expected web-research.json, got %q", art.Title)
	}
}

func TestExtractWebResearchArtifacts_JSONWithoutFilenameDefaults(t *testing.T) {
	input := "```json\n{\"k\":1}\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != webResearchDefaultFilename {
		t.Fatalf("expected %q, got %q", webResearchDefaultFilename, artifacts[0].Title)
	}
}

func TestExtractWebResearchArtifacts_NoJSONBlock(t *testing.T) {
	artifacts := extractWebResearchArtifacts("plain text")
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractWebResearchArtifacts_NonJSONIgnored(t *testing.T) {
	input := "```markdown:note.md\n# title\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractWebResearchArtifacts_MetadataFields(t *testing.T) {
	input := "```json:web-research.json\n{\"x\":1}\n```"
	artifacts := extractWebResearchArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Metadata["format"] != "json" {
		t.Fatalf("unexpected metadata: %#v", artifacts[0].Metadata)
	}
}

func TestSystemPrompt_NoActiveFetchConstraint(t *testing.T) {
	if !strings.Contains(systemPrompt, "can only analyze URL/title/body excerpts that the caller provided") {
		t.Fatalf("systemPrompt must constrain analysis source")
	}
	if !strings.Contains(systemPrompt, "MUST NOT fetch URLs") {
		t.Fatalf("systemPrompt must forbid active networking")
	}
}

func TestNoNetworkOrExecCalls(t *testing.T) {
	contentBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go failed: %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "http.Get") || strings.Contains(content, "http.Client") || strings.Contains(content, ".Do(") {
		t.Fatalf("handler.go must not perform network fetch")
	}
	if strings.Contains(content, "os/exec") || strings.Contains(content, "exec.Command") {
		t.Fatalf("handler.go must not execute commands")
	}
}
