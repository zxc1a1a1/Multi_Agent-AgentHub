# AgentHub v3 架构文档索引

- 版本：v3（第三版修订）
- 状态：**草案，待评审**。本目录只描述“现状事实 + 目标设计 + 实施路径”，不代表任何提案已实现。
- 代码基线：`dev / 18b57df`（2026-10-31 复核）
- 核对方式：逐条对照源码与测试；外部平台能力只引用官方文档，并在文中标注“待接入时复核”。
- 配套文档：`docs/contracts/` 仍是协议事实源；本目录不替代契约，出现冲突时以契约为准并回写本文。

## 0. 阅读约定

### 0.1 事实等级

每篇文档中的结论必须带等级，禁止把提案写成现状：

| 等级 | 含义 | 证据要求 |
|---|---|---|
| `已实现` | 生产代码路径可运行，且有测试或运行记录 | 给出文件与行号 |
| `部分实现` | 代码存在，但未接线、仅测试使用、或只在单条路径生效 | 说明“谁没有调用它” |
| `未实现` | 只有设计或接口，没有可运行路径 | 明确标注为提案 |
| `未验证` | 没有本次可复现的证据 | 不得在验收中算通过 |

### 0.2 声明强度

- 有源码但无调用点 → `部分实现`，不能写成“已接入”。
- 有测试但默认 SKIP → 必须写明触发条件，不能算 CI 通过。
- 只有文档链接 → `未验证`，不能写成“官方推荐”。
- 历史观测值 → 必须带日期与测量方法；环境变化后重新测。

## 1. 结论摘要

1. **演示链路不需要新增中间件**：MySQL 权威存储、Redis outbox relay、gRPC 内部面、AG-UI SSE 公开面已能支撑单机演示。
2. **生产化缺四类能力**：对象存储、可观测性、强隔离沙箱、统一限流与密钥治理；工作流引擎属于条件性缺口。
3. **不建议现在引入 Kafka、Service Mesh、向量数据库**：缺少容量与故障数据，收益无法证明。
4. **前端瓶颈是状态耦合与全量渲染**，不是框架选型；优先拆状态层、做虚拟化，再谈替换组件。
5. **企业微信/飞书应作为 Gateway 的外部渠道适配器**：先做通知推送，再做事件回调与审批卡片；两者都未实现。
6. **外部编码 Agent 只能作为可选 A2A 子 Agent**：默认关闭，需合规、沙箱、预算三项同时满足。
7. **兜底必须 fail-closed**：规划失败不得回退到关键词/规则路由，工具结果未知不得自动重做。
8. **本轮修订的重点是“去夸大”**：上一版把若干“仅有接口/仅测试”的能力写成了已具备，本版逐条更正，见第 2 节。
9. **群聊不是空白**：前端已能多选已注册 Agent、派生 `group_chat`、审批群计划、渲染多轮 `AGENT_TURN`；但群成员不持久化、执行串行、历史会合并成一条消息。
10. **10 个 Agent 的工具大多是占位实现**：返回固定结论或模板，不接真实项目/扫描器；平台真实工具与 Tool Broker 尚未接线。
11. **没有 markdown-agent**：document-agent 承担文档模板生成；前端无 Mermaid/KaTeX/PDF 导出。
12. **多节点被 Orchestrator 内存状态阻塞**：审批、取消、幂等、运行任务引用都在进程内；Gateway 扩副本可分担新请求，但在途投影崩溃后需投影 worker；单服务器推荐“单机单副本 + 垂直扩容”。
13. **路由可借鉴 DSH/AutoGen/LangGraph，但首版不做自由 swarm**：先补路由审计、DAG/并行、群聊发言者选择，再评估受控 handoff。
14. **消息队列与富交互渲染是 P7 提案，不是现状**：先冻结 EventBus/组件白名单，再以 Redis Streams 7.2 + 沙箱 HTML 试点；Kafka/Redpanda/Temporal 不引入。

## 2. 关键事实与更正

说明：C1-C10 是第一版 v3 中写错或写强的结论；C11-C16 是本轮新增调研容易误判的点。

