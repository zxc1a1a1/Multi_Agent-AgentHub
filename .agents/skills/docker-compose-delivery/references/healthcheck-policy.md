# Healthcheck 策略

## 必须有 healthcheck 的服务

- mysql
- gateway
- frontend
- 每个启用的 Child Agent 服务

## healthcheck 要求

healthcheck 必须：

- 快速。
- 稳定。
- 可重复。
- 不修改业务数据。
- 不调用真实 LLM。
- 不依赖外部公网。
- 不输出 secret。
- 能明确反映服务是否可用。

## Gateway 示例

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
  interval: 10s
  timeout: 5s
  retries: 10
```

## MySQL 示例

```yaml
healthcheck:
  test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
  interval: 5s
  timeout: 5s
  retries: 10
```

## 禁止

- 用真实业务请求替代健康检查。
- 用 LLM 调用判断健康。
- 用固定 sleep 代替 readiness。
- healthcheck 打印敏感配置。
