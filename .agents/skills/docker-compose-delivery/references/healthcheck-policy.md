# Healthcheck 策略

## 1. 目的

本文定义服务 ready 判断规则。

## 2. MVP 必需 healthcheck

```text
mysql
gateway
code-agent
```

## 3. MySQL healthcheck

必须验证 MySQL 可连接。

推荐：

```text
mysqladmin ping
```

或等价检查。

## 4. Gateway healthcheck

Gateway 必须暴露：

```text
GET /health
```

至少返回服务存活状态。

正式开发阶段建议检查 DB 连接和 Agent Registry。

## 5. code-agent healthcheck

code-agent 必须暴露：

```text
GET /health
```

至少返回服务存活状态。

正式开发阶段建议检查 AgentCard 和 Runtime 初始化状态。

## 6. 等待规则

smoke test 不得只用 sleep。

必须轮询 health endpoint 或 Docker health status，并设置 timeout。

## 7. 禁止事项

不得：

- 没有 healthcheck。
- 只靠 sleep 等服务 ready。
- healthcheck 永远返回 200 但服务不可用。
- smoke test 不检查 health。
