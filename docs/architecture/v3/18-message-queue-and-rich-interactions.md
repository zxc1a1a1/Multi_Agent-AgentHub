# 18. 消息队列与富交互渲染（P7 提案）

- 版本：v3.4 提案
- 日期：2026-11-02
- 状态：**未实现**。本文只做选型、边界与原子任务设计，不把提案写成现状。
- 前置事实源：`docs/architecture/v3/00-INDEX.md`（C7/C15/C16）、`docs/contracts/event-replay.md`、`docs/contracts/redis-usage.md`、`docs/contracts/artifact-schema.md`
- 关联计划：`docs/plans/v3-atomic-implementation-plan.md` P7

## 1. 为什么现在不做代码

当前 v3 的结论仍然是“单机演示不需要新增中间件”：MySQL 是权威事实，Redis 承担缓存/短时事件副本，Gateway 投影和 Orchestrator 信号都有 SQL 兜底。引入消息队列或交互式渲染前，必须先冻结事件封套、消费语义和前端组件白名单，否则会把“能跑”变成“不可恢复”。

因此本文与 P7 只做：

1. 调研成熟开源项目；
2. 给出推荐默认值与不引入清单；
3. 拆成可独立验收的原子任务；
4. 明确安全、许可与运行时验证门禁。

## 2. 现状证据

| 事实 | 等级 | 证据 |
|---|---|---|
| Redis 已用于 outbox relay / 短时缓存 | 已实现 | `services/orchestrator/relay`、`docs/contracts/redis-usage.md` |
| 当前没有 Redis Stream 消费者；实时链路仍以 SQL 轮询为主 | 部分实现 | `00-INDEX.md` C7；Gateway tail SSE / gRPC SubscribeRun 轮询 SQL |
| MySQL outbox 是当前跨进程投递的可靠边界 | 已实现 | `outbox` 表、relay、`run_signals` |
| 前端已有 Markdown/Shiki/Mermaid/KaTeX/diff2html/iframe 预览 | 已实现 | `StreamingText.tsx`、`CodePreview.tsx`、`DiffPreview.tsx`、`WebPreview.tsx`、`MermaidDiagram.tsx` |
| 前端没有通用“可交互 artifact”协议 | 未实现 | 仅有 `code_preview`/`web_preview`/`form_input` 等固定卡片 |
| 前端没有 Streaming Markdown 专用渲染器 | 未实现 | 当前 `react-markdown` + fork 的流式块解析 |
| 没有消息队列服务 | 未实现 | `docker-compose.new-arch.yml` 只有 MySQL/Redis，无 Kafka/NATS/RabbitMQ |

## 3. 消息队列选型

### 3.1 候选对比

| 方案 | 优势 | 代价/风险 | 适用结论 |
|---|---|---|---|
| **Redis Streams 7.2**（现有 Redis） | 已有部署；`XADD/XREADGROUP/XACK`、consumer group、PEL、`XAUTOCLAIM`；go-redis 已有依赖；运维面最小 | at-least-once，需要业务幂等；无原生 DLQ/延迟队列；内存型存储需 trim 与持久化策略；**当前 Redis 7.2 的许可证在引入前必须复核，禁止未经评审升级到 AGPL/SSPL 版本** | **P7 首选 MVP 适配器** |
| **NATS JetStream** | 单二进制、Go 客户端成熟（`nats.go`）；durable stream、pull consumer、replay、ack/nak；Apache-2.0；适合后续多副本 | 新增一个服务与运维对象；需要自己定义 DLQ/重试策略；团队需学习 subject/stream/consumer 模型 | **第二适配器/scale-up 候选** |
| **RabbitMQ 4.x** | quorum queue/stream、路由、DLX 成熟；生态完善 | 新增 Erlang 运行时；单机运维复杂度高于 Redis/NATS；MVP 收益不足 | 条件性候选，非默认 |
| **Kafka / Redpanda** | 高吞吐、长保留、生态强 | 重；分区/consumer group/运维成本高；Redpanda 许可证需单独审查 | **当前不引入**（与 00-INDEX 结论一致） |
| **Temporal / 工作流引擎** | 长事务/补偿强 | 与现有 SQL 账本/租约重复；引入成本高 | 不引入（D6 延续） |

