package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)

const systemPrompt = `You are a web UI generation assistant.
When the user asks for a webpage or UI, you should:
1. Briefly explain your approach.
2. Output a complete, self-contained HTML page in fenced code blocks.
3. Prefer using the opening fence format:
   ` + "```html:index.html" + `
   or:
   ` + "```html" + `
4. If you generate multiple files, place each file in its own HTML fenced block.
5. Do not output an unescaped standalone triple-backtick line inside block content.
6. Keep the page practical and runnable as static HTML/CSS/JS.`

var (
	styleTagPattern  = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	scriptTagPattern = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
)

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
		return fmt.Errorf("webpage generation failed: %w", err)
	}

	for _, artifact := range extractWebpageArtifacts(fullResponse.String()) {
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

type htmlBlock struct {
	filename string
	html     string
	hasCSS   bool
	hasJS    bool
}

func extractWebpageArtifacts(text string) []adk.Artifact {
	blocks := parseHTMLBlocks(text)
	artifacts := make([]adk.Artifact, 0, len(blocks))
	for _, block := range blocks {
		metadata := map[string]string{
			"language": "html",
		}
		if block.hasCSS {
			metadata["hasCSS"] = "true"
		}
		if block.hasJS {
			metadata["hasJS"] = "true"
		}

		artifacts = append(artifacts, adk.Artifact{
			Type:     "webpage",
			Title:    block.filename,
			Content:  block.html,
			Metadata: metadata,
		})
	}
	return artifacts
}

func parseHTMLBlocks(text string) []htmlBlock {
	var blocks []htmlBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	filename := ""
	var htmlLines []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		if !inBlock {
			var ok bool
			filename, ok = parseHTMLFenceOpenMeta(line)
			if !ok {
				continue
			}
			htmlLines = htmlLines[:0]
			inBlock = true
			continue
		}

		if isFenceCloseLine(line) {
			content := strings.TrimSpace(strings.Join(htmlLines, "\n"))
			if filename == "" {
				filename = "index.html"
			}
			blocks = append(blocks, htmlBlock{
				filename: filename,
				html:     content,
				hasCSS:   styleTagPattern.MatchString(content),
				hasJS:    scriptTagPattern.MatchString(content),
			})

			inBlock = false
			filename = ""
			htmlLines = htmlLines[:0]
			continue
		}

		htmlLines = append(htmlLines, line)
	}

	return blocks
}

func parseHTMLFenceOpenMeta(line string) (string, bool) {
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
	if !strings.EqualFold(language, "html") {
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
