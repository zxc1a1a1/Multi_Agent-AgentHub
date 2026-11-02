package artifacts

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeS3 struct {
	mu       sync.Mutex
	objects  map[string][]byte
	lastList string
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
		prefix := r.URL.Query().Get("prefix")
		keys := make([]string, 0)
		for key := range f.objects {
			if strings.HasPrefix(key, prefix) {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		var body strings.Builder
		body.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
		body.WriteString("<Name>agenthub</Name><Prefix>" + escapeXML(prefix) + "</Prefix>")
		body.WriteString("<KeyCount>" + strconv.Itoa(len(keys)) + "</KeyCount><IsTruncated>false</IsTruncated>")
		for _, key := range keys {
			body.WriteString("<Contents><Key>" + escapeXML(key) + "</Key>")
			body.WriteString("<LastModified>" + time.Now().UTC().Format(time.RFC3339) + "</LastModified>")
			body.WriteString(`<ETag>"fake-etag"</ETag><Size>` + strconv.Itoa(len(f.objects[key])) + `</Size>`)
			body.WriteString("<StorageClass>STANDARD</StorageClass></Contents>")
		}
		body.WriteString("</ListBucketResult>")
		f.lastList = body.String()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(body.String()))
		return
	}
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

func escapeXML(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}

func newFakeS3Store(t *testing.T) (*S3ObjectStore, *fakeS3, func()) {
	t.Helper()
	fake := &fakeS3{objects: map[string][]byte{}}
	server := httptest.NewServer(fake)
	store, err := NewS3ObjectStore(S3Config{
		Endpoint:  strings.TrimPrefix(server.URL, "http://"),
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

func TestS3ObjectStoreList(t *testing.T) {
	store, fake, cleanup := newFakeS3Store(t)
	defer cleanup()
	ctx := context.Background()
	if err := store.Put(ctx, "metadata/art_1.json", []byte("1"), "application/json"); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "metadata/art_2.json", []byte("2"), "application/json"); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "runs/run-1/out.png", []byte("x"), "image/png"); err != nil {
		t.Fatal(err)
	}
	keys, err := store.List(ctx, "metadata/")
	if err != nil {
		t.Fatalf("list: %v last=%s", err, fake.lastList)
	}
	if len(keys) != 2 || keys[0] != "metadata/art_1.json" || keys[1] != "metadata/art_2.json" {
		t.Fatalf("unexpected keys: %#v", keys)
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
