# Gateway-Orchestrator Contract Review Checklist

## 1. 文件完整性

- [ ] `skills/gateway-orchestrator-contract/SKILL.md` 存在。
- [ ] `docs/contracts/gateway-orchestrator.md` 存在。
- [ ] `docs/contracts/gateway-orchestrator-events.md` 存在。
- [ ] 本文件存在。
- [ ] 未将 Gateway-Orchestrator internal contract 写入 Frontend REST OpenAPI。

---

## 2. Skill 与文档一致性

- [ ] `SKILL.md` 中要求的 Gateway 职责已在 `gateway-orchestrator.md` 中说明。
- [ ] `SKILL.md` 中要求的 Orchestrator 职责已在 `gateway-orchestrator.md` 中说明。
- [ ] `SKILL.md` 中要求的 MVP v0.1 同进程模式已说明。
- [ ] `SKILL.md` 中要求的 Post-MVP 独立服务模式已说明。
- [ ] `SKILL.md` 中要求的事件输出规则已在 `gateway-orchestrator-events.md` 中说明。
- [ ] `SKILL.md`、`gateway-orchestrator.md`、`gateway-orchestrator-events.md` 没有互相冲突。

---

## 3. MVP v0.1 检查

- [ ] 允许 Orchestrator 嵌入 Gateway 进程。
- [ ] 明确 `server/internal/orchestrator/` 是 Orchestrator 模块边界。
- [ ] 明确 Gateway 与 Orchestrator 可以同进程但不能合并职责。
- [ ] 明确 MVP v0.1 直接路由到 `code-agent`。
- [ ] 明确 MVP v0.1 不调用 LLM 生成 ExecutionPlan。
- [ ] 明确 MVP v0.1 只要求 `code` Artifact → `code_preview`。
- [ ] 明确 MVP v0.1 不强制群聊、多 Agent 编排、自建 Agent。
- [ ] 明确 MVP v0.1 可以使用 MySQL 8。
- [ ] 明确 MVP v0.1 可以使用固定 Token 或环境变量 Token。

---

## 4. PDR 完整目标检查

- [ ] 保留 Gateway / Orchestrator 清晰分层。
- [ ] 保留 Post-MVP 独立 Orchestrator Service 方向。
- [ ] 保留 internal HTTP / RPC contract 演进方向。
- [ ] 保留 ExecutionPlan、多 Agent、并行 / 串行编排扩展方向。
- [ ] 没有把 MVP 同进程模式写死为长期唯一架构。
- [ ] 明确 `/internal/*` 不暴露给 Frontend。
- [ ] 明确 internal contract 不写入 Frontend REST API OpenAPI。

---

## 5. Gateway 边界检查

Gateway 可以做：

- [ ] HTTP parse。
- [ ] auth。
- [ ] request validate。
- [ ] 保存用户消息。
- [ ] 查询历史消息。
- [ ] 构造 `OrchestratorRequest`。
- [ ] 创建 EventSink / EventChan。
- [ ] 调用 Orchestrator。
- [ ] 写出 SSE。
- [ ] 持久化 OrchestratorResult。

Gateway 不可以做：

- [ ] 意图编排。
- [ ] A2A Client 调用。
- [ ] A2A Stream 解析。
- [ ] ArtifactBuffer。
- [ ] Artifact → `code_preview` 映射。
- [ ] 多 Agent 调度。
- [ ] LLM 调用。
- [ ] 结果聚合。

---

## 6. Orchestrator 边界检查

Orchestrator 可以做：

- [ ] 接收 `OrchestratorRequest`。
- [ ] 选择目标 Agent。
- [ ] 调用 A2A Client。
- [ ] 接收 A2A Stream Event。
- [ ] 调用 ProtocolConverter。
- [ ] 输出 AG-UI Event 到 EventSink。
- [ ] 返回 `OrchestratorResult`。

Orchestrator 不可以做：

- [ ] 直接暴露给 Frontend。
- [ ] 直接写 SSE response。
- [ ] 直接处理用户鉴权。
- [ ] 直接保存用户消息。
- [ ] 直接查询会话列表。
- [ ] 直接操作 React UI。

---

## 7. Contract 结构检查

- [ ] 已定义 `OrchestratorRequest`。
- [ ] 已定义 EventSink / EventChan。
- [ ] 已定义 `OrchestratorResult`。
- [ ] 已定义 OrchestratorError / 错误码。
- [ ] 已定义 context cancellation。
- [ ] 已定义 timeout。
- [ ] 已定义 `traceId`、`requestId`、`runId`、`threadId`。
- [ ] 已定义 A2A 调用边界。
- [ ] 已定义 ProtocolConverter 边界。
- [ ] 已定义 MVP 直接路由规则。
- [ ] 已定义 Post-MVP 编排扩展规则。
- [ ] 已定义 Mock-first 规则。
- [ ] 已定义 Contract Test 规则。
- [ ] 已定义安全规则。

---

## 8. 事件输出检查

- [ ] Orchestrator 输出事件符合 `agui-event-contract`。
- [ ] Gateway 只负责 SSE 包装。
- [ ] A2A `status: working` → `TEXT_MESSAGE_START`。
- [ ] A2A `text` → `TEXT_MESSAGE_CONTENT`。
- [ ] A2A `artifact` → 缓存。
- [ ] A2A `completed` → `TEXT_MESSAGE_END` → flush artifacts → `TOOL_CALL_*` → `RUN_FINISHED`。
- [ ] A2A `failed` → `RUN_ERROR`。
- [ ] `code` Artifact → `code_preview`。
- [ ] `TEXT_MESSAGE_CONTENT` 只传文本 chunk。
- [ ] `TOOL_CALL_ARGS` 可按 `toolCallId` 聚合。
- [ ] `RUN_FINISHED` 后不继续输出内容事件。

---

## 9. 安全与可观测性检查

- [ ] 日志包含 `traceId`。
- [ ] 日志包含 `runId`。
- [ ] 日志包含 `threadId`。
- [ ] 不打印 Authorization token。
- [ ] 不打印 API key。
- [ ] 不泄漏内部堆栈给 Frontend。
- [ ] Orchestrator 不直接信任 Frontend 传来的 AgentName。
- [ ] Tools / Skills 经过白名单校验。
- [ ] Post-MVP internal endpoint 有服务间鉴权。

---

## 10. 阶段越界检查

- [ ] 当前仅生成 Contract 文档，不生成 Go 实现。
- [ ] 未生成 Gateway handler。
- [ ] 未生成 Orchestrator 实现。
- [ ] 未生成 A2A Client。
- [ ] 未生成数据库模型。
- [ ] 未生成 Docker Compose。
- [ ] 未进入下一个 Skill。


## 11. v1.1 对齐补充检查项

- [ ] 是否区分 MVP AG-UI-compatible event 简化与 v1.1 OrchestratorEvent 长期模型？
- [ ] Gateway 是否负责 OrchestratorEvent -> AG-UI Event 映射？
- [ ] Orchestrator 是否没有直接暴露给 Frontend？
- [ ] 是否补充 /internal/runs* 推荐路径？
- [ ] 是否说明 /internal/orchestrator/runs 只是兼容或备选路径？
- [ ] 是否说明 approval.required 与 security / frontend-runtime-skills / data-persistence 的关系？

