# Retry / Rate Limit / Timeout Policy

LLM 调用必须具备可靠性边界。

## Timeout

- 每个请求必须有 timeout。
- timeout 必须小于调用方整体 deadline。
- context cancelled 后必须停止请求。

## Retry

- retry 必须有最大次数。
- 401 / 403 通常不可 retry。
- 429 / 5xx 可按策略 retry。
- 用户取消后不得 retry。

## Rate Limit

- 应支持请求数、token 数和并发数限制。
- rate limit 错误必须被归一化为可理解错误。
