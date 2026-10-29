# Redis Usage Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：从“Post-MVP 可选”提升为辅助能力，仍非权威）

## 1. 文档目的

定义 AgentHub 中 Redis 的使用边界。Redis 是缓存、短期事件副本和心跳工具，**不是持久化事实源**。

## 2. 目标状态

目标启动图同时使用 MySQL（权威）与 Redis（辅助）：

```text
SQL commit → outbox → Redis Streams（短期副本） → Gateway 投影 → 浏览器 AG-UI SSE
```

- Redis 不可用时，核心闭环仍能通过 SQL 有界补读维持，但性能可降级。
- SQL 不可用时，停止接受需要记账的新副作用；不得绕过账本继续产生副作用。

## 3. 允许用途

- 带版本的能力缓存与 Memory 召回缓存（键含 scope/epoch/revision）。
- 运行事件的短期流副本（有长度/过期边界）。
- 实例心跳、租约辅助发现。
- 可丢失的唤醒提示（Pub/Sub）。

## 4. 禁止用途

- 消息历史、Artifact、用户数据、AgentCard、run 结果、审批记录、工具账本、终态、安全日志的唯一存储。
- 用 TTL 提供删除/遗忘正确性。
- 保存 API key、token 明文、完整敏感 prompt、DSN、原始凭据。
- 把多个浏览器订阅者放进同一 `XREADGROUP` 分摊事件。

## 5. Key 命名与版本

```text
agenthub:{env}:{workspaceId}:run:{runId}:events
agenthub:{env}:{workspaceId}:run:{runId}:state
agenthub:{env}:{workspaceId}:memory:{scopeId}:{epoch}
agenthub:{env}:catalog:{revision}
agenthub:{env}:instance:{id}:heartbeat
```

- 缓存键必须携带作用域与版本；Memory 缓存键还须包含 `epoch` 与 `revision`。
- 权威判断（权限、审批、终态、记忆 active 状态）每次从 SQL 复核。

## 6. TTL / 裁剪规则

- 缓存 TTL 上限 5 分钟，且不超过来源剩余有效期。
- 运行事件 Stream 设置明确 `MAXLEN`/过期；可靠补发来自 SQL journal。
- 禁止无 TTL 的业务 key；审计/终态不依赖 Redis 保留期。

## 7. 故障规则

- Redis 重启/清空后：SQL 仍能恢复审批、终态和允许窗口内的事件。
- 缓存失效消息丢失时：下一次新授权仍核验 SQL 权威状态，不能注入旧版本。
- Redis 降级需压测，不能承诺同等性能。

## 8. 安全规则

- 连接信息来自环境变量；不写入代码或文档。
- 不保存密钥/完整 prompt。
- 多实例共享同一 Redis 时按 env/workspace 隔离键空间。
