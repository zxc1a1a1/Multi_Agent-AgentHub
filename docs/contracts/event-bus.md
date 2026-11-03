# EventBus 契约（EventBus Contract）

**status:** `draft-frozen`（契约内容已冻结，实现尚未开始）
**schemaVersion:** `1`
**Owner:** AgentHub Orchestrator / Gateway
**Last updated:** 2026-11-02
**Plan task:** R2-01「冻结 EventBus 契约」（`docs/plans/v3-gap-atomic-plan.md` §4.3）
**对应缺口:** GAP-10 / P7-MQ-01（`docs/architecture/v3/18-message-queue-and-rich-interactions.md`）
**Schema:** `docs/contracts/event-bus.schema.json`（JSON Schema draft 2020-12）
**示例:** `docs/contracts/testdata/event-bus/`（14 个正向示例 + 7 个反向示例，已实跑校验）
**证据:** `docs/reports/evidence/2026-11-02/R2-01/`

---

## 0. 阅读约定：本文不是实现声明

本文件遵循 `docs/architecture/v3/00-INDEX.md` §0.1 的事实等级约定。**除第 9 节「现状」列与第 10 节明确标注为已实现的内容外，本文全部条目等级为 `未实现`**，属于目标契约（提案冻结态），不得在完成度、演示或验收报告中表述为「已有消息队列」。

| 状态 | 含义 | 本文中的位置 |
|---|---|---|
| `已实现` | 生产代码路径可运行且有测试 | 仅 §9 表中标 `已实现` 的行、§10 的旧 per-run stream |
| `部分实现` | 代码存在但语义不完整 | §9 中 `attempt/backoff`、`EXPIRE` |
| `未实现` | 只有本契约，没有可运行路径 | §3~§8 的全部规则、所有 consumer group、`XAUTOCLAIM`、DLQ、幂等表、单写者 lease、指标 |
| `未验证` | 无本次可复现证据 | §11 中所有「计划测试」列（R2-02~R2-08 才产生） |

配套禁令：

- 本文定义的字段与规则**不授予任何实现**；实现顺序见 §13。
- 任何把本契约当作「唯一事实源」的用法都是违规：**SQL outbox / `execution_events` 才是权威**（§1）。
- 本契约不得被引用为 MQ-01~MQ-06 的通过证据；那些用例只能由 R2-08 的真实运行记录证明。

---

## 1. 范围与不变式

### 1.1 范围

EventBus 是 **Orchestrator → 服务端消费者** 的内部事件投递面，承载的是 `execution_events` 的短期副本。

```text
（目标）SQL 事务：executions + execution_events + outbox
   → relay（单写者 lease）→ XADD agenthub:{env}:run:events
   → consumer group：projection / signal / audit（XREADGROUP + XACK）
   → 失败重投：XAUTOCLAIM → 超过 maxAttempts → agenthub:{env}:dlq（仅元数据）
```

不在范围内：

- **浏览器 SSE / AG-UI 公开事件面**。公开事件仍是 `(runId, seq, subIndex)` 标识的 UPPER_SNAKE 名称，由 Gateway 生成，见 `agui-events.md`。队列事件是内部协议，**不得直接透传前端**。
- Gateway ↔ Orchestrator 的 RPC 面（`event-replay.md` §3 的 gRPC 操作集不变）。
- 审批/终态/权限的**判定**：那些永远以 SQL 为准，队列只能提示。

### 1.2 不变式（不可协商）

| ID | 不变式 |
|---|---|
| EB-INV-01 | MySQL `execution_events` + `outbox` 是唯一权威。队列中的每一条事件都必须能在 SQL 中找到同 `(runId, seq)` 的行。 |
| EB-INV-02 | 队列不可用**不得**导致事件丢失：outbox 保持 `pending`，relay 退避重试；禁止绕过 outbox 同步直发。 |
| EB-INV-03 | 删除队列（或清空 Redis）后，审批、终态、投影、历史回放必须仍能由 SQL 恢复。 |
| EB-INV-04 | 投递语义为 at-least-once；一次业务效果由幂等保证，而不是由「恰好一次」假设保证。 |
| EB-INV-05 | 同一 `runId` 的事件在全局只存在**一个合法 writer**（单写者 lease）；lease 失效时立即停止发布。 |
| EB-INV-06 | 队列中不出现 prompt、token、密钥、DSN、原始工具参数、完整 artifact 内容。 |

---

## 2. 事件封套（Envelope）

### 2.1 字段定义

封套是 XADD 条目中 `data` 字段的 JSON 值（见 §4.3 Redis 条目映射）。**13 个字段全部必须存在**；无值的字段写空字符串，不得省略键（保证 schema 可判定、消费者解析路径单一）。

`schemaVersion` 冻结为整数 `1`。

