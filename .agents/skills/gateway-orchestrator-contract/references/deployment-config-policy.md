# 部署配置规则

Gateway 与 Orchestrator 分进程后，必须通过配置建立连接。

## Gateway 环境变量

```text
ORCHESTRATOR_URL=http://orchestrator:8090
ORCHESTRATOR_INTERNAL_TOKEN=change-me
ORCHESTRATOR_TIMEOUT_MS=120000
```

## Orchestrator 环境变量

```text
ORCHESTRATOR_PORT=8090
INTERNAL_SERVICE_TOKEN=change-me
```

## 规则

- Gateway 只能通过 ORCHESTRATOR_URL 调用 Orchestrator。
- 容器间通信使用 service name。
- 容器间不得使用 localhost 调对方服务。
- localhost 只用于宿主机访问。
- Orchestrator 必须有独立 /health。
- Orchestrator 不得暴露到前端网络边界。
- 示例环境变量不得包含真实 token。
