# OrchestrationPlan Contract

## 定义

OrchestrationPlan 是意图编排结果的标准结构。

## 示例

```json
{
  "version": "v1",
  "planId": "plan_001",
  "runId": "run_001",
  "conversationId": "conv_001",
  "planningMode": "auto",
  "strategy": "ordered_parallel",
  "intentSummary": "用户想完成一个多步骤任务",
  "tasks": [
    {
      "taskId": "task_001",
      "agentName": "some-agent",
      "capabilityIds": ["capability_id"],
      "taskContent": "任务描述",
      "dependsOn": [],
      "expectedOutputs": ["markdown"],
      "priority": 1
    }
  ],
  "aggregation": {
    "mode": "message_per_task",
    "summaryRequired": false
  },
  "fallback": {
    "mode": "same_capability_alternative",
    "maxAttempts": 1
  },
  "validation": {
    "schemaVersion": "v1",
    "validated": false
  }
}
```

## 规则

- `tasks` 不得为空。
- `agentName` 必须来自当前可用 Agent 集合。
- `capabilityIds` 必须来自目标 Agent 的能力集合。
- `expectedOutputs` 必须可被目标能力支持。
- `validation.validated` 只能由本地校验器设置。