| # | 常见误判 / 上一版写法 | 源码事实 | 影响 |
|---|---|---|---|
| C1 | “写工具必须经过 Tool Broker（SQL 审批/预算/fencing）” | Broker 已补预算保留/失败语义与单元测试，但生产 Agent 仍未接线（A2-02） | G2 未通过；写工具未开放 |
| C2 | “Gateway-Orchestrator 使用独立内部 token” | HTTP 面有 `Authorization: Bearer` 校验；**gRPC 面 `grpc.NewServer()` 无拦截器、客户端不传 metadata** | gRPC 当前无鉴权，02 列为 P0 缺口 |
| C3 | “公开事件 `(runId, seq, subIndex)` 稳定” | Gateway 分配 `publicSeq`，`SubIndex` 硬编码 0（`httpapi/server.go:897-905`），当前无分片生产者 | 契约允许分片，现状只有单分片；02/03 注明 |
| C4 | “审批由 Orchestrator 账本一次性消费” | 计划级审批已落 SQL（`run_signals`/`hitl_confirmations`）；工具级审批仍由 Agent Guard 校验，未接 SQL `approvals` | A2-02 未完成，两类审批语义仍分开 |
| C5 | “账本含 task_executions” | 表已建（`001_execution_ledger.sql:28`），全仓无写入代码 | 03 标注为未使用表 |
| C6 | “租约带 TTL 与 fencing，可接管” | 已补周期续租与完成释放（A1-02/A1-10）；仍保留 15 分钟 TTL 默认值；`fencing_version=0` 在 Guard 层被拒 | 长跑不再失租；多实例接管有测试 |
| C7 | “Redis Streams 实时投递事件” | relay 每 250ms 发布；**仓库内无 Redis Stream 消费者**；gRPC `SubscribeRun` 轮询 SQL（200ms），Gateway tail SSE 轮询 SQL（250ms） | 02/03 明确“SQL 轮询是现状” |
| C8 | “刷新恢复已在真模型浏览器验证” | `frontend/src/stores/resume.test.ts` 全部 mock `services/api` 与 `agui/client`；真模型浏览器记录见 `docs/reports/p6-real-model-deepseek.md`，但未覆盖 Flash 全路径 | 03 拆分“单测证据/真机证据” |
| C9 | “测试覆盖断连、回放、unknown” | 关键集成测试仍依赖 DSN/Redis；本机 SKIP。已新增 `.github/workflows/v3-integration-gates.yml` 注入 service 且 SKIP 即失败，但尚未在 GitHub Actions 运行 | CI 绿色 ≠ 集成通过；本机仍无集成证据 |
| C10 | “Gateway 已有 `/api/integrations/*`” | 路由表中不存在，仅本目录提案 | 08/09 标注为新增入口 |
| C11 | “群聊只是前端占位/未实现” | 已实现 AgentPicker 多选、`group_chat` 路径、GroupPlanCard、AGENT_TURN 渲染；但 MySQL 路径 `conversation_type` 固定写 `single`，成员不持久化，历史合并成一条消息 | 13 区分“已实现/部分实现” |
| C12 | “10 个 Agent 都有真实工具” | 多数工具返回固定字符串/模板；document-agent 注释明确 Mock；code-agent 默认不加载工具 | 14 逐个标注占位 |
| C13 | “动态 Agent 注册未做” | 已实现：AgentCard 拉取、`agent_catalog` 持久化、启停/健康检查、前端注册面板；但 AgentPicker 未过滤禁用/不健康 | 13 记录为已实现+小缺口 |
| C14 | “ordered_parallel 是真并行” | `sequential` 已切换 DAGExecutor（波次并发+依赖）；`ordered_parallel` 仍走 SerialExecutor 以保持输出顺序 | 17 的并行优化仍对 ordered_parallel 成立 |
| C15 | “已有 Markdown/Mermaid 渲染” | 前端只有 react-markdown+remark-gfm；无 Mermaid、KaTeX、导出；无 markdown-agent | 15/16 给出选型 |
| C16 | “多节点只需多起副本” | 审批/取消/幂等/运行引用在 Orchestrator 内存；制品写临时文件；单服务器扩 Orchestrator 会 404/取消无效 | 12 给出单机/多节点方案 |

