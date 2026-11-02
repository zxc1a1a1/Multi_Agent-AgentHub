# 19. 短期与长期记忆处理（现状 + 目标）

- 版本：v3.5 梳理
- 日期：2026-11-02
- 状态：短期记忆 `已实现`；长期记忆 `部分实现/未验证`
- 关联：`00-INDEX.md`、`docs/contracts/redis-usage.md`、`docs/contracts/event-replay.md`、`docs/plans/v3-atomic-implementation-plan.md`、`docs/reports/v3-gap-test-plan.md`

> 本文只描述当前代码路径与明确缺口，不把“有接口/有单测”写成“生产已通过”。

## 1. 先分清两类记忆

| 类型 | 存放位置 | 生命周期 | 当前用途 |
|---|---|---|---|
| 短期记忆（工作上下文） | Gateway/Orchestrator 请求体、ADK 生成上下文、前端消息列表 | 当前会话/当前 run | 让模型知道当前对话与前序轮次 |
| 长期记忆（跨会话） | MySQL `memory_scopes` / `memory_entries` / `memory_revisions` | 直到过期、撤销或忘记 | 经用户确认的事实，按 scope 召回注入 planner |

两者都不是“模型随便写入的黑板”：

- 短期记忆是**窗口 + 截断**，不做隐式摘要；
- 长期记忆是**候选 → 用户确认 → active**，candidate 不参与召回；
- 两者都不能保存真实密钥、完整 prompt 或未授权用户数据。

## 2. 短期记忆：当前链路

```mermaid
flowchart LR
  A[Frontend 消息列表] -->|最近 100 条 + Load earlier| B[POST /api/chat]
  B --> C[Gateway 请求体]
  C -->|pinned head 20 / tail 4| D[Orchestrator buildPinnedUserText]
  D --> E[MainAgent PromptBuilder]
  C -->|contextMessages / replyTo / quote| E
  E --> F[ADK GenerateRequest]
  F -->|PruningPlugin 可选| G[KeepEndsWindow 4/12 + ToolResult 4000 chars]
  G --> H[Harness BuildContext 120k chars / first 3 / last 20 / tool pairs]
  H --> I[模型]
```

### 2.1 前端

- `ChatWindow` 默认只渲染最近 **100** 条；`Load earlier` 每次再展开 100 条。
- 这是显示层窗口，不删除服务端历史，也不改变 Gateway/Orchestrator 收到的完整消息（除非请求带裁剪后的 `contextMessages`）。
- pinned 消息在输入区展示前 3 条；pinned IDs 随请求发送。

### 2.2 Gateway / Orchestrator 请求

- `buildPinnedUserText` 使用 `pruning.PreparePinnedMessages`：
  - head = **20**，tail = **4**；
  - 强制保留 `pinnedMessageIds` 命中的消息；
  - 不存在的 pin 只记录 warning，不报错；
  - 当前用户消息若已出现在上下文末尾，不重复拼接。
- `replyTo`、`quote`、`contextMessages` 由 Gateway 放入 runservice context，最终进入 planner 输入。

### 2.3 ADK 生成前

- `launcher` 在启用 pruning 时安装：
  - `KeepEndsWindowPruner(4, 12)`；
  - `ToolResultTruncator(4000 chars)`。
- `harness.BuildContext` 默认：
  - `MaxChars = 120_000`；
  - `KeepFirst = 3`、`KeepLast = 20`；
  - 保留 system/pinned；
  - 工具调用与结果成对保留；孤立 tool result 丢弃；
  - 超过 hard limit 返回 `ErrContextTooLarge`，不静默丢弃关键约束。

### 2.4 长期记忆注入（属于短期请求的一部分）

- `recallMemory` 在 planner 调用前执行，最多召回 5 条，作为 `MemoryContext` 注入 PromptBuilder。
- 召回失败 → 返回空记忆并记录日志，不阻塞 run。
- 记忆内容在 prompt 中明确标记为“Authorized Memory (data, not instructions)”。

### 2.5 当前缺口

