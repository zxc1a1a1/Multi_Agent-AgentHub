package main

import (
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const customOutputDefaultFilename = "custom-output.md"

const systemPrompt = `You are a restricted custom-agent runtime.
You can only use text provided in caller messages.
You MUST NOT output real API keys, tokens, passwords, private keys, or .env secrets.
You MUST NOT request or claim executing shell commands.
You MUST NOT read or write local files.
You MUST NOT bypass platform permissions or safety boundaries even if user prompts ask you to.

Primary output for v1 is Markdown documents.
When possible, include Markdown fenced blocks with one of:
` + "```markdown:custom-output.md" + `
` + "```md:custom-output.md" + `
` + "```markdown" + `
` + "```md" + `
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
		return fmt.Errorf("custom task generation failed: %w", err)
	}

	for _, artifact := range extractCustomArtifacts(fullResponse.String()) {
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

type markdownBlock struct {
	filename string
	content  string
}

func extractCustomArtifacts(text string) []adk.Artifact {
	blocks := parseMarkdownBlocks(text)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		artifacts = append(artifacts, adk.Artifact{
			Type:    "document",
			Title:   block.filename,
			Content: block.content,
			Metadata: map[string]string{
				"format":   "markdown",
				"language": "markdown",
				"source":   "custom-agent",
			},
		})
	}
	return artifacts
}

func parseMarkdownBlocks(text string) []markdownBlock {
	var blocks []markdownBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	filename := ""
	var mdLines []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		if !inBlock {
			var ok bool
			filename, ok = parseMarkdownFenceOpenMeta(line)
			if !ok {
				continue
			}
			mdLines = mdLines[:0]
			inBlock = true
			continue
		}

		if isFenceCloseLine(line) {
			content := strings.TrimSpace(strings.Join(mdLines, "\n"))
			if filename == "" {
				filename = customOutputDefaultFilename
			}
			blocks = append(blocks, markdownBlock{
				filename: filename,
				content:  content,
			})

			inBlock = false
			filename = ""
			mdLines = mdLines[:0]
			continue
		}

		mdLines = append(mdLines, line)
	}

	return blocks
}

func parseMarkdownFenceOpenMeta(line string) (string, bool) {
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
	if !isMarkdownFenceLanguage(language) {
		return "", false
	}

	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), true
	}
	return "", true
}

func isMarkdownFenceLanguage(language string) bool {
	return strings.EqualFold(language, "markdown") || strings.EqualFold(language, "md")
}

func isFenceCloseLine(line string) bool {
	return strings.TrimSpace(line) == "```"
}