| 字段 | JSON 类型 / 约束 | 生产者（谁写） | 消费者（谁读） | 现状（代码事实） | 覆盖测试（计划） |
|---|---|---|---|---|---|
| `eventId` | string，pattern `^evt_[A-Za-z0-9_-]{1,80}_[0-9]{1,19}_[0-9]{1,10}$`，≤160 | EventBus publisher（R2-03/R2-04），确定性派生：`evt_<runId>_<seq>_<subIndex>` | 全部 consumer 的去重键；DLQ 条目回填原 `eventId` | `未实现`：relay 当前只写 `runId/seq/type/payload` 四个字段（`services/orchestrator/relay/relay.go:71-74`），无 `eventId` | `TestEnvelopeEventIdIsDeterministic`、`TestEnvelopeRequiredFields`（R2-02）；`MQ-01` |
| `schemaVersion` | integer，`const 1` | publisher 常量 | consumer 先校验版本：未知版本 → `errorClass=unknown_schema_version` 直接进 DLQ，不得猜测解析 | `未实现`：无版本字段 | `TestEnvelopeUnknownSchemaVersionGoesToDlq`（R2-03） |
| `type` | string，allowlist（§3） | 由 `execution_events.event_type` 直传（`services/orchestrator/internal/persistence/mysql/repository.go:215-217`） | 各 consumer 的 handler 路由；`audit` 按类型决定是否记账 | `部分实现`：`event_type` 已落库并透传（`relay.go:73`）；今日实际写入值只有 `run_finished`、`state_update`（`services/orchestrator/httpapi/run_ledger.go:275`、`resume.go:116,130,186`） | `TestEnvelopeTypeNotAllowed`（R2-03）；schema 反向示例 `invalid_public_upper_snake_type`（已实跑，见 §12） |
| `runId` | string，pattern `^run_[A-Za-z0-9_-]{1,64}$`，≤64 | relay 从 `outbox.run_id` 取（`repository.go:694`） | 顺序键（§5）、投影与幂等的分区键 | `已实现`：`outbox.run_id` → XADD values `runId`（`relay.go:72`） | `TestEnvelopeRequiredFields`；`MQ-05` |
| `conversationId` | string，pattern `^[A-Za-z0-9_.:-]{1,64}$`，非空 | relay 从 `executions.conversation_id` JOIN 取（列已存在：`migrations/001_execution_ledger.sql:8`） | `projection` 用于消息归属；`audit` 用于按会话聚合 | `未实现`：`PendingOutbox` 目前只 JOIN 出 `workspace_id`（`repository.go:694-696`），未取 `conversation_id` | `TestEnvelopeConversationIdResolvedFromExecutions`（R2-04） |
| `taskId` | string，pattern `^$\|^[A-Za-z0-9_.:-]{1,64}$`（无关时空字符串） | publisher：A2A/dispatcher 侧的 `taskID`（`services/orchestrator/dispatcher/a2a_dispatcher.go`），无关联时写 `""` | `signal` 用于关联 HITL；`audit` 记账 | `未实现`：outbox 无 task 列，`payload` 里有 `taskId` 但封套层没有 | `TestEnvelopeTaskIdEmptyWhenIrrelevant`（R2-02） |
| `seq` | integer，`≥1`，≤2^63-1 | `outbox.seq`，由 SQL 事务内 `current_seq+1` 分配（`repository.go:203-224`，唯一键 `uniq_outbox_run_seq`） | 顺序判定、缺口检测、游标 | `已实现`：`relay.go:72` | `TestEnvelopeSeqMonotonicPerRun`；`MQ-05` |
| `subIndex` | integer，`≥0` | publisher；一个内部事件映射多个公开事件时的分片号 | 游标 `(seq, subIndex)`；投影去重必须用完整游标 | `部分实现`：SQL 已有 `sub_index`（`services/gateway/store/migrations/002_public_events.sql:4`），但 Gateway 侧硬编码 0（`docs/architecture/v3/00-INDEX.md:55` C3），封套层尚无 | `TestEnvelopeSubIndexParticipationInIdempotencyKey`（R2-02） |
| `occurredAt` | string，UTC RFC3339（pattern 与 `format: date-time` 双约束） | publisher 取 `execution_events.created_at`（列存在：`migrations/001_execution_ledger.sql:107`） | 审计时序、指标延迟计算 | `未实现`：`Event.CreatedAt` 已加载（`repository.go:655,674`）但 relay 未写入 XADD | `TestEnvelopeOccurredAtIsUtcRfc3339`（R2-02） |
| `traceId` | string，pattern `^$\|^[A-Za-z0-9_.:-]{1,128}$` | publisher 从 run 上下文取；跨 gRPC 用 `x-agenthub-trace-id`（`services/orchestrator/grpcapi/auth.go:19-20`） | 全链路串联（`observability-debugging.md`） | `未实现`：outbox/execution_events 无 trace 列，relay 拿不到 → 冻结值：允许 `""`，禁止编造 | `TestEnvelopeTraceIdPassthrough`（R2-04） |
| `idempotencyKey` | string，pattern `^[A-Za-z0-9_.:-]{1,80}:[0-9]{1,19}:[0-9]{1,10}$`，≤160 | publisher 派生：`<runId>:<seq>:<subIndex>` | consumer 幂等表的业务幂等键；与 `run_signals.idempotency_key`、`event_projection_offsets` 游标同构 | `未实现`：封套层无；SQL 侧已有等价唯一键（`migrations/004_run_signals.sql:17,20`） | `TestIdempotencyKeyDerivation`（R2-02）；`MQ-01` |
| `attempt` | integer，`1..32` | **publisher 首次发布恒写 1**；重投次数由 consumer/adapter 侧记录与递增（来源单一，避免双写歧义） | 判定是否达到 `maxAttempts` → DLQ；指标 `retry_total` | `部分实现`：`outbox.attempts` 已存在并用于 backoff（`relay.go:62,78`、`repository.go:686`），语义是**发布**尝试，不等于**消费**尝试 | `TestAttemptCountedByConsumerNotPublisher`（R2-06） |
| `payload` | object，见 §2.3 | `execution_events.payload_json` 原文（`repository.go:654,662`） | 业务 handler | `已实现`：`relay.go:73` 写 `"payload": string(event.Payload)` | `TestPayloadCarriesNoSecrets`（R2-04，含密钥扫描） |

### 2.2 派生规则（必须由实现保证，schema 表达不了）

| ID | 规则 |
|---|---|
| EB-ENV-01 | `eventId` 必须是 `(runId, seq, subIndex)` 的确定性函数，形式为 `evt_<runId>_<seq>_<subIndex>`。**禁止随机 UUID**：relay 崩溃后重复发布同一逻辑事件必须得到同一 `eventId`，否则幂等失效。 |
| EB-ENV-02 | `idempotencyKey == "<runId>:<seq>:<subIndex>"`，且与 `eventId` 携带相同三元组，二者不一致时 publisher 必须拒绝发布。 |
| EB-ENV-03 | `seq ≥ 1`，`subIndex ≥ 0`；同一 run 内 `seq` 严格递增且唯一（SQL `uniq_outbox_run_seq`、`uniq_execution_events_seq` 已保证）。 |
| EB-ENV-04 | `occurredAt` 必须是 UTC（`Z` 结尾）。禁止本地时区偏移字符串。 |
| EB-ENV-05 | 封套字段集闭合：出现未定义字段时，**publisher 侧校验必须拒绝**（schema `additionalProperties: false`）；**consumer 侧必须忽略未知字段但不能据此作判断**（前向兼容）。新增字段属于契约变更，需走 §14。 |
| EB-ENV-06 | 当 `payload` 内含 `type` 时，其值必须等于封套 `type`；不一致视为畸形事件 → `errorClass=decode_error` → DLQ。（JSON Schema 无法比较两个位置的值，必须由实现校验。） |

### 2.3 `payload`

