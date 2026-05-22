# Retry / Rate Limit / Timeout 规则

每个 Provider 请求必须有 timeout。

timeout 来源优先级：

```text
request.timeoutMs
model.defaultTimeoutMs
provider.defaultTimeoutMs
system default
```

如果 ctx 已取消，必须停止新请求、streaming 读取、retry 和 fallback。

可重试错误包括：

- temporary network error。
- provider overload。
- 429。
- 部分 5xx。
- transient timeout。

不可重试错误包括：

- invalid API key。
- permission denied。
- model not found。
- invalid request schema。
- content policy hard failure。
- context cancelled。

retry 必须有上限，推荐：

```text
maxAttempts: 3
backoff: exponential
jitter: true
```

不得无限 retry。
