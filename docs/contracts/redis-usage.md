# Redis Usage Contract

## 1. 文档目的

本文档定义 AgentHub 中 Redis 的使用边界。

Redis 是缓存、临时状态、轻量队列和 rate limit 工具，不是持久化事实源。

## 2. MVP v0.1 状态

MVP v0.1 暂不强制使用 Redis。

不使用 Redis 时，不得影响核心闭环：

```text
用户发消息 → Gateway → Orchestrator → code-agent → AG-UI → code_preview
```

## 3. Post-MVP 可使用场景

Redis 可用于：

- 在线状态。
- 轻量缓存。
- rate limit。
- session 临时状态。
- run 临时状态。
- SSE fanout 辅助。
- 轻量队列。
- 分布式锁。
- Agent health check 缓存。

## 4. 禁止用途

Redis 不允许作为：

- 消息历史唯一存储。
- Artifact 唯一存储。
- 用户数据唯一存储。
- AgentCard 唯一存储。
- Run 结果唯一存储。
- 审批记录唯一存储。
- 可审计安全日志唯一存储。

## 5. Key 命名建议

```text
agenthub:session:{sessionId}
agenthub:run:{runId}:state
agenthub:rate-limit:{userId}
agenthub:agent:{agentName}:health
agenthub:sse:{runId}
```

## 6. TTL 规则

- 临时状态必须设置 TTL。
- rate limit key 必须设置 TTL。
- health cache 必须设置 TTL。
- 不得用无 TTL Redis key 保存长期业务数据。

## 7. 安全规则

- Redis 不保存 API key。
- Redis 不保存用户 token 明文。
- Redis 不保存完整敏感 prompt。
- Redis 连接信息来自环境变量。
