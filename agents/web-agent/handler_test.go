package main

import "testing"

func TestExtractWebpageArtifacts_HTMLWithFilename(t *testing.T) {
	input := "intro\n```html:index.html\n<html><head><style>body{}</style></head><body><script>console.log(1)</script></body></html>\n```\noutro"

	artifacts := extractWebpageArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}

	art := artifacts[0]
	if art.Type != "webpage" {
		t.Fatalf("expected type webpage, got %q", art.Type)
	}
	if art.Title != "index.html" {
		t.Fatalf("expected title index.html, got %q", art.Title)
	}
	if art.Content != "<html><head><style>body{}</style></head><body><script>console.log(1)</script></body></html>" {
		t.Fatalf("unexpected content: %q", art.Content)
	}
	if art.Metadata["language"] != "html" {
		t.Fatalf("expected language html, got %q", art.Metadata["language"])
	}
	if art.Metadata["hasCSS"] != "true" {
		t.Fatalf("expected hasCSS=true, got %q", art.Metadata["hasCSS"])
	}
	if art.Metadata["hasJS"] != "true" {
		t.Fatalf("expected hasJS=true, got %q", art.Metadata["hasJS"])
	}
}

func TestExtractWebpageArtifacts_HTMLWithoutFilenameDefaultsToIndex(t *testing.T) {
	input := "```html\n<html><body>Hello</body></html>\n```"

	artifacts := extractWebpageArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Title != "index.html" {
		t.Fatalf("expected index.html, got %q", artifacts[0].Title)
	}
}

func TestExtractWebpageArtifacts_MultipleHTMLBlocks(t *testing.T) {
	input := "```html:index.html\n<html>A</html>\n```\ntext\n```html:dashboard.html\n<html>B</html>\n```"

	artifacts := extractWebpageArtifacts(input)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].Title != "index.html" || artifacts[1].Title != "dashboard.html" {
		t.Fatalf("unexpected titles: %q, %q", artifacts[0].Title, artifacts[1].Title)
	}
}

func TestExtractWebpageArtifacts_NoHTMLBlocks(t *testing.T) {
	input := "no html fenced block here"
	artifacts := extractWebpageArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestExtractWebpageArtifacts_NonHTMLFenceIgnored(t *testing.T) {
	input := "```markdown:report.md\n# title\n```"
	artifacts := extractWebpageArtifacts(input)
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts for non-html fences, got %d", len(artifacts))
	}
}

func TestParseHTMLBlocks_BackticksInsideContentDoNotClose(t *testing.T) {
	input := "```html:index.html\n<p>`inline` and ``double`` backticks</p>\n<div>ok</div>\n```"

	blocks := parseHTMLBlocks(input)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	expected := "<p>`inline` and ``double`` backticks</p>\n<div>ok</div>"
	if blocks[0].html != expected {
		t.Fatalf("unexpected html: %q", blocks[0].html)
	}
}
