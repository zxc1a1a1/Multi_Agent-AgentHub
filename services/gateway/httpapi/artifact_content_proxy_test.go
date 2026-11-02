package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func TestArtifactContentProxyForwardsToOrchestrator(t *testing.T) {
	runner := &mockRunService{proxyFn: func(_ context.Context, method, internalPath string, _ io.Reader, _ string) (*http.Response, error) {
		if method != http.MethodGet || internalPath != "/internal/orchestrator/artifacts/art_1/content" {
			t.Fatalf("unexpected proxy request: %s %s", method, internalPath)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("png-bytes"))),
		}, nil
	}}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/artifacts/art_1/content", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "png-bytes" || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected response: %d %s %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

func TestArtifactContentProxyRejectsUnsafeOrWrongMethod(t *testing.T) {
	runner := &mockRunService{}
	srv, err := NewServer(store.NewMemoryStore(), runner)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/artifacts/art_1/other", http.StatusNotFound},
		{http.MethodGet, "/api/artifacts/art_1/extra/content", http.StatusNotFound},
		{http.MethodDelete, "/api/artifacts/art_1/content", http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.want {
			t.Fatalf("%s %s: status=%d want=%d body=%s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
