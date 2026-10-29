package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type SandboxSpec struct {
	Executable string
	Args       []string
	WorkingDir string
	Timeout    time.Duration
	Network    bool
}

type SandboxResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

// SandboxExecutor must run the spec in an isolated environment (container or
// equivalent). The default is nil so the tool never falls back to host exec.
type SandboxExecutor interface {
	Run(ctx context.Context, spec SandboxSpec) (SandboxResult, error)
}

type Sandbox struct {
	workspace      *Workspace
	executor       SandboxExecutor
	allowed        map[string]bool
	maxTimeout     time.Duration
	maxOutput      int
	defaultTimeout time.Duration
}

func NewSandbox(workspace *Workspace, executor SandboxExecutor, allowedExecutables []string) *Sandbox {
	allowed := map[string]bool{}
	for _, name := range allowedExecutables {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[name] = true
		}
	}
	return &Sandbox{
		workspace: workspace, executor: executor, allowed: allowed,
		maxTimeout: 60 * time.Second, maxOutput: 64 * 1024, defaultTimeout: 60 * time.Second,
	}
}

func (s *Sandbox) RunTestsTool() *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"executable":{"type":"string"},"args":{"type":"array","items":{"type":"string"},"maxItems":32},"timeoutSeconds":{"type":"integer","minimum":1,"maximum":60},"network":{"type":"boolean"}},"required":["executable"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "sandbox.run_tests", Version: "1",
		Description: "Run an allow-listed test command in an isolated sandbox.",
		InputSchema: schema,
		ReadOnly:    false, Destructive: true, Retryable: false, Idempotent: false,
		Approval: ApprovalAlways, Timeout: 60 * time.Second,
		MaxOutputBytes: 128 * 1024, Provider: "native",
	}
	return &workspaceTool{
		name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: s.runTests,
	}
}

func (s *Sandbox) runTests(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
	if s.executor == nil {
		return nil, fmt.Errorf("sandbox unavailable: no isolated executor configured")
	}
	var args struct {
		Executable     string   `json:"executable"`
		Args           []string `json:"args"`
		TimeoutSeconds int      `json:"timeoutSeconds"`
		Network        bool     `json:"network"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	executable := strings.TrimSpace(args.Executable)
	if executable == "" || strings.ContainsAny(executable, `/\`) || filepath.Base(executable) != executable {
		return nil, fmt.Errorf("executable must be a bare allow-listed name")
	}
	if !s.allowed[executable] {
		return nil, fmt.Errorf("executable %q is not allow-listed", executable)
	}
	if len(args.Args) > 32 {
		return nil, fmt.Errorf("too many arguments")
	}
	for _, arg := range args.Args {
		if strings.ContainsRune(arg, 0) || len(arg) > 256 {
			return nil, fmt.Errorf("invalid argument")
		}
	}
	timeout := s.defaultTimeout
	if args.TimeoutSeconds > 0 {
		timeout = time.Duration(args.TimeoutSeconds) * time.Second
	}
	if timeout > s.maxTimeout {
		timeout = s.maxTimeout
	}

	result, err := s.executor.Run(ctx, SandboxSpec{
		Executable: executable, Args: append([]string(nil), args.Args...),
		WorkingDir: s.workspace.rootReal, Timeout: timeout, Network: args.Network,
	})
	if err != nil {
		return nil, err
	}
	stdout := truncate(result.Stdout, s.maxOutput)
	stderr := truncate(result.Stderr, s.maxOutput)
	return jsonResult(map[string]any{
		"exitCode": result.ExitCode, "stdout": stdout, "stderr": stderr,
		"timedOut": result.TimedOut,
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
