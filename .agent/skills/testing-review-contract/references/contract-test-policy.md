# Contract Test Policy

所有跨边界通信必须有契约测试或 schema validation 测试。

## 必测边界

- Frontend ↔ Gateway。
- Gateway ↔ Orchestrator。
- Orchestrator ↔ Child Agent。
- Orchestrator ↔ LLM Provider Adapter。
- Artifact / Runtime Capability。
- Observability / Safe Error。

## 规则

- 每个 contract 至少有 valid fixture 和 invalid fixture。
- Producer 与 Consumer 视角都要能被验证。
- 删除字段必须 Review。
- 新增字段默认 optional。
- mock 数据不得超出 contract。
