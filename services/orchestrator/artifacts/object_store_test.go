package artifacts

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeS3 struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 1 {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	key := parts[1]
	if key == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodPut:
		data, _ := io.ReadAll(r.Body)
		if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
			data = decodeAWSChunks(data)
		}
		f.objects[key] = data
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		data, ok := f.objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`))
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Set("ETag", `"fake-etag"`)
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	case http.MethodDelete:
		delete(f.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func newFakeS3Store(t *testing.T) (*S3ObjectStore, *fakeS3, func()) {
	t.Helper()
	fake := &fakeS3{objects: map[string][]byte{}}
	server := httptest.NewServer(fake)
	store, err := NewS3ObjectStore(S3Config{
		Endpoint: strings.TrimPrefix(server.URL, "http://"),
		AccessKey: "test-access", SecretKey: "test-secret",
		Bucket: "agenthub", Region: "us-east-1",
	})
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return store, fake, server.Close
}

func TestS3ObjectStoreRoundTripAndPresign(t *testing.T) {
	store, _, cleanup := newFakeS3Store(t)
	defer cleanup()
	ctx := context.Background()
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}
	if err := store.Put(ctx, "runs/run-1/out.txt", []byte("hello"), "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}
	data, err := store.Get(ctx, "runs/run-1/out.txt")
	if err != nil || string(data) != "hello" {
		t.Fatalf("get: %q err=%v", data, err)
	}
	signed, err := store.PresignGet(ctx, "runs/run-1/out.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if !strings.Contains(signed, "X-Amz-Signature") || !strings.Contains(signed, "runs/run-1/out.txt") {
		t.Fatalf("unexpected signed URL: %s", signed)
	}
	if err := store.Delete(ctx, "runs/run-1/out.txt"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, "runs/run-1/out.txt"); err == nil {
		t.Fatal("deleted object must not be readable")
	}
}

func TestS3ObjectStoreRejectsUnsafeInputs(t *testing.T) {
	store, _, cleanup := newFakeS3Store(t)
	defer cleanup()
	ctx := context.Background()
	for _, key := range []string{"", "/abs", "../escape", `a\b`} {
		if err := store.Put(ctx, key, []byte("x"), "text/plain"); err == nil {
			t.Fatalf("unsafe key %q must be rejected", key)
		}
	}
	if _, err := store.PresignGet(ctx, "ok", 8*24*time.Hour); err == nil {
		t.Fatal("presign ttl over 7 days must be rejected")
	}
}

func decodeAWSChunks(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for len(data) > 0 {
		lineEnd := bytes.Index(data, []byte("\r\n"))
		if lineEnd < 0 {
			break
		}
		header := string(data[:lineEnd])
		data = data[lineEnd+2:]
		sizePart := strings.SplitN(header, ";", 2)[0]
		size, err := strconv.ParseInt(sizePart, 16, 64)
		if err != nil || size == 0 {
			break
		}
		if int64(len(data)) < size {
			break
		}
		out = append(out, data[:size]...)
		data = data[size:]
		if len(data) >= 2 && data[0] == '\r' && data[1] == '\n' {
			data = data[2:]
		}
	}
	return out
}
