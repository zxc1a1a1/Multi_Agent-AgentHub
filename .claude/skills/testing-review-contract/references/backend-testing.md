# 后端测试策略

## 1. 目的

本文定义 AgentHub Go 后端测试重点。

## 2. 必测模块

后端必须测试：

```text
Gateway handler
Orchestrator direct routing
Agent Registry lookup
A2A client mock
A2A stream converter
Artifact buffer
mapArtifactToSkill
buildToolArgs
message persistence
auth middleware
error normalization
```

## 3. MVP 必测路径

MVP 至少测试：

- 保存用户消息。
- 根据 req.AgentName 或 conversation.agentName 路由到 code-agent。
- A2A text 转 AG-UI text。
- A2A artifact 进入 buffer。
- completed 后 flush artifact 为 code_preview Tool Call。
- agent not found 返回 RUN_ERROR。
- A2A stream failed 返回 RUN_ERROR。
- DB 写入失败返回安全错误。

## 4. Mock / Fake

后端测试默认使用：

```text
fake LLM provider
mock A2A agent
test database
fixture event stream
```

不得在单元测试中调用真实 LLM。

## 5. 禁止事项

不得：

- 让测试依赖生产服务。
- 在测试中使用真实 API key。
- 只测 handler happy path。
- 忽略 context cancellation。
