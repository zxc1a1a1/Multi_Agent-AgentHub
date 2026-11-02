# R2-01 冻结 EventBus 契约 —— 证据摘要

- task: `R2-01`（`docs/plans/v3-gap-atomic-plan.md` §4.3）
- testId: `P7-MQ-01` / GAP-10
- 类型: **DOCUMENTS ONLY**（无 Go 代码、无前端代码、无运行时行为改动）
- commit baseline: `150019f2da917abcbaabc47f6595aa58bc442057`（branch `dev`）
  - 说明：本次工作期间 HEAD 从 `0271a38` 前进到 `150019f`（队友提交 `test(memory): 认证记忆 scope/确认/遗忘/缓存路径`，只含 `docs/reports/evidence/2026-11-02/R1-01/` 与一个 memory 测试）。本任务引用的源码文件（`relay.go`、`event_types.go`、`repository.go`、`store.go`）经 `git diff --name-only HEAD -- <files>` 确认**未被该提交改动**，文档中的行号引用对 `150019f` 成立。
- 执行时间: 2026-11-02T09:11Z 起，校验于 09:34Z 完成（本地时区 UTC）
- 环境: Windows / git-bash；`python 3.11.5` + `jsonschema 4.19.1`（校验器）；`node v20.12.2`（仅 JSON 解析，路径 `/e/node_js/node.exe`，PATH 上的 `node` 已损坏）
- 未连接任何外部服务：无 MySQL、无 Redis、无 HTTP、无 LLM 调用；未读取 `.env`

## 1. 产出文件

| 文件 | 动作 |
|---|---|
| `docs/contracts/event-bus.md` | 新增（冻结契约正文） |
| `docs/contracts/event-bus.schema.json` | 新增（JSON Schema draft 2020-12） |
| `docs/contracts/testdata/event-bus/*.json` | 新增 21 个示例（14 正向 / 7 反向） |
| `docs/contracts/redis-usage.md` | 最小改动：header 交叉引用、§5 键定位、新增 §9（3 处编辑，未重写） |
| `docs/contracts/event-replay.md` | 最小改动：header 交叉引用、§4 定位说明、§7 两条要点（3 处编辑，未重写） |
| `docs/contracts/README.md` | 最小改动：core contracts 列表加入 `event-bus.md`/`event-bus.schema.json` 并标注 `draft-frozen` |

**关于 `docs/contracts/00-INDEX.md`**：计划 §4.3 R2-01 把改动范围写成 `00-INDEX.md`，但 `docs/contracts/` 下**不存在** `00-INDEX.md`（`ls docs/contracts | grep -i index` 只有 `constraint-index.md`，它是「约束状态表」而非契约索引）。仓库里真实的契约索引是 `docs/contracts/README.md`（标题 "AgentHub Contract Directory"，含 "Current core contracts" 列表），真实名为 `00-INDEX.md` 的文件是 `docs/architecture/v3/00-INDEX.md`（架构文档索引，且当前工作区已被队友改动）。为遵守「只改列出文件、不做范围扩张」，本任务更新 `docs/contracts/README.md` 承担索引职责；`docs/architecture/v3/00-INDEX.md` **未修改**，需评审确认是否由后续任务补引用（见 §5 未决）。

## 2. 验收结果（计划 §4.3「验收」逐条）

| 验收项 | 结论 | 依据 |
|---|---|---|
| 封套 13 字段全部定义，且每字段有生产者/消费者/测试 | **PASS** | `event-bus.md` §2.1 表；交叉检查 `envelope required == plan R2-01 field list: True (13 fields)` |
| `type` allowlist 来自真实代码，未发明 | **PASS** | 交叉检查 `schema enum n=14 go internal n=14 sets equal: True`（对照 `pkg/runtime/agui/event_types.go:4-19`） |
| 示例通过 schema 校验 | **PASS** | 14/14 正向 `exit=0`，7/7 反向 `exit=1`（`logs/schema-validation.txt`） |
| schema 自身是合法 draft 2020-12 | **PASS** | `Draft202012Validator.check_schema` → `OK meta=https://json-schema.org/draft/2020-12/schema` |
| 与现有 `event-replay.md` 无矛盾 | **PASS** | `event-bus.md` §15 明确继承权威顺序 / `(runId,seq,subIndex)` / SQL 补读 / 双游标；`EB-GRP-04` 原样继承「浏览器订阅不得同 `XREADGROUP`」；`redis-usage.md` §9 与 `event-replay.md` §4/§7 已把旧 per-run 键标为兼容路径 |
| 明确旧 per-run stream 为兼容/回滚路径 | **PASS** | `event-bus.md` §10（含 EB-CMP-01..05 与逐项差异表）；`redis-usage.md` §5/§9；`event-replay.md` §4 |
| 单写者 lease 失效即停止发布的规则被写死 | **PASS** | `EB-ORD-03`（「立即停止发起新的 XADD；已发出的 pipeline 结果不得再触发 MarkOutboxPublished；不得写完这一批再停」）+ `EB-ORD-04/05` fencing 与 TTL |
| 不夸大：文档不被读成实现声明 | **PASS** | `status: draft-frozen` + §0「未实现/已实现」分级表 + §9 现状核查 + §12「schema 管不了的规则」+ §11 测试列标注为「计划」 |
| 密钥/凭据扫描 | **PASS（含说明）** | 见 §4 |

