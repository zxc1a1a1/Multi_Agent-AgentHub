package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dockerAvailable() bool {
	if os.Getenv("AGENTHUB_DOCKER_SANDBOX_TEST") != "1" {
		return false
	}
	return exec.Command("docker", "info").Run() == nil
}

func TestDockerSandboxRunsAllowListedCommand(t *testing.T) {
	w, _ := newTestWorkspace(t)
	if !dockerAvailable() {
		t.Skip("SKIP docker sandbox: set AGENTHUB_DOCKER_SANDBOX_TEST=1 and start Docker")
	}
	executor := &DockerExecutor{Image: "alpine:3"}
	sb := NewSandbox(w, executor, []string{"echo"})
	tool := sb.RunTestsTool()
	result, err := tool.Execute(context.Background(), []byte(`{"executable":"echo","args":["hello-sandbox"],"timeoutSeconds":30}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("sandbox failed: %s", result.Content)
	}
	out := decodeResult(t, result.Content)
	if !strings.Contains(out["stdout"].(string), "hello-sandbox") {
		t.Fatalf("sandbox stdout missing output: %#v", out)
	}
	if out["exitCode"] != float64(0) {
		t.Fatalf("unexpected exit code: %#v", out)
	}
}

func TestDockerExecutorUnavailableDoesNotRunOnHost(t *testing.T) {
	w, _ := newTestWorkspace(t)
	marker := filepath.Join(w.root, "host-ran")
	executor := &DockerExecutor{
		Image:     "alpine:3",
		DockerBin: filepath.Join(t.TempDir(), "missing-docker"),
	}
	_, err := executor.Run(context.Background(), SandboxSpec{
		Executable: "sh", Args: []string{"-c", "touch " + marker},
		WorkingDir: w.root, Timeout: 5 * time.Second,
	})
	if err == nil {
		t.Fatal("missing docker binary must fail closed")
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("host command was executed despite unavailable sandbox: %v", statErr)
	}
}

func TestDockerSandboxTimeout(t *testing.T) {
	w, _ := newTestWorkspace(t)
	if !dockerAvailable() {
		t.Skip("SKIP docker sandbox: set AGENTHUB_DOCKER_SANDBOX_TEST=1 and start Docker")
	}
	sb := NewSandbox(w, &DockerExecutor{Image: "alpine:3"}, []string{"sleep"})
	tool := sb.RunTestsTool()
	result, _ := tool.Execute(context.Background(), []byte(`{"executable":"sleep","args":["5"],"timeoutSeconds":1}`))
	if result.IsError {
		t.Fatalf("sandbox failed: %s", result.Content)
	}
	out := decodeResult(t, result.Content)
	if out["timedOut"] != true {
		t.Fatalf("expected timeout: %#v", out)
	}
}

func TestDockerSandboxRejectsNetworkByDefault(t *testing.T) {
	w, _ := newTestWorkspace(t)
	executor := &DockerExecutor{Image: "alpine:3"}
	if _, err := executor.Run(context.Background(), SandboxSpec{
		Executable: "echo", WorkingDir: w.rootReal, Network: true, Timeout: time.Second,
	}); err == nil {
		t.Fatal("network must be rejected unless explicitly allowed")
	}
}
