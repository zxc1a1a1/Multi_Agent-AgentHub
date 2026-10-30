# AgentHub v3 架构文档索引（草案）

- 日期：2026-10-30
- 状态：**待评审（Draft）**；本文只做架构与选型提案，不代表已实现
- 基线：`dev / 4913747`；Go 1.26、React 18、MySQL 8.4、Redis 7.2、gRPC

## 1. 结论摘要

1. **当前演示栈不缺少“跑起来”的中间件**：MySQL 权威存储、Redis Streams 事件中继、gRPC 内部面、SSE 公开面已经够用。
2. **生产化仍缺四类能力**：对象存储、可观测性、强隔离沙箱、统一限流/密钥治理；工作流引擎属于条件性缺口。
3. **不建议现在引入 Kafka、Service Mesh、向量数据库**：当前事件量和单集群规模不足以证明收益，先保留 Redis Streams + SQL 账本。
4. **前端优先做状态分层与列表虚拟化**，而不是整体重写；TanStack Query/Virtual、Shiki/Monaco 按需引入。
5. **企业微信、飞书应作为 Gateway 的外部渠道适配器**，先做 webhook 推送，再做事件回调与审批卡片。
6. **Claude Code 等外部编码 Agent 应封装为可选 A2A 子 Agent**，通过 sidecar/CLI 或 Agent SDK 适配，不能让浏览器直连。
7. **Harness 兜底是 fail-closed + 明确降级**，不得回退到关键词/规则路由。

## 2. 文档地图

| 编号 | 文件 | 内容 | 主要读者 |
|---|---|---|---|
| 01 | `01-overall-architecture.md` | 目标架构、模块边界、部署形态 | 全员 |
| 02 | `02-service-connectivity.md` | 联通协议、端口、鉴权、时序 | 后端/前端/运维 |
| 03 | `03-link-failure-and-recovery.md` | 断链、重试、恢复、联调矩阵 | 后端/测试/运维 |
| 04 | `04-harness-fallback.md` | Harness 兜底与禁止回退边界 | 后端/评审 |
| 05 | `05-middleware-gap-assessment.md` | 中间件盘点与缺口判断 | 架构/运维 |
| 06 | `06-frontend-optimization.md` | 前端优化与组件选型 | 前端 |
| 07 | `07-tooling-selection.md` | 工具、MCP、沙箱、检索选型 | 后端/安全 |
| 08 | `08-wecom-integration.md` | 企业微信接入方案 | 后端/产品 |
| 09 | `09-feishu-integration.md` | 飞书接入方案 | 后端/产品 |
| 10 | `10-external-code-agents.md` | Claude Code 等外部编码 Agent 接入 | 后端/架构 |
| 11 | `11-reusable-components.md` | 可复用组件与许可证核验清单 | 全员 |

## 3. 当前事实基线

| 项 | 当前值 | 证据 |
|---|---|---|
| Gateway 存储 | MySQL 8.4 权威；内存/SQLite 仅测试 | `docker-compose.new-arch.yml`、`services/gateway/store` |
| Redis | 7.2；outbox relay + Memory 缓存 | `services/orchestrator/relay`、`cache_redis.go` |
| 内部协议 | gRPC start/subscribe/control；HTTP 兼容 | `pkg/runtime/proto/orchestrator/v1` |
| 公开协议 | AG-UI SSE；`(runId, seq, subIndex)` 游标 | `docs/contracts/agui-events.md`、`event-replay.md` |
| 规划器 | LLM-only，无规则回退 | `services/orchestrator/planner/main_agent.go` |
| 前端 | React 18、Zustand、Vite、自定义 SSE | `frontend/package.json`、`frontend/src/agui/client.ts` |
| 模型 | DeepSeek Flash `deepseek-flash` | `docs/reports/p6-real-model-deepseek.md` |
| 工具 | Tool Broker/Opt-in code-agent tools；Flash 带 tools HTTP 400 | `pkg/runtime/tools`、`services/agents/code-agent/server.go` |

## 4. 选型原则

1. **权威数据只放 MySQL**；Redis、对象存储、缓存都不承担唯一事实。
2. **协议适配层可替换**：外部渠道、模型、Agent、工具都通过 adapter 隔离。
3. **安全默认关闭**：写工具、外网访问、模型工具、外部 Agent 默认 opt-in。
4. **先量化再引入**：没有压测/故障数据，不引入 Kafka、Mesh、向量库。
5. **许可证与运维成本先审**：AGPL/SSPL 组件不得未经评审进入交付。

## 5. 官方来源（访问于 2026-10-30）

- AG-UI：https://docs.ag-ui.com/concepts/architecture
- MCP：https://modelcontextprotocol.io/docs/2026-07-28/getting-started/intro
- 企业微信回调：https://developer.work.weixin.qq.com/document/path/90930
- 企业微信接收消息：https://developer.work.weixin.qq.com/document/path/100719
- 企业微信消息推送（群机器人）：https://developer.work.weixin.qq.com/document/path/91770
- 飞书自定义机器人：https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
- 飞书卡片回调：https://open.feishu.cn/document/uAjLw4CM/ukzMukzMukzM/feishu-cards/handle-card-callbacks
- Claude Agent SDK：https://code.claude.com/docs/en/agent-sdk/overview
- OpenAI function calling：https://developers.openai.com/api/docs/guides/function-calling
- Anthropic tool use：https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview
- OpenTelemetry：https://opentelemetry.io/docs/
- Temporal：https://docs.temporal.io/
- gVisor：https://gvisor.dev/docs/
- TanStack Virtual：https://tanstack.com/virtual/latest/docs/introduction
- MinIO：https://min.io/

## 6. 评审问题

- 生产部署是单机持久卷还是多节点对象存储？
- 企业微信/飞书是否需要“接收消息 + 卡片审批”，还是只做通知推送？
- Claude Code 是否允许外发代码/仓库内容？预算与合规边界是什么？
- 是否接受 Temporal 作为新的运维依赖，还是继续使用 SQL 账本 + lease？
