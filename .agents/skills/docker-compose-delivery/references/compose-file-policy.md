# Compose 文件策略

## 目的

统一 AgentHub 本地交付所使用的 Compose 文件，避免多个 Compose 文件分叉维护。

## 推荐文件

推荐主文件：

```text
compose.yaml
```

兼容文件：

```text
docker-compose.yml
```

可选覆盖：

```text
compose.override.yaml
compose.dev.yaml
compose.demo.yaml
compose.mock.yaml
```

## 规则

- 主 Compose 文件只能有一个事实源。
- 如果同时存在 `compose.yaml` 和 `docker-compose.yml`，二者不得长期分叉。
- 本地开发差异放入 override 文件。
- Demo 默认启动路径必须清晰。
- 生产部署配置不得混进本地 Compose 主文件。
- 每次修改 Compose 后必须能通过 `docker compose config`。
- 新增服务时必须同步更新文档、环境变量和 smoke test。

## 禁止

- 同时维护两个不同主 Compose 文件。
- 默认 Compose 文件依赖某个人机器的绝对路径。
- 把真实 secret 写入 Compose。
- 使用不稳定的临时容器名作为服务名。
