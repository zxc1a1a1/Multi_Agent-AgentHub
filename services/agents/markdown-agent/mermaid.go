package markdownagent

import (
	"strings"
)

var diagramKeywords = map[string]bool{
	"graph": true, "flowchart": true, "sequencediagram": true, "classdiagram": true,
	"statediagram": true, "gantt": true, "pie": true, "erdiagram": true,
	"journey": true, "gitgraph": true, "mindmap": true, "timeline": true,
}

// ValidateMermaid performs deterministic, non-rendering checks on fenced
// ```mermaid blocks: diagram keyword, balanced brackets and terminated fences.
// It does not claim full Mermaid grammar compatibility.
func ValidateMermaid(source string) []Issue {
	issues := make([]Issue, 0)
	lines := strings.Split(source, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "```mermaid" {
			continue
		}
		openLine := i + 1
		var body []string
		closed := false
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "```" {
				closed = true
				i = j
				break
			}
			body = append(body, lines[j])
		}
		if !closed {
			issues = append(issues, Issue{
				Line: openLine, Column: 1, Code: "MERMAID_UNTERMINATED",
				Message: "mermaid block is not closed",
			})
			break
		}
		issues = append(issues, validateMermaidBody(body, openLine)...)
	}
	return issues
}

func validateMermaidBody(body []string, fenceLine int) []Issue {
	issues := make([]Issue, 0)
	firstLine := 0
	for idx, line := range body {
		if strings.TrimSpace(line) != "" {
			firstLine = idx
			break
		}
	}
	if firstLine == 0 && strings.TrimSpace(body[0]) == "" {
		issues = append(issues, Issue{
			Line: fenceLine + 1, Column: 1, Code: "MERMAID_EMPTY",
			Message: "mermaid block has no diagram",
		})
		return issues
	}
	keyword := strings.Fields(strings.TrimSpace(body[firstLine]))[0]
	keyword = strings.TrimSuffix(strings.ToLower(keyword), ";")
	if !diagramKeywords[keyword] {
		issues = append(issues, Issue{
			Line: fenceLine + firstLine + 1, Column: 1, Code: "MERMAID_UNKNOWN_DIAGRAM",
			Message: "unknown diagram type " + keyword,
		})
	}
	for idx, line := range body {
		if column := unbalancedBracketColumn(line); column > 0 {
			issues = append(issues, Issue{
				Line: fenceLine + idx + 1, Column: column, Code: "MERMAID_UNBALANCED_BRACKET",
				Message: "unbalanced bracket, brace or parenthesis",
			})
		}
	}
	return issues
}

func unbalancedBracketColumn(line string) int {
	type opening struct {
		char   rune
		column int
	}
	stack := make([]opening, 0)
	pairs := map[rune]rune{'}': '{', ']': '[', ')': '('}
	inQuote := false
	var quote rune
	for index, char := range line {
		if char == '"' || char == '\'' {
			if inQuote && char == quote {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quote = char
			}
			continue
		}
		if inQuote {
			continue
		}
		switch char {
		case '{', '[', '(':
			stack = append(stack, opening{char: char, column: index + 1})
		case '}', ']', ')':
			if len(stack) == 0 || stack[len(stack)-1].char != pairs[char] {
				return index + 1
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		return stack[len(stack)-1].column
	}
	if inQuote {
		return len(line)
	}
	return 0
}
