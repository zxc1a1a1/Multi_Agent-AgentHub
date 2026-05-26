package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const securityReportFilename = "security-report.json"

const systemPrompt = `You are a security review assistant.
You can only analyze text provided in user messages and local rule-scan findings from this runtime context.
You MUST NOT claim reading local files, .env files, or executing shell/CLI commands.
Provide:
1. Risk summary.
2. Findings with severity.
3. Recommended mitigations.

When possible, include JSON fenced blocks with:
` + "```json:security-report.json" + `
or:
` + "```json" + `
Do not output an unescaped standalone triple-backtick line inside block content.`

var (
	secretTokenPattern   = regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)
	dotEnvPattern        = regexp.MustCompile(`(?i)\.env`)
	openAIKeyPattern     = regexp.MustCompile(`(?i)OPENAI_API_KEY`)
	anthropicKeyPattern  = regexp.MustCompile(`(?i)ANTHROPIC_API_KEY`)
	agentHubTokenPattern = regexp.MustCompile(`(?i)AGENTHUB_API_TOKEN`)
	dbPasswordPattern    = regexp.MustCompile(`(?i)DB_PASSWORD`)
	mysqlRootPattern     = regexp.MustCompile(`(?i)MYSQL_ROOT_PASSWORD`)
	databaseURLPattern   = regexp.MustCompile(`(?i)DATABASE_URL`)
	privateKeyPattern    = regexp.MustCompile(`(?i)PRIVATE KEY`)
	beginRSAPattern      = regexp.MustCompile(`(?i)BEGIN RSA`)
	rmRFPattern          = regexp.MustCompile(`(?i)\brm\s+-rf\b`)
	gitResetHardPattern  = regexp.MustCompile(`(?i)\bgit\s+reset\s+--hard\b`)
	dockerPrunePattern   = regexp.MustCompile(`(?i)\bdocker\s+system\s+prune\b`)
	allowedTokenReplacer = strings.NewReplacer("${AGENTHUB_API_TOKEN}", "<allowed-placeholder>", "${MYSQL_ROOT_PASSWORD}", "<allowed-placeholder>", "test-token", "<allowed-test-token>", "<本地测试token>", "<allowed-local-token>")
)

type scanRule struct {
	name     string
	severity string
	pattern  *regexp.Regexp
}

type securityFinding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type securityFallbackReport struct {
	Source    string            `json:"source"`
	Summary   string            `json:"summary"`
	HighRisk  bool              `json:"highRisk"`
	Findings  []securityFinding `json:"findings"`
	NextSteps []string          `json:"nextSteps"`
}

func handleTask(ctx *adk.Context, messages []a2a.Message) error {
	return defaultTaskHandler(ctx, messages)
}

func handleTaskWithLLM(llm *adk.LLMClient) adk.TaskHandler {
	return func(ctx *adk.Context, messages []a2a.Message) error {
		return processTaskWithLLM(llm, ctx, messages)
	}
}

func processTaskWithLLM(llm *adk.LLMClient, ctx *adk.Context, messages []a2a.Message) error {
	llmMsgs := make([]adk.LLMMessage, 0, len(messages)+1)
	var combinedText strings.Builder
	for _, msg := range messages {
		role := string(msg.Role)
		content := extractTextFromParts(msg.Parts)
		if content != "" {
			llmMsgs = append(llmMsgs, adk.LLMMessage{Role: role, Content: content})
			combinedText.WriteString(content)
			combinedText.WriteString("\n")
		}
	}

	if len(llmMsgs) == 0 {
		return fmt.Errorf("no messages to process")
	}

	findings := scanSecurityRisks(combinedText.String())
	if len(findings) > 0 {
		findingsJSON, _ := json.Marshal(findings)
		llmMsgs = append(llmMsgs, adk.LLMMessage{
			Role:    "user",
			Content: "Rule-scan findings from provided text (pre-scan only): " + string(findingsJSON),
		})
	}

	var fullResponse strings.Builder
	err := llm.StreamCompletion(
		ctx.Context(),
		systemPrompt,
		llmMsgs,
		func(chunk string) {
			fullResponse.WriteString(chunk)
			ctx.StreamText(chunk)
		},
	)
	if err != nil {
		return fmt.Errorf("security report generation failed: %w", err)
	}

	artifacts := extractSecurityArtifacts(fullResponse.String())
	if len(artifacts) == 0 && hasHighRiskFindings(findings) {
		artifacts = append(artifacts, buildFallbackSecurityArtifact(findings))
	}

	for _, artifact := range artifacts {
		ctx.AddArtifact(artifact)
	}

	return nil
}

var defaultTaskHandler = handleTaskWithLLM(adk.NewLLMClient())