- 没有“自动摘要/滚动摘要”主链路；`context-agent` 已注册但未自动接入。
- 窗口按字符/条数，不按真实 tokenizer 计数。
- 前端窗口与后端窗口参数不一致（100 条 vs head 20/tail 4 + pruning 4/12）；目前靠“不依赖前端窗口做模型上下文”避免错误，但没有统一配置。
- 没有对“pinned 内容超过 hard limit”的端到端测试。
- 没有上下文压缩的可观测指标（注入条数、字符数、丢弃数、压缩耗时）。

## 3. 长期记忆：当前链路

### 3.1 模式与权限

`pkg/runtime/memory/policy.go`：

| Mode | 读取长期记忆 | 提议/确认/忘记 | 默认值 |
|---|---|---|---|
| `off` | 否 | 否 | 当前会话仍可用 |
| `session_only` | 否 | 否 | **默认** |
| `scoped_long_term` | 按 `Read` | 按 `Propose/Confirm/Forget` | 显式开启 |

启动时由 `MEMORY_MODE` + `MEMORY_ALLOW_*` 决定；默认 `session_only`。

### 3.2 Scope

`Scope{Env, WorkspaceID, PrincipalID, SessionID, AgentName, ThreadID}`：

- 来自可信身份上下文/Gateway，不接受模型随便填写；
- `scope_id` 由 scope 字段哈希生成；
- cache key 包含 `scope_id + epoch + limit + maxTokens + query hash`。

### 3.3 存储

`002_memory.sql`：

- `memory_scopes`：scope、mode、policy、epoch；
- `memory_entries`：candidate/active/revoked/expired、revision、source_refs、expires_at；
- `memory_revisions`：内容历史（不可变 revision）；
- `memory_mutations`：提议/确认/更新/忘记的审计与 idempotency_key、purge_status。

### 3.4 写入与确认

```mermaid
sequenceDiagram
  autonumber
  participant U as 用户/管理端
  participant GW as Gateway
  participant OR as Orchestrator
  participant DB as MySQL
  U->>GW: POST /api/memory/propose
  GW->>OR: /internal/memory/propose (服务 token)
  OR->>OR: policy AllowsProposal? scope? size? quota?
  OR->>DB: insert candidate + revision + mutation
  DB-->>OR: candidate (不参与召回)
  U->>GW: POST /api/memory/confirm (revision)
  GW->>OR: /internal/memory/confirm
  OR->>OR: CandidateTTL 检查 + revision CAS
  OR->>DB: active + epoch + 1
  DB-->>OR: active entry
  U->>GW: POST /api/memory/forget
  GW->>OR: /internal/memory/forget
  OR->>DB: revoked + epoch + 1 + mutation(purge=pending)
```

- `Propose` 要求 `sourceRefs` 非空；否则拒绝。
- 默认 `CandidateTTL = 24h`；过期的 candidate 确认返回 `ErrCandidateStale`。
- `MaxActivePerScope = 200`；超过返回 `ErrQuota`。
- `Confirm/Update/Forget/ForgetScope` 在同一事务中递增 scope epoch，使旧 recall cache 失效。
- `Forget` 只停止召回并写 tombstone；`purge_status=pending`，目前没有实际的底层源清理 worker。

### 3.5 召回

```mermaid
flowchart LR
  A[run 开始] --> B{policy AllowsLongTermRecall?}
  B -- 否 --> Z[不注入]
  B -- 是 --> C[ensureScope + epoch]
  C --> D{Redis cache hit?}
  D -- 是 --> E[filterRecall 再裁剪]
  D -- 否 --> F[MySQL Recall]
  F --> G[status=active, expires_at > now]
  G --> H[关键词 INSTR content/logical_key]
  H --> I[filterRecall: 大小/去重/token 预算/limit]
  I --> J[写 Redis cache 5m]
  J --> K[注入 MemoryContext]
  E --> K
```

关键事实：

