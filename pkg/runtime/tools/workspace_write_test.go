package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

func sha(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func TestApplyPatchHashChecked(t *testing.T) {
	w, root := newTestWorkspace(t)
	tool := w.ApplyPatchTool()
	d := tool.Descriptor()
	if d.ReadOnly || d.Approval != ApprovalAlways {
		t.Fatalf("patch descriptor must be write+approval: %#v", d)
	}
	before := "hello world\n"
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt","expectedSha256":"`+sha(before)+`","content":"patched\n"}`))
	if err != nil || res.IsError {
		t.Fatalf("patch failed: err=%v result=%#v", err, res)
	}
	out := decodeResult(t, res.Content)
	if out["previousSha256"] != sha(before) || out["sha256"] != sha("patched\n") {
		t.Fatalf("unexpected patch result: %#v", out)
	}
	data, _ := os.ReadFile(filepath.Join(root, "hello.txt"))
	if string(data) != "patched\n" {
		t.Fatalf("file not patched: %q", data)
	}
}

func TestApplyPatchRejectsStaleHashAndLeavesFile(t *testing.T) {
	w, root := newTestWorkspace(t)
	tool := w.ApplyPatchTool()
	res, _ := tool.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt","expectedSha256":"`+sha("other")+`","content":"x"}`))
	if !res.IsError {
		t.Fatal("stale hash must be rejected")
	}
	data, _ := os.ReadFile(filepath.Join(root, "hello.txt"))
	if string(data) != "hello world\n" {
		t.Fatalf("stale patch must not modify file: %q", data)
	}
	res, _ = tool.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt","content":"x"}`))
	if !res.IsError {
		t.Fatal("existing file without expected hash must be rejected")
	}
}

func TestApplyPatchCreatesNewFile(t *testing.T) {
	w, root := newTestWorkspace(t)
	tool := w.ApplyPatchTool()
	res, _ := tool.Execute(context.Background(), json.RawMessage(`{"path":"nested/new.txt","content":"new content"}`))
	if res.IsError {
		t.Fatalf("new file patch failed: %s", res.Content)
	}
	data, err := os.ReadFile(filepath.Join(root, "nested", "new.txt"))
	if err != nil || string(data) != "new content" {
		t.Fatalf("new file missing: %v %q", err, data)
	}
}

func TestApplyPatchRejectsTraversalCredentialAndSymlink(t *testing.T) {
	w, root := newTestWorkspace(t)
	tool := w.ApplyPatchTool()
	for _, path := range []string{"../outside.txt", ".env", "id_rsa"} {
		res, _ := tool.Execute(context.Background(), json.RawMessage(`{"path":"`+path+`","content":"x"}`))
		if !res.IsError {
			t.Fatalf("path %q must be rejected", path)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	_ = os.WriteFile(outside, []byte("secret"), 0o644)
	if err := os.Symlink(outside, filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, _ := tool.Execute(context.Background(), json.RawMessage(`{"path":"escape.txt","expectedSha256":"`+sha("secret")+`","content":"x"}`))
	if !res.IsError {
		t.Fatal("symlink escaping root must be rejected")
	}
}

type countingTool struct {
	calls int
}

func (c *countingTool) Name() string            { return "workspace.apply_patch" }
func (c *countingTool) Description() string     { return "test" }
func (c *countingTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (c *countingTool) Execute(context.Context, json.RawMessage) (*adk.ToolResult, error) {
	c.calls++
	return &adk.ToolResult{Content: `{}`}, nil
}

func TestBrokerBlocksUnapprovedWrite(t *testing.T) {
	w, _ := newTestWorkspace(t)
	d := w.ApplyPatchTool().Descriptor()
	tool := &countingTool{}
	env := Envelope{RunID: "r", ToolCallID: "tc", ToolName: d.Name, ToolVersion: d.Version, ArgsHash: "h", PrincipalID: "p", FencingVersion: 1}
	if _, err := Execute(context.Background(), Guard{}, env, d, tool, json.RawMessage(`{}`)); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired, got %v", err)
	}
	if tool.calls != 0 {
		t.Fatal("unapproved write reached the tool")
	}
	env.ApprovalID = "appr-1"
	if _, err := Execute(context.Background(), Guard{}, env, d, tool, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("approved write should execute: %v", err)
	}
	if tool.calls != 1 {
		t.Fatal("approved write did not execute")
	}
}
