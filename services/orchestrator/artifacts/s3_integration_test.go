package artifacts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// R0-07 integration test: two independent stores (two Orchestrator replicas)
// share one S3-compatible bucket. It runs only against a real endpoint and
// skips with an explicit reason otherwise.
func s3IntegrationConfig(t *testing.T) S3Config {
	t.Helper()
	endpoint := strings.TrimSpace(os.Getenv("AGENTHUB_S3_TEST_ENDPOINT"))
	if endpoint == "" {
		t.Skip("AGENTHUB_S3_TEST_ENDPOINT is not set: a real S3-compatible endpoint is required for R0-07")
	}
	access := strings.TrimSpace(os.Getenv("AGENTHUB_S3_TEST_ACCESS_KEY"))
	secret := strings.TrimSpace(os.Getenv("AGENTHUB_S3_TEST_SECRET_KEY"))
	if access == "" {
		access = "agenthub-test"
	}
	if secret == "" {
		secret = "agenthub-test-secret"
	}
	bucket := strings.TrimSpace(os.Getenv("AGENTHUB_S3_TEST_BUCKET"))
	if bucket == "" {
		bucket = fmt.Sprintf("agenthub-r007-%d", time.Now().UnixNano())
	}
	secure := strings.HasPrefix(strings.ToLower(endpoint), "https://")
	return S3Config{
		Endpoint: endpoint, AccessKey: access, SecretKey: secret,
		Bucket: bucket, Region: "us-east-1", Secure: secure,
	}
}

func TestIntegrationS3CrossInstanceArtifact(t *testing.T) {
	cfg := s3IntegrationConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	storeA, err := NewS3ObjectStore(cfg)
	if err != nil {
		t.Fatalf("store A: %v", err)
	}
	storeB, err := NewS3ObjectStore(cfg)
	if err != nil {
		t.Fatalf("store B: %v", err)
	}

	// Outage phase: the harness stops the object store and reruns this phase.
	if os.Getenv("AGENTHUB_S3_TEST_EXPECT_DOWN") == "1" {
		if _, err := storeB.Get(ctx, "r007/probe.bin"); err == nil {
			t.Fatal("object store outage must fail the read, not silently fall back")
		}
		t.Log("outage read failed as required")
		return
	}

	if err := storeA.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure bucket A: %v", err)
	}
	if err := storeB.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure bucket B: %v", err)
	}

	key := fmt.Sprintf("r007/%d.bin", time.Now().UnixNano())
	payload := []byte("cross-instance artifact payload")
	sum := sha256.Sum256(payload)
	wantChecksum := "sha256:" + hex.EncodeToString(sum[:])

	if err := storeA.Put(ctx, key, payload, "text/plain"); err != nil {
		t.Fatalf("put on A: %v", err)
	}
	got, err := storeB.Get(ctx, key)
	if err != nil {
		t.Fatalf("get on B: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("B read %d bytes, want %d", len(got), len(payload))
	}
	gotSum := sha256.Sum256(got)
	if "sha256:"+hex.EncodeToString(gotSum[:]) != wantChecksum {
		t.Fatalf("checksum mismatch: got %x", gotSum)
	}

	// Presigned URL must be usable and bounded.
	presigned, err := storeB.PresignGet(ctx, key, 2*time.Minute)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	resp, err := http.Get(presigned)
	if err != nil {
		t.Fatalf("presigned GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Equal(body, payload) {
		t.Fatalf("presigned GET status=%d bytes=%d", resp.StatusCode, len(body))
	}
	if _, err := storeB.PresignGet(ctx, key, 8*24*time.Hour); err == nil {
		t.Fatal("presign ttl above 7 days must be rejected")
	}

	// Metadata records written by A must be visible to B.
	metaA, err := NewS3Store(storeA, "metadata/")
	if err != nil {
		t.Fatalf("metadata store A: %v", err)
	}
	metaB, err := NewS3Store(storeB, "metadata/")
	if err != nil {
		t.Fatalf("metadata store B: %v", err)
	}
	artifactID := fmt.Sprintf("art_r007_%d", time.Now().UnixNano())
	if err := metaA.Create(ctx, ArtifactRecord{
		ID: artifactID, RunID: "run_r007", TaskID: "task_r007",
		Name: "probe.txt", Kind: "text", MimeType: "text/plain",
		Size: int64(len(payload)), DownloadPath: "probe.txt",
	}); err != nil {
		t.Fatalf("metadata create: %v", err)
	}
	record, ok, err := metaB.Get(ctx, artifactID)
	if err != nil || !ok {
		t.Fatalf("metadata cross-instance read ok=%v err=%v", ok, err)
	}
	if record.Size != int64(len(payload)) || record.Name != "probe.txt" {
		t.Fatalf("metadata mismatch: %#v", record)
	}

	// Key traversal and absolute paths must be rejected, never normalized.
	for _, bad := range []string{"../escape", "/etc/passwd", `a\b`, ""} {
		if _, err := storeB.Get(ctx, bad); err == nil {
			t.Fatalf("unsafe key %q must be rejected", bad)
		}
	}
	// 10MB + 1 byte is rejected before reaching the network.
	oversized := make([]byte, maxObjectBytes+1)
	if err := storeA.Put(ctx, key+"-big", oversized, "application/octet-stream"); err == nil {
		t.Fatal("object above the 10MB limit must be rejected")
	}

	_ = storeA.Delete(ctx, key)
	_ = metaA.Delete(ctx, artifactID)
}
