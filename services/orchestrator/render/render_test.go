package render

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("AGENTHUB_FAKE_RENDER") == "1" {
		output, format := "", "png"
		args := os.Args
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--output":
				if i+1 < len(args) {
					output = args[i+1]
				}
			case "--format":
				if i+1 < len(args) {
					format = args[i+1]
				}
			}
		}
		payload := []byte("\x89PNG\r\n\x1a\nfake")
		if format == "pdf" {
			payload = []byte("%PDF-1.4 fake")
		}
		if output != "" {
			_ = os.WriteFile(output, payload, 0o600)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestValidateURLRejectsInternalTargets(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/", "http://localhost:3000/", "http://app.local/",
		"http://10.1.2.3/", "http://192.168.1.1/", "http://169.254.169.254/latest/meta-data",
		"http://100.64.0.1/", "file:///etc/passwd", "https://user:pass@example.com/",
	}
	for _, raw := range blocked {
		if err := ValidateURL(raw); err == nil {
			t.Fatalf("expected rejection: %s", raw)
		}
	}
	if err := ValidateURL("https://93.184.216.34/path"); err != nil {
		t.Fatalf("public IP should be allowed: %v", err)
	}
}

func TestPlaywrightRendererRunsCommandAndReadsOutput(t *testing.T) {
	t.Setenv("AGENTHUB_FAKE_RENDER", "1")
	script := filepath.Join(t.TempDir(), "render.mjs")
	if err := os.WriteFile(script, []byte("// fake"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := NewPlaywrightRenderer(PlaywrightConfig{
		NodePath: executable, ScriptPath: script, Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, contentType, err := renderer.Render(context.Background(), Request{
		URL: "https://93.184.216.34/", Format: FormatPNG,
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if contentType != "image/png" || !strings.HasPrefix(string(data), "\x89PNG") {
		t.Fatalf("unexpected png: %s %q", contentType, data)
	}
	data, contentType, err = renderer.Render(context.Background(), Request{
		URL: "https://93.184.216.34/", Format: FormatPDF,
	})
	if err != nil {
		t.Fatalf("pdf render failed: %v", err)
	}
	if contentType != "application/pdf" || !strings.HasPrefix(string(data), "%PDF") {
		t.Fatalf("unexpected pdf: %s %q", contentType, data)
	}
}

func TestPlaywrightRendererEndToEnd(t *testing.T) {
	if os.Getenv("AGENTHUB_RENDER_E2E") != "1" {
		t.Skip("set AGENTHUB_RENDER_E2E=1 with PLAYWRIGHT_RENDER_SCRIPT and AGENTHUB_RENDER_E2E_URL")
	}
	script := strings.TrimSpace(os.Getenv("PLAYWRIGHT_RENDER_SCRIPT"))
	target := strings.TrimSpace(os.Getenv("AGENTHUB_RENDER_E2E_URL"))
	if script == "" || target == "" {
		t.Skip("PLAYWRIGHT_RENDER_SCRIPT and AGENTHUB_RENDER_E2E_URL are required")
	}
	renderer, err := NewPlaywrightRenderer(PlaywrightConfig{
		NodePath: os.Getenv("PLAYWRIGHT_NODE_PATH"), ScriptPath: script,
		Channel: os.Getenv("PLAYWRIGHT_CHANNEL"), Timeout: 90 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	png, mime, err := renderer.Render(context.Background(), Request{URL: target, Format: FormatPNG})
	if err != nil {
		t.Fatalf("png render failed: %v", err)
	}
	if mime != "image/png" || len(png) < 1000 || !strings.HasPrefix(string(png[:8]), "\x89PNG\r\n\x1a\n") {
		t.Fatalf("unexpected png: mime=%s bytes=%d", mime, len(png))
	}
	pdf, mime, err := renderer.Render(context.Background(), Request{URL: target, Format: FormatPDF})
	if err != nil {
		t.Fatalf("pdf render failed: %v", err)
	}
	if mime != "application/pdf" || len(pdf) < 1000 || !strings.HasPrefix(string(pdf[:4]), "%PDF") {
		t.Fatalf("unexpected pdf: mime=%s bytes=%d", mime, len(pdf))
	}
	t.Logf("rendered png=%d bytes pdf=%d bytes", len(png), len(pdf))
}

func TestPlaywrightRendererRejectsFormatAndURLBeforeExec(t *testing.T) {
	script := filepath.Join(t.TempDir(), "render.mjs")
	if err := os.WriteFile(script, []byte("// fake"), 0o600); err != nil {
		t.Fatal(err)
	}
	renderer, err := NewPlaywrightRenderer(PlaywrightConfig{ScriptPath: script})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := renderer.Render(context.Background(), Request{URL: "https://93.184.216.34/", Format: "svg"}); err == nil {
		t.Fatal("unsupported format must fail")
	}
	if _, _, err := renderer.Render(context.Background(), Request{URL: "http://127.0.0.1/", Format: FormatPNG}); err == nil {
		t.Fatal("private URL must fail before exec")
	}
}
