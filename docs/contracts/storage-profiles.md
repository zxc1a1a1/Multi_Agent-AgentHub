# Storage Profiles 契约

## Current: MySQL 8

MySQL 8 是当前 v1.0 关系型事实源。

用于：

- conversations
- messages
- agents
- runs
- tasks
- tool calls
- artifacts metadata

## Planned: PostgreSQL

PostgreSQL 是长期可选演进方向。

规则：

- 不得影响当前 MySQL 落地。
- JSONB 不能替代必要关系建模。
- 高频 JSON 查询必须设计索引。

## Cache: Redis

Redis 只用于缓存和短期状态。

不得作为：

- 消息事实源。
- Run 事实源。
- Artifact 事实源。
- Agent Registry 事实源。

## Large Object: Object Storage

Object Storage 用于大型产物。

数据库保存：

- content_ref
- metadata
- ownership
- trace id
- permission state

禁止：

- 永久公开 URL。
- 未授权直连下载。
- 只靠路径表达归属。
