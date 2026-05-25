# Secret Config Policy

LLM Provider secret 必须安全管理。

## 规则

- API key 只能来自环境变量、secret manager 或等价机制。
- 配置文件只能保存 env var 名称。
- 不得把真实 key 写入 YAML / JSON / Prompt / Artifact / 日志 / 数据库普通字段。
- 不得把用户 token 当 Provider API key。
- 错误信息不得包含 Authorization header。
- trace metadata 不得保存 secret。
