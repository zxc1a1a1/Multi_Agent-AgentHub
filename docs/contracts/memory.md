# Memory Contract

**Status:** Active (v2, P0 frozen)
**Owner:** AgentHub Orchestrator `MemoryController`
**Last updated:** 2026-10-29

## 1. 分层

| 层次 | 内容与范围 | 存储 |
|---|---|---|
| 工作记忆 | 当前 run 目标、工具 observation、临时计划 | 进程内；安全边界写 checkpoint；不自动转长期 |
| 短期会话记忆 | 同一授权 thread/Agent 分支的消息与上下文摘要 | MySQL `session_events` + 版本化 `session_context_snapshots` |
| 执行 checkpoint | 游标、pending calls、预算、权限/记忆版本 | MySQL；不当作事实注入模型 |
| 长期记忆 | 已确认偏好、项目事实、带来源观察 | MySQL `memory_entries/revisions/scopes/mutations` |
| 缓存 | 已授权条目/上下文窗口加速副本 | Redis，键含 scope/epoch/revision，短 TTL |

`MemoryStore`（Gateway 进程内存储）不是本契约的 MemoryController，也不是长期记忆。

## 2. MemoryScope（冻结）

```text
env + workspaceId + principalId（可信身份上下文生成）
可选收窄：sessionId / agentName / threadId
```

- 服务端生成范围，不接受模型或浏览器 `user_id` 自行扩大。
- 固定 Token 无法区分真实用户时，不宣称个人隔离；个人长期记忆必须先有可验证主体映射。
- 项目共享范围需单独授权；默认不跨项目继承“全局用户记忆”。
- 相同关键词不能绕过 scope 过滤；检索前先按 scope/status/expiry 过滤。

## 3. MemoryPolicy 模式（冻结）

| 模式 | 读写行为 |
|---|---|
| `off` | 不复用过往会话/长期记忆；当前请求、必需系统约束、当前 run 工具结果仍可用 |
| `session_only`（**默认**） | 只使用当前授权 thread 历史/摘要；不跨会话召回、不写长期 |
| `scoped_long_term` | 在允许范围读取 active 记忆；新建/更新先提候选，按策略确认 |

- 读、写提议、候选确认、修改、遗忘分别授权，不用一个 enabled 开关包办。
- 模型不能修改 MemoryPolicy；由可信配置/用户控制 API 生成并版本化。
- 用户偏好不等于系统政策；召回记忆不得覆盖安全规则、审批策略、工具权限或预算。
- 敏感信息、凭据、模型隐藏思维链禁止入库；用户确认不能绕过密钥禁令。

## 4. 长期记忆条目（冻结）

```text
memoryId, scopeId, kind, logicalKey, content, contentHash,
sourceRefs[], revision, status(active|candidate|revoked|expired),
createdAt, verifiedAt, expiresAt, lastUsedAt
```

- 每个范围最多 200 条 active；单条正文最多 4 KiB。
- 候选 24 小时过期；未指定期限的长期条目默认 30 天后需复核。
- 到期/复核是策略状态，不自动删除原始聊天。
- 查看/命中记忆不自动续期。

## 5. 召回（冻结）

```text
模型调用前有界自动预召回 / 模型 memory.search、memory.read
→ 身份与范围过滤（先授权再检索）
→ SQL 核验 policy epoch、active 状态、有效期、条目 revision
→ 有界结构化/全文检索 → 排序合并去重 (memoryId, revision) → token 限额
```

- 每轮最终注入最多 5 条（固定核心最多 2 条），最多占可用输入预算 10% 且不超过 2,048 tokens。
- 自动预召回与显式工具召回共用同一管线与预算，不重复附加。
- 初版只做结构化字段 + MySQL 全文检索，不宣称 embedding 语义检索。
- SQL 不可用时省略长期记忆或明确阻止依赖它的任务，不读旧缓存凑答案。
- 条目/摘要/checkpoint/检索结果携带 `memoryId/revision/source` lineage；删除或更新后阻止旧派生片段再次注入。

## 6. 写入 / 更新（冻结）

```text
用户明确请求记住 / 工具观察候选
→ memory.propose（不立即生效）
→ schema、敏感性、来源、作用域、重复/冲突检查
→ 需要时用户确认精确内容与范围
→ 同事务校验 expectedRevision/权限/审批、幂等写入、scope epoch++、outbox
→ active 新版本 → 下一次授权召回可见
```

- 候选包含拟新增/替代内容、来源 message/artifact、时间、expectedRevision。
- 模型自评置信度不是确认依据。
- 同一 `logicalKey` 唯一约束 + CAS 更新；两个并发更新使用同一 expectedRevision 时只有一个成功，另一个返回冲突，不覆盖新事实。
- 失败/回滚不增加可见版本；候选/accepted 状态不能被 search 或下一轮模型读到。
- `memory.propose` 由 Tool Broker 管理；模型不能执行 approve、无审批 update/delete 或修改策略。
- 确认绑定候选、目标 revision、内容 hash、toolCallId 与范围。
- 数据库提交后才报告“已记住”。

## 7. 遗忘与清除（冻结）

1. 遗忘事务将条目标记 `revoked`/tombstone，`scope epoch++`，失效相关候选，产生仅含 ID/版本的清理事件。
2. 返回 `recallDisabled=true` 与独立 `purgeStatus`（`pending | in_progress | completed | failed`），不把异步物理清除写成已完成。
3. 新授权以 SQL 校验/登记的事务时点为界；已发送给外部模型的在途调用不能撤回，须记录在途边界并在下一轮重新组装。
4. 旧 Redis 值、旧摘要、旧 checkpoint、延迟候选、重复事件都不得重新激活记忆。
5. “忘记长期记忆” ≠ “删除原始聊天及全部来源”；UI 必须区分并显示来源保留提示。
6. 删除长期记忆不冒充删除聊天/第三方/备份；备份恢复后先重放删除标记再开放召回。

## 8. API（P0 冻结操作面）

Gateway 代理到 Orchestrator MemoryController 的内部 API：

```text
GET    /internal/memory/entries?scope=&status=&limit=
POST   /internal/memory/propose
POST   /internal/memory/confirm
POST   /internal/memory/update
POST   /internal/memory/forget          # 单条或范围
GET    /internal/memory/policy
PUT    /internal/memory/policy
GET    /internal/memory/mutations/{id}  # purgeStatus
```

- 所有操作携带请求主体上下文；Gateway 不做记忆内容判断。
- 错误不得显示成成功；`purgeStatus` 与 `recallDisabled` 分开返回。
- 公开 API 与前端交互在 P4 接入，P3c 只完成 Controller/Repository/管理操作。

## 9. 测试门禁

- 同会话可恢复；不同主体/工作区/Agent 私有分支不能串读。
- session_only 不跨会话召回；scoped_long_term 只召回同授权范围 active；off 不复用旧记忆。
- 候选不可被 search/模型读取；确认后 SQL 真值立即可读，不等待 Redis TTL。
- CAS 冲突、过期、权限收窄、删除后旧缓存/旧 checkpoint/重复事件均不得复活记忆。
- 摘要/抽取计入总模型预算；注入条数/token 限制可实际触发。
