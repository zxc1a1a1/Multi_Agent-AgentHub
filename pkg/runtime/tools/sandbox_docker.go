package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// DockerExecutor runs allow-listed commands inside a short-lived container.
// The image is platform-configured; the model never supplies it.
type DockerExecutor struct {
	Image        string
	DockerBin    string
	MemoryMB     int
	CPUs         float64
	AllowNetwork bool
	OutputLimit  int
}

func (d *DockerExecutor) Run(ctx context.Context, spec SandboxSpec) (SandboxResult, error) {
	image := strings.TrimSpace(d.Image)
	bin := strings.TrimSpace(d.DockerBin)
	if bin == "" {
		bin = "docker"
	}
	if image == "" {
		return SandboxResult{}, fmt.Errorf("sandbox image is not configured")
	}
	if spec.Network && !d.AllowNetwork {
		return SandboxResult{}, fmt.Errorf("sandbox network is disabled by policy")
	}
	if strings.TrimSpace(spec.WorkingDir) == "" {
		return SandboxResult{}, fmt.Errorf("working directory is required")
	}
	memory := d.MemoryMB
	if memory <= 0 {
		memory = 256
	}
	cpus := d.CPUs
	if cpus <= 0 {
		cpus = 0.5
	}
	limit := d.OutputLimit
	if limit <= 0 {
		limit = 64 * 1024
	}
	network := "none"
	if spec.Network && d.AllowNetwork {
		network = "bridge"
	}
	args := []string{
		"run", "--rm",
		"--network", network,
		"--memory", fmt.Sprintf("%dm", memory),
		"--cpus", fmt.Sprintf("%g", cpus),
		"--pids-limit", "128",
		"--read-only",
		"--tmpfs", "/tmp:rw,size=16m",
		"--security-opt", "no-new-privileges",
		"-v", spec.WorkingDir + ":/workspace:ro",
		"-w", "/workspace",
		image,
		spec.Executable,
	}
	args = append(args, spec.Args...)

	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, args...)
	stdout := &limitedBuffer{limit: limit}
	stderr := &limitedBuffer{limit: limit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	result := SandboxResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if runCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = -1
		return result, nil
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("sandbox execution failed: %w", err)
	}
	return result, nil
}

type limitedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.buf.String()
	if b.truncated {
		out += "\n[output truncated]"
	}
	return out
}
