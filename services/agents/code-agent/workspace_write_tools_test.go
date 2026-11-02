package codeagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
)

type fakeSandbox struct {
	calls int
	spec  tools.SandboxSpec
}

func (f *fakeSandbox) Run(_ context.Context, spec tools.SandboxSpec) (tools.SandboxResult, error) {
	f.calls++
	f.spec = spec
	return tools.SandboxResult{ExitCode: 0, Stdout: "ok"}, nil
}

func TestWorkspaceWriteToolsAreApprovalGatedAndSandboxed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	executor := &fakeSandbox{}
	toolSet, err := WorkspaceWriteTools(root, executor, []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(toolSet) != 2 {
		t.Fatalf("expected apply_patch and sandbox tool: %#v", toolSet)
	}
	names := map[string]adk.Tool{}
	for _, tool := range toolSet {
		names[tool.Name()] = tool
		described, ok := tool.(interface{ Descriptor() tools.Descriptor })
		if !ok {
			t.Fatalf("tool %s lacks descriptor", tool.Name())
		}
		if described.Descriptor().ReadOnly || described.Descriptor().Approval != tools.ApprovalAlways {
			t.Fatalf("write tool must require approval: %#v", described.Descriptor())
		}
	}
	patch, ok := names["workspace.apply_patch"]
	if !ok {
		t.Fatal("apply_patch missing")
	}
	args, _ := json.Marshal(map[string]any{
		"path": "a.txt", "expectedSha256": sha256Hex("before"), "content": "after",
	})
	result, err := patch.Execute(context.Background(), args)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("patch failed: %#v err=%v", result, err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(data) != "after" {
		t.Fatalf("patch did not write: %q", data)
	}

	sandbox, ok := names["sandbox.run_tests"]
	if !ok {
		t.Fatal("sandbox.run_tests missing")
	}
	result, err = sandbox.Execute(context.Background(), json.RawMessage(`{"executable":"go","args":["test","./..."]}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("sandbox failed: %#v err=%v", result, err)
	}
	if executor.calls != 1 || executor.spec.Executable != "go" || executor.spec.WorkingDir != root {
		t.Fatalf("sandbox not used as expected: %#v", executor.spec)
	}
	if executor.spec.Timeout <= 0 || executor.spec.Timeout > time.Minute {
		t.Fatalf("sandbox timeout not bounded: %s", executor.spec.Timeout)
	}
}

func TestWorkspaceWriteToolsRejectUnlistedExecutable(t *testing.T) {
	executor := &fakeSandbox{}
	toolSet, err := WorkspaceWriteTools(t.TempDir(), executor, []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range toolSet {
		if tool.Name() == "sandbox.run_tests" {
			result, err := tool.Execute(context.Background(), json.RawMessage(`{"executable":"sh","args":["-c","id"]}`))
			if err != nil || result == nil || !result.IsError || !strings.Contains(result.Content, "not allow-listed") {
				t.Fatalf("unlisted executable must be rejected: %#v err=%v", result, err)
			}
		}
	}
	if executor.calls != 0 {
		t.Fatal("unlisted executable reached the sandbox executor")
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
