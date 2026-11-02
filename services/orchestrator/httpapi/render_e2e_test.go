package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/render"
)

// TestRenderEndpointE2EPlaywright runs the full render -> blob -> metadata ->
// content path with the real Playwright script. Set AGENTHUB_RENDER_E2E=1,
// PLAYWRIGHT_RENDER_SCRIPT and the target URL to run it.
func TestRenderEndpointE2EPlaywright(t *testing.T) {
	if os.Getenv("AGENTHUB_RENDER_E2E") != "1" {
		t.Skip("set AGENTHUB_RENDER_E2E=1 with PLAYWRIGHT_RENDER_SCRIPT")
	}
	script := strings.TrimSpace(os.Getenv("PLAYWRIGHT_RENDER_SCRIPT"))
	if script == "" {
		t.Skip("PLAYWRIGHT_RENDER_SCRIPT is required")
	}
	renderer, err := render.NewPlaywrightRenderer(render.PlaywrightConfig{
		NodePath: os.Getenv("PLAYWRIGHT_NODE_PATH"), ScriptPath: script,
		Channel: os.Getenv("PLAYWRIGHT_CHANNEL"), Timeout: 90 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := artifacts.NewJSONStore(filepath.Join(t.TempDir(), "artifacts.json"))
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := artifacts.NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(WithArtifactStore(metadata), WithArtifactBlobs(blobs), WithPageRenderer(renderer))

	assertRender := func(body string, magic string) {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render", strings.NewReader(body)))
		if rec.Code != http.StatusOK {
			t.Fatalf("render status=%d body=%s", rec.Code, rec.Body.String())
		}
		var published struct {
			ArtifactID   string `json:"artifactId"`
			DownloadPath string `json:"downloadPath"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &published); err != nil {
			t.Fatal(err)
		}
		content := httptest.NewRecorder()
		srv.Handler().ServeHTTP(content, httptest.NewRequest(http.MethodGet, published.DownloadPath, nil))
		if content.Code != http.StatusOK || !strings.HasPrefix(content.Body.String(), magic) {
			t.Fatalf("content status=%d body=%q", content.Code, content.Body.String())
		}
	}

	target := strings.TrimSpace(os.Getenv("AGENTHUB_RENDER_E2E_URL"))
	if target == "" {
		target = "https://example.com"
	}
	assertRender(`{"runId":"run-e2e-render","url":`+strconv.Quote(target)+`,"format":"png"}`, "\x89PNG\r\n\x1a\n")
	assertRender(`{"runId":"run-e2e-render","html":"<h1>Markdown export</h1><p>Hello</p>","format":"pdf"}`, "%PDF")
}
