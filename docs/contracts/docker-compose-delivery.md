# Docker Compose Delivery Contract

## 目的

本文是 AgentHub 本地 Compose 交付的项目级契约。

它定义：

- Compose 文件策略。
- 服务拓扑。
- Child Agent 服务模式。
- 环境变量和 secrets。
- healthcheck。
- Makefile 命令。
- smoke test。
- Demo checklist。

## 当前交付目标

当前交付目标支持：

- frontend
- gateway
- mysql
- 2+ Child Agent 服务
- 一键启动
- healthcheck
- smoke test
- Demo 稳定运行

## 原则

- 本地 Compose 是开发和 Demo 交付入口。
- 不固定具体 Agent 名称。
- 新增 Agent 必须遵守通用服务模式。
- 不提交真实 secret。
- 所有长期运行服务必须能健康检查。
- Smoke test 不依赖真实 LLM 随机输出。
