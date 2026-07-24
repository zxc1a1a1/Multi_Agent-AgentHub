# AgentHub Test Matrix

本文件定义 AgentHub v1.0 的测试矩阵。测试矩阵不是任务清单，而是质量门禁分类。

## 必须分层

1. Schema / Contract Tests。
2. Unit Tests。
3. Component / Store Tests。
4. Adapter / Converter Tests。
5. Service Integration Tests。
6. Cross-process Integration Tests。
7. Streaming Replay Tests。
8. Failure / Fallback Tests。
9. Security Regression Tests。
10. Docker Smoke Tests。
11. Demo E2E Tests。
12. Documentation Gates。

## v1.0 必测能力

- Gateway 与 Orchestrator 分进程。
- 2+ Agent。
- group conversation。
- LLM Planner 与 fallback。
- Registry / Health Check。
- Artifact / Runtime Capability。
- Streaming / STATE_UPDATE。
- Docker smoke 与 Demo path。

## 禁止

- 只测 happy path。
- 用真实 LLM 替代 fake LLM。
- 把具体 Agent 名称写成长期唯一测试对象。