- 内容就是现有内部事件 JSON（`agui.InternalStreamEvent`，`pkg/runtime/agui/internal_event.go:10-47`），不新增包装层，避免二次编码。
- `run_finished` / `state_update` 现有生产者写入的 payload 只有 `{status, reason}`，**不含 `type`**（`services/orchestrator/httpapi/run_ledger.go:184-193`）。这是允许的（EB-ENV-06 只在含 `type` 时生效）；R2-04 若统一补 `type`，必须保持两者一致。
- 顶层禁止键（schema 已硬约束，见 `event-bus.schema.json` `payload.not`）：`prompt`、`prompts`、`systemPrompt`、`token`、`tokens`、`apiKey`、`accessToken`、`refreshToken`、`clientSecret`、`secret`、`secrets`、`password`、`credential`、`credentials`、`privateKey`、`dsn`、`databaseUrl`、`cookie`、`authorization`、`headers`、`blob`。
- 已知缺口：`agui.Artifact`（`pkg/runtime/agui/events.go:120-125`）内联 `Content` 且**没有 `artifactId` 字段**，`metadata` 是自由 map。R2-04 落地前，任何带 artifact 的事件都必须先改造为引用形式（EB-SEC-03），否则不得进入队列。

---

## 3. 事件类型 allowlist（冻结）

`type` 只允许 `pkg/runtime/agui/event_types.go:4-19` 定义的**内部**（lower_snake）名称。这是仓库中真实存在的唯一内部事件名集合，未新增、未改名。

| # | `type` | 语义 | 生产者（真实代码位置） | 主要消费者 | 现状：是否已进 outbox | 覆盖测试 |
|---|---|---|---|---|---|---|
| 1 | `run_started` | run 被接受、流开始 | `services/orchestrator/httpapi/handler_run_stream.go:119`、`resume.go:109` | projection / audit | `未实现`（只出现在 SSE 流，未写 execution_events） | `TestEnvelopeAllowlistCoversInternalTypes` |
| 2 | `run_finished` | run 终态 | `run_ledger.go:275`、`resume.go:116,186` | projection / signal / audit | `已实现`（当前 outbox 实际类型之一） | `MQ-05`、schema 正向示例 |
| 3 | `run_error` | 脱敏错误终态 | `handler_run_stream.go:655,985` | signal / audit | `未实现` | `TestRunErrorPayloadIsRedacted` |
| 4 | `message_start` | 助手消息流开始 | `handler_run_stream.go:806` | projection | `未实现` | schema 正向示例 |
| 5 | `message_delta` | 文本增量 | `executor/executor.go:127`、`serial_executor.go:101`、`dag_executor.go:228,328`、`ordered_parallel_executor.go:135,239`、`agent_turn.go:42`、`handler_run_stream.go:819` | projection | `未实现` | `TestBatchedDeltaWithinSizeCap` |
| 6 | `message_end` | 助手消息流结束 | `handler_run_stream.go:827` | projection | `未实现` | schema 正向示例 |
| 7 | `state_update` | 计划/审批/进度状态 | `run_ledger.go:197`（经 `state()`）、`resume.go:130`、`handler_run_stream.go:326,370,437,586,614,642,692,716,877` | signal / audit | `已实现`（当前 outbox 实际类型之一） | `MQ-04`、schema 正向示例 |
| 8 | `activity_snapshot` | 计划审批/群聊活动卡片快照 | `services/orchestrator/httpapi/handler_run_stream.go:1092` | projection / signal | `未实现`（有 SSE 生产点，未写 execution_events） | `TestActivitySnapshotSizeWithinCap` |
| 9 | `tool_call_start` | 工具调用开始 | **无内部事件生产点**；公开名 `TOOL_CALL_START` 由 `pkg/runtime/agui/translator.go:186,256,365` 从 ADK 工具事件翻译产生。Gateway 侧已按内部名分派（`services/gateway/orchestratorclient/client.go:358`） | audit / projection | `未实现` | `TestToolCallEventsCarryToolCallIdOnly` |
| 10 | `tool_call_args` | 工具参数分片 | 同上（Gateway 分派点 `client.go:368`） | audit | `未实现`（参数原文禁止入队，见 EB-SEC-05） | `TestToolCallArgsIsHashOnly` |
| 11 | `tool_call_end` | 工具调用结束 | 同上（Gateway 分派点 `client.go:384`） | audit / signal | `未实现` | `TestToolResultUnknownNotRetried` |
| 12 | `agent_turn_started` | 群聊某轮开始 | `services/orchestrator/executor/agent_turn.go:33` | projection | `未实现` | `TestAgentTurnRequiresTurnIndex` |
| 13 | `agent_turn_content` | 群聊轮内增量 | `services/orchestrator/executor/agent_turn.go:40` | projection | `未实现` | `MQ-05` |
| 14 | `agent_turn_finished` | 群聊轮结束（投影单位） | `services/orchestrator/executor/agent_turn.go:47` | projection / audit | `未实现` | `TestProjectionConsumesFinishedTurnOnly`（对齐 `services/gateway/projection/worker.go`） |

冻结规则：

| ID | 规则 |
|---|---|
| EB-TYPE-01 | allowlist 是封闭集合。出现表外类型：publisher 拒绝发布；consumer 不处理，直接 `errorClass=type_not_allowed` → DLQ。 |
| EB-TYPE-02 | **禁止**公开 AG-UI 的 UPPER_SNAKE 名称（`RUN_FINISHED` 等）作为封套 `type`。公开名只在 Gateway→前端 SSE 面存在（`agui-events.md`）。 |
| EB-TYPE-03 | **禁止** `pkg/runtime/agui/translator.go` 的点号名（`artifact.delta`、`message.end`、`tool.call`、`tool.result`、`state.delta`、`thinking`）作为封套 `type`；它们是 ADK→runtime 的翻译中间名，不是队列协议。 |
| EB-TYPE-04 | `services/orchestrator/grpcapi/server.go:129-130` 在缺失类型时回退写 `"message"`，该值进入的是 `orchestrator_public_events`（另一张表），**不是**合法封套类型；任何复用该回退到 EventBus 的实现都是违规。 |
| EB-TYPE-05 | 新增类型 = 契约变更：先改本表与 schema `enum`，再改实现，再补测试（§14）。 |

---

## 4. Stream 与 consumer group

### 4.1 Stream key（冻结）

