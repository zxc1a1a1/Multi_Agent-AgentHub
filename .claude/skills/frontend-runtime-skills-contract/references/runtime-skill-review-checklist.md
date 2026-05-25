# Runtime Skill Review Checklist

## 通用性

- [ ] 是否没有通过 `agentName` 决定 Runtime Capability？
- [ ] 是否没有把某个 Agent 写成某个 capability 的专属来源？
- [ ] 是否只通过 `toolName` 查询 registry？
- [ ] 是否没有通过 Agent 名称绕过参数校验？

## Registry

- [ ] `toolName` 是否唯一？
- [ ] `toolName` 是否使用 `snake_case`？
- [ ] `status` 是否明确？
- [ ] `behavior` 是否明确？
- [ ] `riskLevel` 是否明确？
- [ ] `requiresConfirmation` 是否明确？
- [ ] `failureMode` 是否明确？

## Tool Call

- [ ] 是否按 `toolCallId` 聚合 args？
- [ ] 是否等待 `TOOL_CALL_END` 后解析？
- [ ] 是否兼容 `delta` / `content`？
- [ ] malformed JSON 是否安全降级？
- [ ] 重复 END 是否不会重复执行？
- [ ] 未知 `toolName` 是否不会执行？

## 安全

- [ ] `web_preview` 是否 iframe 隔离？
- [ ] Markdown 是否默认禁用 raw HTML？
- [ ] `code_preview` 是否只展示不执行？
- [ ] side_effect 是否默认 disabled？
- [ ] 用户可见错误是否脱敏？
