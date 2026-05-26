package main

import (
	"strings"
	"sync"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func TestParseCodeBlocksStandardWithFilename(t *testing.T) {
	input := "intro\n```go:main.go\npackage main\nfunc main() {}\n```\noutro"

	blocks := parseCodeBlocks(input)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].language != "go" {
		t.Fatalf("expected language go, got %q", blocks[0].language)
	}
	if blocks[0].filename != "main.go" {
		t.Fatalf("expected filename main.go, got %q", blocks[0].filename)
	}
	if blocks[0].code != "package main\nfunc main() {}" {
		t.Fatalf("unexpected code: %q", blocks[0].code)
	}
}

func TestParseCodeBlocksWithoutFilename(t *testing.T) {
	input := "```go\npackage main\n```\n"

	blocks := parseCodeBlocks(input)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].language != "go" {
		t.Fatalf("expected language go, got %q", blocks[0].language)
	}
	if blocks[0].filename != "untitled.go" {
		t.Fatalf("expected filename untitled.go, got %q", blocks[0].filename)
	}
	if blocks[0].code != "package main" {
		t.Fatalf("unexpected code: %q", blocks[0].code)
	}
}

func TestParseCodeBlocksMultipleBlocks(t *testing.T) {
	input := "```go:main.go\npackage main\n```\ntext\n```python:app.py\nprint('x')\n```\n"

	blocks := parseCodeBlocks(input)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].filename != "main.go" || blocks[1].filename != "app.py" {
		t.Fatalf("unexpected filenames: %q, %q", blocks[0].filename, blocks[1].filename)
	}
}

func TestParseCodeBlocksNoBlocks(t *testing.T) {
	input := "no fenced code blocks here"

	blocks := parseCodeBlocks(input)
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestParseCodeBlocksInvalidOrEmptyLanguage(t *testing.T) {
	input := "``` :main.go\nbad\n```\n```go! :main.go\nbad2\n```\n"

	blocks := parseCodeBlocks(input)
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks for invalid language headers, got %d", len(blocks))
	}
}

func TestParseCodeBlocksBackticksInsideCodeDoNotCloseBlock(t *testing.T) {
	input := "```go:main.go\ns := \"value with `backtick` and ``double``\"\nfmt.Println(s)\n```\n"

	blocks := parseCodeBlocks(input)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	expected := "s := \"value with `backtick` and ``double``\"\nfmt.Println(s)"
	if blocks[0].code != expected {
		t.Fatalf("unexpected code: %q", blocks[0].code)
	}
}

func TestCrossAgent_VisionToCode_ConsumesVisionOutput(t *testing.T) {
	// Simulate vision-agent output describing a UI component
	visionOutput := "I analyzed the design mockup.\n\n" +
		"```json:vision-analysis.json\n" +
		`{"summary":"A React component with a form containing name and email fields, plus a submit button",` + "\n" +
		`"uiElements":["text input x2","button"],` + "\n" +
		`"layout":"stacked form in a card",` + "\n" +
		`"suggestedNextAgents":["code-agent"]}` + "\n" +
		"```\n\n" +
		"The design shows a clean form component."

	// code-agent should be able to extract text from vision-agent output
	parts := a2a.ContentParts{a2a.NewTextPart(visionOutput)}
	textContent := extractTextFromParts(parts)
	if textContent == "" {
		t.Fatal("code-agent should extract text from vision-agent output")
	}

	// Verify code-agent can parse code blocks from a response that includes vision context.
	// Note: parseCodeBlocks also matches the vision-analysis JSON fenced block (language=json),
	// so we get 2 blocks. Search for the tsx block among them.
	mixedResponse := visionOutput + "\n\nHere is the React component:\n\n" +
		"```tsx:Form.tsx\nimport React from 'react';\n" +
		"const Form = () => {\n  return <form><input name='email'/><input name='name'/><button>Submit</button></form>;\n};\n" +
		"export default Form;\n```"
	blocks := parseCodeBlocks(mixedResponse)
	var tsxBlock *codeBlock
	for i := range blocks {
		if blocks[i].language == "tsx" {
			tsxBlock = &blocks[i]
			break
		}
	}
	if tsxBlock == nil {
		t.Fatalf("expected a tsx code block in vision→code chain output, got %d blocks", len(blocks))
	}
	if tsxBlock.filename != "Form.tsx" {
		t.Errorf("expected filename Form.tsx, got %q", tsxBlock.filename)
	}
	if !strings.Contains(tsxBlock.code, "export default Form") {
		t.Errorf("code block should contain the component code")
	}
}

func TestCrossAgent_VisionToCode_MetadataChain(t *testing.T) {
	// Verify vision→code chain: vision description → code generation
	visionContent := `{"summary":"A Go HTTP server with /health and /api endpoints"}`

	codeResponse := "Based on the description: " + visionContent + "\n\n" +
		"```go:main.go\npackage main\nimport \"net/http\"\nfunc main() {\n  http.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {\n    w.Write([]byte(\"ok\"))\n  })\n  http.ListenAndServe(\":8080\", nil)\n}\n```"
	blocks := parseCodeBlocks(codeResponse)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 code block, got %d", len(blocks))
	}
	if blocks[0].language != "go" {
		t.Errorf("expected language go, got %q", blocks[0].language)
	}
}

// TestConcurrent_ParseCodeBlocksParallel verifies parseCodeBlocks is safe
// under concurrent calls — no data races when parsing code blocks from
// multiple goroutines simultaneously.
func TestConcurrent_ParseCodeBlocksParallel(t *testing.T) {
	inputs := []string{
		"```go:main.go\npackage main\nfunc main() {}\n```",
		"```python:app.py\nprint('hello')\n```",
		"```tsx:Form.tsx\nimport React from 'react';\nconst Form = () => <form/>;\nexport default Form;\n```",
		"```go:handler.go\npackage main\nfunc handler(w http.ResponseWriter, r *http.Request) {}\n```",
	}

	var wg sync.WaitGroup
	type parseResult struct {
		idx       int
		numBlocks int
		language  string
	}
	results := make(chan parseResult, len(inputs)*10)

	for i, input := range inputs {
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func(idx int, in string) {
				defer wg.Done()
				blocks := parseCodeBlocks(in)
				if len(blocks) > 0 {
					results <- parseResult{
						idx:       idx,
						numBlocks: len(blocks),
						language:  blocks[0].language,
					}
				}
			}(i, input)
		}
	}

	wg.Wait()
	close(results)

	expectedLangs := []string{"go", "python", "tsx", "go"}
	for r := range results {
		if r.numBlocks != 1 {
			t.Errorf("input %d: expected 1 block, got %d", r.idx, r.numBlocks)
		}
		if r.language != expectedLangs[r.idx] {
			t.Errorf("input %d: expected language %q, got %q", r.idx, expectedLangs[r.idx], r.language)
		}
	}
}