## 3. 文档地图

| 编号 | 文件 | 回答的问题 | 主要读者 |
|---|---|---|---|
| 01 | `01-overall-architecture.md` | 模块边界、数据所有权、部署形态、演进顺序 | 全员 |
| 02 | `02-service-connectivity.md` | 谁和谁说话、协议/端口/鉴权、事件游标、联通验收 | 后端/前端/运维 |
| 03 | `03-link-failure-and-recovery.md` | 断在哪里、如何恢复、幂等键、测试证据等级 | 后端/测试/运维 |
| 04 | `04-harness-fallback.md` | 哪些兜底禁止、分层降级、预算/租约/工具语义 | 后端/评审 |
| 05 | `05-middleware-gap-assessment.md` | 现有中间件、缺口优先级、引入条件 | 架构/运维 |
| 06 | `06-frontend-optimization.md` | 前端瓶颈、状态分层、虚拟化、迁移步骤与指标 | 前端 |
| 07 | `07-tooling-selection.md` | 工具、沙箱、MCP、制品、模型工具能力 | 后端/安全 |
| 08 | `08-wecom-integration.md` | 企业微信接入边界、回调、去重、审批 | 后端/产品 |
| 09 | `09-feishu-integration.md` | 飞书接入边界、事件、卡片回调、审批 | 后端/产品 |
| 10 | `10-external-code-agents.md` | Claude Code 等外部 Agent 的 A2A 适配与合规 | 后端/架构 |
| 11 | `11-reusable-components.md` | 可复用组件、许可证、准入清单 | 全员 |
| 12 | `12-deployment-topologies.md` | 多节点定义、单服务器方案、多副本阻塞点 | 运维/后端 |
| 13 | `13-group-chat-and-agent-selection.md` | 注册 Agent 选择、群聊现状、G1/G2 设计 | 前端/后端/产品 |
| 14 | `14-agent-tool-catalog.md` | 10 个 Agent 的工具盘点与补齐顺序 | 后端/安全 |
| 15 | `15-markdown-agent.md` | Markdown Agent、Mermaid/KaTeX/导出选型 | 后端/前端 |
| 16 | `16-frontend-rendering.md` | 代码/Diff/Markdown/网页预览/浏览器渲染 | 前端 |
| 17 | `17-routing-harness-patterns.md` | DSH/Swarm/LangGraph/AutoGen 路由对照与演进 | 架构/后端 |
| 18 | `18-message-queue-and-rich-interactions.md` | P7 提案：Redis Streams/NATS、Streamdown/Sandpack/RJSF/图表与 G7 门禁 | 架构/后端/前端/运维 |

配套实施计划：`docs/plans/v3-atomic-implementation-plan.md`（按最小原子功能拆分，含门禁与验收）。

## 4. 当前事实基线

