package agui

import (
	"errors"
	"regexp"
	"strings"
)

const truncatedSuffix = "...[truncated]"

var (
	secretAssignmentPattern = regexp.MustCompile("(?i)\\b(OPENAI_API_KEY|ANTHROPIC_API_KEY|AGENTHUB_API_TOKEN|DATABASE_URL|DB_PASSWORD|MYSQL_ROOT_PASSWORD)\\s*=\\s*([^\\s\"'`]+|\"[^\"]*\"|'[^']*')")
	privateKeyPattern       = regexp.MustCompile("(?i)-----BEGIN [A-Z ]*PRIVATE KEY-----|\\bPRIVATE KEY\\b|\\bBEGIN RSA\\b|\\bBEGIN OPENSSH\\b")
	skTokenPattern          = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`)
	windowsPathPattern      = regexp.MustCompile(`(?i)[a-z]:\\[^\s]+`)
	unixPathPattern         = regexp.MustCompile(`/(?:[^/\s]+/)+[^/\s]+`)
	stackLikePattern        = regexp.MustCompile(`(?i)\bpanic\b|\bstack\b|\btrace\b`)
)

type TextStreamFilter struct {
	maxTextChars int
}

type FilterOption func(*TextStreamFilter)

func NewTextStreamFilter(opts ...FilterOption) *TextStreamFilter {
	filter := &TextStreamFilter{
		maxTextChars: 0,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(filter)
	}
	return filter
}

func WithMaxTextChars(n int) FilterOption {
	return func(f *TextStreamFilter) {
		if f == nil {
			return
		}
		f.maxTextChars = n
	}
}

func (f *TextStreamFilter) FilterText(text string) string {
	if text == "" {
		return ""
	}

	out := secretAssignmentPattern.ReplaceAllString(text, "$1=[redacted]")
	out = privateKeyPattern.ReplaceAllString(out, "[redacted]")
	out = skTokenPattern.ReplaceAllString(out, "[redacted]")

	maxChars := 0
	if f != nil {
		maxChars = f.maxTextChars
	}
	if maxChars <= 0 {
		return out
	}

	runes := []rune(out)
	if len(runes) <= maxChars {
		return out
	}
	return string(runes[:maxChars]) + truncatedSuffix
}

func (f *TextStreamFilter) FilterError(err error) string {
	if err == nil {
		return ""
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "internal error"
	}

	if stackLikePattern.MatchString(msg) {
		msg = "internal error"
	}
	msg = windowsPathPattern.ReplaceAllString(msg, "[redacted]")
	msg = unixPathPattern.ReplaceAllString(msg, "[redacted]")
	msg = strings.TrimSpace(f.FilterText(msg))

	if msg == "" {
		return "internal error"
	}

	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "internal error"
}

func sanitizeError(err error, filter *TextStreamFilter) error {
	if err == nil {
		return nil
	}
	if filter == nil {
		filter = NewTextStreamFilter()
	}
	return errors.New(filter.FilterError(err))
}
