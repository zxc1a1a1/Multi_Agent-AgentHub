package markdownagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderMarkdownToHTML(t *testing.T) {
	html, err := Render("# Title\n\n**bold** and [docs](https://example.com)\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<h1 id="title">Title</h1>`, "<strong>bold</strong>", `<a href="https://example.com">docs</a>`, "<table>"} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered HTML missing %q:\n%s", want, html)
		}
	}
}

func TestRenderDropsRawHTML(t *testing.T) {
	html, err := Render("<script>alert(1)</script>\n\n<img src=x onerror=alert(2)>\n\nsafe")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(html), "<script") || strings.Contains(strings.ToLower(html), "<img") {
		t.Fatalf("raw HTML must not reach the rendered output: %s", html)
	}
	if !strings.Contains(html, "safe") {
		t.Fatalf("safe text lost: %s", html)
	}
}

func TestLintReportsLineAndColumn(t *testing.T) {
	source := "#Bad heading\n\n[]()\n\n```\ncode\n```\n"
	issues := Lint(source)
	codes := map[string]Issue{}
	for _, issue := range issues {
		codes[issue.Code] = issue
	}
	heading, ok := codes["HEADING_MISSING_SPACE"]
	if !ok || heading.Line != 1 || heading.Column != 2 {
		t.Fatalf("heading issue missing or wrong position: %#v", issues)
	}
	link, ok := codes["EMPTY_LINK"]
	if !ok || link.Line != 3 {
		t.Fatalf("empty link issue missing: %#v", issues)
	}
	fence, ok := codes["CODE_BLOCK_NO_LANGUAGE"]
	if !ok || fence.Line != 5 {
		t.Fatalf("code fence issue missing: %#v", issues)
	}
}

func TestParseFrontmatterValidAndInvalid(t *testing.T) {
	valid, err := ParseFrontmatter("---\ntitle: Hello\ncount: 2\n---\n# Body\n")
	if err != nil {
		t.Fatal(err)
	}
	if valid.Meta["title"] != "Hello" || !strings.Contains(valid.Body, "# Body") {
		t.Fatalf("unexpected frontmatter: %#v", valid)
	}
	if _, err := ParseFrontmatter("---\ntitle: [\n---\nbody"); err == nil {
		t.Fatal("invalid YAML must return an error")
	}
	if _, err := ParseFrontmatter("---\ntitle: x\n"); err == nil {
		t.Fatal("unterminated frontmatter must return an error")
	}
}

func TestTOCAnchorsMatchRenderedIDs(t *testing.T) {
	source := "# Section One\n\n## Sub Section\n"
	html, err := Render(source)
	if err != nil {
		t.Fatal(err)
	}
	toc := GenerateTOC(source)
	if len(toc) != 2 || toc[0].Anchor != "section-one" || toc[1].Anchor != "sub-section" {
		t.Fatalf("unexpected TOC: %#v", toc)
	}
	if !strings.Contains(html, `id="section-one"`) || !strings.Contains(html, `id="sub-section"`) {
		t.Fatalf("rendered anchors do not match TOC:\n%s", html)
	}
}

func TestToolsReturnJSON(t *testing.T) {
	args := json.RawMessage(`{"markdown":"# Hi"}`)
	result, err := (LintTool{}).Execute(context.Background(), args)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("lint tool failed: %#v err=%v", result, err)
	}
	if !json.Valid([]byte(result.Content)) {
		t.Fatalf("lint output is not JSON: %s", result.Content)
	}
	renderResult, err := (RenderTool{}).Execute(context.Background(), args)
	if err != nil || renderResult.IsError || !strings.Contains(renderResult.Content, "<h1") {
		t.Fatalf("render tool failed: %#v err=%v", renderResult, err)
	}
}
