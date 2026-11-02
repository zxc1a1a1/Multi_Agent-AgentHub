package codeagent

import (
	adk "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/tools"
)

// WorkspaceTools returns the read-only workspace tool set. ApplyPatch and
// sandbox tools are intentionally excluded: code-agent must not gain write
// capability through this path.
func WorkspaceTools(root string) ([]adk.Tool, error) {
	workspace, err := tools.NewWorkspace(root)
	if err != nil {
		return nil, err
	}
	return []adk.Tool{
		workspace.ListTool(),
		workspace.ReadTool(),
		workspace.SearchTool(),
	}, nil
}

// WorkspaceWriteTools returns approval-gated write tools. The ADK guard must
// authorize every call before it reaches ApplyPatch or the sandbox executor.
func WorkspaceWriteTools(root string, executor tools.SandboxExecutor, allowedExecutables []string) ([]adk.Tool, error) {
	workspace, err := tools.NewWorkspace(root)
	if err != nil {
		return nil, err
	}
	sandbox := tools.NewSandbox(workspace, executor, allowedExecutables)
	return []adk.Tool{workspace.ApplyPatchTool(), sandbox.RunTestsTool()}, nil
}