### 3.2 推荐决策（D14）

**新增 `EventBus` 接口，Redis Streams 7.2 作为 P7 第一适配器；NATS JetStream 作为同接口的第二适配器，只有达到明确容量/运维阈值才启用；Kafka/Redpanda 不在本计划内。**

理由：

- AgentHub 当前吞吐是“每次 run 少量事件”，瓶颈不在分区吞吐，而在恢复语义；
- Redis 已在 compose 中，先复用能减少一个中间件；
- MySQL outbox 继续作为唯一可靠生产源，消息队列只做“更快投递”，不能成为唯一事实；
- 用接口隔离后，后续替换 NATS 不改业务 handler。

阈值（需在 P7-MQ-01 冻结）：

- Redis Stream 持续积压 > 10k 或 p99 投递 > 2s，且 SQL 轮询已无法满足；
- 需要跨区域复制/长保留/独立扩容；
- 运维团队明确接受 NATS 服务。

### 3.3 事件封套（建议冻结）

```json
{
  "eventId": "evt_...",
  "schemaVersion": 1,
  "type": "run.finished",
  "runId": "run_...",
  "conversationId": "conv_...",
  "taskId": "task_...",
  "seq": 12,
  "subIndex": 0,
  "occurredAt": "2026-11-02T00:00:00Z",
  "traceId": "trace_...",
  "idempotencyKey": "run_...:12:0",
  "attempt": 1,
  "payload": {}
}
```

规则：

- `runId` 作为分区/顺序键；同一 run 的事件至少保持发布顺序。
- `eventId` 全局唯一；消费者以 `eventId` 或 `runId+seq+subIndex` 去重。
- payload 只放事件数据，不放 prompt、token、密钥、完整工具参数；大对象存 artifact，消息只带引用。
- 事件类型与 AG-UI 公开事件分离：队列事件是内部协议，不能直接透传前端。

### 3.4 Stream / consumer 建议

| Stream | 生产者 | 消费者 | 说明 |
|---|---|---|---|
| `agenthub.run.events` | outbox relay | Gateway projection worker、replay indexer | 运行事件 |
| `agenthub.approvals` | Orchestrator HITL | 审批通知/审计 | 不能替代 SQL approval |
| `agenthub.tool.results` | Tool Broker | ledger reconciler | unknown 仍需人工/对账 |
| `agenthub.dlq` | 消费者重试耗尽 | 运维/补偿任务 | 保存原始 eventId 与错误类别 |

消费者组按职责拆分；每个 consumer 有独立 offset/PEL；ack 只能在业务事务提交后执行。

### 3.5 失败语义

- at-least-once：允许重复投递，不允许丢消息。
- 消费失败：`XAUTOCLAIM` 重试；超过 `maxAttempts` 进 DLQ；DLQ 不自动重放。
- 消费者崩溃：未 ack 消息留在 PEL，由其他实例 claim。
- 重复消费：以幂等键查 SQL/幂等表，重复事件返回原结果。
- 队列不可用：outbox 继续写 MySQL，relay 退避；**不允许**改为同步直写并绕过 outbox。
- 顺序：同一 run 的单消费者串行；跨 run 可并行。
- 可观测：publish/consume/ack/retry/dlq/pending 计数 + traceId 串联。

## 4. HTML / 富交互渲染选型

### 4.1 候选对比

| 能力 | 候选开源项目 | 结论 |
|---|---|---|
| 流式 Markdown | **Streamdown**（Vercel，面向 AI streaming；处理未闭合代码块/数学/CJK）；现用 `react-markdown` + 自定义分块 | 先做 flag 试点；license 以 GitHub LICENSE 复核；不直接替换现有渲染，保留回滚 |
| 交互式代码/HTML/React 预览 | **Sandpack**（CodeSandbox；React 包，Nodebox 隔离）；`react-live` 更轻但能力有限 | P7 先支持单文件 HTML/CSS/JS；React/TSX 需评估打包器与 CSP；沙箱 iframe 不允许同源 |
| 生成式 UI / agent 前端栈 | **CopilotKit**（开源，支持 AG-UI；generative UI） | 只做兼容性评估；当前已有 AG-UI + 自定义事件，不整体替换聊天栈 |
| 声明式表单 | **react-jsonschema-form (RJSF)**（JSON Schema → React 表单） | 用于 `form_input`/工具参数；schema 大小与字段白名单受限 |
| 图表 | **Recharts**（MIT 候选；React 原生）与 **Vega-Lite**（声明式 spec，BSD-3 候选） | 普通图表用 Recharts；模型生成图表用 Vega-Lite JSON，禁执行任意 JS |
| 代码编辑 | **CodeMirror 6**（MIT 候选） | 用于 artifact 编辑/对照；按需懒加载 |
| 现有基础 | Shiki、diff2html、Mermaid、KaTeX、DOMPurify | 保留，不在 P7 重复替换 |