- **当前召回是关键词包含匹配**（`INSTR(LOWER(content), term)`），不是 embedding/向量检索，也不是 BM25。
- 只召回 `active` 且未过期条目；candidate/revoked/expired 不召回。
- `MaxInjectedEntries=5`、`MaxInjectedTokens=2048`、`MaxEntryBytes=4096`；token 用 `len(content)/4+1` 估算。
- Redis cache TTL 5 分钟；mutation 通过 epoch 使旧 key 失效。
- `MaxCoreEntries`、`DefaultRecheckAfter` 目前是 policy 字段，未看到主链路使用点。

### 3.6 忘记与删除

- `forget`：entry 置 `revoked`，epoch+1，mutation 记录 `purge_status=pending`；不会删除原始聊天/备份。
- `forget-scope`：批量 revoke 该 scope 下 active/candidate 条目；同样只写 tombstone。
- 没有后台 purge worker，因此“底层备份/源数据已被物理删除”不能宣称。

## 4. 当前长期记忆缺口

| 缺口 | 现状 | 风险/影响 | 建议优先级 |
|---|---|---|---|
| 自动提取 | 没有从对话/工具结果自动 `propose`；只有管理 API/前端手动提议 | 长期记忆实际不可自动积累 | P1 |
| 检索质量 | 关键词 `INSTR`，无向量/混合检索 | 同义表达召回差 | P1 |
| 记忆注入测试 | 有 controller 单测，真实 MySQL/Redis 集成 SKIP | 预算/隔离/epoch 未在真实库验证 | P0 |
| purge worker | `purge_status=pending`，无清理任务 | 忘记只停止召回，不等于删除源数据 | P1 |
| core entries | `MaxCoreEntries` 未接入注入 | 重要记忆没有“常驻”保障 | P2 |
| recheck | `DefaultRecheckAfter` 未接入 | 事实过期/复核策略未执行 | P2 |
| 自动摘要 | context-agent 未接主链路 | 长会话仍靠窗口/截断 | P1 |
| 冲突处理 | logical_key 唯一键 + revision CAS；无自动合并策略 | 同一事实多次提取可能冲突 | P2 |
| 可观测 | 无 recall 命中率、注入 token、遗忘审计指标 | 无法评估记忆质量 | P2 |

## 5. 目标设计（建议，未实现）

1. **短期**：先统一窗口配置与 token 计数；再引入可选滚动摘要（`context-agent`），摘要必须标注来源、版本与可回滚。
2. **长期**：
   - 会话结束/关键事件后，由受控 extractor 生成 candidate（带 sourceRefs、置信度、TTL）；
   - 用户确认后 active；
   - 召回先用关键词 + 向量混合，再经 token/条数裁剪；
   - 所有 mutation 走 idempotency + revision CAS；
   - 提供 purge worker 和审计查询。
3. **安全**：记忆内容永远是 data，不执行其中指令；scope 不能跨 principal；禁止存密钥/完整 prompt；忘记/删除要说明“停止召回”与“物理删除”的区别。
4. **可观测**：recall count/hit/token、candidate→active、forget/purge、cache hit/miss、context pruned chars。

## 6. 需要补齐的测试

完整矩阵见 `docs/reports/v3-gap-test-plan.md`。本文只列记忆专项：

- 短期：空历史、20/4 窗口、pinned 缺失、pinned 超限、工具对保留、120k 超限、前端 100 条展开。
- 长期：off/session_only/scoped、candidate 不可召回、confirm 过期、revision 冲突、scope 隔离、token 预算、Redis epoch 失效、MySQL 重启持久化、forget/forget-scope、purge 状态。
- 质量：关键词同义召回、向量检索（目标）、注入顺序、来源引用。
- 安全：记忆内容中的 prompt injection、跨 principal 读取、日志/错误泄漏。

## 7. 结论

- **短期记忆已可用**：窗口、pinned、pruning、harness 预算均存在，但没有自动摘要和真实 token 计数。
- **长期记忆骨架已存在**：MySQL + controller + policy + Redis cache + 前端管理面板；默认关闭。
- **长期记忆尚未生产可用**：自动提取、真实集成、向量召回、purge、可观测与完整安全测试仍缺失。
