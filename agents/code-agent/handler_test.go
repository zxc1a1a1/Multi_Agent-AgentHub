package main

import "testing"

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
