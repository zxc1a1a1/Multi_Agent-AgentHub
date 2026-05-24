# AG-UI Error Redaction

来源：`docs/contracts/agui-events.md`、`docs/contracts/security-boundaries.md`。

`RUN_ERROR` 对外不得泄漏：

- stack trace
- token / API key
- 数据库连接串
- 内部地址 / 内部路径
- 完整 system prompt

建议字段：

- `type`
- `runId`
- `code`
- `message`
