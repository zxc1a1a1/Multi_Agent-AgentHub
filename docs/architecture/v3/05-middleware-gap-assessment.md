# 05 中间件缺口评估

- 状态：待评审
- 结论：**演示链路不缺；生产化缺对象存储、可观测性、强隔离沙箱、限流/密钥治理；工作流引擎条件性引入；Kafka/向量库暂不引入。**

## 1. 现有中间件

| 类别 | 现状 | 用途 | 评价 |
|---|---|---|---|
| 关系库 | MySQL 8.4 | 会话/消息/账本/事件/审批 | 权威存储，继续 |
| 缓存/事件 | Redis 7.2 + Streams | outbox relay、Memory 缓存 | 足够当前规模 |
| 内部 RPC | gRPC 1.75 | Gateway↔Orchestrator 控制/订阅 | 保留，补 live 推送 |
| 公开流 | SSE（HTTP） | 浏览器事件 | 保留，补背压 |
| 容器 | Docker + compose | Agent/沙箱 | 演示可用，生产补隔离 |
| 模型接入 | OpenAI 兼容/Anthropic | DeepSeek/Claude | 已有适配层 |

## 2. 缺口矩阵

| 中间件/能力 | 当前 | 缺口 | 风险 | 建议 | 优先级 |
|---|---|---|---|---|---|
| 对象存储 | 单机卷/内存 artifact | 多节点共享、签名 URL、生命周期 | 制品丢失/不可扩展 | MinIO/S3 适配器 | P0（多节点） |
| 可观测性 | 结构化日志/health | traces、metrics、集中日志 | 故障定位慢 | OpenTelemetry + Prometheus/Grafana/Loki/Tempo | P0 |
| 沙箱隔离 | Docker 限权 | 内核逃逸面、恶意代码 | 宿主风险 | gVisor（或 Firecracker）独立池 | P0（开放写工具后） |
| 入口治理 | Gateway 简单中间件 | 限流、WAF、请求体上限、配额 | 滥用/成本 | API Gateway/Nginx/Envoy | P1 |
| 密钥治理 | 环境变量 | 轮换、审计、最小权限 | 泄露面 | Vault/KMS/云 Secret Manager | P1 |
| 工作流引擎 | SQL 账本+lease | 跨服务长事务/定时/补偿 | 复杂流程维护成本 | 先评估 Temporal | P2（条件） |
| 消息总线 | Redis Streams | 跨集群回放/多租户 | 规模上限 | Kafka 仅在有数据后 | P3（暂不） |
| 向量检索 | 无 | 语义 Memory 召回 | 召回质量 | pgvector/Qdrant 按评测 | P3（条件） |
| MCP 网关 | adapter allowlist | 服务器注册、审计、配额 | 工具蔓延 | MCP Gateway/Registry | P2 |
| IM 渠道 | 无 | 企业微信/飞书适配 | 重复建设 | Gateway adapter | P1 |

## 3. 是否缺少中间件（判断）

### 3.1 当前阶段

- **不缺少跑通多 Agent 的中间件**：MySQL + Redis + gRPC + SSE 足以支撑演示与单机生产。
- **缺少生产保障**：没有 traces/metrics、对象存储、强沙箱、限流与密钥治理。
- **缺少外部渠道接入层**：企业微信/飞书没有 adapter。

### 3.2 不建议现在引入

| 组件 | 不引入理由 | 重新评估条件 |
|---|---|---|
| Kafka | Redis Streams 已满足当前事件量；运维成本高 | 跨集群、回放保留、吞吐达到瓶颈 |
| Service Mesh | 服务数量少，gRPC 直连足够 | 多集群/多团队/细粒度 mTLS |
| Temporal | 现有 SQL 账本+lease 已覆盖核心恢复 | 跨服务补偿、定时器、长流程显著增多 |
| 向量数据库 | Memory 数据量与评测不足 | 召回评测证明收益 |
| 完整 Agent 框架 | 现有 ADK/A2A/AG-UI 边界清晰 | 不自建重复能力时 |

## 4. 推荐演进

### P0（生产前）

1. OpenTelemetry SDK + trace 上下文贯穿 Gateway/Orchestrator/Agent。
2. 对象存储抽象（本地卷与 S3/MinIO 双实现）。
3. 沙箱执行池（gVisor/Firecracker），控制面与执行面隔离。
4. 入口限流、请求体上限、超时、并发配额。

### P1

1. 企业微信/飞书 adapter（通知 → 审批卡片）。
2. 密钥管理接入与轮换。
3. MCP 服务器注册/审计/配额。

### P2

1. Temporal 评估（仅当工作流复杂度/补偿需求成立）。
2. 多租户与配额计费。

### P3

1. Kafka/向量库按数据决定。

## 5. 选型标准

- Go 生态成熟、可观测、可水平扩展。
- 许可证可接受（AGPL/SSPL 需法务评审）。
- 有托管或成熟运维方案。
- 可被 adapter 隔离，不侵入领域代码。
- 引入前必须有压测/故障演练数据。

## 6. 参考来源

- OpenTelemetry：https://opentelemetry.io/docs/
- Temporal：https://docs.temporal.io/
- gVisor：https://gvisor.dev/docs/
- MinIO：https://min.io/
- MCP：https://modelcontextprotocol.io/docs/2026-07-28/getting-started/intro
- AG-UI：https://docs.ag-ui.com/concepts/architecture

## 7. 待补数据

- 当前 MySQL/Redis 容量、P95 延迟、事件保留天数。
- 单 run 事件数、工具调用数、并发 run 数。
- 制品平均大小与增长速率。
- 没有这些数据前，不承诺 Kafka/Temporal/向量库的收益。
