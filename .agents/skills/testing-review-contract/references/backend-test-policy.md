# Backend Test Policy

后端测试必须覆盖纯逻辑、服务边界和失败路径。

## 必测项

- Planner validation。
- Registry filtering。
- Converter。
- Safe error mapping。
- Orchestrator strategy。
- Gateway internal client timeout。
- Cancellation。
- Persistence roundtrip。

## 禁止

单元测试不得启动 Docker Compose，不得依赖真实 LLM，不得使用真实 API key。
