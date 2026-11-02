package markdownagent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	// AutoHeadingID makes rendered anchors match GenerateTOC output.
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// Render converts Markdown to HTML. Raw HTML is never rendered: goldmark
// without html.WithUnsafe() drops it, so scripts cannot reach the preview DOM.
func Render(source string) (string, error) {
	var out strings.Builder
	if err := markdown.Convert([]byte(source), &out); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Issue is one deterministic lint finding.
type Issue struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	emptyLinkPattern = regexp.MustCompile(`\]\(\s*\)`)
	headingPattern   = regexp.MustCompile(`^(#{1,6})([^#\s].*)$`)
)

// Lint reports heading spacing, empty links and fenced code blocks without a
// language. Line is 1-based; Column is 1-based byte position.
func Lint(source string) []Issue {
	issues := make([]Issue, 0)
	lines := strings.Split(source, "\n")
	inFence := false
	for i, line := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				if lang == "" {
					issues = append(issues, Issue{
						Line: lineNo, Column: strings.Index(line, "```") + 1,
						Code: "CODE_BLOCK_NO_LANGUAGE", Message: "code fence has no language",
					})
				}
			}
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if match := headingPattern.FindStringSubmatch(line); match != nil {
			issues = append(issues, Issue{
				Line: lineNo, Column: len(match[1]) + 1,
				Code: "HEADING_MISSING_SPACE", Message: "heading marker must be followed by a space",
			})
		}
		for _, loc := range emptyLinkPattern.FindAllStringIndex(line, -1) {
			issues = append(issues, Issue{
				Line: lineNo, Column: loc[0] + 1,
				Code: "EMPTY_LINK", Message: "link target is empty",
			})
		}
	}
	return issues
}

// Frontmatter is the parsed YAML header plus the remaining Markdown body.
type Frontmatter struct {
	Meta map[string]any `json:"meta"`
	Body string         `json:"body"`
}

// ParseFrontmatter parses an optional leading YAML frontmatter block. A
// document without frontmatter returns an empty map and the original body.
func ParseFrontmatter(source string) (Frontmatter, error) {
	normalized := strings.ReplaceAll(source, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return Frontmatter{Meta: map[string]any{}, Body: source}, nil
	}
	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return Frontmatter{}, fmt.Errorf("frontmatter is not terminated")
	}
	header := rest[:end]
	body := rest[end+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	meta := map[string]any{}
	if err := yaml.Unmarshal([]byte(header), &meta); err != nil {
		return Frontmatter{}, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	return Frontmatter{Meta: meta, Body: body}, nil
}

// TOCEntry is one heading entry; Anchor matches the rendered heading id.
type TOCEntry struct {
	Level  int    `json:"level"`
	Text   string `json:"text"`
	Anchor string `json:"anchor"`
}

// GenerateTOC parses headings with the same parser/options used by Render, so
// anchor values match the generated HTML ids.
func GenerateTOC(source string) []TOCEntry {
	doc := markdown.Parser().Parse(text.NewReader([]byte(source)))
	entries := make([]TOCEntry, 0)
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		anchor, _ := heading.AttributeString("id")
		anchorText := ""
		if raw, ok := anchor.([]byte); ok {
			anchorText = string(raw)
		}
		entries = append(entries, TOCEntry{
			Level:  heading.Level,
			Text:   headingText(heading, source),
			Anchor: anchorText,
		})
		return ast.WalkContinue, nil
	})
	return entries
}

func headingText(heading *ast.Heading, source string) string {
	var builder strings.Builder
	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			builder.Write(textNode.Segment.Value([]byte(source)))
		}
	}
	return builder.String()
}
