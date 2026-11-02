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
