# 服务拓扑

## 基础拓扑

AgentHub 本地交付基础服务包括：

```text
frontend
gateway
mysql
one or more child-agent services
```

## 可选服务

可选服务包括：

```text
mock-llm
mock-agent
redis
object-storage
observability
reverse-proxy
```

## 服务通信规则

- 容器之间通过 Compose service name 通信。
- 宿主机访问容器才使用 `localhost`。
- Agent 服务优先只在 Compose 网络内部暴露。
- Gateway 通过环境变量或配置文件发现 Agent。
- 可选服务不应破坏默认启动路径。

## 示例

容器内：

```text
http://gateway:8080
http://some-agent:8083
```

宿主机：

```text
http://localhost:3000
http://localhost:8080
```
