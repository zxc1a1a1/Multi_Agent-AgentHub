# Frontend Runtime Capability Registry

## 注册项结构

```ts
type FrontendRuntimeSkill = {
  toolName: string
  description: string
  status: 'implemented' | 'seeded' | 'reserved' | 'disabled' | 'deprecated'
  component?: string
  behavior: 'render' | 'interactive' | 'side_effect'
  riskLevel: 'low' | 'medium' | 'high'
  requiresConfirmation: boolean
  acceptsStreamingArgs: boolean
  parseAt: 'tool_call_end'
  parametersSchema: unknown
  resultSchema?: unknown
  failureMode: 'hide' | 'placeholder' | 'error_card' | 'text_fallback'
}
```

## 状态规则

- `implemented`：当前可执行。
- `seeded`：推荐注册，实际实现以代码为准。
- `reserved`：已占名，不执行。
- `disabled`：存在但禁用。
- `deprecated`：兼容旧能力。

## 硬性规则

- `toolName` 唯一。
- `toolName` 使用 `snake_case`。
- Registry 不得使用 `agentName`。
- Registry 不得通过历史阶段字段决定当前执行。