| 用途 | Key | 类型 | 写入者 | 读取者 | 保留策略 |
|---|---|---|---|---|---|
| 运行事件（目标契约） | `agenthub:{env}:run:events` | Stream | relay/publisher（单写者） | `projection` / `signal` / `audit` 三个 group | `XADD MAXLEN ~ <EVENTBUS_STREAM_MAXLEN>`（建议默认 100000，必须显式配置）；**不设 per-key TTL**（共享长生命周期键，`EXPIRE` 会整键丢失） |
| 死信队列 | `agenthub:{env}:dlq` | Stream | 任一 consumer（超 `maxAttempts` 或畸形） | 运维/排查（只读） | `XADD MAXLEN ~ <EVENTBUS_DLQ_MAXLEN>`（默认 10000）；无 TTL；**禁止任何自动重放**（EB-DEL-05） |
| 兼容/回滚路径 | `agenthub:{env}:{workspaceId}:run:{runId}:events` | Stream | 现有 relay | 现有实现（仓库内**无消费者**） | `MAXLEN 10000` + `EXPIRE 1h`（`relay.go:38,69-76`）；见 §10 |

`{env}` 取值与 `environment-variables.md` / compose 中的 env 标识一致（`dev` / `prod` 等），必须小写且匹配 `[a-z0-9_-]{1,32}`。

**与旧文档的差异（必须知道）**：本文把 stream 从「按 run 一个键」改为「按 env 一个共享键」，因此键里**不再有 `{workspaceId}`**。工作区隔离改为规则 EB-ISO-01 承担。`redis-usage.md` §5 与 `event-replay.md` §7 列出的旧键继续有效，但被标注为兼容路径。

### 4.2 Consumer group（冻结）

group 名固定三个，按职责拆分，一个 group 内可多实例并行消费**不同** run：

| group | `consumerKey`（幂等/游标键） | 职责 | 是否权威 | 停机后果 |
|---|---|---|---|---|
| `projection` | `assistant_message.v1`（复用现值：`services/gateway/store/store.go:69`） | 把已提交轮次投影为 assistant 消息、维护 `event_projection_offsets` | 否，可全量由 SQL 重建（`services/gateway/projection/worker.go`） | 刷新后历史缺失，SQL 路径可补齐 |
| `signal` | `run_signal.v1`（新增） | 唤醒/加速 HITL 与状态推进的**提示** | 否。审批/取消判定必须读 `run_signals` / `approvals` | 退化为现有 SQL 轮询（`docs/architecture/v3/00-INDEX.md:59` C7） |
| `audit` | `audit.v1`（新增） | 错误轨迹、路由决策、指标聚合（只读记账） | 否 | 审计延迟，不阻塞链路 |

规则：

| ID | 规则 |
|---|---|
| EB-GRP-01 | 三个 group 独立 `XPENDING`/PEL，互不影响；一个 group 的积压不得阻塞另一个。 |
| EB-GRP-02 | consumer 名 = `{instanceId}`（如 `gateway-a`），必须稳定可辨识，用于 §11 多实例证据。 |
| EB-GRP-03 | `XGROUP CREATE ... mkstream` 在启动时幂等执行（`BUSYGROUP` 视为成功）。 |
| EB-GRP-04 | **禁止**把多个浏览器订阅者放进同一个 `XREADGROUP` 分摊事件（沿用 `redis-usage.md` §4、`event-replay.md` §7）。浏览器订阅继续用独立游标 `XREAD` 或 SQL tail。 |
| EB-GRP-05 | `signal`/`audit` 消费到 `state_update`/`run_error` 后，任何对外可见判定（是否放行、是否终态）必须回到 SQL 复核。 |
| EB-ISO-01 | 封套不携带 `workspaceId`；共享键意味着跨工作区事件同流。任何可见性/权限判断必须由 `runId` 回 SQL `executions.workspace_id`（`migrations/001_execution_ledger.sql:9`）解析，**禁止**依据 payload 内容做租户判断。 |
| EB-ISO-02 | Redis ACL 最小权限：publisher 凭证只允许目标 stream 的 `xadd`；各 consumer 凭证只允许自身 group 的 `xreadgroup/xack/xautoclaim`；DLQ 写入权限只给 consumer。`未实现`，R2-03 落地项。 |

### 4.3 Redis 条目映射

XADD 条目字段固定为：

```text
XADD agenthub:{env}:run:events MAXLEN ~ <N>
     data '<封套 JSON，UTF-8，单行>'
```

- 只使用 `data` 一个字段；不再沿用 `relay.go:71-74` 的四字段散列形式（那套形式没有 `eventId`，无法承载幂等）。
- 检索用字段不重复存：`runId`/`seq` 需要按 key 查询时由 consumer 解析封套（`未实现` 侧要求提供 `XLEN`/lag 指标而非二级索引）。
- `XADD` 必须带 `MAXLEN ~`，禁止无上限写入。

---

## 5. 顺序与单写者 lease

| ID | 规则 |
|---|---|
| EB-ORD-01 | **同一 run 单消费者有序**：对任一 group，同一 `runId` 的事件必须按 `(seq, subIndex)` 递增、且由**单一** consumer 实例串行处理。实现要求 consumer 侧按 `runId` 做路由/持锁，禁止同 run 并发处理。 |
| EB-ORD-02 | 为保证 EB-ORD-01 的发布端前提，relay 必须持有**单写者 lease**：同一 `{env}` 的 `agenthub:{env}:run:events` 全局只有一个合法 writer。 |
| EB-ORD-03 | **lease 失效即停止发布（硬性规则）**：续约失败、续约超时、`ctx` 取消或 fencing token 不再等于当前有效值时，publisher 必须**立即**停止发起新的 `XADD`；已经发出的 pipeline 结果不得再触发 `MarkOutboxPublished`；不得「写完这一批再停」。 |
| EB-ORD-04 | lease 必须带 **fencing token**（单调递增）。任何携带旧 token 的写入被判定为非法；被 fencing 拒绝后不得重试原批次，必须重新排队。 |
| EB-ORD-05 | lease 参数：`TTL` 默认 15s（与 `event-replay`/租约现状同量级），续约周期 ≤ `TTL/3`，续约失败一次即视为失效（不做「连续 N 次失败才失效」的宽松策略）。 |
| EB-ORD-06 | 跨 run **可并行**：不同 `runId` 可由不同 consumer/不同 group 实例并行处理，无任何跨 run 顺序承诺。 |
| EB-ORD-07 | relay 的 lease **不是** run 的 lease。run 状态迁移仍由 `executions.lease_owner` / `fencing_version` 判定（`migrations/001_execution_ledger.sql:16-18`、`repository.go:199-213`），两套 fencing 不得混用或互相替代。 |
| EB-ORD-08 | 缺口不得被跳过：consumer 发现同一 run 的 `seq` 不连续时，必须按 `event-replay.md` §6 回退 SQL tail 补读，禁止「按收到的顺序就是权威」的假设继续投影。 |

