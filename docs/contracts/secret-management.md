# Secret Management

## Secret 来源

Secret 只能来自环境变量、secret manager 或等价机制。

## 禁止位置

Secret 不得出现于：

- Git 仓库
- `.env.example` 的真实值
- OpenAPI 示例
- AgentCard
- Artifact metadata
- 日志
- trace
- debug dump
- Prompt
- 前端事件

## 最小要求

- 最小权限。
- 可轮换。
- 可审计。
- 泄漏检查。
