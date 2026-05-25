# Docker Compose Review Checklist

## 文件

- 主 Compose 文件是否明确？
- 是否能通过 `docker compose config`？
- 是否没有真实 secret？
- 是否没有本机绝对路径？

## 服务

- 是否有 frontend？
- 是否有 gateway？
- 是否有 mysql？
- 是否有 1 个或多个 Child Agent？
- 新增 Agent 是否有 healthcheck？

## 环境

- `.env.example` 是否完整？
- `.env` 是否未提交？
- Dockerfile 是否没有真实 secret？

## 验收

- Makefile 命令是否存在？
- smoke test 是否存在？
- Demo checklist 是否存在？
- reset 是否明确危险？
