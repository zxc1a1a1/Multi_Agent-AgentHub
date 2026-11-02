package securityagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const secretFixture = `aws_access_key_id = AKIAIOSFODNN7EXAMPLE
github_token: ghp_abcdefghijklmnopqrstuvwxyz0123456789
password = "supersecretpassword123"
-----BEGIN RSA PRIVATE KEY-----
clean line
`

func TestScanSecretsFindsAndRedacts(t *testing.T) {
	findings := ScanSecrets(secretFixture)
	if len(findings) < 3 {
		t.Fatalf("expected at least 3 findings, got %#v", findings)
	}
	rules := map[string]bool{}
	for _, finding := range findings {
		rules[finding.Rule] = true
		if finding.Line <= 0 || finding.Column <= 0 || finding.Redacted == "" {
			t.Fatalf("incomplete finding: %#v", finding)
		}
	}
	for _, want := range []string{"AWS_ACCESS_KEY", "GITHUB_TOKEN", "PRIVATE_KEY", "CREDENTIAL_ASSIGNMENT"} {
		if !rules[want] {
			t.Fatalf("rule %s not matched: %#v", want, findings)
		}
	}
	encoded, _ := json.Marshal(findings)
	for _, secret := range []string{"AKIAIOSFODNN7EXAMPLE", "ghp_abcdefghijklmnopqrstuvwxyz0123456789", "supersecretpassword123"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("secret leaked in findings: %s", encoded)
		}
	}
}

func TestScanSecretsCleanContent(t *testing.T) {
	if findings := ScanSecrets("func main() {}\n"); len(findings) != 0 {
		t.Fatalf("unexpected findings: %#v", findings)
	}
}

func TestParseVulnReportPlainAndJSON(t *testing.T) {
	plain := "package golang.org/x/net\nCVE-2023-44487\nCVE-2023-44487\nCVE-2024-1234\n"
	findings, err := ParseVulnReport(plain)
	if err != nil || len(findings) != 2 {
		t.Fatalf("plain parse: %#v err=%v", findings, err)
	}
	if findings[0].ID != "CVE-2023-44487" || findings[0].Package == "" {
		t.Fatalf("unexpected plain findings: %#v", findings)
	}

	jsonReport := `{"findings":[{"osv":{"id":"GO-2024-0001","aliases":["CVE-2024-9999"]},"trace":[{"module":"example.com/vuln"}]}]}`
	findings, err = ParseVulnReport(jsonReport)
	if err != nil || len(findings) != 1 || findings[0].ID != "CVE-2024-9999" {
		t.Fatalf("json parse: %#v err=%v", findings, err)
	}
	if _, err := ParseVulnReport("no vulnerabilities here"); err == nil {
		t.Fatal("report without CVE must error")
	}
}

func TestDetectSecretsToolReturnsRedactedJSON(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"content": secretFixture})
	result, err := (DetectSecretsTool{}).Execute(context.Background(), args)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("tool failed: %#v err=%v", result, err)
	}
	if strings.Contains(result.Content, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("tool leaked secret: %s", result.Content)
	}
	if !strings.Contains(result.Content, "AWS_ACCESS_KEY") {
		t.Fatalf("tool missed AWS key: %s", result.Content)
	}
}