func extractTextFromParts(parts a2a.ContentParts) string {
	var texts []string
	for _, part := range parts {
		if text, ok := part.Content.(a2a.Text); ok {
			texts = append(texts, string(text))
		}
	}
	return strings.Join(texts, "\n")
}

type jsonBlock struct {
	filename string
	content  string
}

func extractSecurityArtifacts(text string) []adk.Artifact {
	blocks := parseJSONBlocks(text, securityReportFilename)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		filename := block.filename
		if filename == "" {
			filename = securityReportFilename
		}
		artifacts = append(artifacts, adk.Artifact{
			Type:    "security_report",
			Title:   filename,
			Content: block.content,
			Metadata: map[string]string{
				"format":   "json",
				"language": "json",
			},
		})
	}
	return artifacts
}

func parseJSONBlocks(text, defaultFilename string) []jsonBlock {
	var blocks []jsonBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	filename := ""
	var jsonLines []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		if !inBlock {
			var ok bool
			filename, ok = parseJSONFenceOpenMeta(line)
			if !ok {
				continue
			}
			jsonLines = jsonLines[:0]
			inBlock = true
			continue
		}

		if isFenceCloseLine(line) {
			content := strings.TrimSpace(strings.Join(jsonLines, "\n"))
			if filename == "" {
				filename = defaultFilename
			}
			blocks = append(blocks, jsonBlock{
				filename: filename,
				content:  content,
			})

			inBlock = false
			filename = ""
			jsonLines = jsonLines[:0]
			continue
		}

		jsonLines = append(jsonLines, line)
	}

	return blocks
}

func parseJSONFenceOpenMeta(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return "", false
	}

	header := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	if header == "" {
		return "", false
	}

	parts := strings.SplitN(header, ":", 2)
	language := strings.TrimSpace(parts[0])
	if !strings.EqualFold(language, "json") {
		return "", false
	}

	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), true
	}
	return "", true
}

func isFenceCloseLine(line string) bool {
	return strings.TrimSpace(line) == "```"
}

func scanSecurityRisks(text string) []securityFinding {
	normalized := allowedTokenReplacer.Replace(text)
	rules := []scanRule{
		{name: ".env", severity: "high", pattern: dotEnvPattern},
		{name: "OPENAI_API_KEY", severity: "high", pattern: openAIKeyPattern},
		{name: "ANTHROPIC_API_KEY", severity: "high", pattern: anthropicKeyPattern},
		{name: "AGENTHUB_API_TOKEN", severity: "high", pattern: agentHubTokenPattern},
		{name: "DB_PASSWORD", severity: "high", pattern: dbPasswordPattern},
		{name: "MYSQL_ROOT_PASSWORD", severity: "high", pattern: mysqlRootPattern},
		{name: "DATABASE_URL", severity: "high", pattern: databaseURLPattern},
		{name: "PRIVATE KEY", severity: "critical", pattern: privateKeyPattern},
		{name: "BEGIN RSA", severity: "critical", pattern: beginRSAPattern},
		{name: "sk-token", severity: "critical", pattern: secretTokenPattern},
		{name: "rm -rf", severity: "critical", pattern: rmRFPattern},
		{name: "git reset --hard", severity: "high", pattern: gitResetHardPattern},
		{name: "docker system prune", severity: "high", pattern: dockerPrunePattern},
	}

	findings := make([]securityFinding, 0)
	for _, rule := range rules {
		if rule.pattern.MatchString(normalized) {
			findings = append(findings, securityFinding{
				Rule:     rule.name,
				Severity: rule.severity,
				Message:  "Matched risky pattern in provided text.",
			})
		}
	}
	return findings
}

func hasHighRiskFindings(findings []securityFinding) bool {
	for _, f := range findings {
		if f.Severity == "high" || f.Severity == "critical" {
			return true
		}
	}
	return false
}

func buildFallbackSecurityArtifact(findings []securityFinding) adk.Artifact {
	report := securityFallbackReport{
		Source:   "rule_scan_fallback",
		Summary:  "High-risk patterns detected in provided text. LLM did not return a JSON security report block.",
		HighRisk: true,
		Findings: findings,
		NextSteps: []string{
			"Remove or redact secrets and credentials from shared content.",
			"Avoid destructive commands unless explicitly approved.",
			"Re-run security review with sanitized inputs.",
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		raw = []byte(`{"source":"rule_scan_fallback","summary":"failed to serialize fallback report","highRisk":true}`)
	}

	return adk.Artifact{
		Type:    "security_report",
		Title:   securityReportFilename,
		Content: string(raw),
		Metadata: map[string]string{
			"format":   "json",
			"language": "json",
		},
	}
}
