# Smoke Test 策略

## 目标

Smoke test 用于验证本地交付是否可用，而不是验证所有业务细节。

## 分层

### Level 0: Compose config

```text
docker compose config
```

### Level 1: Service health

- mysql healthy
- gateway `/health`
- frontend reachable
- enabled child agents `/health`

### Level 2: Registry / API

- Gateway 能列出启用 Agent。
- v1.0 Demo profile 至少 2 个 healthy Agent，除非当前 profile 明确是单 Agent 回归。

### Level 3: Minimal run

- 创建会话。
- 发送最小消息。
- 收到流式响应或 mock 响应。

### Level 4: Demo path

- 单 Agent 对话。
- 多 Agent / 群聊路径。
- 产物预览元数据存在。

## 规则

- 不依赖真实 LLM 随机输出。
- 不输出 secret。
- 失败必须返回非零退出码。
- 失败必须显示失败项。
- 本机和 CI 都应可运行。
