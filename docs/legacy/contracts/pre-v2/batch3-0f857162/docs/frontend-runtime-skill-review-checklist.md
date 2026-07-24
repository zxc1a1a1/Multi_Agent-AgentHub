# Frontend Runtime Skill Review Checklist

## 通用性

- [ ] 没有通过 `agentName` 决定 Runtime Capability。
- [ ] 没有把某个 capability 绑定到具体 Agent。
- [ ] 只通过 `toolName` 查询 registry。
- [ ] 没有通过 Agent 名称绕过 schema validation。

## Registry

- [ ] `toolName` 唯一且为 `snake_case`。
- [ ] `status` 明确。
- [ ] `behavior` 明确。
- [ ] `riskLevel` 明确。
- [ ] `requiresConfirmation` 明确。
- [ ] `failureMode` 明确。

## Tool Call

- [ ] 按 `toolCallId` 聚合 args。
- [ ] 等待 `TOOL_CALL_END` 后解析。
- [ ] 兼容 `delta` / `content`。
- [ ] malformed JSON 安全降级。
- [ ] 重复 END 不会重复执行。
- [ ] 未知 `toolName` 不执行。

## Schema

- [ ] 有参数 schema。
- [ ] 校验必填字段。
- [ ] 限制字符串长度。
- [ ] 限制 URL 协议。
- [ ] 不把未校验 args 传给组件。

## 安全

- [ ] `web_preview` 使用隔离渲染。
- [ ] 不默认启用 `allow-same-origin`。
- [ ] Markdown 默认禁用 raw HTML。
- [ ] `code_preview` 只展示不执行。
- [ ] side_effect 默认 disabled。
