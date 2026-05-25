# Service Topology

目标拓扑：

```text
Frontend App
  ↓ Public REST / Public SSE or stream
Gateway Service
  ↓ Protected Internal API / Internal stream
Orchestrator Service
  ↓ Agent protocol / Registry / LLM Provider / Artifact normalization
Child Agent Services
  ↓ Attached Resources
Data Layer / Object Storage / Cache / LLM Providers
```

## 规则

- Frontend 只能访问 Gateway。
- Gateway 是外部入口与前端连接层。
- Orchestrator 是内部编排服务。
- Child Agent 是能力提供服务。
- Data Layer、Object Storage、Cache、LLM Provider 是 attached resources。

## 禁止

- Frontend 直连 Orchestrator 或 Child Agent。
- Gateway 直接调用 Child Agent。
- Gateway 直接调用 LLM Provider。
- Orchestrator 直接暴露给浏览器。
