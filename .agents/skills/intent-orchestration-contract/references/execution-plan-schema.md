# ExecutionPlan Schema 规则

## 1. 目的

本文定义 ExecutionPlan 的结构化规则。

ExecutionPlan 是 Orchestrator 执行计划的唯一结构化表达。

任何 plan 必须先通过 schema validation，才能被执行。

## 2. MVP 最小结构

MVP 阶段只允许 `single` 策略。

示例：

```json
{
  "version": "v0.1",
  "strategy": "single",
  "reason": "Direct routing to selected code-agent.",
  "tasks": [
    {
      "id": "task_1",
      "agent": "code-agent",
      "skill": "code_generate",
      "input": {
        "message": "帮我写一个 Go HTTP 服务器"
      },
      "dependsOn": [],
      "requiredArtifacts": ["code"]
    }
  ],
  "fallback": null
}
```

## 3. 长期字段

长期 ExecutionPlan 可包含：

| 字段 | 含义 |
|---|---|
| `planId` | plan ID |
| `runId` | run ID |
| `conversationId` | 会话 ID |
| `messageId` | 用户消息 ID |
| `traceId` | 链路追踪 ID |
| `version` | plan schema 版本 |
| `strategy` | `single` / `parallel` / `sequential` |
| `reason` | 生成计划的原因 |
| `tasks` | 任务列表 |
| `aggregation` | 聚合规则 |
| `fallback` | fallback 配置 |
| `approvalRequired` | 是否需要审批 |
| `createdAt` | 创建时间 |

## 4. Task 字段

每个 task 必须包含：

| 字段 | 含义 |
|---|---|
| `id` | task ID |
| `agent` | 目标 Agent 名称 |
| `skill` | 目标 Agent skill |
| `input` | 任务输入 |
| `dependsOn` | 依赖 task ID 列表 |
| `requiredArtifacts` | 期望产物类型 |

## 5. 校验规则

必须校验：

- `strategy` 是否支持。
- `tasks` 是否非空。
- `task.agent` 是否存在于 Agent Registry。
- `task.skill` 是否存在于目标 AgentCard.skills。
- `dependsOn` 是否引用存在的 task。
- `parallel` 任务不得存在非法依赖。
- `sequential` 任务依赖必须形成 DAG。
- `requiredArtifacts` 是否为已知 Artifact type。

## 6. 禁止事项

不得：

- 执行自然语言 plan。
- 执行 schema validation 失败的 plan。
- 执行包含未知 Agent 的 plan。
- 执行包含未知 skill 的 plan。
- 让 LLM planner 输出绕过 schema validation。
