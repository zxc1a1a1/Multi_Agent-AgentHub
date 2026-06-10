package planner

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PlanParser parses raw LLM output into a validated PlanSchema.
// It handles plain JSON, markdown-fenced JSON, leading/trailing whitespace,
// and extracts the first JSON object from surrounding text.
//
// The parser rejects: empty input, non-JSON, JSON array roots,
// missing mode, and missing (or empty) steps.
// It does NOT validate mode values or agent names — that is the
// Normalizer/Validator's responsibility.
type PlanParser struct{}

// NewPlanParser creates a new PlanParser.
func NewPlanParser() *PlanParser {
	return &PlanParser{}
}

// Parse extracts and validates a PlanSchema from raw LLM output.
func (p *PlanParser) Parse(raw string) (*PlanSchema, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("parse: empty input")
	}

	// Step 1: strip markdown code fences if present.
	cleaned := stripFences(text)

	// Step 2: find the first JSON structural character.
	objStart := strings.Index(cleaned, "{")
	arrStart := strings.Index(cleaned, "[")

	// No JSON at all.
	if objStart < 0 && arrStart < 0 {
		return nil, fmt.Errorf("parse: no JSON found in input")
	}

	// Array root found before any object — reject.
	if arrStart >= 0 && (objStart < 0 || arrStart < objStart) {
		return nil, fmt.Errorf("parse: JSON array root is not allowed")
	}

	// Step 3: extract the JSON object string using brace-depth matching.
	jsonText, err := extractBracedObject(cleaned, objStart)
	if err != nil {
		return nil, err
	}

	// Step 4: decode into PlanSchema.
	var schema PlanSchema
	if err := json.Unmarshal([]byte(jsonText), &schema); err != nil {
		return nil, fmt.Errorf("parse: invalid JSON: %w", err)
	}

	// Step 5: reject missing required fields.
	if strings.TrimSpace(schema.Mode) == "" {
		return nil, fmt.Errorf("parse: missing mode field")
	}
	if len(schema.Steps) == 0 {
		return nil, fmt.Errorf("parse: missing steps (must be non-empty array)")
	}

	return &schema, nil
}

// stripMarkdownFences removes ```json / ``` wrappers if present.
// Exported for test visibility.
func stripMarkdownFences(text string) string {
	return stripFences(text)
}

// stripFences removes ```json / ``` wrappers if the text starts and ends with them.
func stripFences(text string) string {
	t := text

	// Case: ```json ... ``` or ``` ... ```
	if strings.HasPrefix(t, "```json") {
		t = strings.TrimPrefix(t, "```json")
		t = strings.TrimSpace(t)
		if strings.HasSuffix(t, "```") {
			t = strings.TrimSuffix(t, "```")
			t = strings.TrimSpace(t)
		}
		return t
	}
	if strings.HasPrefix(t, "```") {
		t = strings.TrimPrefix(t, "```")
		t = strings.TrimSpace(t)
		if strings.HasSuffix(t, "```") {
			t = strings.TrimSuffix(t, "```")
			t = strings.TrimSpace(t)
		}
		return t
	}

	return t
}

// extractBracedObject extracts a balanced JSON object starting at position start.
// It handles nested braces, escaped characters, and strings.
func extractBracedObject(text string, start int) (string, error) {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(text); i++ {
		c := text[i]

		if escaped {
			escaped = false
			continue
		}

		if inString {
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1], nil
			}
		}
	}

	return "", fmt.Errorf("parse: unclosed JSON object (depth still %d)", depth)
}
