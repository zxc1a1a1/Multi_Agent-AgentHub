# Error Redaction

来源：`docs/contracts/security-boundaries.md`、`docs/contracts/security-review-checklist.md`。

对外错误禁止泄漏：

- stack trace
- API key / token
- 数据库连接串
- 内部地址、内部路径

`RUN_ERROR` 与 REST 错误都必须安全降级输出。
