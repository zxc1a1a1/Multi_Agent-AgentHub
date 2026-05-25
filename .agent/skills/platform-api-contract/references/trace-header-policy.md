# Trace Header 策略

Frontend 可以传入：

```text
X-Request-Id
X-Trace-Id
```

Gateway 必须接受或生成这些 ID，在 response header 中返回 `X-Request-Id`，并在日志中记录。`traceId` 不能作为鉴权依据。
