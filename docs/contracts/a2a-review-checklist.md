# A2A Agent Contract Review Checklist

## 1. 文件完整性

- [ ] `skills/a2a-agent-contract/SKILL.md` 存在。
- [ ] `docs/contracts/a2a-agent-card.md` 存在。
- [ ] `docs/contracts/a2a-task.md` 存在。
- [ ] `docs/contracts/a2a-errors.md` 存在。
- [ ] 本文件存在。
- [ ] 未把 A2A endpoint 写入 Frontend REST OpenAPI。

---

## 2. Skill 与文档一致性

- [ ] `SKILL.md` 中要求的 AgentCard 已在 `a2a-agent-card.md` 中定义。
- [ ] `SKILL.md` 中要求的 A2A task 已在 `a2a-task.md` 中定义。
- [ ] `SKILL.md` 中要求的 A2A errors 已在 `a2a-errors.md` 中定义。
- [ ] 三份 Contract 文档没有互相冲突。
- [ ] 三份 Contract 文档均区分 MVP v0.1 与 Post-MVP。

---

## 3. MVP v0.1 检查

- [ ] MVP v0.1 只强制实现 `code-agent`。
- [ ] `code-agent` 必须暴露 AgentCard。
- [ ] `code-agent` 必须提供 A2A Server。
- [ ] `code-agent` 必须支持 `/a2a/tasks/sendSubscribe`。
- [ ] `code-agent` 必须使用 ADK Runtime 约定。
- [ ] `code-agent` 必须能接收用户消息和历史上下文。
- [ ] `code-agent` 必须能流式输出文本。
- [ ] `code-agent` 必须能生成 `code` Artifact。
- [ ] `code` Artifact metadata 至少包含 `language`。
- [ ] `code` Artifact title 可作为 filename。
- [ ] `code` Artifact 最终可映射为 `code_preview`。
- [ ] MVP v0.1 不强制 `web-agent` / `doc-agent` / `custom-agent`。
- [ ] MVP v0.1 不强制复杂 Agent Registry。

---

## 4. PDR 完整目标检查

- [ ] 保留多 Agent 扩展方向。
- [ ] 保留 `web-agent` / `doc-agent` / `custom-agent` 扩展方向。
- [ ] 保留 Agent Registry 扩展方向。
- [ ] 保留 AgentCard 动态拉取方向。
- [ ] 保留 capability 匹配方向。
- [ ] 保留多 Agent ExecutionPlan 方向。
- [ ] 没有把单 `code-agent` 写死为长期唯一架构。
- [ ] 遵守 Contract first / Mock first / Real integration later / Review always。

---

## 5. AgentCard 检查

- [ ] `GET /.well-known/agent.json` 是否存在？
- [ ] AgentCard 是否包含 `name`？
- [ ] AgentCard 是否包含 `description`？
- [ ] AgentCard 是否包含 `url`？
- [ ] AgentCard 是否包含 `version`？
- [ ] AgentCard 是否包含 `capabilities`？
- [ ] AgentCard 是否包含 `skills`？
- [ ] AgentCard 是否包含 `inputModes`？
- [ ] AgentCard 是否包含 `outputModes`？
- [ ] `code-agent` 的 `inputModes` 是否包含 `text`？
- [ ] `code-agent` 的 `outputModes` 是否包含 `text` 和 `code`？
- [ ] `capabilities.streaming` 是否为 true？
- [ ] `capabilities.artifacts` 是否为 true？
- [ ] `skills` 是否包含代码生成能力？
- [ ] AgentCard 是否不泄漏 API key / token / system prompt？

---

## 6. A2A Endpoint / Task 检查

- [ ] `POST /a2a/tasks/sendSubscribe` 是否存在？
- [ ] request 是否包含 task id？
- [ ] request 是否包含 messages？
- [ ] metadata 是否包含 `runId`？
- [ ] metadata 是否包含 `threadId`？
- [ ] metadata 是否包含 `traceId`？
- [ ] metadata 是否不携带用户 token？
- [ ] metadata 是否不携带 API key？
- [ ] A2A endpoint 是否不暴露给 Frontend？
- [ ] Gateway handler 是否没有直接调用 A2A endpoint？
- [ ] Orchestrator 是否通过 A2A Client 调用？

