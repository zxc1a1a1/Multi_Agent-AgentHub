# Artifact Lifecycle 规则

## 1. 目的

本文定义 Artifact 从 A2A 输出到前端预览的生命周期。

## 2. 长期生命周期

```text
1. Child Agent 产出 A2A Artifact。
2. Orchestrator / Converter 接收 A2A Artifact。
3. normalize 为 AgentHub Artifact。
4. 分配 artifactId。
5. 关联 conversationId / messageId / runId / a2aTaskId / stepId。
6. 判断 content 或 contentRef。
7. 持久化 metadata。
8. 持久化 content 或 object key。
9. 根据 artifact.type 映射 Frontend Runtime Skill。
10. 发送 AG-UI artifact summary / Tool Call。
11. 前端按需通过 REST 查询完整内容。
12. 前端 Runtime Skill 渲染。
13. Artifact 状态更新为 previewed。
```

## 3. MVP 生命周期

MVP 阶段允许简化为：

```text
1. A2A artifact received。
2. ArtifactBuffer 缓存 artifact。
3. 文本流继续输出。
4. A2A completed。
5. 发送 TEXT_MESSAGE_END。
6. flush ArtifactBuffer。
7. code → code_preview。
8. 发送 TOOL_CALL_START。
9. 发送 TOOL_CALL_ARGS {code, language, filename}。
10. 发送 TOOL_CALL_END。
11. 发送 RUN_FINISHED。
12. 将 artifacts JSON 临时保存到 messages。
```

## 4. 状态

推荐状态：

```text
created
normalized
persisted
preview_mapped
previewed
failed
```

## 5. 错误处理

Artifact 生命周期任一阶段失败时，必须：

- 保留 errorCode。
- 保留 artifactId 或临时 ID。
- 保留 runId / messageId。
- 不让主文本流崩溃。
- 不把内部错误暴露给用户。

## 6. 禁止事项

不得：

- A2A artifact 直接不校验转给前端。
- ArtifactBuffer 丢失 completed 前的产物。
- completed 后忘记 flush。
- Artifact 失败后没有可追踪 ID。
- preview 失败导致整个 run 无法结束。