| 项 | 值 | 证据 |
|---|---|---|
| 语言/构建 | Go 1.26.0；`go.work` 多模块 | `go.work:1` |
| 关系库 | MySQL 8.4（固定 digest） | `docker-compose.new-arch.yml:10` |
| 缓存/事件 | Redis 7.2-alpine（固定 digest） | `docker-compose.new-arch.yml:31` |
| 内部 RPC | gRPC v1.75.1；HTTP 兼容面并存 | `pkg/runtime/go.mod:3,16`；`services/orchestrator/httpapi/server.go:253-267` |
| 服务数 | compose 15 个服务：MySQL、Redis、10 个 Agent、Orchestrator、Gateway、Frontend | `docker-compose.new-arch.yml:9-343` |
| 公开协议 | AG-UI 公开事件 14 类；SSE `POST /api/chat` + `GET /events/stream` | `pkg/runtime/agui/event_types.go:22-37`；`services/gateway/httpapi/server.go:708` |
| 事件持久化 | Gateway `run_public_events(run_id,seq,sub_index,...)`；Orchestrator `orchestrator_public_events` | `services/gateway/store/migrations/002_public_events.sql:1-10`；`grpcapi/server.go:114,129` |
| 规划器 | LLM-only + 一次修复；无规则回退 | `services/orchestrator/planner/main_agent.go:40-95`；`repair.go:9-12` |
| 前端 | React 18.3.1、Zustand 4.5.5、Vite 6.0.3；无 TanStack 依赖 | `frontend/package.json:14-38` |
| 前端产物 | JS 1.4MB（gzip 435KB）+ CSS 49KB（gzip 8.4KB），2026-10-29 构建 | `frontend/dist/assets/`；本次 `gzip -c` 实测 |
| 模型工具 | provider 已支持发送 `tools`；DeepSeek Flash 带 tools 返回 HTTP 400，默认关闭 | `docs/reports/p6-real-model-deepseek.md:22,62` |
| 工具账本 | `toolruntime.Broker` 仅测试引用；ADK Runner 直接执行工具，Guard/Broker 均未接线 | `services/orchestrator/internal/toolruntime/broker.go`；`pkg/adk/runner.go:475` |
| 群聊 | AgentPicker 多选 + `group_chat` + GroupPlanCard + AGENT_TURN；成员不持久化 | `AgentPicker.tsx`；`executionpath.go:17-46`；`store/mysql.go:145-163` |
| Agent 工具 | 10 个 Agent 默认工具多为固定输出；code-agent 需 `CODE_AGENT_ENABLE_TOOLS=true` | `services/agents/*/tools.go`；`code-agent/main.go:51-53` |
| 前端渲染 | react-markdown+remark-gfm；CodePreview 用 hljs；DiffPreview 自定义；WebPreview 严格 iframe | `StreamingText.tsx`；`CodePreview.tsx`；`DiffPreview.tsx`；`WebPreview.tsx` |
| 部署 | 单机 15 服务、各 1 副本；Orchestrator 审批/取消状态在内存 | `docker-compose.new-arch.yml`；`handler_hitl.go:105-106`；`run_task_registry.go:15-27` |

## 5. 选型原则

1. **权威数据只放 MySQL**：Redis、对象存储、缓存都不承担唯一事实。
2. **适配层隔离**：外部渠道、模型、Agent、工具都通过 adapter 进入，不侵入领域代码。
3. **安全默认关闭**：写工具、外网访问、模型工具、外部 Agent 默认 opt-in。
4. **先量化再引入**：没有压测/故障数据，不引入 Kafka、Mesh、向量库。
5. **许可证先审**：AGPL/SSPL 组件未经评审不得进入交付物。
6. **提案必须能给验收命令**：给不出验证方式的设计不进实施排期。

## 6. 如何复核本文档

```bash
# 事实核对（示例）
grep -n "SubscribeRun" pkg/runtime/proto/orchestrator/v1/orchestrator.proto
grep -rn "NewLedgerBroker\|toolruntime" --include=*.go services | grep -v _test
grep -n "ticker\|200 \* time.Millisecond" services/orchestrator/grpcapi/server.go
grep -rn "PathGroupChat\|group_chat" services/orchestrator/internal/executionpath
grep -rn "pendingPlanStates\|runTaskRegistry" services/orchestrator/httpapi services/orchestrator/registry
grep -rn "ToolResult{Content" services/agents/*/tools.go
grep -rn "mermaid\|katex" frontend/src frontend/package.json

# 集成测试（需要真实 MySQL/Redis，否则只能得到 SKIP）
cd services/gateway && AGENTHUB_MYSQL_TEST_DSN='<test-dsn>' go test ./httpapi -run 'ReplaySameRun'
cd services/orchestrator && AGENTHUB_MYSQL_TEST_DSN='<test-dsn>' go test ./internal/toolruntime -run 'Unknown'
cd services/orchestrator && AGENTHUB_REDIS_TEST_ADDR='<test-redis>' go test ./relay -run 'PublishesInOrder'

# 前端
cd frontend && npm test -- --run && npm run build
```

> 命令中的 `<test-dsn>`、`<test-redis>` 只是占位符，必须由执行者从本地环境注入；本文档不记录任何真实连接串或密钥。

## 7. 决策记录（A0-01 冻结，2026-10-31）

