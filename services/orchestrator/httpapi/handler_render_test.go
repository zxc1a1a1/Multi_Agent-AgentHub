package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/render"
)

type fakePageRenderer struct {
	requests []render.Request
	data     []byte
	mime     string
	err      error
}

func (f *fakePageRenderer) Render(_ context.Context, req render.Request) ([]byte, string, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, "", f.err
	}
	return f.data, f.mime, nil
}

func renderTestServer(t *testing.T, renderer render.Renderer) (*Server, *artifacts.LocalObjectStore) {
	t.Helper()
	metadata, err := artifacts.NewJSONStore(filepath.Join(t.TempDir(), "artifacts.json"))
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := artifacts.NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(WithArtifactStore(metadata), WithArtifactBlobs(blobs))
	if renderer != nil {
		srv = NewServer(WithArtifactStore(metadata), WithArtifactBlobs(blobs), WithPageRenderer(renderer))
	}
	return srv, blobs
}

func TestRenderEndpointPublishesArtifactAndServesContent(t *testing.T) {
	payload := []byte("\x89PNG\r\n\x1a\nartifact")
	renderer := &fakePageRenderer{data: payload, mime: "image/png"}
	srv, _ := renderTestServer(t, renderer)
	body := `{"runId":"run-render","taskId":"task-1","url":"https://93.184.216.34/","format":"png","fullPage":true}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var published struct {
		ArtifactID   string `json:"artifactId"`
		Kind         string `json:"kind"`
		MimeType     string `json:"mimeType"`
		Size         int    `json:"size"`
		Checksum     string `json:"checksum"`
		DownloadPath string `json:"downloadPath"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &published); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(published.ArtifactID, "art_") || published.Kind != "image" ||
		published.MimeType != "image/png" || published.Size != len(payload) ||
		!strings.HasPrefix(published.Checksum, "sha256:") {
		t.Fatalf("unexpected metadata: %#v", published)
	}
	if len(renderer.requests) != 1 || renderer.requests[0].URL != "https://93.184.216.34/" ||
		!renderer.requests[0].FullPage || renderer.requests[0].Format != "png" {
		t.Fatalf("render request not forwarded: %#v", renderer.requests)
	}

	content := httptest.NewRecorder()
	srv.Handler().ServeHTTP(content, httptest.NewRequest(http.MethodGet, published.DownloadPath, nil))
	if content.Code != http.StatusOK || content.Body.String() != string(payload) {
		t.Fatalf("content status=%d body=%q", content.Code, content.Body.String())
	}
	if got := content.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type=%s", got)
	}
}

func TestRenderEndpointAcceptsPreRenderedHTML(t *testing.T) {
	renderer := &fakePageRenderer{data: []byte("%PDF-1.4 artifact"), mime: "application/pdf"}
	srv, _ := renderTestServer(t, renderer)
	body := `{"runId":"run-md","taskId":"task-md","html":"<h1>Hello</h1>","format":"pdf"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(renderer.requests) != 1 || !strings.Contains(renderer.requests[0].HTML, "<h1>Hello</h1>") ||
		renderer.requests[0].URL != "" || renderer.requests[0].Format != "pdf" {
		t.Fatalf("html request not forwarded: %#v", renderer.requests)
	}
}

func TestRenderEndpointRejectsAmbiguousTarget(t *testing.T) {
	renderer := &fakePageRenderer{data: []byte("x"), mime: "image/png"}
	srv, _ := renderTestServer(t, renderer)
	for _, body := range []string{
		`{"runId":"run","format":"png"}`,
		`{"runId":"run","url":"https://93.184.216.34/","html":"<h1>x</h1>","format":"png"}`,
	} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
	if len(renderer.requests) != 0 {
		t.Fatal("ambiguous render requests must not reach the renderer")
	}
}

func TestRenderEndpointRejectsInternalURL(t *testing.T) {
	renderer := &fakePageRenderer{data: []byte("x"), mime: "image/png"}
	srv, _ := renderTestServer(t, renderer)
	body := `{"runId":"run-render","url":"http://169.254.169.254/latest/meta-data","format":"png"}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(renderer.requests) != 0 {
		t.Fatal("renderer must not be called for internal URLs")
	}
}

func TestRenderEndpointUnavailableWithoutRenderer(t *testing.T) {
	srv, _ := renderTestServer(t, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render",
		strings.NewReader(`{"runId":"run-render","url":"https://93.184.216.34/","format":"png"}`)))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestArtifactContentDetectsChecksumMismatch(t *testing.T) {
	payload := []byte("original")
	renderer := &fakePageRenderer{data: payload, mime: "image/png"}
	srv, blobs := renderTestServer(t, renderer)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/render",
		strings.NewReader(`{"runId":"run-render","url":"https://93.184.216.34/","format":"png"}`)))
	var published struct {
		ArtifactID string `json:"artifactId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &published); err != nil {
		t.Fatal(err)
	}
	record, ok, err := srv.ensureArtifactStore().Get(context.Background(), published.ArtifactID)
	if err != nil || !ok {
		t.Fatalf("artifact missing: %v", err)
	}
	key := strings.TrimPrefix(record.StorageRef, "obj://")
	if err := blobs.Put(context.Background(), key, []byte("tampered"), "image/png"); err != nil {
		t.Fatal(err)
	}
	content := httptest.NewRecorder()
	srv.Handler().ServeHTTP(content, httptest.NewRequest(http.MethodGet, record.DownloadPath, nil))
	if content.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", content.Code, content.Body.String())
	}
}
