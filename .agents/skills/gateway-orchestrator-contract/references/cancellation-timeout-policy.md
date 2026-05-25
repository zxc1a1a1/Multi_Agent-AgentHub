# 取消与超时规则

Gateway 与 Orchestrator 分进程后，取消必须跨网络传播。

## Gateway 必须

- 浏览器断连时取消本地 context。
- 关闭 Orchestrator 内部 stream 请求。
- 如有 runId，调用 cancel endpoint 或等价机制。
- timeout 后不再写普通事件。
- 释放连接和 goroutine。

## Orchestrator 必须

- 检测内部请求断开。
- 收到 cancel 后停止未完成 task。
- 不再启动新 task。
- 将取消信号传递给下游调用。
- 结束 run，并返回 cancelled 或 failed。

## 推荐 Endpoint

```text
POST /internal/orchestrator/runs/{runId}/cancel
```

## 禁止

- 取消后继续输出 message_delta。
- 取消后继续调用新 Agent。
- timeout 后让 stream 悬挂。
- 泄漏 goroutine。
