# 01 总体架构

- 状态：待评审
- 目标：在保留现有可运行链路的前提下，给出可生产化的模块边界与部署演进路径

## 1. 范围

本文覆盖 Frontend、Gateway、Orchestrator、Child Agent、Tool Runtime、Memory、事件与存储。  
不覆盖具体 prompt、模型参数与 UI 视觉稿。

## 2. 目标拓扑

```text
Browser / WeCom / Feishu / API Client
                |
                v
        +----------------+
        |     Gateway    |  HTTP API / SSE / channel adapters
        |  auth, idempot |  conversation & message projection
        +-------+--------+
                | gRPC（内部）
                v
        +----------------+
        |  Orchestrator  |  planner, ledger, approval, budget, resume
        |  harness loop  |  memory controller, relay, tool broker
        +--+----+----+---+
           |    |    |
     A2A   |    |    | MySQL / Redis
           v    v    v
     Child Agents  Tool Sandbox / MCP
     (code, web,   (workspace, docker,
      document...)  web, artifact)
```

## 3. 模块职责

| 模块 | 必须做 | 不得做 |
|---|---|---|
| Frontend | 展示、输入、审批、游标恢复、流式渲染 | 直接调用 A2A/MCP、保存密钥、执行生成代码 |
| Gateway | 鉴权、限流、协议转换、SSE、会话/消息投影、外部渠道适配 | 规划、调度子 Agent、直接执行工具 |
| Orchestrator | 规划、计划校验、审批、预算、派发、账本、恢复 | 持久化会话消息、渲染 UI |
| Child Agent | 在自身沙箱内执行任务、返回事件/制品 | 读取平台密钥、绕过 Broker 执行写操作 |
| Tool Runtime | 工具描述、授权、审批、执行、账本 | 把未授权调用透传给宿主 |
| Memory | 授权范围内的召回、候选/确认、遗忘、epoch | 跨 scope 注入、用缓存替代 SQL 权威 |
| Infra | MySQL 权威、Redis 辅助、对象存储、观测 | 用缓存/消息队列充当唯一事实 |

## 4. 权威数据与派生数据

| 数据 | 权威 | 派生/缓存 | 约束 |
|---|---|---|---|
| 会话、用户消息 | Gateway MySQL | 前端缓存 | 用户消息不可被事件投影覆盖 |
| 运行、计划、审批、工具调用 | Orchestrator MySQL 账本 | Redis Streams | 账本事务先于派发 |
| 公开事件 | SQL 事件日志 | Redis / 浏览器内存 | `(runId, seq, subIndex)` 稳定 |
| Assistant 消息 | Gateway MySQL 投影 | 前端消息列表 | 投影与 offset 同事务 |
| 制品 | 对象存储（待补） | 本地卷（演示） | 元数据入 MySQL，内容不入库 |
| Memory | MySQL | Redis 缓存 | 缓存键含 scope/epoch/policy |

## 5. 部署形态

### 5.1 演示/单机

- 单 compose：MySQL + Redis + Gateway + Orchestrator + 2~10 个 Agent。
- 制品落单机持久卷；接受单点故障。
- 观测只保留结构化日志与健康检查。

### 5.2 生产/多节点

- Gateway、Orchestrator 无状态多副本；Agent 按能力池部署。
- MySQL 主从/托管实例；Redis 集群或托管；对象存储 S3/MinIO。
- 统一入口网关：TLS、限流、WAF、请求体上限。
- OpenTelemetry Collector + Prometheus/Grafana/Loki/Tempo。
- 沙箱节点独立池（gVisor 或 Firecracker），不与控制面混部。

## 6. 关键边界

1. 浏览器只认 Gateway 公共 API，不直连 Orchestrator gRPC、Child Agent A2A、MCP。
2. Gateway 不持有模型密钥；模型调用只发生在 Orchestrator/Agent 侧。
3. 写工具必须经过 Tool Broker：审批、预算、fencing、幂等键、结果账本缺一不可。
4. 外部 IM 渠道只做“消息入口/出口”，不复制编排逻辑。
5. 外部编码 Agent 作为 Child Agent 的一种实现，不进入 Gateway 进程。

## 7. 演进顺序

| 阶段 | 目标 | 前置条件 |
|---|---|---|
| A | 稳定现有单机链路与真实模型 | 当前 P4/P5 门禁 |
| B | 对象存储 + OTel + 限流 + 密钥治理 | 生产部署决策 |
| C | IM 渠道（企业微信/飞书）通知与审批 | Gateway adapter 契约 |
| D | Claude Code 等外部 Agent 可选接入 | 沙箱与合规批准 |
| E | 工作流引擎/向量库按数据决定 | 压测与容量数据 |

## 8. 待评审决策

- 是否接受“Gateway 作为唯一外部入口”的长连接成本。
- 制品是否必须跨节点共享；若是，对象存储为硬门槛。
- 外部 Agent 的代码/仓库外发是否允许，决定 Claude Code 方案是否可行。
