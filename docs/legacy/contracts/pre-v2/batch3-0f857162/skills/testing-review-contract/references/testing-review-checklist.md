# Testing Review Checklist

## 事实源

- 是否参考 SPRINT-v1.0-Plan.md？
- 是否把 MVP v0.1 放入 Historical Profile？

## v1.0 能力

- 2+ Agent 是否覆盖？
- group conversation 是否覆盖？
- Planner fallback 是否覆盖？
- Registry / Health 是否覆盖？
- code / webpage / markdown 是否覆盖？
- fallback / retry 是否覆盖？

## 边界

- Gateway / Orchestrator 是否分进程？
- `/internal/**` 是否未进入公开 OpenAPI？
- Gateway 是否没有直接调 Child Agent 或 LLM？

## 安全

- 错误是否脱敏？
- iframe 是否 sandbox？
- fixtures 是否无 secret？
