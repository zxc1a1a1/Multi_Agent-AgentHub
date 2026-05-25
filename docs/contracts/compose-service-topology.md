# Compose Service Topology

## 基础服务

```text
frontend
gateway
mysql
child-agent services
```

## 可选服务

```text
mock-llm
mock-agent
redis
object-storage
observability
reverse-proxy
```

## 规则

- 默认拓扑必须能运行主 Demo。
- 可选服务通过 profile 启用。
- 服务之间通过 service name 访问。
- Agent 服务 service name 供 Orchestrator / Registry 使用，Gateway 不得直接持有。
- Gateway 只通过 `ORCHESTRATOR_URL` 访问 Orchestrator，不直接访问 Child Agent。
- Agent 服务不固定名称。
- 新增 Agent 必须有 healthcheck。
