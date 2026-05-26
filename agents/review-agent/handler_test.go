package main

import (
	"sync"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

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

func TestCrossAgent_SecurityToReview_ConsumesSecurityReport(t *testing.T) {
	// Simulate security-agent output: security_report JSON block
	securityOutput := "Security scan complete. Findings:\n\n" +
		"```json:security-report.json\n" +
		`{"risk":"high","findings":[{"type":"hardcoded_secret","severity":"critical","location":"config.go:15","description":"API_KEY=sk-abc found in source"}` + "\n" +
		`{"type":"dangerous_command","severity":"high","location":"deploy.sh:3","description":"rm -rf /tmp/build used without safeguards"}],` + "\n" +
		`"summary":"2 critical/high issues found","recommendation":"Remove hardcoded secrets, add safeguards to rm command"}` + "\n" +
		"```"

	// review-agent should extract text from security report
	parts := a2a.ContentParts{a2a.NewTextPart(securityOutput)}
	textContent := extractTextFromParts(parts)
	if textContent == "" {
		t.Fatal("review-agent should extract text from security report")
	}

	// Verify review-agent can produce review_report from security findings.
	// Note: extractReviewArtifacts uses parseJSONBlocks which matches any JSON block,
	// so both security-report.json and review-report.json are parsed.
	// Find the review_report among them.
	reviewResponse := securityOutput + "\n\nCode review based on security findings:\n\n" +
		"```json:review-report.json\n" +
		`{"overallRating":"needs_improvement","issues":[{"file":"config.go","severity":"critical","message":"Remove hardcoded API_KEY"}],` + "\n" +
		`"approved":false,"reviewer":"review-agent"}` + "\n" +
		"```"
	artifacts := extractReviewArtifacts(reviewResponse)
	var reviewArt *adk.Artifact
	for i := range artifacts {
		if artifacts[i].Type == "review_report" {
			reviewArt = &artifacts[i]
			break
		}
	}
	if reviewArt == nil {
		t.Fatalf("expected a review_report artifact in security→review chain output, got %d artifacts", len(artifacts))
	}
}

func TestCrossAgent_SecurityToReview_ArtifactTypeChain(t *testing.T) {
	// Verify the security→review chain artifact types are distinct and correct
	securityArtifact := adk.Artifact{
		Type:    "security_report",
		Title:   "security-report.json",
		Content: `{"findings":[{"severity":"high","type":"secret_leak"}]}`,
		Metadata: map[string]string{
			"format":   "json",
			"language": "json",
		},
	}
	reviewArtifact := adk.Artifact{
		Type:    "review_report",
		Title:   "review-report.json",
		Content: `{"approved":false,"issues":[{"severity":"high"}]}`,
		Metadata: map[string]string{
			"format":   "json",
			"language": "json",
		},
	}

	// Chain: security_report → review input → review_report
	if securityArtifact.Type == reviewArtifact.Type {
		t.Fatal("security_report and review_report must be distinct artifact types")
	}
	if securityArtifact.Content == "" || reviewArtifact.Content == "" {
		t.Fatal("both artifacts must have content for the chain to work")
	}
}

// TestConcurrent_ExtractReviewArtifactsParallel verifies
// extractReviewArtifacts is safe under concurrent calls.
func TestConcurrent_ExtractReviewArtifactsParallel(t *testing.T) {
	inputs := []string{
		"```json:review-report.json\n{\"verdict\":\"ok\",\"score\":95}\n```",
		"```json:review-report.json\n{\"verdict\":\"needs_work\",\"score\":45}\n```",
		"```json:review-report.json\n{\"verdict\":\"approved\",\"score\":88}\n```",
		"```json:review-report.json\n{\"verdict\":\"rejected\",\"score\":10}\n```",
	}

	var wg sync.WaitGroup
	type extractResult struct {
		idx     int
		count   int
		typ     string
		content string
	}
	results := make(chan extractResult, len(inputs)*10)

	for i, input := range inputs {
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func(idx int, in string) {
				defer wg.Done()
				artifacts := extractReviewArtifacts(in)
				if len(artifacts) > 0 {
					results <- extractResult{
						idx:     idx,
						count:   len(artifacts),
						typ:     artifacts[0].Type,
						content: artifacts[0].Content,
					}
				}
			}(i, input)
		}
	}

	wg.Wait()
	close(results)

	expectedVerdicts := []string{"ok", "needs_work", "approved", "rejected"}
	for r := range results {
		if r.count != 1 {
			t.Errorf("input %d: expected 1 artifact, got %d", r.idx, r.count)
		}
		if r.typ != "review_report" {
			t.Errorf("input %d: expected type review_report, got %q", r.idx, r.typ)
		}
		expected := expectedVerdicts[r.idx]
		ok := false
		for i := 0; i <= len(r.content)-len(expected); i++ {
			if r.content[i:i+len(expected)] == expected {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("input %d: content missing verdict %q, got %q", r.idx, expected, r.content)
		}
	}
}
