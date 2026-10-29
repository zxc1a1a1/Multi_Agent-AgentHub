package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestWorkspace(t *testing.T) (*Workspace, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "nested.txt"), []byte("nested content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := NewWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	return w, root
}

func decodeResult(t *testing.T, content string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		t.Fatalf("decode tool result: %v (%s)", err, content)
	}
	return out
}

func TestWorkspaceReadAndList(t *testing.T) {
	w, _ := newTestWorkspace(t)
	ctx := context.Background()

	read, err := w.ReadTool().Execute(ctx, json.RawMessage(`{"path":"hello.txt"}`))
	if err != nil || read.IsError {
		t.Fatalf("read failed: err=%v result=%#v", err, read)
	}
	out := decodeResult(t, read.Content)
	if out["content"] != "hello world\n" || out["truncated"] != false {
		t.Fatalf("unexpected read result: %#v", out)
	}

	list, err := w.ListTool().Execute(ctx, json.RawMessage(`{}`))
	if err != nil || list.IsError {
		t.Fatalf("list failed: err=%v result=%#v", err, list)
	}
	listed := decodeResult(t, list.Content)
	entries, _ := listed["entries"].([]any)
	names := map[string]bool{}
	for _, e := range entries {
		names[e.(map[string]any)["name"].(string)] = true
	}
	if !names["hello.txt"] || !names["sub"] {
		t.Fatalf("list missing expected entries: %#v", names)
	}
}

func TestWorkspaceRejectsTraversalAndAbsolutePaths(t *testing.T) {
	w, _ := newTestWorkspace(t)
	ctx := context.Background()
	for _, path := range []string{"../outside.txt", "sub/../../outside.txt", filepath.Join(w.root, "hello.txt")} {
		res, err := w.ReadTool().Execute(ctx, json.RawMessage(`{"path":`+quote(path)+`}`))
		if err != nil {
			t.Fatalf("unexpected go error: %v", err)
		}
		if !res.IsError {
			t.Fatalf("path %q must be rejected", path)
		}
	}
	list, _ := w.ListTool().Execute(ctx, json.RawMessage(`{"path":".."}`))
	if !list.IsError {
		t.Fatal("list path traversal must be rejected")
	}
	search, _ := w.SearchTool().Execute(ctx, json.RawMessage(`{"query":"x","path":"../"}`))
	if !search.IsError {
		t.Fatal("search path traversal must be rejected")
	}
}