现状（`未实现` 项清单）：仓库内**没有** Redis 单写者 lease 实现，也没有任何 `XREADGROUP`/`XACK`/`XAUTOCLAIM` 调用（`grep -rn "XReadGroup\|XAutoClaim\|XGroupCreate\|XAck" --include=*.go services/ pkg/` 无命中）。现有 relay 用「首次失败即中止本批」实现同 run 不乱序（`relay.go:48-56,77-80`），这在单副本下够用，在多副本下**不成立**，因此 EB-ORD-02/03 是引入多副本 relay 的前置条件。

---

## 6. 投递语义

| ID | 规则 |
|---|---|
| EB-DEL-01 | **at-least-once**：允许重复投递，不允许丢消息。消费者必须把重复当常态（崩溃窗口见 `relay.go:81-85` 的既有注释）。 |
| EB-DEL-02 | **ack 只在业务事务提交之后**：handler 内业务写库事务 `COMMIT` 成功后才允许 `XACK`。禁止「先 ack 后执行」「在事务内 ack」。 |
| EB-DEL-03 | 失败处理顺序：handler 失败 → 不 ack（留在 PEL）→ 由 `XAUTOCLAIM` 在 `idle ≥ <EVENTBUS_CLAIM_MIN_IDLE>` 时重投，`attempt+1` → 仍失败继续。禁止消费失败静默 ack。 |
| EB-DEL-04 | 达到 `maxAttempts`（默认 5，`1..32` 封顶）→ 写 `agenthub:{env}:dlq`（仅元数据）→ **然后** `XACK` 原条目，避免 PEL 永久增长。 |
| EB-DEL-05 | **DLQ 永不自动重放**：禁止任何 `XAUTOCLAIM`/定时任务/后台循环把 DLQ 条目重新注入 run:events；禁止 DLQ 的 consumer group。恢复只能走「SQL 重放 + 人工触发」，且必须作为独立运维动作留审计。 |
| EB-DEL-06 | 解析失败、`schemaVersion` 未知、`type` 不在 allowlist、封套超 64KB、`payload` 含 EB-SEC-02 禁止键：视为畸形，`attempt=1` 即直接进 DLQ（不做无意义重试），随后 `XACK`。对应 `errorClass` 分别为 `decode_error` / `unknown_schema_version` / `type_not_allowed` / `size_exceeded` / `security_redaction_required`。 |
| EB-DEL-07 | 队列不可用（Redis down）时：outbox 保持 `pending`、`available_at` 退避（沿用 `relay.go:109-115` 的 backoff 语义），**SQL outbox 仍是权威来源**，恢复后从 SQL 续投；禁止改为同步直写绕过 outbox（EB-INV-02）。 |
| EB-DEL-08 | DLQ 条目结构必须是 `event-bus.schema.json` `$defs.dlqEntry`：`eventId`、`consumerKey`、`runId`、`seq`、`type`、`attempts`、`errorClass`、`firstSeen`、`lastSeen`（可选 `streamKey`、`streamEntryId`），**不含 payload**。 |
| EB-DEL-09 | `errorClass` 只允许封闭枚举：`decode_error`、`unknown_schema_version`、`type_not_allowed`、`size_exceeded`、`security_redaction_required`、`handler_error`、`store_error`、`idempotency_conflict`、`timeout`、`unknown`。禁止写入原始错误文本、堆栈、DSN、provider 响应原文（对齐 `error-taxonomy.md`：对外只给 `safeMessage`）。`security_redaction_required` 与 `error-taxonomy.md` 的 `SECURITY_REDACTION_REQUIRED` 同义，其余为队列侧专用类别。 |

---

## 7. 幂等

| ID | 规则 |
|---|---|
| EB-IDM-01 | `eventId` 全局唯一，且由 `(runId, seq, subIndex)` 确定性派生（EB-ENV-01）。同一逻辑事件的多次发布必须命中同一 `eventId`。 |
| EB-IDM-02 | 消费者按 `(consumer_key, event_id)` 去重，唯一键落在 SQL 幂等表（R2-05 引入 `008_event_consumer_idempotency.sql`；现状 `未实现`，可参考已存在的同构唯一键 `run_signals(run_id, idempotency_key)`、`event_projection_offsets(run_id, consumer_key)`）。 |
| EB-IDM-03 | **重复事件返回原结果**：若幂等表状态为 `done`，直接 ack 并返回已存结果引用，不得再次执行业务副作用。 |
| EB-IDM-04 | 并发重复：唯一键冲突时读取现有行状态；`processing` 状态不得重复执行，返回「稍后重试」，由 `XAUTOCLAIM` 后续处理。 |
| EB-IDM-05 | 同 `eventId` 不同 `payload` 视为异常：必须告警（`errorClass=idempotency_conflict`）并进 DLQ，禁止「后写覆盖」。 |
| EB-IDM-06 | 幂等表只增不删；禁止用进程内 map 作为唯一去重手段（重启即失效）。 |
| EB-IDM-07 | 投影类消费者额外受现有游标语义约束：`event_projection_offsets(run_id, consumer_key)` 记录 `last_seq/last_sub_index`（`services/gateway/store/migrations/001_gateway.sql:59-63`），同一 `(seq, subIndex)` 不得重复写库。偏移只用于避免重复写，不得用于抑制其他订阅者的发送（`event-replay.md` §5）。 |

---

## 8. 安全

