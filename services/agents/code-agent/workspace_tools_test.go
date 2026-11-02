package codeagent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
)

func TestWorkspaceToolsAreReadOnlyAndReadRealFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("real file content"), 0o644); err != nil {
		t.Fatal(err)
	}
	toolSet, err := WorkspaceTools(root)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	var readTool interface {
		Execute(context.Context, json.RawMessage) (*adk.ToolResult, error)
	}
	for _, tool := range toolSet {
		names[tool.Name()] = true
		described, ok := tool.(interface{ Descriptor() tools.Descriptor })
		if !ok {
			t.Fatalf("tool %s has no descriptor", tool.Name())
		}
		if !described.Descriptor().ReadOnly {
			t.Fatalf("tool %s must be read-only", tool.Name())
		}
		if tool.Name() == "workspace.read" {
			readTool = tool
		}
	}
	if names["workspace.apply_patch"] || names["sandbox.run_tests"] {
		t.Fatalf("write tools must not be exposed: %#v", names)
	}
	if readTool == nil {
		t.Fatal("workspace.read missing")
	}
	result, err := readTool.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt"}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("read failed: %#v err=%v", result, err)
	}
	if !strings.Contains(result.Content, "real file content") {
		t.Fatalf("read did not return the real file: %s", result.Content)
	}
}
