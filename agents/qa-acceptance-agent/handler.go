package main

import (
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const acceptanceReportDefaultFilename = "acceptance-report.json"

const systemPrompt = `You are a QA acceptance assistant.
Use only the provided text context (requirements, artifacts, tests, reviews) to decide acceptance.
Return:
1. Final acceptance decision.
2. Passed items.
3. Failed items.
4. Items requiring manual confirmation.
Do not claim access to external systems.

When possible, include JSON fenced blocks with:
` + "```json:acceptance-report.json" + `
or:
` + "```json" + `
Do not output an unescaped standalone triple-backtick line inside block content.`

func handleTask(ctx *adk.Context, messages []a2a.Message) error {
	return defaultTaskHandler(ctx, messages)
}

func handleTaskWithLLM(llm *adk.LLMClient) adk.TaskHandler {
	return func(ctx *adk.Context, messages []a2a.Message) error {
		return processTaskWithLLM(llm, ctx, messages)
	}
}

func processTaskWithLLM(llm *adk.LLMClient, ctx *adk.Context, messages []a2a.Message) error {
	llmMsgs := make([]adk.LLMMessage, 0, len(messages))
	for _, msg := range messages {
		role := string(msg.Role)
		content := extractTextFromParts(msg.Parts)
		if content != "" {
			llmMsgs = append(llmMsgs, adk.LLMMessage{Role: role, Content: content})
		}
	}

	if len(llmMsgs) == 0 {
		return fmt.Errorf("no messages to process")
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
		return fmt.Errorf("acceptance report generation failed: %w", err)
	}

	for _, artifact := range extractAcceptanceArtifacts(fullResponse.String()) {
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

func extractAcceptanceArtifacts(text string) []adk.Artifact {
	blocks := parseJSONBlocks(text, acceptanceReportDefaultFilename)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		artifacts = append(artifacts, adk.Artifact{
			Type:    "acceptance_report",
			Title:   block.filename,
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
