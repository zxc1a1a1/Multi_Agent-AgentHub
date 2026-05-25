# Gateway-Orchestrator Test Policy

Gateway 与 Orchestrator 必须作为两个独立服务测试。

## 必测项

- Gateway 通过内部 URL 调用 Orchestrator。
- Gateway 不 import Orchestrator 业务包。
- Orchestrator health 独立可测。
- 内部 stream 可断开、超时、取消。
- `/internal/**` 不出现在公开 OpenAPI。
- requestId / traceId / runId 不丢失。

## 禁止

不得用同进程函数调用冒充分进程集成测试。