---

## 7. A2A Streaming Event 检查

- [ ] 是否支持 `status: working`？
- [ ] 是否支持 `text`？
- [ ] 是否支持 `artifact`？
- [ ] 是否支持 `status: completed`？
- [ ] 是否支持 `status: failed`？
- [ ] `completed` 后是否不继续输出 text / artifact？
- [ ] `failed` 后是否不继续输出正常事件？
- [ ] A2A event 是否没有直接暴露给 Frontend？
- [ ] A2A event 是否先经过 ProtocolConverter？
- [ ] Gateway handler 是否没有直接解析 A2A event？

---

## 8. Artifact 检查

- [ ] MVP 是否只强制 `type = code`？
- [ ] `code` Artifact 是否包含 `type`？
- [ ] `code` Artifact 是否包含 `title`？
- [ ] `code` Artifact 是否包含 `content`？
- [ ] `code` Artifact 是否包含 `metadata.language`？
- [ ] `title` 是否可作为 filename？
- [ ] 大代码是否作为 Artifact，而不是只塞进 text chunk？
- [ ] `code` Artifact 是否映射到 `code_preview`？
- [ ] Agent 是否没有直接决定前端组件？

---

## 9. 错误 / 安全 / Trace 检查

- [ ] A2A failed 是否能映射到 `RUN_ERROR`？
- [ ] 是否定义 A2A 错误码？
- [ ] 错误是否不泄漏 API key？
- [ ] 错误是否不泄漏 token？
- [ ] 错误是否不泄漏内部堆栈？
- [ ] AgentCard 是否不泄漏敏感信息？
- [ ] A2A metadata 是否不携带用户 token？
- [ ] traceId 是否贯穿 Gateway / Orchestrator / Child Agent？
- [ ] runId 是否可关联 AG-UI Run？
- [ ] taskId 是否可关联 A2A Task？
- [ ] agentName 是否可定位失败 Agent？

---

## 10. Mock-first / Contract Test 检查

- [ ] Mock Agent 是否暴露 AgentCard？
- [ ] Mock Agent 是否暴露 `/a2a/tasks/sendSubscribe`？
- [ ] Mock Agent 是否输出合法 A2A stream event？
- [ ] Mock Agent 是否能模拟 `status: working`？
- [ ] Mock Agent 是否能模拟 `text`？
- [ ] Mock Agent 是否能模拟 `code` Artifact？
- [ ] Mock Agent 是否能模拟 `status: completed`？
- [ ] Mock Agent 是否能模拟 `status: failed`？
- [ ] Mock Agent 是否不直接输出 AG-UI Event？
- [ ] Contract Test 是否覆盖 AgentCard？
- [ ] Contract Test 是否覆盖 sendSubscribe？
- [ ] Contract Test 是否覆盖 Artifact？
- [ ] Contract Test 是否覆盖边界检查？

---

## 11. 阶段越界检查

- [ ] 当前仅生成 Contract 文档，不生成 Go 实现。
- [ ] 未生成 ADK Runtime 实现。
- [ ] 未生成 code-agent 业务实现。
- [ ] 未生成 A2A Client 实现。
- [ ] 未生成 Gateway handler。
- [ ] 未生成 Docker Compose。
- [ ] 未进入下一个 Skill。


## 12. v1.1 对齐补充检查项

- [ ] 是否包含 `GET /health`？
- [ ] 是否说明 MVP 最小 endpoint 与 Post-MVP 完整 endpoint 范围？
- [ ] 是否说明 `DELETE /a2a/tasks/:id/cancel` 是 v1.1 推荐路径？
- [ ] 是否说明 `POST /a2a/tasks/{id}/cancel` 仅为兼容路径？
- [ ] 是否说明 `/.well-known/agent.json` 与 `/.well-known/agent-card.json` 的兼容关系？
- [ ] A2A metadata 是否不携带用户 token？
- [ ] A2A 错误是否不泄漏 token、API key、stack trace、内部地址、完整 system prompt？

