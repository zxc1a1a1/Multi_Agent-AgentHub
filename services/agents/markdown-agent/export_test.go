package markdownagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func exportIdentity() context.Context {
	return adk.WithRunIdentity(context.Background(), adk.RunIdentity{
		RunID: "run-export", TaskID: "task-1", PrincipalID: "user-1", FencingVersion: 1,
	})
}

func TestExportToolRendersSanitizedMarkdown(t *testing.T) {
	var received exportRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artifactId":"art_pdf","kind":"document","mimeType":"application/pdf","size":123,"checksum":"sha256:abc"}`))
	}))
	defer server.Close()
	tool, err := NewExportTool(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Export(exportIdentity(), "# Title\n\n<script>alert(1)</script>", "pdf", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || !strings.Contains(result.Content, "artifact=art_pdf") ||
		!strings.Contains(result.Content, "download=/api/artifacts/art_pdf/content") {
		t.Fatalf("unexpected result: %#v", result)
	}
	if received.RunID != "run-export" || received.TaskID != "task-1" || received.Format != "pdf" || !received.FullPage {
		t.Fatalf("identity/format not forwarded: %#v", received)
	}
	if !strings.Contains(received.HTML, "<h1") || !strings.Contains(received.HTML, "Title") {
		t.Fatalf("markdown not rendered: %s", received.HTML)
	}
	if strings.Contains(received.HTML, "<script") {
		t.Fatalf("raw HTML must be stripped: %s", received.HTML)
	}
}

func TestExportToolRejectsMissingIdentityAndBadInput(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer server.Close()
	tool, _ := NewExportTool(server.URL, "")
	cases := []struct {
		ctx      context.Context
		markdown string
		format   string
		want     string
	}{
		{context.Background(), "# Title", "pdf", "identity"},
		{exportIdentity(), "", "pdf", "required"},
		{exportIdentity(), "# Title", "svg", "format"},
	}
	for _, tc := range cases {
		result, err := tool.Export(tc.ctx, tc.markdown, tc.format, false)
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if !result.IsError || !strings.Contains(result.Content, tc.want) {
			t.Fatalf("unexpected result: %#v", result)
		}
	}
	if calls != 0 {
		t.Fatalf("rejected exports reached the platform: %d", calls)
	}
}

func TestExportToolExecuteArgs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artifactId":"art_1","kind":"image","mimeType":"image/png","size":10,"checksum":"sha256:x"}`))
	}))
	defer server.Close()
	tool, _ := NewExportTool(server.URL, "")
	result, err := tool.Execute(exportIdentity(), json.RawMessage(`{"markdown":"# Hi","format":"png"}`))
	if err != nil || result.IsError || !strings.Contains(result.Content, "artifact=art_1") {
		t.Fatalf("unexpected execute: %#v err=%v", result, err)
	}
}
