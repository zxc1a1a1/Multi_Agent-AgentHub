# A2A Error Policy

来源：`docs/contracts/a2a-errors.md`、`docs/contracts/security-boundaries.md`。

## 1. 映射链路

```text
A2A failed
→ OrchestratorError
→ AG-UI RUN_ERROR
```

## 2. 错误字段

推荐包含：`code`、`message`、`retryable`。

## 3. 脱敏

不得向前端暴露：

- API key / token
- stack trace
- 内部地址
- 完整敏感 prompt
