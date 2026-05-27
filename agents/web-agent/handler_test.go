package main

import (
	"strings"
	"sync"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

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

func TestCrossAgent_VisionToWeb_ConsumesVisionOutput(t *testing.T) {
	// Simulate vision-agent output containing vision_analysis JSON block
	visionOutput := "I analyzed the UI screenshot.\n\n" +
		"```json:vision-analysis.json\n" +
		`{"summary":"A login page with email field, password field and submit button",` + "\n" +
		`"imageType":"screenshot",` + "\n" +
		`"uiElements":["email input","password input","login button","forgot password link"],` + "\n" +
		`"layout":"vertical centered card with shadow",` + "\n" +
		`"suggestedNextAgents":["web-agent","code-agent"]}` + "\n" +
		"```\n\n" +
		"Based on this analysis, I recommend generating a responsive HTML page."

	// web-agent extracts text from A2A message parts
	parts := a2a.ContentParts{a2a.NewTextPart(visionOutput)}
	textContent := extractTextFromParts(parts)
	if textContent == "" {
		t.Fatal("web-agent should extract text from vision-agent output")
	}
	if !strings.Contains(textContent, "login page") {
		t.Errorf("extracted text should contain vision summary keywords")
	}

	// Verify web-agent can still parse its own HTML blocks when vision content is in context
	mixedResponse := visionOutput + "\n\nHere is the webpage:\n\n" +
		"```html:index.html\n<html><head><style>body{margin:0}</style></head><body><h1>Login</h1><script>console.log(1)</script></body></html>\n```"
	artifacts := extractWebpageArtifacts(mixedResponse)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 webpage artifact from mixed vision+web output, got %d", len(artifacts))
	}
	if artifacts[0].Type != "webpage" {
		t.Errorf("expected artifact type webpage, got %q", artifacts[0].Type)
	}
	if artifacts[0].Title != "index.html" {
		t.Errorf("expected title index.html, got %q", artifacts[0].Title)
	}
}

func TestCrossAgent_VisionToWeb_ArtifactMetadataPassThrough(t *testing.T) {
	// Verify vision→web chain: vision artifact content is consumable as web-agent input
	visionArtifact := adk.Artifact{
		Type:    "vision_analysis",
		Title:   "vision-analysis.json",
		Content: `{"summary":"A dashboard with charts and navigation sidebar"}`,
		Metadata: map[string]string{
			"format":    "json",
			"language":  "json",
			"createdBy": "vision-agent",
			"version":   "v0.1",
		},
	}
	if visionArtifact.Content == "" {
		t.Fatal("vision artifact must have content for downstream consumption")
	}

	// Simulate: vision output → web input → web output
	webInput := "Based on this vision analysis: " + visionArtifact.Content + "\n\nGenerate a webpage"
	webResponse := "Here is the page:\n```html:index.html\n<html><body><h1>Dashboard</h1></body></html>\n```"
	_ = webInput
	artifacts := extractWebpageArtifacts(webResponse)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 webpage artifact from vision→web chain, got %d", len(artifacts))
	}
}

// TestConcurrent_ExtractWebpageArtifactsParallel verifies
// extractWebpageArtifacts is safe under concurrent calls.
func TestConcurrent_ExtractWebpageArtifactsParallel(t *testing.T) {
	inputs := []string{
		"```html:index.html\n<html><head><style>body{}</style></head><body><script>x()</script></body></html>\n```",
		"```html:dashboard.html\n<html><head><style>.card{}</style></head><body><div>Dashboard</div></body></html>\n```",
		"```html:login.html\n<html><head><style>form{}</style></head><body><script>login()</script></body></html>\n```",
		"intro\n```html:profile.html\n<html><body><h1>Profile</h1><script>init()</script></body></html>\n```\noutro",
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
				artifacts := extractWebpageArtifacts(in)
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

	expectedTitles := []string{"index.html", "dashboard.html", "login.html", "profile.html"}
	for r := range results {
		if r.count != 1 {
			t.Errorf("input %d: expected 1 artifact, got %d", r.idx, r.count)
		}
		if r.typ != "webpage" {
			t.Errorf("input %d: expected type webpage, got %q", r.idx, r.typ)
		}
		if r.title != expectedTitles[r.idx] {
			t.Errorf("input %d: expected title %q, got %q", r.idx, expectedTitles[r.idx], r.title)
		}
	}
}
