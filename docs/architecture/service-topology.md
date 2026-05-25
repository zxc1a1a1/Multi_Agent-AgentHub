# Service Topology

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

服务之间必须保持单向职责：Frontend 不越过 Gateway，Gateway 不越过 Orchestrator 直接编排，Orchestrator 不直接暴露给浏览器。