| ID | 规则 | 可判定形式 |
|---|---|---|
| EB-SEC-01 | 单个封套序列化后（UTF-8 字节）**≤ 65536 B（64KB）**。publisher 在 `XADD` 前检查；超限不得静默截断：能改引用就改引用，否则 `errorClass=size_exceeded` 进 DLQ 并保留 outbox `pending`。 | 长度检查在实现层；schema 无法表达 → §12 |
| EB-SEC-02 | `payload` 不含 prompt、token、密钥、DSN、连接串、完整工具参数、Cookie/Authorization。 | schema `payload.not`（顶层键）+ 密钥扫描（§12 测试） |
| EB-SEC-03 | 大对象（代码、HTML、截图、PDF、长文本 blob）只以 artifact 引用出现，引用形式为 `$defs/artifactRef`：至少 `artifactId`，可选 `type/title/mime/size/checksum`。禁止内联 `Content`。 | schema `$defs/artifactRef`；已知冲突见 §2.3 |
| EB-SEC-04 | DLQ 只存元数据（§6 EB-DEL-08），**永不**存 payload、参数或错误原文。 | schema `$defs.dlqEntry.additionalProperties=false` |
| EB-SEC-05 | `tool_call_args` 事件的 payload 只允许参数摘要/长度/`argsHash`（`sha256:<64 hex>`），不允许参数原文。 | 契约测试 `TestToolCallArgsIsHashOnly` |
| EB-SEC-06 | `run_error` 的 payload 必须是脱敏后的 `SafeError`（`pkg/runtime/agui/events.go:98-102`）投影：`code` + `safeMessage`；禁止 `stackTrace`、`rawError`、provider 原始响应。 | schema `allOf[0].then`（禁止 `stackTrace`/`rawError`） |
| EB-SEC-07 | 封套 `type` 必须来自 §3 allowlist（EB-TYPE-01），未知类型不入队、不消费。 | schema `enum` |
| EB-SEC-08 | 队列条目不得包含内部服务 URL（Child Agent endpoint、A2A 地址）。 | 契约测试 + 日志扫描 |
| EB-SEC-09 | 事件不得成为租户/权限判定依据（EB-ISO-01）。 | 契约测试 `TestWorkspaceScopeResolvedFromSqlOnly` |
| EB-SEC-10 | 证据与日志中记录 MQ 交互时遵守计划 §5.4/§5.7：`action=mq_*`，只写 `eventId`/`idempotencyKey`/`attempt`/`pending`/`errorClass`，不写 payload。 | 证据扫描脚本 |

---

## 9. 现状核查（本文档撰写时逐条对照代码）

| 主题 | 现状 | 证据 |
|---|---|---|
| relay 能力 | 只做 `XADD` + `EXPIRE`，按 run 分键；无 `eventId`、无封套、无 consumer group、无 DLQ、无 ack/claim | `services/orchestrator/relay/relay.go:41-46,66-80` |
| stream key | `agenthub:{env}:{workspaceId}:run:{runId}:events`，`maxLen=10000`、`streamTTL=1h` | `relay.go:38,45` |
| 同 run 不乱序的实现方式 | 遍历按 `o.id ASC`；首次失败即 `MarkOutboxAttempt` 并 `return`，后续同 run 事件不得超越 | `relay.go:50-56,77-80`；`repository.go:694-698` |
| 重复投递窗口 | `XADD` 成功但 `MarkOutboxPublished` 前崩溃 → 重复发布；消费者需按 `(runId, seq)` 去重 | `relay.go:81-85` |
| Redis 消费者 | **不存在**。仓库内无 `XREADGROUP`/`XACK`/`XAUTOCLAIM`/`XGROUP` 调用；SSE/gRPC 现状是 SQL 轮询（250ms / 200ms） | `grep -rn "XReadGroup\|XAutoClaim\|XGroupCreate\|XAck" --include=*.go services/ pkg/` 无命中；`docs/architecture/v3/00-INDEX.md:59`（C7） |
| outbox 权威写入 | 与 `execution_events`、`executions.current_seq` 同事务写入，SQL 唯一键 `(run_id, seq)` | `repository.go:203-227`；`migrations/001_execution_ledger.sql:101-124` |
| 实际入队事件类型 | 只有 `run_finished`、`state_update`；且 payload 不含 `type` | `run_ledger.go:184-199,275`；`resume.go:114-131,184-187` |
| 投影 consumer key | `assistant_message.v1`，游标表 PK `(run_id, consumer_key)` | `services/gateway/store/store.go:69`；`migrations/001_gateway.sql:59-63` |
| 投影幂等（现状） | 已实现基于 offset 的幂等；无通用 `event_id` 幂等表 | `services/gateway/projection/worker.go`；`event-replay.md` §9 |
| HITL 信号 | 走 SQL `run_signals`（`UNIQUE(run_id, idempotency_key)`），**不经队列** | `migrations/004_run_signals.sql:6-21` |
| Eventbus 抽象 | 无 `pkg/runtime/eventbus` 包（`find . -type d -name eventbus` 无命中） | `docs/plans/v3-gap-atomic-plan.md` §4.3 R2-02 |
| 已有 relay 测试 | `TestIntegrationRelayPreservesPerRunOrder`、`...RedeliversWhenMarkIsLost`、`...KeepsOutboxPendingOnPublishFailure`、`...RebuildsStreamFromSQLAfterDataLoss`（需注入 Redis 地址，否则 SKIP） | `services/orchestrator/relay/redis_integration_test.go`；`docs/reports/evidence/2026-11-02/R0-03/` |

> 结论：**当前不存在消息队列**。本文冻结的是目标契约；旧 per-run stream 是当前唯一真实存在的路径。

### 9.1 实现状态更新（2026-11-03，R2-02/R2-03/R2-04）

上表是 R2-01 撰写时的事实。以下任务已实现，但**生产默认路径未切换**：

| 能力 | 状态 | 位置 / 开关 |
|---|---|---|
| 封套 + 内存传输 + per-run 保序 dispatcher | 已实现（R2-02） | `pkg/runtime/eventbus/{eventbus,memory,dispatch}.go` |
| Redis Streams adapter（XADD/XREADGROUP/XACK/XAUTOCLAIM/DLQ） | 已实现并过真实 Redis 7.2 测试（R2-03） | `pkg/runtime/eventbus/redis.go` |
| relay 通过 EventBus 发布 outbox | 已实现，**默认关闭**（R2-04） | `EVENT_BUS=legacy|redis_streams`（默认 `legacy`）；`OUTBOX_RELAY_LEASE=1` 时用 MySQL `GET_LOCK` 单写者租约，丢失即停止 |
| 消费者幂等表 | 已实现（R2-05） | `event_consumer_idempotency`（orchestrator 008 / gateway 004）+ `pkg/runtime/eventbus/sqlidem` |
| Gateway 投影消费 | flag 试点（R2-07） | `EVENTBUS_PROJECTION=1` + `REDIS_URL`；消费只触发 SQL recovery，offset 幂等，SQL worker 仍是回退路径 |
| 重试/DLQ 指标与运维接口 | 部分（adapter 有钩子；无 Prometheus 文本/统计接口，R2-06） | `pkg/runtime/eventbus/metrics.go` |

EVENT_BUS=redis_streams 下 relay 的封套由 `(runId, seq, subIndex=0)` 派生 `eventId`，
conversationId 取自 `executions.conversation_id`；SQL outbox 仍是权威，publish 成功后才
`MarkOutboxPublished`，失败只记 attempt/backoff。旧 per-run stream 保留为回滚路径。

---

