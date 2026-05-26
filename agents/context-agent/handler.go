package main

import (
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const contextBundleDefaultFilename = "context-bundle.json"

const systemPrompt = `You are a context preparation assistant for multi-agent workflows.
Given conversation text, produce:
1. A concise summary of current context.
2. Key requirements.
3. Hard constraints and risks.
4. Related messages/artifacts references when available.

When possible, include one or more JSON fenced blocks with:
` + "```json:context-bundle.json" + `
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
		return fmt.Errorf("context bundle generation failed: %w", err)
	}

	for _, artifact := range extractContextArtifacts(fullResponse.String()) {
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

func extractContextArtifacts(text string) []adk.Artifact {
	blocks := parseJSONBlocks(text, contextBundleDefaultFilename)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		artifacts = append(artifacts, adk.Artifact{
			Type:    "context_bundle",
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
