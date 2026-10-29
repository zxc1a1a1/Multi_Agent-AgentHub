# Planner Input Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：冻结允许集合、模型能力与失败前置条件）

## Authoritative source order

1. `docs/contracts/intent-orchestration.md`
2. `docs/contracts/chat-execution-path.md`
3. `docs/contracts/participant-boundary.md`
4. `docs/contracts/orchestration-plan.md`

## PlannerInput

```json
{
  "runId": "run_...",
  "conversationId": "conv_...",
  "executionPath": "single_chat|group_chat|main_agent_orchestration",
  "allowedAgents": ["code-agent"],
  "userMessage": "...",
  "history": [{"role": "assistant", "content": "...", "senderName": "code-agent"}],
  "agents": [{"name": "code-agent", "description": "...", "capabilities": ["code_generation"], "outputModes": ["text", "code"]}],
  "tools": [{"name": "workspace.read", "description": "...", "readOnly": true}],
  "observations": [{"toolCallId": "tc_...", "status": "succeeded|failed|unknown", "summary": "..."}],
  "budget": {"modelCallsRemaining": 12, "toolCallsRemaining": 20},
  "metadata": {"conversationType": "single|group"}
}
```

## Rules

1. Planner must never inspect Gateway DB directly.
2. Planner must not call Child Agents or execute tools.
3. Planner output must be validated before execution.
4. `allowedAgents` 由服务端身份/会话边界生成，不接受浏览器任意扩大；模型不能修改。
5. `tools` 只包含本次运行授权且可用的工具描述；未配置后端的工具不得出现在列表中。
6. `observations` 是已完成工具/委派的证据摘要；大结果以 artifact 引用传递，不整段重放。
7. 模型能力缺失（无 provider/model/key/结构化输出能力）时，Planner 入口必须失败，不得构造空模型 Planner。
8. 历史与 observation 必须有界；超过上下文预算时按 `runtime-harness.md` 的 ContextBuilder 规则裁剪，但不得拆散 tool call-result 配对。
9. 预算字段是只读视图；Planner 不能自报预算或审批状态。
