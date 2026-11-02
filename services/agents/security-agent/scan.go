package securityagent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SecretFinding never contains the full secret value.
type SecretFinding struct {
	Rule     string `json:"rule"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Redacted string `json:"redacted"`
}

type secretRule struct {
	name    string
	pattern *regexp.Regexp
}

var secretRules = []secretRule{
	{"PRIVATE_KEY", regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
	{"AWS_ACCESS_KEY", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"GITHUB_TOKEN", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`)},
	{"SLACK_TOKEN", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`)},
	{"JWT", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
	{"CREDENTIAL_ASSIGNMENT", regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|passwd)\s*[:=]\s*["']?([A-Za-z0-9_\-./+=]{12,})`)},
}

// ScanSecrets returns findings with redacted values; the original secret is
// never echoed in the result.
func ScanSecrets(content string) []SecretFinding {
	findings := make([]SecretFinding, 0)
	lines := strings.Split(content, "\n")
	for lineIndex, line := range lines {
		for _, rule := range secretRules {
			for _, match := range rule.pattern.FindAllStringSubmatchIndex(line, -1) {
				start := match[0]
				value := line[match[0]:match[1]]
				if rule.name == "CREDENTIAL_ASSIGNMENT" && len(match) >= 6 {
					value = line[match[4]:match[5]]
					start = match[4]
				}
				findings = append(findings, SecretFinding{
					Rule: rule.name, Line: lineIndex + 1, Column: start + 1,
					Redacted: redact(value),
				})
			}
		}
	}
	return findings
}

func redact(value string) string {
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + "***" + value[len(value)-2:]
}

// CVEFinding is one CVE identified in a vulnerability report.
type CVEFinding struct {
	ID      string `json:"id"`
	Package string `json:"package,omitempty"`
	Summary string `json:"summary,omitempty"`
}

var (
	cvePattern     = regexp.MustCompile(`CVE-\d{4}-\d{4,}`)
	packagePattern = regexp.MustCompile(`(?i)(?:package|module)\s*[:=]?\s*([A-Za-z0-9_.\-/@]+)`)
)

// ParseVulnReport extracts CVE ids from plain text or JSON reports (including
// govulncheck-style aliases). Duplicate ids are collapsed.
func ParseVulnReport(report string) ([]CVEFinding, error) {
	if strings.TrimSpace(report) == "" {
		return nil, fmt.Errorf("vulnerability report is empty")
	}
	byID := map[string]CVEFinding{}
	collect := func(text, packageName string) {
		for _, id := range cvePattern.FindAllString(text, -1) {
			id = strings.ToUpper(id)
			existing := byID[id]
			if existing.ID == "" {
				existing.ID = id
			}
			if existing.Package == "" && packageName != "" {
				existing.Package = packageName
			}
			byID[id] = existing
		}
	}

	var parsed any
	if json.Unmarshal([]byte(report), &parsed) == nil {
		walkJSON(parsed, "", collect)
	} else {
		lines := strings.Split(report, "\n")
		lastPackage := ""
		for _, line := range lines {
			if match := packagePattern.FindStringSubmatch(line); match != nil {
				lastPackage = strings.Trim(match[1], `"'`)
			}
			collect(line, lastPackage)
		}
	}
	if len(byID) == 0 {
		return nil, fmt.Errorf("no CVE ids found")
	}
	out := make([]CVEFinding, 0, len(byID))
	for _, finding := range byID {
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func walkJSON(value any, packageName string, collect func(string, string)) {
	switch typed := value.(type) {
	case map[string]any:
		currentPackage := packageName
		if name, ok := typed["module"].(string); ok {
			currentPackage = name
		}
		if name, ok := typed["package"].(string); ok {
			currentPackage = name
		}
		if id, ok := typed["id"].(string); ok {
			collect(id, currentPackage)
		}
		if aliases, ok := typed["aliases"].([]any); ok {
			for _, alias := range aliases {
				if text, ok := alias.(string); ok {
					collect(text, currentPackage)
				}
			}
		}
		for _, child := range typed {
			walkJSON(child, currentPackage, collect)
		}
	case []any:
		for _, child := range typed {
			walkJSON(child, packageName, collect)
		}
	case string:
		collect(typed, packageName)
	}
}