## 3. 校验矩阵（真实执行）

| # | 命令（详见 commands.txt） | 结果 |
|---|---|---|
| V1 | `Draft202012Validator.check_schema(event-bus.schema.json)` | PASS |
| V2 | `python -m jsonschema -o plain -i <14 正向示例> <schema>` | PASS（14/14 exit=0） |
| V3 | `python -m jsonschema -o plain -i <7 反向示例> <schema>` | PASS（7/7 被正确拒绝 exit=1） |
| V4 | `/e/node_js/node.exe -e "...JSON.parse..."` | PASS（schema + 21 examples） |
| V5 | allowlist / required-fields 与 Go 源码、计划字段表交叉比对 | PASS（无差集） |
| V6 | `events.jsonl` 结构校验（必填键 / `detail` 白名单键 / `errorClass` 取值 ∈ 契约枚举） | PASS（29 rows） |
| V7 | `grep -rn "XReadGroup\|XAutoClaim\|XGroupCreate\|XAck" --include=*.go services/ pkg/` | PASS（无命中，支撑「未实现」表述） |
| V8 | `find . -type d -name eventbus` | PASS（0 个，`pkg/runtime/eventbus` 尚不存在） |
| V9 | `git ls-files .env` | PASS（空输出：`.env` 未被跟踪） |
| V10 | 密钥模式扫描（见 §4） | PASS（无真实凭据） |
| V11 | `go test ./...`（orchestrator / gateway） | **SKIP**：R2-01 不改 Go 代码，无被测变更；跑一次只会给出与任务无关的既有状态 |
| V12 | `npm test -- --run` / `npm run build`（frontend） | **SKIP**：无前端改动；且 `npm` 需经 `/e/node_js/node.exe /e/node_js/node_modules/npm/bin/npm-cli.js` 调用，本任务无前端产物可验证 |
| V13 | Redis/MySQL 集成测试 | **SKIP**：本任务不改变 relay 行为，也未注入操作者提供的环境变量（`AGENTHUB_REDIS_TEST_ADDR`、`AGENTHUB_MYSQL_TEST_DSN` 均只引用变量名） |

反向示例与对应拒绝理由（来自 `logs/schema-validation.txt` 实际输出）：

| 反向示例 | 拒绝理由 | 冻结规则 |
|---|---|---|
| `invalid_public_upper_snake_type` | `'RUN_FINISHED' is not one of [...]` | EB-TYPE-02 |
| `invalid_missing_event_id` | `'eventId' is a required property` | §2.1 全字段必填 |
| `invalid_prompt_in_payload` | `should not be valid under {...'required':['prompt']...}` | EB-SEC-02 |
| `invalid_seq_zero` | `0 is less than the minimum of 1` | EB-ENV-03 |
| `invalid_agent_turn_without_turn_index` | `'turnIndex' is a required property`（payload） | §3 #12-14 |
| `invalid_unknown_envelope_field` | `Additional properties are not allowed ('unknownField' was unexpected)` | EB-ENV-05 |
| `invalid_event_id_derivation` | `'evt_run_0123456789abcdef_13' does not match '^evt_...$'` | EB-ENV-01 |

## 4. 安全扫描记录

- 模式：`sk-[A-Za-z0-9_-]{20,}` / `BEGIN ... PRIVATE KEY` / `api_key=<20+>` / `password=<非占位>` / `user:pass@host`。
- 结果：**0 处真实凭据**。
- 命中说明（如实记录）：宽松扫描报出 2 行，内容均为**扫描模式本身的字面文本**（`event-bus.md` §12 的 EB-SEC-02 说明行、`commands.txt` §4 的 grep 命令行），不是密钥。
- 额外用 40+ 字符泛匹配复核时命中 20 处，逐条确认全部为测试函数名与 commit SHA（例如 `TestRelayStopsPublishingImmediatelyOnLeaseLoss`），非凭据。
- 全部示例使用合成 ID（`run_0123456789abcdef`、`conv_1730500000000`、`task_…`、`trace_…`），不含可读文本、prompt 或 URL。
- 未读取、未打印任何 `.env` 内容；`git ls-files .env` 为空。
- 未执行 `git add` / `git commit` / `git push`。

## 5. 交互覆盖检查（计划 §5.5）