## 10. 兼容与回滚：旧 per-run stream

**明确声明**：现有 per-run stream

```text
agenthub:{env}:{workspaceId}:run:{runId}:events      （由 services/orchestrator/relay/relay.go 写入）
```

是 **兼容 / 回滚路径（compatibility / rollback path）**，**不是**目标契约。理由与边界：

| 维度 | 旧 per-run stream（兼容路径） | 本文契约（目标） |
|---|---|---|
| 键粒度 | 每 run 一键，含 `{workspaceId}` | 每 `{env}` 一个共享键 |
| 条目字段 | `runId`/`seq`/`type`/`payload` 散字段 | 单一 `data` 字段承载 13 字段封套 |
| 幂等标识 | 无 `eventId`，只能 `(runId, seq)` | `eventId` + `idempotencyKey` + `(consumer_key, event_id)` |
| 消费方式 | 无 consumer group、无 ack、无 PEL | `XREADGROUP` + `XACK` + `XAUTOCLAIM` + 三 group |
| 失败 | 无重试上限、无 DLQ | `maxAttempts` → DLQ（仅元数据，不自动重放） |
| 生命周期 | `MAXLEN 10000` + `EXPIRE 1h` | `MAXLEN ~`，无 per-key TTL |
| 契约地位 | 保留一个版本，供 `EVENT_BUS=legacy` 回滚 | `schemaVersion=1` 冻结目标 |
| 事实等级 | `已实现`（仅发布端；无消费者） | `未实现` |

回滚规则：

| ID | 规则 |
|---|---|
| EB-CMP-01 | `EVENT_BUS=legacy`（默认值，R2-04 引入 flag）时行为与今天完全一致：只写 per-run stream，不写共享 stream。 |
| EB-CMP-02 | 共享 stream 上线后，旧路径至少保留**一个版本**；不得在同一批次删除 per-run 写入代码。 |
| EB-CMP-03 | 两种路径**都不改变** SQL 权威地位：回滚不丢事件（outbox 仍是真源），前滚不需要数据迁移。 |
| EB-CMP-04 | 现有 relay 测试（§9 的 4 个 `TestIntegrationRelay*`）在引入 EventBus 后必须继续通过；它们就是兼容路径的回归门禁。 |
| EB-CMP-05 | 文档与演示中不得把旧 per-run stream 描述为「消息队列」；它只是 Redis 短期副本写入。 |

---

## 11. 规则 → 验收/测试映射

R2-01 不产出任何代码或测试（`未验证`）。下表列的是**后续任务必须落地的测试点**，命名即为计划中的测试函数，供 R2-02~R2-08 直接对照实现。

| 规则 | 验收用例（计划） | 落地任务 | 对应测试族 |
|---|---|---|---|
| EB-ENV-01/02/03/04、EB-IDM-01 | `TestEnvelopeEventIdIsDeterministic`、`TestEnvelopeRequiredFields`、`TestIdempotencyKeyDerivation` | R2-02 | `pkg/runtime/eventbus/envelope_test.go` |
| schema 一致性 | `TestEnvelopeSchemaExamplesMatchContract`（加载 `docs/contracts/testdata/event-bus/*.json` 全量校验） | R2-02 | 契约测试 |
| EB-TYPE-01/02/03、EB-SEC-07 | `TestEnvelopeTypeNotAllowed`、`TestEnvelopeAllowlistCoversInternalTypes` | R2-02/R2-03 | `pkg/runtime/eventbus/redis_test.go` |
| EB-STR（键/MAXLEN/`data` 单字段） | `TestRedisAdapterUsesSingleDataFieldAndMaxLen` | R2-03 | 同上（真实 Redis，无 Redis 则 SKIP） |
| EB-GRP-01/02/03/04 | `TestConsumerGroupsAreIndependent` | R2-03/R2-05 | 同上 |
| EB-ORD-01/02/08 | `TestPerRunOrderingUnderSingleWriter`、`TestGapFallsBackToSqlTail` | R2-03/R2-07 | `MQ-05` |
| EB-ORD-03/04/05/06/07 | `TestRelayStopsPublishingImmediatelyOnLeaseLoss`、`TestStaleFencingTokenRejected`、`TestCrossRunParallelismDoesNotBreakPerRunOrder` | R2-04 | relay 单测 + 多副本证据 |
| EB-DEL-01/02 | `TestAckOnlyAfterTransactionCommit`、`TestDuplicateDeliveryYieldsOneBusinessEffect` | R2-05 | `MQ-01` |
| EB-DEL-03、EB-IDM-04 | `TestUnackedMessageIsReclaimedByXAutoClaim`、`TestConcurrentDuplicateDoesNotReexecute` | R2-05 | `MQ-02` |
| EB-DEL-04/05、EB-DEL-09 | `TestMaxAttemptsMovesToDlqAndNeverReplays`、`TestDlqEntryCarriesNoPayload`、`TestDlqErrorClassEnumIsClosed` | R2-06 | `MQ-04` |
| EB-DEL-06 | `TestMalformedEnvelopeGoesStraightToDlq` | R2-03 | `MQ-04` |
| EB-DEL-07、EB-INV-02/03 | `TestOutboxSurvivesRedisOutage`、`TestSqlRecoveryAfterRedisFlush` | R2-04/R2-08 | `MQ-03`、`MQ-06` |
| EB-IDM-02/03/05/06/07 | `TestIdempotencyTableIsTheOnlyDedupSource`、`TestSameEventIdDifferentPayloadConflicts` | R2-05 | `MQ-01`、F-10 |
| EB-SEC-01 | `TestOversizedEnvelopeIsRejectedNotTruncated` | R2-03/R2-04 | `S-*` |
| EB-SEC-02/03/05/06/08 | `TestPayloadCarriesNoSecrets`（含密钥模式扫描）、`TestArtifactIsReferenceOnly`、`TestToolCallArgsIsHashOnly`、`TestRunErrorPayloadIsRedacted` | R2-04 | `S-*`、`security-test-policy.md` |
| EB-ISO-01/02 | `TestWorkspaceScopeResolvedFromSqlOnly`、`TestRedisAclMinimumPermissions` | R2-03/R2-07 | `S-*` |
| EB-CMP-01/02/04 | `TestEventBusLegacyFlagKeepsPerRunStream`（flag off 行为不变）+ 现有 4 个 relay 集成测试 | R2-04 | 回归门禁 |
| MQ-01~MQ-06、P-06 端到端 | `tests/eventbus/` 故障注入脚本 | R2-08 | 见计划 §4.3 |

