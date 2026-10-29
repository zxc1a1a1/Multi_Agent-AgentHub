package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeSandboxExecutor struct {
	calls int
	spec  SandboxSpec
	err   error
	out   SandboxResult
}

func (f *fakeSandboxExecutor) Run(_ context.Context, spec SandboxSpec) (SandboxResult, error) {
	f.calls++
	f.spec = spec
	if f.err != nil {
		return SandboxResult{}, f.err
	}
	return f.out, nil
}

func TestSandboxRunsAllowListedCommand(t *testing.T) {
	w, root := newTestWorkspace(t)
	exec := &fakeSandboxExecutor{out: SandboxResult{ExitCode: 0, Stdout: "PASS"}}
	sb := NewSandbox(w, exec, []string{"go", "npm"})
	tool := sb.RunTestsTool()
	d := tool.Descriptor()
	if d.ReadOnly || d.Approval != ApprovalAlways {
		t.Fatalf("sandbox descriptor must be write+approval: %#v", d)
	}
	res, _ := tool.Execute(context.Background(), json.RawMessage(`{"executable":"go","args":["test","./..."],"timeoutSeconds":99}`))
	if res.IsError {
		t.Fatalf("sandbox run failed: %s", res.Content)
	}
	if exec.calls != 1 || exec.spec.Executable != "go" || len(exec.spec.Args) != 2 {
		t.Fatalf("unexpected executor call: %#v", exec.spec)
	}
	if exec.spec.WorkingDir != root {
		t.Fatalf("working dir must be the workspace root, got %s", exec.spec.WorkingDir)
	}
	if exec.spec.Timeout != 60*time.Second {
		t.Fatalf("timeout must be clamped to 60s, got %s", exec.spec.Timeout)
	}
	out := decodeResult(t, res.Content)
	if out["stdout"] != "PASS" || out["exitCode"] != float64(0) {
		t.Fatalf("unexpected result: %#v", out)
	}
}

func TestSandboxRejectsUnlistedExecutableAndUnavailable(t *testing.T) {
	w, _ := newTestWorkspace(t)
	exec := &fakeSandboxExecutor{}
	sb := NewSandbox(w, exec, []string{"go"})
	tool := sb.RunTestsTool()
	for _, raw := range []string{
		`{"executable":"sh","args":["-c","rm -rf /"]}`,
		`{"executable":"/bin/sh"}`,
		`{"executable":"../go"}`,
	} {
		res, _ := tool.Execute(context.Background(), json.RawMessage(raw))
		if !res.IsError {
			t.Fatalf("executable must be rejected: %s", raw)
		}
	}
	if exec.calls != 0 {
		t.Fatal("rejected commands reached the executor")
	}

	unavailable := NewSandbox(w, nil, []string{"go"})
	res, _ := unavailable.RunTestsTool().Execute(context.Background(), json.RawMessage(`{"executable":"go"}`))
	if !res.IsError {
		t.Fatal("sandbox without executor must be unavailable")
	}
}

func TestSandboxRequiresApprovalBeforeExecution(t *testing.T) {
	w, _ := newTestWorkspace(t)
	exec := &fakeSandboxExecutor{}
	sb := NewSandbox(w, exec, []string{"go"})
	tool := sb.RunTestsTool()
	d := tool.Descriptor()
	env := Envelope{RunID: "r", ToolCallID: "tc", ToolName: d.Name, ToolVersion: d.Version, ArgsHash: "h", PrincipalID: "p", FencingVersion: 1}
	if _, err := Execute(context.Background(), Guard{}, env, d, tool, json.RawMessage(`{"executable":"go"}`)); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("expected approval requirement, got %v", err)
	}
	if exec.calls != 0 {
		t.Fatal("unapproved sandbox command executed")
	}
	env.ApprovalID = "appr-1"
	if _, err := Execute(context.Background(), Guard{}, env, d, tool, json.RawMessage(`{"executable":"go"}`)); err != nil {
		t.Fatalf("approved sandbox should run: %v", err)
	}
	if exec.calls != 1 {
		t.Fatal("approved sandbox command did not run")
	}
}
