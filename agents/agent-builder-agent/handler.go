package main

import (
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const agentProfileDefaultFilename = "agent-profile.yaml"

const systemPrompt = `You are an agent profile design assistant.
Generate draft custom-agent profiles from caller text.
Profile draft should include:
- name
- description
- skills
- allowedArtifactTypes
- riskLevel
- systemPromptDraft

You MUST NOT create files, directories, or real deployment assets.
You MUST only provide profile drafts as text/yaml output.

When possible, include YAML fenced blocks with one of:
` + "```yaml:agent-profile.yaml" + `
` + "```yml:agent-profile.yaml" + `
` + "```yaml" + `
` + "```yml" + `
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
		return fmt.Errorf("agent profile generation failed: %w", err)
	}

	for _, artifact := range extractAgentProfileArtifacts(fullResponse.String()) {
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

type yamlBlock struct {
	filename string
	content  string
}

func extractAgentProfileArtifacts(text string) []adk.Artifact {
	blocks := parseYAMLBlocks(text)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		artifacts = append(artifacts, adk.Artifact{
			Type:    "agent_profile",
			Title:   block.filename,
			Content: block.content,
			Metadata: map[string]string{
				"format":   "yaml",
				"language": "yaml",
			},
		})
	}
	return artifacts
}

func parseYAMLBlocks(text string) []yamlBlock {
	var blocks []yamlBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	filename := ""
	var yamlLines []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		if !inBlock {
			var ok bool
			filename, ok = parseYAMLFenceOpenMeta(line)
			if !ok {
				continue
			}
			yamlLines = yamlLines[:0]
			inBlock = true
			continue
		}

		if isFenceCloseLine(line) {
			content := strings.TrimSpace(strings.Join(yamlLines, "\n"))
			if filename == "" {
				filename = agentProfileDefaultFilename
			}
			blocks = append(blocks, yamlBlock{
				filename: filename,
				content:  content,
			})

			inBlock = false
			filename = ""
			yamlLines = yamlLines[:0]
			continue
		}

		yamlLines = append(yamlLines, line)
	}

	return blocks
}

func parseYAMLFenceOpenMeta(line string) (string, bool) {
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
	if !isYAMLFenceLanguage(language) {
		return "", false
	}

	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), true
	}
	return "", true
}

func isYAMLFenceLanguage(language string) bool {
	return strings.EqualFold(language, "yaml") || strings.EqualFold(language, "yml")
}

func isFenceCloseLine(line string) bool {
	return strings.TrimSpace(line) == "```"
}