许可证说明：上表“候选”不是最终许可结论；进入交付前必须逐项核对 `LICENSE`，AGPL/SSPL 未经评审不得引入（与 `00-INDEX.md` 选型原则一致）。

### 4.2 推荐决策（D15/D16）

**D15：渲染保留 `react-markdown` 作为基线，Streamdown 以 feature flag 试点；**
**D16：交互能力先做“沙箱化单文件 HTML + 白名单 JSON 表单/图表”，再评估 Sandpack React/TSX。**

交互 artifact 的统一协议（P7-UI-06 冻结）：

```json
{
  "type": "ui.component",
  "component": "html_sandbox|json_form|chart|code_editor",
  "props": {},
  "schemaVersion": 1,
  "artifactId": "art_...",
  "capabilities": ["read_only"]
}
```

规则：

- `component` 必须命中前端 registry 白名单；未知组件降级为文本卡片。
- 禁止 `eval`、`new Function`、动态 `import()` 任意 URL。
- HTML 在 `iframe sandbox="allow-scripts"`（**不加 `allow-same-origin`**）+ CSP 中运行；父子消息使用 `postMessage` 且校验 origin。
- 交互事件通过 AG-UI `CUSTOM`/内部事件回传，不能直接访问 Gateway/Orchestrator 内部接口。
- 图表使用 JSON spec + 数据上限；表单使用 JSON Schema + 字段白名单；代码编辑器只编辑文本，不自动执行。
- 重组件全部动态加载；单组件 chunk 设预算，首屏不引入 Sandpack/图表库。

## 5. 原子任务（P7 提案，未实施）

这些任务**尚未写入执行证据**，不得计入 A0–A6 完成度。

### 5.1 消息队列

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| P7-MQ-01 | 冻结 EventBus 封套/stream/consumer 契约 | `docs/contracts/event-bus.md`、schema | 字段、idempotency、DLQ、顺序、安全规则明确 | G0/G1 |
| P7-MQ-02 | Redis Streams adapter + outbox relay | `pkg/runtime/eventbus`、orchestrator relay | outbox→stream→consumer 可重放；队列失败不丢 outbox | A1-09b |
| P7-MQ-03 | 幂等消费者 + ack/claim | consumer wrapper、SQL 幂等表 | 重复投递只产生一次业务效果；崩溃后 XAUTOCLAIM | P7-MQ-02 |
| P7-MQ-04 | 重试/DLQ/pending 指标 | eventbus、metrics | 超限进 DLQ；pending/retry/dlq 可查 | P7-MQ-03 |
| P7-MQ-05 | Gateway projection / Orchestrator 新异步消费试点 | projection、handler | 只在非权威路径启用；SQL 仍可恢复 | P7-MQ-03 |
| P7-MQ-06 | 故障/负载验收 | tests、runbook | consumer kill、Redis 重启、重复发布、积压恢复均有记录 | P7-MQ-04 |
| P7-MQ-07 | NATS JetStream 第二适配器评估 | `pkg/runtime/eventbus` | 同接口可切换；仅在阈值触发后实施 | P7-MQ-01 |

