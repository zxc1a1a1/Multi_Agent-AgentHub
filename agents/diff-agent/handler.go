package main

import (
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const diffDefaultFilename = "patch.diff"

const systemPrompt = `You are a diff generation and review assistant.
You may generate unified diffs and explain patch intent/risk.
You MUST NOT claim that you executed git apply, git checkout, git reset, or any shell command.
You MUST NOT apply patches.
When possible, include diff fenced blocks with:
` + "```diff:patch.diff" + `
or:
` + "```diff" + `
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
		return fmt.Errorf("diff generation failed: %w", err)
	}

	for _, artifact := range extractDiffArtifacts(fullResponse.String()) {
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

type diffBlock struct {
	filename string
	content  string
}

func extractDiffArtifacts(text string) []adk.Artifact {
	blocks := parseDiffBlocks(text)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		artifacts = append(artifacts, adk.Artifact{
			Type:    "diff",
			Title:   block.filename,
			Content: block.content,
			Metadata: map[string]string{
				"format":      "unified_diff",
				"applyStatus": "not_applied",
				"language":    "diff",
			},
		})
	}
	return artifacts
}

func parseDiffBlocks(text string) []diffBlock {
	var blocks []diffBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	filename := ""
	var diffLines []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		if !inBlock {
			var ok bool
			filename, ok = parseDiffFenceOpenMeta(line)
			if !ok {
				continue
			}
			diffLines = diffLines[:0]
			inBlock = true
			continue
		}

		if isFenceCloseLine(line) {
			content := strings.TrimSpace(strings.Join(diffLines, "\n"))
			if filename == "" {
				filename = diffDefaultFilename
			}
			blocks = append(blocks, diffBlock{
				filename: filename,
				content:  content,
			})

			inBlock = false
			filename = ""
			diffLines = diffLines[:0]
			continue
		}

		diffLines = append(diffLines, line)
	}

	return blocks
}

func parseDiffFenceOpenMeta(line string) (string, bool) {
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
	if !strings.EqualFold(language, "diff") {
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