func TestWorkspaceDeniesCredentialFiles(t *testing.T) {
	w, root := newTestWorkspace(t)
	ctx := context.Background()
	for name, content := range map[string]string{
		".env":             "SECRET=top-secret-value",
		".env.local":       "SECRET=top-secret-value",
		"server.pem":       "SECRET",
		"id_rsa":           "SECRET",
		"credentials.json": "SECRET",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name := range map[string]bool{".env": true, ".env.local": true, "server.pem": true, "id_rsa": true, "credentials.json": true} {
		res, _ := w.ReadTool().Execute(ctx, json.RawMessage(`{"path":`+quote(name)+`}`))
		if !res.IsError {
			t.Fatalf("credential file %q must be denied", name)
		}
	}
	list, _ := w.ListTool().Execute(ctx, json.RawMessage(`{}`))
	listed := decodeResult(t, list.Content)
	for _, e := range listed["entries"].([]any) {
		name := e.(map[string]any)["name"].(string)
		if deniedName(name) {
			t.Fatalf("credential entry %q must not be listed", name)
		}
	}
	search, _ := w.SearchTool().Execute(ctx, json.RawMessage(`{"query":"SECRET"}`))
	searched := decodeResult(t, search.Content)
	if len(searched["matches"].([]any)) != 0 {
		t.Fatalf("search must not read credential files: %#v", searched)
	}
}

func TestWorkspaceRejectsSymlinkEscape(t *testing.T) {
	w, root := newTestWorkspace(t)
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, _ := w.ReadTool().Execute(context.Background(), json.RawMessage(`{"path":"escape.txt"}`))
	if !res.IsError {
		t.Fatal("symlink escaping the root must be rejected")
	}
	list, _ := w.ListTool().Execute(context.Background(), json.RawMessage(`{}`))
	listed := decodeResult(t, list.Content)
	for _, e := range listed["entries"].([]any) {
		if e.(map[string]any)["name"] == "escape.txt" {
			t.Fatal("escaping symlink must not be listed")
		}
	}
}

func TestWorkspaceReadTruncatesAndRejectsBinary(t *testing.T) {
	w, root := newTestWorkspace(t)
	big := filepath.Join(root, "big.txt")
	if err := os.WriteFile(big, []byte(strings.Repeat("a", 200_000)), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _ := w.ReadTool().Execute(context.Background(), json.RawMessage(`{"path":"big.txt","maxBytes":1024}`))
	if res.IsError {
		t.Fatalf("read failed: %s", res.Content)
	}
	out := decodeResult(t, res.Content)
	if out["truncated"] != true || len(out["content"].(string)) != 1024 {
		t.Fatalf("expected 1024-byte truncation, got %#v", out)
	}

	if err := os.WriteFile(filepath.Join(root, "binary.bin"), []byte{'a', 0, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	bin, _ := w.ReadTool().Execute(context.Background(), json.RawMessage(`{"path":"binary.bin"}`))
	if !bin.IsError {
		t.Fatal("binary file must be rejected")
	}
}

func TestWorkspaceSearchBoundsAndSkipsSecrets(t *testing.T) {
	w, root := newTestWorkspace(t)
	for i, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("needle line\nneedle again\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = i
	}
	res, _ := w.SearchTool().Execute(context.Background(), json.RawMessage(`{"query":"needle","maxResults":2}`))
	out := decodeResult(t, res.Content)
	matches := out["matches"].([]any)
	if len(matches) != 2 {
		t.Fatalf("expected 2 bounded matches, got %d", len(matches))
	}
	if out["truncated"] != true {
		t.Fatal("expected truncated search result")
	}
}

func TestGuardAuthorization(t *testing.T) {
	read := Descriptor{Name: "workspace.read", Version: "1", ReadOnly: true, Approval: ApprovalNone, Timeout: time.Second, MaxOutputBytes: 1024, Provider: "native", InputSchema: json.RawMessage(`{"type":"object"}`)}
	write := Descriptor{Name: "workspace.apply_patch", Version: "1", ReadOnly: false, Approval: ApprovalAlways, Timeout: time.Second, MaxOutputBytes: 1024, Provider: "native", InputSchema: json.RawMessage(`{"type":"object"}`)}
	if err := read.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := write.Validate(); err != nil {
		t.Fatal(err)
	}
	base := Envelope{RunID: "r", ToolCallID: "tc", ToolName: read.Name, ToolVersion: read.Version, ArgsHash: "h", PrincipalID: "p", FencingVersion: 1}
	if err := (Guard{}).Authorize(context.Background(), base, read); err != nil {
		t.Fatalf("read-only call should authorize: %v", err)
	}
	if err := (Guard{}).Authorize(context.Background(), Envelope{RunID: "r"}, read); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("missing identity must be rejected, got %v", err)
	}
	writeEnvelope := Envelope{RunID: "r", ToolCallID: "tc", ToolName: write.Name, ToolVersion: write.Version, ArgsHash: "h", PrincipalID: "p", FencingVersion: 1}
	if err := (Guard{}).Authorize(context.Background(), writeEnvelope, write); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("write without approval must be rejected, got %v", err)
	}
	writeEnvelope.ApprovalID = "a1"
	if err := (Guard{}).Authorize(context.Background(), writeEnvelope, write); err != nil {
		t.Fatalf("approved write should authorize: %v", err)
	}
	if err := (Guard{}).Authorize(context.Background(), base, write); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("tool name mismatch must be rejected, got %v", err)
	}
}

func TestDescriptorRejectsUnsafeWrites(t *testing.T) {
	d := Descriptor{Name: "workspace.apply_patch", Version: "1", ReadOnly: false, Approval: ApprovalNone, Timeout: time.Second, MaxOutputBytes: 1024, Provider: "native", InputSchema: json.RawMessage(`{}`)}
	if err := d.Validate(); err == nil {
		t.Fatal("write tool without approval must be invalid")
	}
	d.Approval = ApprovalAlways
	if err := d.Validate(); err != nil {
		t.Fatalf("valid write descriptor rejected: %v", err)
	}
	d.InputSchema = json.RawMessage(`not-json`)
	if err := d.Validate(); err == nil {
		t.Fatal("invalid schema must be rejected")
	}
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
