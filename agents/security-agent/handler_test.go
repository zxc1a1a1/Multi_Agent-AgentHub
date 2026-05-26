package main

import "testing"

func TestScanSecurityRisks_DetectDotEnv(t *testing.T) {
	findings := scanSecurityRisks("please commit .env file")
	if len(findings) == 0 {
		t.Fatalf("expected findings for .env")
	}
}

func TestScanSecurityRisks_DetectSKTokenPattern(t *testing.T) {
	fakeToken := "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
	findings := scanSecurityRisks("token: " + fakeToken)
	if len(findings) == 0 {
		t.Fatalf("expected findings for sk-token pattern")
	}
}

func TestScanSecurityRisks_DetectPrivateKey(t *testing.T) {
	findings := scanSecurityRisks("-----BEGIN PRIVATE KEY-----")
	if len(findings) == 0 {
		t.Fatalf("expected findings for private key")
	}
}

func TestScanSecurityRisks_DetectDangerousCommandRmRF(t *testing.T) {
	findings := scanSecurityRisks("run: rm -rf /tmp/demo")
	if len(findings) == 0 {
		t.Fatalf("expected findings for rm -rf")
	}
}

func TestScanSecurityRisks_AllowsPlaceholderToken(t *testing.T) {
	findings := scanSecurityRisks("use ${AGENTHUB_API_TOKEN} in template")
	if len(findings) != 0 {
		t.Fatalf("expected no findings for allowed placeholder, got %d", len(findings))
	}
}

func TestScanSecurityRisks_AllowsTestToken(t *testing.T) {
	findings := scanSecurityRisks("this is test-token for docs")
	if len(findings) != 0 {
		t.Fatalf("expected no findings for test-token, got %d", len(findings))
	}
}

func TestBuildFallbackSecurityArtifactWhenNoLLMJSON(t *testing.T) {
	findings := scanSecurityRisks("danger: rm -rf /")
	if !hasHighRiskFindings(findings) {
		t.Fatalf("expected high-risk findings")
	}

	art := buildFallbackSecurityArtifact(findings)
	if art.Type != "security_report" {
		t.Fatalf("expected security_report, got %q", art.Type)
	}
	if art.Title != securityReportFilename {
		t.Fatalf("expected %q, got %q", securityReportFilename, art.Title)
	}
	if art.Metadata["format"] != "json" || art.Metadata["language"] != "json" {
		t.Fatalf("unexpected metadata: %#v", art.Metadata)
	}
	if art.Content == "" {
		t.Fatalf("expected non-empty fallback content")
	}
}

func TestExtractSecurityArtifacts_MetadataAndType(t *testing.T) {
	input := "```json:security-report.json\n{\"risk\":\"high\"}\n```"
	artifacts := extractSecurityArtifacts(input)
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	art := artifacts[0]
	if art.Type != "security_report" {
		t.Fatalf("expected security_report, got %q", art.Type)
	}
	if art.Title != "security-report.json" {
		t.Fatalf("expected security-report.json, got %q", art.Title)
	}
	if art.Metadata["format"] != "json" {
		t.Fatalf("expected format json, got %q", art.Metadata["format"])
	}
}