本任务不产生运行时交互；`events.jsonl` 记录的是**契约校验用例结果行**（`action=test_case`，沿用 `docs/reports/evidence/README.md` §3 约定），共 29 行。

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作（点击/输入/选择/按键） | events.jsonl | 0 | not_applicable: 文档任务，无 UI 与浏览器参与 |
| API 请求/响应 | events.jsonl | 0 | not_applicable: 无 HTTP 调用，未启动任何服务 |
| SSE 连接/事件/断线 | events.jsonl | 0 | not_applicable: 无流式运行；`eventType`/`seq`/`subIndex` 字段仅出现在 14 条 schema 示例校验行中，来自合成测试值而非真实流 |
| HITL 请求/信号/Claim | events.jsonl | 0 | not_applicable: 契约规定审批仍走 SQL `run_signals`，不入队（§3/§15），无审批交互 |
| 工具授权/结果 | events.jsonl | 0 | not_applicable: 无工具执行；`tool_call_*` 行只是封套 schema 校验 |
| 记忆召回 | events.jsonl | 0 | not_applicable: 与记忆链路无关 |
| Artifact | events.jsonl | 0 | not_applicable: 无制品生成；`artifactRef` 只是 schema `$defs` 定义 |
| MQ publish/consume/ack/claim/DLQ | events.jsonl | 0 | not_applicable: **R2-01 只冻结契约，不实现队列**；真实 `mq_*` 交互由 R2-03/R2-04/R2-05/R2-08 产生。`action=mq_publish` 等在本任务写出来即为伪造 |
| 多实例（instanceId/role） | events.jsonl | 0 | not_applicable: 单进程本地文档校验；`instanceId` 无真实值可填（§5.2 禁止编造） |
| MySQL 事务/幂等键冲突 | events.jsonl | 0 | not_applicable: 未连接数据库 |
| Redis 命令（stream/pending/DLQ） | events.jsonl | 0 | not_applicable: 未连接 Redis；§9 的「无消费者」结论由静态 grep 支撑 |
| 契约/schema 校验用例 | events.jsonl | 29 | yes（V1~V6 全覆盖：14 正向 + 7 反向 + 8 项交叉/现状核查） |
| 跨层串联（`runId`/`traceId` 一致） | events.jsonl | 14 | yes（同一合成 `run_0123456789abcdef` 贯穿 14 条示例行，`traceId` 与示例文件一致）；说明：这是示例数据的一致性，不是真实链路串联 |

无法关联层的原因说明：本任务的全部结论都来自**源码静态核对 + JSON Schema 校验**，不存在跨层运行时链路可串联；因此不填写真实 `requestId`/`runId`（§5.2 要求没有就写空字符串，不得编造）。

## 6. 冻结内容摘要（便于评审）

- 封套：`eventId`、`schemaVersion(=1)`、`type`、`runId`、`conversationId`、`taskId`、`seq`、`subIndex`、`occurredAt`、`traceId`、`idempotencyKey`、`attempt`、`payload`（13 字段全必填，`additionalProperties: false`）。
- allowlist（14，取自 `pkg/runtime/agui/event_types.go`）：`run_started`、`run_finished`、`run_error`、`message_start`、`message_delta`、`message_end`、`state_update`、`activity_snapshot`、`tool_call_start`、`tool_call_args`、`tool_call_end`、`agent_turn_started`、`agent_turn_content`、`agent_turn_finished`。
- stream：`agenthub:{env}:run:events`、`agenthub:{env}:dlq`；consumer group：`projection`（`assistant_message.v1`）、`signal`（`run_signal.v1`）、`audit`（`audit.v1`）。
- 顺序：同 run 单消费者有序；单写者 lease + fencing token；**lease 失效立即停止发布**（EB-ORD-03）。
- 投递：at-least-once；业务事务提交后 `XACK`；`XAUTOCLAIM` 重投；`maxAttempts` → DLQ（仅元数据）且**永不自动重放**；队列不可用时 SQL outbox 仍权威。
- 幂等：`eventId` 确定性派生 `evt_<runId>_<seq>_<subIndex>`；`(consumer_key, event_id)` 去重；重复返回原结果。
- 安全：单事件 ≤ 64KB；payload 禁 prompt/token/密钥；大对象只带 artifact 引用；DLQ 只存元数据；type 走 allowlist。
- 兼容：旧 `agenthub:{env}:{workspaceId}:run:{runId}:events`（`relay.go` 现写入）= 兼容/回滚路径，非目标契约。

## 7. 未决 / 需评审

1. `docs/architecture/v3/00-INDEX.md` 是否需要补 D14/P7 行的 EventBus 引用（本任务未改，避免与队友未提交改动混淆）。
2. `docs/architecture/v3/18-message-queue-and-rich-interactions.md` §3.3/§3.4 的提案值（点号 `type`、`agenthub.approvals`/`agenthub.tool.results` 流名）已被本契约**取代**，差异在 `event-bus.md` §15 逐条列出；该架构文档同属队友在改的文件，未编辑。
3. `agui.Artifact` 内联 `Content` 且无 `artifactId`（`pkg/runtime/agui/events.go:120-125`）与 EB-SEC-03 冲突，R2-04 落地前必须改造为引用形式（已写入 §2.3 缺口说明）。
4. 共享 stream 去掉 `{workspaceId}` 后的租户隔离依赖 EB-ISO-01（回 SQL 取 `executions.workspace_id`）+ EB-ISO-02（Redis ACL 最小权限，未实现）。
5. `subIndex` 目前恒为 0（`00-INDEX.md:55` C3），契约保留分片语义；真正启用需与 Gateway 公开事件映射一起评审。
