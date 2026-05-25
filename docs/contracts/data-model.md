# AgentHub 数据模型契约

## 目的

本文定义 AgentHub v1.0 的数据模型事实源。

## 当前 Profile

```text
profile = v1.0-generic-persistence
storage = MySQL 8
supports = 2+ agents, single/group conversations, runs, steps, tasks, artifacts, tool calls
```

## 核心实体

| 表 | 说明 |
|---|---|
| users | 用户 |
| conversations | 会话 |
| conversation_participants | 会话参与者 |
| messages | 消息 |
| agents | Agent 注册信息 |
| agent_health_checks | 健康检查 |
| runs | 一次运行 |
| run_steps | 运行步骤 |
| agent_tasks | 子 Agent 任务 |
| tool_calls | 工具调用 |
| artifacts | 产物 |

## 关系

- User 1:N Conversation。
- Conversation 1:N Message。
- Conversation 1:N Run。
- Conversation 1:N ConversationParticipant。
- Agent 1:N AgentHealthCheck。
- Run 1:N RunStep。
- Run 1:N AgentTask。
- Run 1:N Message。
- Message 1:N Artifact。
- Message 1:N ToolCall。
- Artifact 1:N ToolCall 可选。

## 通用字段

推荐所有核心表具备：

```text
id
created_at
updated_at
```

用户可见历史表建议具备：

```text
deleted_at
```

## 状态分层

### Agent 状态分离

- `agents.status`：生命周期/启用状态，枚举 `enabled | disabled | experimental | deprecated`。
- `agents.health`：当前健康状态，由 Registry 周期性探测 `/health` 后归一化写入，枚举 `healthy | degraded | unhealthy | unknown`。
- `disabled` 属于 `agents.status`，不属于 `agents.health`。
- `agent_health_checks.status`：归一化后健康检查记录状态，枚举 `healthy | degraded | unhealthy | unknown`。

### Run 状态分层

- `runs.status`：粗粒度生命周期状态，只使用 `accepted | running | completed | failed | cancelled`。不得写入细粒度阶段。
- `runs.phase`：可选细粒度当前阶段（如 `planning`、`dispatching`、`aggregating`），用于展示运行细节。
- `run_steps.step_type`：持久化详细步骤类型（如 `planning | dispatch | agent_call | tool_call | artifact | retry | fallback | aggregate`），用于审计和排障。
- `STATE_UPDATE.state.phase`：前端 SSE 事件中的阶段提示，不持久化。

## 命名

- 数据库字段使用 snake_case。
- API 字段使用 camelCase。
- 表名使用复数 snake_case。
- JSON 字段必须有结构说明。
