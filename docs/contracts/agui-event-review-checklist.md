# AG-UI Event Review Checklist

本文档用于检查 `docs/contracts/agui-events.md`、`docs/contracts/agui-events.schema.json` 以及后续 AG-UI 相关实现是否符合 AgentHub 的 PDR、MVP v0.1、UML 和项目级 Skills 约束。

---

## 1. MVP 必须事件链检查

- [ ] 是否支持 `RUN_STARTED`？
- [ ] 是否支持 `TEXT_MESSAGE_START`？
- [ ] 是否支持 `TEXT_MESSAGE_CONTENT`？
- [ ] 是否支持 `TEXT_MESSAGE_END`？
- [ ] 是否支持 `TOOL_CALL_START`？
- [ ] 是否支持 `TOOL_CALL_ARGS`？
- [ ] 是否支持 `TOOL_CALL_END`？
- [ ] 是否支持 `RUN_FINISHED`？
- [ ] 是否支持 `RUN_ERROR`？
- [ ] 是否保留 `STATE_UPDATE` schema？

MVP v0.1 成功链路应为：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START?
TOOL_CALL_ARGS*
TOOL_CALL_END?
RUN_FINISHED
```

---

## 2. 协议边界检查

- [ ] 是否明确 AG-UI 只负责 Frontend ↔ Gateway 实时事件？
- [ ] 是否没有把 AG-UI 事件写成普通 REST response？
- [ ] 是否没有把 REST API response schema 混进 AG-UI Event Contract？
- [ ] 是否没有把 A2A event 暴露给 Frontend？
- [ ] 是否没有把 A2A endpoint 暴露为 Frontend API？
- [ ] 是否没有把 Gateway ↔ Orchestrator 内部协议混入 AG-UI？
- [ ] 是否没有修改 `docs/contracts/openapi.yaml` 的职责边界？

---

## 3. A2A → AG-UI 映射检查

- [ ] A2A `status:working` 是否转换为 `TEXT_MESSAGE_START`？
- [ ] A2A `text` 是否转换为 `TEXT_MESSAGE_CONTENT`？
- [ ] A2A `artifact` 是否进入 `artifactBuffer`，而不是立即输出到前端？
- [ ] A2A `status:completed` 是否触发 `TEXT_MESSAGE_END`？
- [ ] `TEXT_MESSAGE_END` 后是否 flush artifacts？
- [ ] Artifact 是否转换为 `TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END`？
- [ ] Tool Call 完成后是否输出 `RUN_FINISHED`？
- [ ] A2A `status:failed` 是否转换为 `RUN_ERROR`？
- [ ] A2A 原始事件是否没有直接透传到 Frontend？

---

## 4. Artifact 与 Frontend Skill 检查

- [ ] 是否没有把大产物塞进 `TEXT_MESSAGE_CONTENT`？
- [ ] `TEXT_MESSAGE_CONTENT` 是否只传文本 chunk？
- [ ] 是否明确 Artifact 必须通过 Tool Call 映射到 Frontend Skill？
- [ ] `code` Artifact 是否映射为 `code_preview`？
- [ ] 未注册 Frontend Skill 的 Artifact 是否不会被强行触发？
- [ ] 大型 Artifact 是否预留给 Artifact Contract / fileUrl 处理？

---

## 5. code_preview 参数检查

- [ ] `code_preview` args 是否包含 `code`？
- [ ] `code_preview` args 是否包含 `language`？
- [ ] `code_preview` args 是否包含 `filename`？
- [ ] `code` 是否来自 `artifact.content`？
- [ ] `language` 是否来自 `artifact.metadata.language`？
- [ ] `filename` 是否来自 `artifact.title`？
- [ ] `code` 为空时是否不会触发 `code_preview`？
- [ ] `language` 缺失时是否有合理兜底？
- [ ] `filename` 缺失时是否有合理兜底？

---

## 6. Tool Call 顺序检查

- [ ] `TOOL_CALL_ARGS` 是否不会早于 `TOOL_CALL_START`？
- [ ] `TOOL_CALL_END` 是否不会早于 `TOOL_CALL_START`？
- [ ] `TOOL_CALL_ARGS` 是否都携带 `toolCallId`？
- [ ] `TOOL_CALL_END` 是否携带 `toolCallId`？
- [ ] `TOOL_CALL_ARGS` 是否允许分片？
- [ ] Frontend 是否按 `toolCallId` 聚合参数？
- [ ] Frontend 是否在 `TOOL_CALL_END` 后再解析 args JSON？
- [ ] `RUN_FINISHED` 是否不会早于未完成的 Tool Call？

---

## 7. Text Message 顺序检查

- [ ] `TEXT_MESSAGE_CONTENT` 是否不会早于 `TEXT_MESSAGE_START`？
- [ ] `TEXT_MESSAGE_END` 是否不会早于 `TEXT_MESSAGE_START`？
- [ ] `TEXT_MESSAGE_CONTENT` 是否都携带 `messageId`？
- [ ] `TEXT_MESSAGE_END` 是否携带 `messageId`？
- [ ] Frontend 是否按 `messageId` 聚合文本？
- [ ] Markdown 是否建议在 `TEXT_MESSAGE_END` 后再统一解析？

---

## 8. Schema 检查

- [ ] `agui-events.schema.json` 是否是合法 JSON？
- [ ] 是否使用 JSON Schema draft 2020-12？
- [ ] 是否包含 `AGUIEvent`？
- [ ] 是否包含 `RunStartedEvent`？
- [ ] 是否包含 `RunFinishedEvent`？
- [ ] 是否包含 `RunErrorEvent`？
- [ ] 是否包含 `TextMessageStartEvent`？
- [ ] 是否包含 `TextMessageContentEvent`？
- [ ] 是否包含 `TextMessageEndEvent`？
- [ ] 是否包含 `ToolCallStartEvent`？
- [ ] 是否包含 `ToolCallArgsEvent`？
- [ ] 是否包含 `ToolCallEndEvent`？
- [ ] 是否包含 `StateUpdateEvent`？
- [ ] 是否包含 `CodePreviewArgs`？
- [ ] 所有事件是否都有 `type`？
- [ ] 所有字段是否使用 camelCase？
- [ ] 是否没有 snake_case 字段？
- [ ] 是否可以用 schema 校验每个 AG-UI event？

---

## 9. 错误处理检查

- [ ] 是否支持 `RUN_ERROR`？
- [ ] SSE 建立后是否用 `RUN_ERROR` 表达运行失败？
- [ ] `RUN_ERROR` 后是否不会继续输出 `RUN_FINISHED`？
- [ ] 错误事件是否不泄漏 Token、LLM API Key、数据库连接串？
- [ ] A2A 连接失败是否能映射为 `RUN_ERROR`？
- [ ] Child Agent 失败是否能映射为 `RUN_ERROR`？
- [ ] Tool Call 参数构造失败是否能映射为 `RUN_ERROR` 或被安全跳过？

---

## 10. Frontend 处理检查

- [ ] Frontend 是否通过 SSE 读取 `data:` 行？
- [ ] Frontend 是否根据 `type` 分发事件？
- [ ] Frontend 是否按 `messageId` 管理流式消息？
- [ ] Frontend 是否按 `toolCallId` 管理 Tool Call？
- [ ] Frontend 是否在 `TEXT_MESSAGE_START` 时创建空 Agent 气泡？
- [ ] Frontend 是否在 `TEXT_MESSAGE_CONTENT` 时追加文本？
- [ ] Frontend 是否在 `TEXT_MESSAGE_END` 时完成消息？
- [ ] Frontend 是否在 `TOOL_CALL_END` 时执行 `code_preview`？
- [ ] Frontend 是否在 `RUN_FINISHED` 时关闭 loading？
- [ ] Frontend 是否在 `RUN_ERROR` 时显示错误状态？

---

## 11. Gateway / Orchestrator 检查

- [ ] Gateway 是否设置 `Content-Type: text/event-stream`？
- [ ] Gateway 是否每个事件单独 flush？
- [ ] Gateway 是否不输出 A2A 原始事件？
- [ ] Gateway Handler 是否不承担复杂协议转换？
- [ ] Orchestrator / Converter 是否维护 `artifactBuffer`？
- [ ] Orchestrator / Converter 是否维护 `messageId`？
- [ ] Orchestrator / Converter 是否在 completed 后 flush artifacts？
- [ ] Orchestrator / Converter 是否最后输出 `RUN_FINISHED`？

---

## 12. MVP / UML 一致性检查

- [ ] 是否符合 MVP v0.1 的 `/api/agui/run` SSE 事件流？
- [ ] 是否支持 `code-agent` 生成文本和 `code` Artifact？
- [ ] 是否支持 `code` Artifact 转 `code_preview`？
- [ ] 是否符合 UML 中 A2A → AG-UI 协议转换时序？
- [ ] 是否符合 UML 中前端流式渲染和 Tool Call 聚合时序？
- [ ] 是否没有提前要求复杂 `STATE_UPDATE`、多 Agent 编排或群聊事件？


## 13. v1.1 对齐补充检查项

- [ ] 是否统一使用 Frontend Runtime Skills / frontend-runtime-skills-contract 术语？
- [ ] 是否明确 v1.1 RUN_STARTED 映射与 MVP TEXT_MESSAGE_START 映射可以共存？
- [ ] RUN_ERROR 是否不泄漏 stack trace、token、API key、内部地址、完整 system prompt？
- [ ] 大 Artifact 是否没有塞进 TEXT_MESSAGE_CONTENT？

