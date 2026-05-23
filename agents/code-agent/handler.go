package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/agenthub/agents/adk"
)

// systemPrompt defines the code-agent's behavior.
// Per task-handler-contract section 5 (MVP code-agent handler):
//   read user message → construct LLM request → stream LLM text
//   → ctx.StreamText(chunk) → collect full reply → parse code blocks
//   → ctx.AddArtifact(type=code)
const systemPrompt = `You are a helpful code generation assistant. When the user asks you to write code:
1. Provide a brief explanation of your approach.
2. Output the complete code in fenced code blocks with the language identifier and filename like this:

` + "```go:main.go" + `
package main
// ...code here
` + "```" + `

Always specify the language and an appropriate filename after the triple backticks, separated by a colon.
If you generate multiple files, put each in its own code block.
Keep explanations concise and focus on delivering working code.`

// handleTask implements the adk.TaskHandler signature.
//
// Per task-handler-contract:
//   - Must respect context cancellation
//   - Must not return AG-UI events directly
//   - Must not expose secrets in errors
//   - Uses ctx.StreamText for text output
//   - Uses ctx.AddArtifact for code artifacts
func handleTask(ctx *adk.Context, messages []a2a.Message) error {
	llm := adk.NewLLMClient()

	// Convert A2A Messages to LLM messages
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

	// Per streaming-output rules: stream text chunks via ctx.StreamText
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
		// Per adk-runtime-contract section 20: errors must be user-safe
		return fmt.Errorf("code generation failed: %w", err)
	}

	// Per task-handler-contract: parse code blocks → ctx.AddArtifact(type=code)
	// Per adk-runtime-contract section 18: MVP only allows artifact.type = "code"
	blocks := parseCodeBlocks(fullResponse.String())
	for _, block := range blocks {
		ctx.AddArtifact(adk.Artifact{
			Type:    "code",
			Title:   block.filename,
			Content: block.code,
			Metadata: map[string]string{
				"language": block.language,
			},
		})
	}

	return nil
}

// extractTextFromParts extracts plain text from A2A message parts.
func extractTextFromParts(parts a2a.ContentParts) string {
	var texts []string
	for _, part := range parts {
		if text, ok := part.Content.(a2a.Text); ok {
			texts = append(texts, string(text))
		}
	}
	return strings.Join(texts, "\n")
}

type codeBlock struct {
	language string
	filename string
	code     string
}

// codeBlockRegex matches fenced code blocks with optional language:filename
var codeBlockRegex = regexp.MustCompile("(?s)```(\\w+)(?::([^\\n]+))?\\n(.*?)```")

func parseCodeBlocks(text string) []codeBlock {
	matches := codeBlockRegex.FindAllStringSubmatch(text, -1)
	var blocks []codeBlock
	for _, m := range matches {
		lang := m[1]
		filename := strings.TrimSpace(m[2])
		code := strings.TrimSpace(m[3])

		if filename == "" {
			filename = "untitled." + langExtension(lang)
		}

		blocks = append(blocks, codeBlock{
			language: lang,
			filename: filename,
			code:     code,
		})
	}
	return blocks
}

func langExtension(lang string) string {
	extensions := map[string]string{
		"go": "go", "python": "py", "javascript": "js", "typescript": "ts",
		"java": "java", "rust": "rs", "c": "c", "cpp": "cpp",
		"html": "html", "css": "css", "sql": "sql", "bash": "sh",
		"shell": "sh", "yaml": "yaml", "json": "json", "ruby": "rb",
		"php": "php", "swift": "swift", "kotlin": "kt",
	}
	if ext, ok := extensions[lang]; ok {
		return ext
	}
	return lang
}