---

## 12. schema 无法表达、必须由实现与测试保证的规则

明确列出，避免「schema 过了就算合规」的误读：

| 规则 | 为什么 schema 管不了 | 由谁保证 |
|---|---|---|
| EB-SEC-01（64KB 上限） | JSON Schema 约束结构不约束序列化字节数 | publisher 前置检查 + `TestOversizedEnvelopeIsRejectedNotTruncated` |
| EB-ENV-01/02（`eventId`/`idempotencyKey` 派生） | 需要跨字段计算与等值比较 | publisher + `TestEnvelope*Derivation` |
| EB-ENV-06（`payload.type == type`） | schema 不支持两个位置的值相等 | publisher/consumer + `TestPayloadTypeMismatchIsDecodeError` |
| EB-SEC-02 深层敏感键 | `payload.not.required` 只覆盖顶层键 | publisher 脱敏 + 证据/日志密钥扫描（`sk-[A-Za-z0-9_-]{20,}`、`BEGIN ... PRIVATE KEY`、`user:pass@`） |
| EB-ORD-03（lease 失效即停） | 属运行时行为 | relay 实现 + `TestRelayStopsPublishingImmediatelyOnLeaseLoss` |
| EB-DEL-02（ack 时机）/EB-DEL-05（DLQ 不重放） | 属事务与运维行为 | consumer/adapter 实现 + 故障注入用例 |
| `occurredAt` 为「事件发生时间」而非「发布时间」 | schema 只校验格式 | publisher 取 `execution_events.created_at` |

---

## 13. 实现顺序（本文之后）

```text
R2-01（本文）契约冻结
  → R2-02 接口 + 内存实现（无 Redis 依赖）
  → R2-03 Redis Streams adapter（XADD/XREADGROUP/XACK/XAUTOCLAIM/DLQ）
  → R2-04 outbox relay v2（EVENT_BUS flag、单写者 lease、封套构造）
  → R2-05 幂等消费者 + 008_event_consumer_idempotency.sql
  → R2-06 重试/DLQ/指标
  → R2-07 Gateway 投影消费试点（SQL 仍权威）
  → R2-08 MQ 故障/重复/顺序验收
```

每步都必须只依赖本文，不得反向修改本文语义来迁就实现；确需修改走 §14。

---

## 14. 变更管理（冻结之后）

| 变更类型 | 流程 | 是否 bump `schemaVersion` |
|---|---|---|
| 新增可选封套字段 | 契约 MR（含 §2 表 + schema + 示例 + 对应测试点）→ 评审 → 实现 | 否（minor）；但 EB-ENV-05 要求先改 schema 再发布 |
| 新增事件类型 | 同上，必须说明生产者/消费者/测试 | 否 |
| 修改既有字段语义 / 删除字段 / 改 allowlist 语义 | 需架构评审签字（`project-architecture`）+ 迁移与回滚说明 | 是（`schemaVersion` → 2） |
| 引入 NATS 第二适配器 | 不改封套，只改 adapter；见 `docs/architecture/v3/18-...md` D14 | 否 |
| 引入 Kafka/NATS 之外的新中间件 | 本契约禁止未经评审引入 | — |

消费者兼容性要求：`schemaVersion` 不匹配时**拒绝处理并进 DLQ**，禁止「尽力解析」。生产者兼容性要求：不得发布 `additionalProperties` 之外的字段。

---

## 15. 与其他文档的关系

| 文档 | 关系 |
|---|---|
| `redis-usage.md` | Redis 仍是辅助、非权威（EB-INV-01 与之完全一致）。旧键列表保留为兼容路径；共享 stream/DLQ 新键以本文为准。 |
| `event-replay.md` | 权威顺序、`(runId, seq, subIndex)` 游标、SQL 补读、投影与订阅双游标语义**全部继承不变**。本文只新增队列侧封套/ack/DLQ/lease 规则。§4（outbox relay）与本文 §6 是同一件事的两个层次：SQL 侧不变，队列侧按本文升级。 |
| `agui-events.md` / `agui-events.schema.json` | 公开前端事件协议，独立于本文；两者不得混用名称（EB-TYPE-02）。 |
| `gateway-orchestrator-events.md` | 描述 Gateway↔Orchestrator 的公开事件序列，仍是事实源；EventBus 不改变该映射。 |
| `docs/architecture/v3/18-message-queue-and-rich-interactions.md` | 其 §3.3/§3.4 为**提案建议**，本文是**冻结结果**，冲突处以本文为准：① stream 名带 `{env}` 前缀且不引入 `agenthub.approvals`/`agenthub.tool.results`（v1 只有 `run:events` + `dlq`）；② `type` 用 lower_snake 内部名而非点号名（提案示例 `run.finished` 作废）；③ 审批仍走 SQL `run_signals`，不入队。 |
| `error-taxonomy.md` | DLQ `errorClass` 是队列侧封闭枚举，与对外错误码分开；映射关系在 R2-06 指标落地时补。 |
| `security-boundaries.md` / `secret-management.md` | EB-SEC-* 是它们在本契约的具体化；任何密钥、DSN 入队都是安全违规。 |

---

## 16. 校验与复现

```bash
# 1) schema 自身必须是合法的 draft 2020-12 schema
PYTHONUTF8=1 python -c "import json;from jsonschema import Draft202012Validator;Draft202012Validator.check_schema(json.load(open('docs/contracts/event-bus.schema.json',encoding='utf-8')));print('OK')"

# 2) 逐条校验示例（exit 0 = 合法，exit 1 = 非法）
for f in docs/contracts/testdata/event-bus/*.json; do
  PYTHONUTF8=1 python -m jsonschema -o plain -i "$f" docs/contracts/event-bus.schema.json
  echo "$(basename "$f") exit=$?"
done

# 3) 现状复核（确认「未实现」陈述仍然成立）
grep -rn "XReadGroup\|XAutoClaim\|XGroupCreate\|XAck" --include=*.go services/ pkg/     # 应为空
grep -n "agenthub:%s:%s:run:%s:events" services/orchestrator/relay/relay.go             # 兼容路径仍在
```

> `PYTHONUTF8=1` 在 Windows 上是必需的：不带它时 CLI 会以 gbk 打开含中文描述的 schema 并报 `UnicodeDecodeError`（已在证据中复现）。

实际执行结果与命令输出：`docs/reports/evidence/2026-11-02/R2-01/`（`commands.txt`、`summary.md`、`events.jsonl`、`logs/schema-validation.txt`）。