用户指令为实现 `docs/plans/v3-atomic-implementation-plan.md`，即接受该计划
第 3 节推荐默认值。以下决策视为已批准；未执行部分在“落地状态”列明。

| ID | 决策 | 采纳值 | 落地状态 | 影响文档 |
|---|---|---|---|---|
| D1 | gRPC 内部鉴权 | 立即加 token interceptor + trace metadata | 已实现（A1-07） | 02、04 |
| D2 | Tool Broker 接线 | 本轮接线，先于真实写工具 | 部分实现：ADK Guard 已强制；SQL 授权入口未接线（A2-02） | 04、07 |
| D3 | 制品存储 | 先本地卷 adapter，再做 S3/MinIO | 未实现（A6-02/A6-03） | 01、05、07 |
| D4 | IM 渠道 | 本计划不实施，另立计划 | 不实施 | 08、09 |
| D5 | 外部编码 Agent | 本计划不实施，另立计划 | 不实施 | 10 |
| D6 | Temporal | 不引入，继续 SQL 账本 + 租约 | 已采纳 | 05 |
| D7 | TanStack Query | 不阻塞本计划，单独前端优化 | 不实施 | 06 |
| D8 | 长连接/入口 | 单机先上 Nginx，SSE 关闭缓冲 | 未实现（A6-01） | 01、08、09 |
| D9 | 群聊 | 先 G1 计划扇出，后 G2 讨论 | G1 进行中（成员持久化已落地） | 13 |
| D10 | Markdown | 新增独立 markdown-agent（确定性工具） | 未实现（A5） | 15 |
| D11 | 前端渲染 | Shiki + diff2html + Mermaid + KaTeX | 未实现（A5-08..A5-10b） | 16 |
| D12 | 部署目标 | 单机单副本优先，A1 完成后再多副本 | 单机 | 12 |
| D13 | 路由 | supervisor + G1；受控 handoff 后置，不做自由 swarm | 仅 supervisor | 17 |
| D14 | 消息队列 | `EventBus` 接口；Redis Streams 7.2 先试；NATS JetStream 第二适配器；不引入 Kafka | P7 提案，未实施 | 18 |
| D15 | 流式 Markdown | react-markdown 基线；Streamdown flag 试点 | P7 提案，未实施 | 18 |
| D16 | 富交互渲染 | 沙箱单文件 HTML + RJSF 表单 + 声明式图表；Sandpack 单独评审 | P7 提案，未实施 | 18 |
| D17 | MQ/UI 安全 | 队列非唯一事实源；禁 eval；iframe 不加 allow-same-origin；组件白名单 | P7 提案，未实施 | 18 |

### 未决项

- D1 的 token 存放方式（环境变量 vs 文件挂载）需在 A1-07 实施时确认。
- D3 的 S3 兼容实现（MinIO/云厂商）需在 A6-03 前确认目标环境。
- D9 的 G2 主持人选择策略需在 A4-05 前确认预算上限。

## 8. 变更记录

| 版本 | 日期 | 内容 |
|---|---|---|
| v3.0 | 2026-10-30 | 初版 12 篇，建立结论与文档地图 |
| v3.1 | 2026-10-30 | 逐篇补充源码证据；更正 C1-C10；新增事实等级、验收命令与决策表 |
| v3.2 | 2026-10-31 | 新增 12-17：部署拓扑、群聊与注册 Agent、Agent 工具目录、Markdown Agent、前端渲染、路由 Harness 对照；更正 C11-C16 |
| v3.3 | 2026-10-31 | A0-01 冻结 D1-D13；A0-02 会话参与者契约；A0-03 路由决策契约；A0-04 制品元数据；A0-05 群聊投影语义；A0-06 工具执行位置与账本语义；A0-07 集成测试 CI 门禁 |
| v3.4 | 2026-11-02 | 新增 18 与 P7 提案：消息队列（Redis Streams/NATS JetStream）、流式 Markdown（Streamdown）、沙箱 HTML/交互组件（Sandpack/RJSF/图表）；新增 D14-D17 与 G7，均未实施 |
