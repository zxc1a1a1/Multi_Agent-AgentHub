package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
)

type headerBlobStore struct{ data map[string][]byte }

func (s *headerBlobStore) EnsureBucket(context.Context) error { return nil }
func (s *headerBlobStore) Put(_ context.Context, key string, data []byte, _ string) error {
	s.data[key] = append([]byte(nil), data...)
	return nil
}
func (s *headerBlobStore) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := s.data[key]
	if !ok {
		return nil, artifacts.ErrObjectNotFound
	}
	return data, nil
}
func (s *headerBlobStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
func (s *headerBlobStore) List(context.Context, string) ([]string, error) { return nil, nil }
func (s *headerBlobStore) Delete(context.Context, string) error           { return nil }

// R3-02: artifact HTML must not be sniffed into an active document by the
// browser; the iframe sandbox is the execution boundary.
func TestArtifactContentSetsNosniff(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_TOKEN", "artifact-test-token")
	payload := []byte("<html><body><script>parent.alert(1)</script></body></html>")
	sum := sha256.Sum256(payload)
	key := "runs/run_r302/art_probe.html"
	blobs := &headerBlobStore{data: map[string][]byte{key: payload}}
	meta, err := artifacts.NewJSONStore(t.TempDir() + "/artifacts.json")
	if err != nil {
		t.Fatalf("metadata store: %v", err)
	}
	if err := meta.Create(context.Background(), artifacts.ArtifactRecord{
		ID: "art_probe", RunID: "run_r302", Name: "probe.html", Kind: "document",
		MimeType: "text/html", ContentType: "text/html", Size: int64(len(payload)),
		StorageRef: "obj://" + key, Checksum: "sha256:" + hex.EncodeToString(sum[:]),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create metadata: %v", err)
	}
	server := NewServer(WithArtifactStore(meta), WithArtifactBlobs(blobs))
	req := httptest.NewRequest(http.MethodGet, "/internal/orchestrator/artifacts/art_probe/content", nil)
	req.Header.Set("Authorization", "Bearer artifact-test-token")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options=%q, want nosniff", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got == "" {
		t.Fatal("Content-Disposition must keep the artifact as an attachment")
	}
	if rec.Body.String() != string(payload) {
		t.Fatal("content bytes changed")
	}
}
