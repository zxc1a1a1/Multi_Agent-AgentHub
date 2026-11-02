package markdownagent

import "testing"

func TestValidateMermaidAcceptsValidDiagram(t *testing.T) {
	source := "```mermaid\nflowchart TD\n  A[Start] --> B{Decision}\n  B -->|\"yes\"| C[Done]\n```\n"
	if issues := ValidateMermaid(source); len(issues) != 0 {
		t.Fatalf("valid diagram reported issues: %#v", issues)
	}
}

func TestValidateMermaidReportsLineAndColumn(t *testing.T) {
	source := "```mermaid\nflowchart TD\n  A[Start\n```\n"
	issues := ValidateMermaid(source)
	if len(issues) != 1 || issues[0].Code != "MERMAID_UNBALANCED_BRACKET" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if issues[0].Line != 3 || issues[0].Column != 4 {
		t.Fatalf("wrong position: %#v", issues[0])
	}
}

func TestValidateMermaidReportsUnknownAndUnterminated(t *testing.T) {
	unknown := "```mermaid\nnotadiagram\nA --> B\n```\n"
	issues := ValidateMermaid(unknown)
	if len(issues) != 1 || issues[0].Code != "MERMAID_UNKNOWN_DIAGRAM" || issues[0].Line != 2 {
		t.Fatalf("unknown diagram not reported: %#v", issues)
	}
	unterminated := "```mermaid\nflowchart TD\nA --> B\n"
	issues = ValidateMermaid(unterminated)
	if len(issues) != 1 || issues[0].Code != "MERMAID_UNTERMINATED" {
		t.Fatalf("unterminated block not reported: %#v", issues)
	}
}
