# Secret Management

来源：`docs/contracts/secret-management.md`、`docs/contracts/security-boundaries.md`。

- `.env` 不提交。
- API Key / Token / DB 密码只来自环境变量或安全密钥管理。
- 不写入代码、日志、AgentCard、示例响应。
- `.env.example` 仅保留占位符。
