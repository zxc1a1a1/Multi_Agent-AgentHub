# Deployment / Demo Boundary

v1.0 必须能演示完整链路。

## 交付边界

- docker compose 一键启动。
- 服务健康检查。
- smoke test。
- 单聊 Demo。
- 群聊多 Agent Demo。
- 产物预览 Demo。
- 历史消息持久化验证。
- 错误降级提示。

## 服务拓扑建议

演示部署应体现 Frontend、Gateway、Orchestrator、Child Agent、Database 的职责分离。具体 Docker Compose、Dockerfile、环境变量属于部署契约或实现，不在本 Skill 写实现代码。
