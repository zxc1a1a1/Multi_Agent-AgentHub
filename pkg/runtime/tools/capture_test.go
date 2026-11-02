package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func captureIdentity() context.Context {
	return adk.WithRunIdentity(context.Background(), adk.RunIdentity{
		RunID: "run-capture", TaskID: "task-1", PrincipalID: "user-1", FencingVersion: 1,
	})
}

func TestCaptureToolPublishesArtifactReference(t *testing.T) {
	var received captureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artifactId":"art_123","kind":"image","mimeType":"image/png","size":12,"checksum":"sha256:abc","downloadPath":"/internal/orchestrator/artifacts/art_123/content"}`))
	}))
	defer server.Close()
	tool, err := NewCaptureTool(CaptureConfig{Endpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Execute(captureIdentity(), json.RawMessage(`{"url":"https://93.184.216.34/","format":"pdf","fullPage":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || !strings.Contains(result.Content, "artifact=art_123") || !strings.Contains(result.Content, "checksum=sha256:abc") {
		t.Fatalf("unexpected result: %#v", result)
	}
	if received.RunID != "run-capture" || received.TaskID != "task-1" || received.URL != "https://93.184.216.34/" ||
		received.Format != "pdf" || !received.FullPage {
		t.Fatalf("request not forwarded: %#v", received)
	}
}

func TestCaptureToolRejectsUnsafeInputsWithoutRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer server.Close()
	tool, _ := NewCaptureTool(CaptureConfig{Endpoint: server.URL})

	cases := []struct {
		ctx  context.Context
		args string
		want string
	}{
		{context.Background(), `{"url":"https://93.184.216.34/"}`, "identity"},
		{captureIdentity(), `{"url":"file:///etc/passwd"}`, "http(s)"},
		{captureIdentity(), `{"url":"https://93.184.216.34/","format":"svg"}`, "format"},
	}
	for _, tc := range cases {
		result, err := tool.Execute(tc.ctx, json.RawMessage(tc.args))
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if !result.IsError || !strings.Contains(result.Content, tc.want) {
			t.Fatalf("case %s: %#v", tc.args, result)
		}
	}
	if calls != 0 {
		t.Fatalf("rejected captures reached the platform: %d", calls)
	}
}

func TestCaptureToolMapsPolicyDenial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"url is not allowed"}`))
	}))
	defer server.Close()
	tool, _ := NewCaptureTool(CaptureConfig{Endpoint: server.URL})
	result, err := tool.Execute(captureIdentity(), json.RawMessage(`{"url":"https://93.184.216.34/"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(result.Content, "not allowed") {
		t.Fatalf("denial not mapped: %#v", result)
	}
}
