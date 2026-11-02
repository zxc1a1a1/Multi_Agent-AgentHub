// Package render produces screenshot/PDF bytes from a URL. The production
// renderer shells out to the bundled Playwright script in an isolated temp dir;
// URL validation rejects loopback/private targets before the browser starts.
package render

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	FormatPNG = "png"
	FormatPDF = "pdf"
	maxBytes  = 10 << 20
)

type Request struct {
	URL      string
	Format   string
	FullPage bool
}

// Renderer is the browser boundary used by the artifact endpoint.
type Renderer interface {
	Render(ctx context.Context, req Request) ([]byte, string, error)
}

type PlaywrightConfig struct {
	NodePath   string
	ScriptPath string
	Channel    string
	Timeout    time.Duration
}

type PlaywrightRenderer struct {
	nodePath   string
	scriptPath string
	channel    string
	timeout    time.Duration
}

func NewPlaywrightRenderer(cfg PlaywrightConfig) (*PlaywrightRenderer, error) {
	nodePath := strings.TrimSpace(cfg.NodePath)
	if nodePath == "" {
		nodePath = "node"
	}
	scriptPath := strings.TrimSpace(cfg.ScriptPath)
	if scriptPath == "" {
		return nil, fmt.Errorf("playwright render script is required")
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("playwright render script is unavailable")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &PlaywrightRenderer{
		nodePath: nodePath, scriptPath: scriptPath,
		channel: strings.TrimSpace(cfg.Channel), timeout: timeout,
	}, nil
}

func (r *PlaywrightRenderer) Render(ctx context.Context, req Request) ([]byte, string, error) {
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format != FormatPNG && format != FormatPDF {
		return nil, "", fmt.Errorf("unsupported render format")
	}
	if err := ValidateURL(req.URL); err != nil {
		return nil, "", err
	}
	dir, err := os.MkdirTemp("", "agenthub-render-*")
	if err != nil {
		return nil, "", fmt.Errorf("render workspace unavailable")
	}
	defer os.RemoveAll(dir)
	ext := "." + format
	output := filepath.Join(dir, "out"+ext)
	args := []string{r.scriptPath, "--url", req.URL, "--output", output, "--format", format}
	if req.FullPage {
		args = append(args, "--full-page")
	}
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	command := exec.CommandContext(runCtx, r.nodePath, args...)
	command.Env = append(os.Environ(), "AGENTHUB_RENDER_OUTPUT="+output)
	if r.channel != "" {
		command.Env = append(command.Env, "PLAYWRIGHT_CHANNEL="+r.channel)
	}
	var stderr bytes.Buffer
	command.Stderr = &limitedBuffer{buf: &stderr, limit: 4096}
	command.Stdout = nil
	if err := command.Run(); err != nil {
		return nil, "", fmt.Errorf("page render failed")
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		return nil, "", fmt.Errorf("page render produced no output")
	}
	if info.Size() > maxBytes {
		return nil, "", fmt.Errorf("rendered output exceeds size limit")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		return nil, "", fmt.Errorf("rendered output unavailable")
	}
	contentType := "image/png"
	if format == FormatPDF {
		contentType = "application/pdf"
	}
	return data, contentType, nil
}

// ValidateURL rejects non-http(s), credentials, loopback, private and
// link-local targets. DNS names are resolved once here; the browser still
// performs its own resolution, so a determined DNS-rebinding attacker would
// need network-level egress controls as well.
func ValidateURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid render URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("render URL must be http(s)")
	}
	if parsed.User != nil {
		return fmt.Errorf("render URL must not contain credentials")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return fmt.Errorf("render URL host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("render URL host is not allowed")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("render URL host cannot be resolved")
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("render URL host is not allowed")
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	// Carrier-grade NAT and metadata ranges are not covered by IsPrivate.
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
		if v4[0] == 169 && v4[1] == 254 {
			return true
		}
	}
	return false
}

type limitedBuffer struct {
	buf   *bytes.Buffer
	limit int
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	if w.buf.Len() < w.limit {
		remaining := w.limit - w.buf.Len()
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = w.buf.Write(p)
	}
	return len(p), nil
}
