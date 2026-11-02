package testagent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TestFailure is one failing test extracted from a log.
type TestFailure struct {
	Test    string   `json:"test"`
	Package string   `json:"package,omitempty"`
	Message string   `json:"message,omitempty"`
	Details []string `json:"details,omitempty"`
}

// TestSummary is the deterministic result of parsing a test log.
type TestSummary struct {
	Framework string        `json:"framework"`
	Total     int           `json:"total"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	Failures  []TestFailure `json:"failures"`
	Coverage  *float64      `json:"coveragePercent,omitempty"`
}

var (
	goTestResultPattern   = regexp.MustCompile(`^\s*--- (PASS|FAIL|SKIP): ([^\s]+)`)
	goTestPackagePattern  = regexp.MustCompile(`^(ok|FAIL)\s+([^\s]+)`)
	goTestErrorLine       = regexp.MustCompile(`(?:_test\.go:\d+:|\bpanic:|Error Trace:|Error:)`)
	goCoveragePercent    = regexp.MustCompile(`coverage:\s*([0-9]+(?:\.[0-9]+)?)%`)
	goCoverFuncTotal      = regexp.MustCompile(`total:\s*\(statements\)\s*([0-9]+(?:\.[0-9]+)?)%`)
	coverProfileLine     = regexp.MustCompile(`^(.+?):(\d+)\.(\d+),(\d+)\.(\d+)\s+(\d+)\s+(\d+)$`)
)

// ParseTestLog extracts pass/fail/skip counts, failure details and coverage from
// `go test -v` output. Unknown lines are ignored.
func ParseTestLog(log string, framework string) TestSummary {
	summary := TestSummary{Framework: framework, Failures: make([]TestFailure, 0)}
	if summary.Framework == "" {
		summary.Framework = "go test"
	}
	lines := strings.Split(strings.ReplaceAll(log, "\r\n", "\n"), "\n")
	current := -1
	currentPackage := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if match := goTestPackagePattern.FindStringSubmatch(trimmed); match != nil {
			if match[1] == "ok" || match[1] == "FAIL" {
				currentPackage = match[2]
			}
		}
		if match := goTestResultPattern.FindStringSubmatch(line); match != nil {
			summary.Total++
			switch match[1] {
			case "PASS":
				summary.Passed++
				current = -1
			case "SKIP":
				summary.Skipped++
				current = -1
			case "FAIL":
				summary.Failed++
				summary.Failures = append(summary.Failures, TestFailure{
					Test: match[2], Package: currentPackage,
				})
				current = len(summary.Failures) - 1
			}
			continue
		}
		if current >= 0 {
			if goTestErrorLine.MatchString(trimmed) && len(summary.Failures[current].Details) < 5 {
				summary.Failures[current].Details = append(summary.Failures[current].Details, trimmed)
				if summary.Failures[current].Message == "" {
					summary.Failures[current].Message = trimmed
				}
			}
		}
		if percent := coverageFromLine(trimmed); percent != nil {
			summary.Coverage = percent
		}
	}
	return summary
}

func coverageFromLine(line string) *float64 {
	if match := goCoveragePercent.FindStringSubmatch(line); match != nil {
		value, err := strconv.ParseFloat(match[1], 64)
		if err == nil {
			return &value
		}
	}
	if match := goCoverFuncTotal.FindStringSubmatch(line); match != nil {
		value, err := strconv.ParseFloat(match[1], 64)
		if err == nil {
			return &value
		}
	}
	return nil
}

// ParseCoverage computes statement coverage from a Go cover profile. Returns an
// error when the profile contains no statements.
func ParseCoverage(profile string) (float64, error) {
	total, covered := 0, 0
	for _, line := range strings.Split(strings.ReplaceAll(profile, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		match := coverProfileLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		statements, err := strconv.Atoi(match[6])
		if err != nil {
			return 0, err
		}
		count, err := strconv.Atoi(match[7])
		if err != nil {
			return 0, err
		}
		total += statements
		if count > 0 {
			covered += statements
		}
	}
	if total == 0 {
		return 0, fmt.Errorf("coverage profile contains no statements")
	}
	return float64(covered) / float64(total) * 100, nil
}

// AssessCoverage compares coverage against a threshold.
func AssessCoverage(percent, threshold float64) map[string]any {
	if threshold <= 0 {
		threshold = 80
	}
	return map[string]any{
		"coveragePercent": percent,
		"threshold":       threshold,
		"pass":            percent >= threshold,
		"delta":           percent - threshold,
	}
}
