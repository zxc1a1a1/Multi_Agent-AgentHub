# Schema Validation Policy

Schema validation 是跨服务执行前的门禁。

## 必须校验

- OrchestrationPlan。
- PlannerInput。
- Gateway-Orchestrator Request / Result。
- Stream Event。
- AgentCard / Registry Summary。
- Artifact Metadata。
- ToolCall Args。
- LLM Request / Response。

## 失败处理

schema 校验失败不得进入真实执行。必须返回 safe error，并记录 requestId / runId / errorCode。
