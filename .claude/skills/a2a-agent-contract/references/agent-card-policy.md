# Agent Card Policy

来源：`docs/contracts/a2a-agent-card.md`。

## 1. 端点

- `GET /.well-known/agent.json`（可兼容 `.well-known/agent-card.json` 作为历史路径）。

## 2. 关键字段

- `name`、`description`、`url`、`version`
- `capabilities`、`skills`
- `inputModes`、`outputModes`

## 3. 安全

- AgentCard 不得包含密钥、token、内部敏感配置。
