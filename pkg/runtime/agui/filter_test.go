package agui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTextStreamFilter_AllowsNormalChinese(t *testing.T) {
	filter := NewTextStreamFilter()
	input := "这是普通中文内容，不应被误伤。"
	got := filter.FilterText(input)
	if got != input {
		t.Fatalf("unexpected mutation: got %q want %q", got, input)
	}
}

func TestTextStreamFilter_RedactsAPIKey(t *testing.T) {
	filter := NewTextStreamFilter()
	got := filter.FilterText("OPENAI_API_KEY=abc123")
	if !strings.Contains(got, "OPENAI_API_KEY=[redacted]") {
		t.Fatalf("expected redacted api key, got %q", got)
	}
	if strings.Contains(got, "abc123") {
		t.Fatalf("raw secret leaked: %q", got)
	}
}

func TestTextStreamFilter_RedactsDatabaseURL(t *testing.T) {
	filter := NewTextStreamFilter()
	got := filter.FilterText("DATABASE_URL=mysql://user:pass@localhost:3306/db")
	if !strings.Contains(got, "DATABASE_URL=[redacted]") {
		t.Fatalf("expected redacted database url, got %q", got)
	}
	if strings.Contains(got, "mysql://") {
		t.Fatalf("raw database url leaked: %q", got)
	}
}

func TestTextStreamFilter_RedactsPrivateKey(t *testing.T) {
	filter := NewTextStreamFilter()
	got := filter.FilterText("-----BEGIN RSA PRIVATE KEY-----")
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected private key redacted, got %q", got)
	}
}

func TestTextStreamFilter_RedactsSKToken(t *testing.T) {
	filter := NewTextStreamFilter()
	token := "sk-" + strings.Repeat("x", 30)
	got := filter.FilterText("token=" + token)
	if strings.Contains(got, token) {
		t.Fatalf("raw sk token leaked: %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected redacted token, got %q", got)
	}
}

func TestTextStreamFilter_TruncatesLongText(t *testing.T) {
	filter := NewTextStreamFilter(WithMaxTextChars(4))
	got := filter.FilterText("123456789")
	if got != "1234"+truncatedSuffix {
		t.Fatalf("unexpected truncation result: %q", got)
	}
}

func TestTextStreamFilter_DisableTruncation(t *testing.T) {
	filter := NewTextStreamFilter(WithMaxTextChars(0))
	input := "123456789"
	got := filter.FilterText(input)
	if got != input {
		t.Fatalf("expected no truncation, got %q", got)
	}
}

func TestTextStreamFilter_FilterError(t *testing.T) {
	filter := NewTextStreamFilter()
	err := errors.New(`failed: DB_PASSWORD=abc C:\secret\file.go`)
	got := filter.FilterError(err)
	if strings.Contains(got, "abc") {
		t.Fatalf("password leaked: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "secret") || strings.Contains(got, `C:\`) {
		t.Fatalf("path leaked: %q", got)
	}
	if !strings.Contains(got, "DB_PASSWORD=[redacted]") {
		t.Fatalf("expected redacted field, got %q", got)
	}
}

func TestTextStreamFilter_FilterErrorNoStack(t *testing.T) {
	filter := NewTextStreamFilter()
	err := errors.New("panic: boom\nstack trace:\nC:\\work\\x.go:10")
	got := filter.FilterError(err)
	if strings.Contains(strings.ToLower(got), "panic") || strings.Contains(strings.ToLower(got), "stack") {
		t.Fatalf("stack details leaked: %q", got)
	}
}

func TestTextStreamFilter_DoesNotReadDotEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("A=from-dotenv"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	filter := NewTextStreamFilter()
	input := "${A}"
	got := filter.FilterText(input)
	if got != input {
		t.Fatalf("text should remain unchanged, got %q", got)
	}
}
