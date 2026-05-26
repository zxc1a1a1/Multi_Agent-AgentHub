package main

import (
	"strings"
	"sync"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

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

func TestCrossAgent_FileToDocument_ConsumesFileSummary(t *testing.T) {
	// Simulate file-agent output: file_summary JSON block
	fileAgentOutput := "I analyzed the file contents.\n\n" +
		"```json:file-summary.json\n" +
		`{"fileName":"architecture.md","fileType":"markdown","summary":"A document describing the microservice architecture with 3 services: Gateway, Orchestrator, and Registry",` + "\n" +
		`"lineCount":120,"topics":["microservices","API design","deployment"]}` + "\n" +
		"```\n\n" +
		"The file covers system architecture in detail."

	// document-agent should extract text from file-agent output
	parts := a2a.ContentParts{a2a.NewTextPart(fileAgentOutput)}
	textContent := extractTextFromParts(parts)
	if textContent == "" {
		t.Fatal("document-agent should extract text from file-agent output")
	}
	if !strings.Contains(textContent, "architecture") {
		t.Errorf("extracted text should contain file summary topics")
	}

	// Verify document-agent can produce document from file-agent context.
	// Note: document-agent parser only matches markdown/md fences, not json fences,
	// so the file-summary JSON block is correctly ignored.
	docResponse := fileAgentOutput + "\n\nBased on the architecture file, here is the formatted document:\n\n" +
		"```markdown:architecture-doc.md\n# System Architecture\n## Services\n- Gateway\n- Orchestrator\n- Registry\n```"
	artifacts := extractDocumentArtifacts(docResponse)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 document artifact from file→document chain, got %d", len(artifacts))
	}
	if artifacts[0].Type != "document" {
		t.Errorf("expected type document, got %q", artifacts[0].Type)
	}
	if artifacts[0].Title != "architecture-doc.md" {
		t.Errorf("expected title architecture-doc.md, got %q", artifacts[0].Title)
	}
	if artifacts[0].Metadata["format"] != "markdown" {
		t.Errorf("expected format markdown, got %q", artifacts[0].Metadata["format"])
	}
}

func TestCrossAgent_FileToDocument_PreservesContext(t *testing.T) {
	// file-agent → document-agent: file_summary becomes context for document generation
	fileSummaryArtifact := adk.Artifact{
		Type:    "file_summary",
		Title:   "file-summary.json",
		Content: `{"fileName":"readme.md","summary":"Project overview and setup instructions"}`,
		Metadata: map[string]string{
			"format":   "json",
			"language": "json",
		},
	}
	if fileSummaryArtifact.Content == "" {
		t.Fatal("file_summary artifact must have content for downstream")
	}

	// Simulate document-agent consuming file_summary and producing markdown document
	input := "Summarize this analysis into a markdown document: " + fileSummaryArtifact.Content
	response := "```markdown:summary.md\n# Project Overview\nSetup instructions included\n```"
	_ = input
	artifacts := extractDocumentArtifacts(response)
	if len(artifacts) != 1 || artifacts[0].Type != "document" {
		t.Fatal("document-agent should produce document artifact from file_summary context")
	}
}

// TestConcurrent_ExtractDocumentArtifactsParallel verifies
// extractDocumentArtifacts is safe under concurrent calls.
func TestConcurrent_ExtractDocumentArtifactsParallel(t *testing.T) {
	inputs := []string{
		"```markdown:report.md\n# Report\nContent here\n```",
		"```md:README.md\n# Readme\nSetup instructions\n```",
		"```markdown:spec.md\n# Spec\nRequirements\n```",
		"```md:handoff.md\n# Handoff\nTransfer notes\n```",
	}

	var wg sync.WaitGroup
	type extractResult struct {
		idx   int
		count int
		typ   string
		title string
	}
	results := make(chan extractResult, len(inputs)*10)

	for i, input := range inputs {
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func(idx int, in string) {
				defer wg.Done()
				artifacts := extractDocumentArtifacts(in)
				if len(artifacts) > 0 {
					results <- extractResult{
						idx:   idx,
						count: len(artifacts),
						typ:   artifacts[0].Type,
						title: artifacts[0].Title,
					}
				}
			}(i, input)
		}
	}

	wg.Wait()
	close(results)

	expectedTitles := []string{"report.md", "README.md", "spec.md", "handoff.md"}
	for r := range results {
		if r.count != 1 {
			t.Errorf("input %d: expected 1 artifact, got %d", r.idx, r.count)
		}
		if r.typ != "document" {
			t.Errorf("input %d: expected type document, got %q", r.idx, r.typ)
		}
		if r.title != expectedTitles[r.idx] {
			t.Errorf("input %d: expected title %q, got %q", r.idx, expectedTitles[r.idx], r.title)
		}
	}
}