### 5.2 富交互渲染

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| P7-UI-01 | Streamdown flag 试点 + 回滚 | `StreamingText.tsx`、package | 未闭合 Markdown/代码块/CJK 不崩；关闭 flag 回退旧渲染 | A5-08 |
| P7-UI-02 | 沙箱 HTML artifact | `SandboxedHtml.tsx`、iframe CSP | 脚本不能读父页/同源存储；外网请求受控；XSS 样例 0 执行 | A5-07 |
| P7-UI-03 | 交互式代码预览（Sandpack 评估） | `InteractiveCodePreview.tsx` | 首屏不加载；单文件 HTML/JS 可运行；React/TSX 需单独评审 | P7-UI-02 |
| P7-UI-04 | JSON Schema 表单 | `JsonSchemaForm.tsx`、RJSF | schema/字段白名单；提交走工具输入校验 | A5-07 |
| P7-UI-05 | 声明式图表 | `ChartCard.tsx`、Recharts/Vega-Lite | 数据/spec 上限；无任意 JS；懒加载 | A5-07 |
| P7-UI-06 | AG-UI 交互组件 registry | `agui/skills.ts`、事件契约 | 未知组件降级；组件事件 schema 测试 | P7-UI-02..05 |
| P7-UI-07 | 性能/可访问性验收 | frontend build、Lighthouse | 首屏 gzip 预算、重组件 lazy、键盘可操作 | P7-UI-01..06 |

## 6. G7 门禁（提案）

- **消息**：重复发布、consumer 崩溃、Redis 重启、DLQ 超限均有测试；不丢 outbox；SQL 仍可恢复。
- **顺序**：同 run 顺序可验证，跨 run 可并行；不承诺全局顺序。
- **安全**：事件不含 secret/prompt/token；HTML/表单/图表 XSS 样例 0 执行；iframe 无同源逃逸。
- **性能**：重组件不进入首屏；首屏 gzip ≤500KB；交互包按需加载。
- **许可**：所有新增依赖的 LICENSE 复核记录；AGPL/SSPL 未经评审不得进入。

## 7. 调研来源（2026-11-02 访问）

| 项目/主题 | 来源 | 用途 |
|---|---|---|
| NATS JetStream | https://docs.nats.io/concepts/jetstream | durable stream、at-least-once、replay |
| nats.go | https://github.com/nats-io/nats.go | Go client / JetStream API |
| Redis Streams | https://redis.io/docs/latest/develop/data-types/streams/ | consumer group、PEL、XACK/XAUTOCLAIM、trim |
| RabbitMQ quorum queues | https://www.rabbitmq.com/docs/quorum-queues | quorum/stream 语义与运维代价 |
| Streamdown | https://streamdown.ai/ ；https://vercel.com/changelog/introducing-streamdown | AI streaming Markdown |
| Sandpack | https://sandpack.codesandbox.io/ | 浏览器内交互式代码预览与隔离 |
| CopilotKit | https://github.com/copilotkit/copilotkit ；https://www.copilotkit.ai/blog/the-developer-s-guide-to-generative-ui-in-2026 | AG-UI / generative UI 兼容性评估 |
| react-jsonschema-form | https://github.com/rjsf-team/react-jsonschema-form | JSON Schema 表单 |
| Vega-Lite | https://vega.github.io/vega-lite/ | 声明式图表 spec |
| Recharts | https://recharts.org/ | React 图表候选 |
| CodeMirror | https://codemirror.net/ | 代码编辑候选 |

许可证与版本需在执行 P7-MQ-01/P7-UI-01 前逐项复核；本表不构成许可结论。

## 8. 验证方式（提案，尚未执行）

```bash
# 消息队列（Redis Streams）
# 1) 启动现有 Redis 7.2；不得依赖未评审的 Redis 8
# 2) 同一 event 发布两次，断言业务只执行一次
# 3) consumer 在 ack 前 kill，重启后 XAUTOCLAIM 重新投递
# 4) 超过 maxAttempts 后进入 DLQ，且不自动重放
# 5) outbox 写入失败/Redis 不可用时，relay 退避且 SQL 仍可恢复

# 前端
# 1) Streamdown flag 开关对比；旧渲染回滚
# 2) iframe sandbox XSS/父页访问样例必须失败
# 3) Sandpack/图表/编辑器不在首屏 chunk
# 4) 首屏 gzip ≤500KB；组件懒加载
```

## 9. 明确不做的范围

- 不引入 Kafka/Redpanda、Temporal、Service Mesh；
- 不把消息队列设为唯一事实源；
- 不允许在 HTML artifact 内执行未白名单脚本或访问内部接口；
- 不整体替换现有 AG-UI/聊天栈为 CopilotKit；
- 不在 A0–A6 证据表中把 P7 任务写成已完成。
