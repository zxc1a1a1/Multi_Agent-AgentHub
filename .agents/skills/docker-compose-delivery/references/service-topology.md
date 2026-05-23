# 服务拓扑

## 1. 目的

本文定义 AgentHub 本地 Compose 服务拓扑。

## 2. MVP 拓扑

```text
frontend → gateway
gateway → mysql
gateway → code-agent
code-agent → LLM Provider
```

## 3. 容器 DNS

Compose network 内服务名即 DNS 名。

MVP 推荐：

```text
mysql
gateway
code-agent
frontend
```

## 4. URL 约定

```text
DATABASE_URL=mysql://agenthub:agenthub@mysql:3306/agenthub
AGENT_CODE_URL=http://code-agent:8081
GATEWAY_URL=http://gateway:8080
```

前端浏览器访问 Gateway 时，应使用浏览器可访问地址，例如：

```text
http://localhost:8080
```

## 5. 禁止事项

不得：

- 容器之间使用 localhost 连接其他容器。
- 前端浏览器使用容器内部 DNS。
- Gateway 直连前端。
- 前端直连 code-agent。
- code-agent 直连 MySQL，除非 contract 明确允许。
