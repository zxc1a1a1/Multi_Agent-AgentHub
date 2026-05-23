# Smoke Test 策略

## 1. 目的

本文定义 AgentHub 本地 Demo smoke test。

## 2. MVP smoke test

必须验证：

```text
docker compose config
docker compose up -d --build
mysql healthy
gateway /health
code-agent /health
frontend reachable
minimal API path
no obvious 500
```

## 3. 推荐步骤

```text
1. 检查 docker compose config。
2. 启动服务。
3. 等待 mysql healthy。
4. 访问 gateway /health。
5. 访问 code-agent /health。
6. 访问 frontend /。
7. 发送最小 API 请求或 mock run。
8. 检查日志中没有明显 panic / fatal / 500。
```

## 4. LLM 规则

CI / smoke test 默认使用 mock LLM 或固定响应。

手动 Demo 可以使用真实 API key，但不得提交 key。

## 5. 禁止事项

不得：

- smoke test 只检查容器启动，不检查服务可用。
- smoke test 依赖真实 LLM 随机输出。
- smoke test 使用真实生产 API key。
- smoke test 失败仍返回 0。
- 用固定 sleep 代替健康检查。
