# Child Agent 服务策略

## 目的

定义新增任意 Child Agent 时，Docker Compose 需要遵守的通用规则。

本文件不固定具体 Agent 名称。

## 必须满足

每个 Child Agent 服务必须：

- 有稳定 service name。
- 有 `build` 或 `image`。
- 有明确内部端口。
- 提供 `/health`。
- 可被 Orchestrator / Registry 通过 service name 访问。
- 不把 secret bake 进镜像。
- 日志能标识 agent name。
- 能被 smoke test 检查。

## 推荐服务模板

```yaml
services:
  some-agent:
    build:
      context: ./agents
      dockerfile: some-agent/Dockerfile
    expose:
      - "8083"
    environment:
      - AGENT_PORT=8083
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8083/health"]
      interval: 10s
      timeout: 5s
      retries: 10
```

## 新增 Agent 必做

新增 Agent 时必须同步：

- Compose 服务定义。
- `.env.example` 中的 Agent URL 或 registry 配置。
- Orchestrator / Registry 可读取的 Agent 配置。
- smoke test。
- Demo 文档。
- healthcheck。

## 禁止

- 新增 Agent 但没有 `/health`。
- 新增 Agent 但没有文档说明。
- 新增 Agent 必须改 Gateway 源码才能发现。
- Agent 服务默认映射过多宿主机端口。
- 把 Agent URL 直接配置给 Gateway。
