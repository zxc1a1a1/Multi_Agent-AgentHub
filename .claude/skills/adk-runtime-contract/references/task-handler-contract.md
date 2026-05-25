# task-handler-contract

## 目的

本文定义 Child Agent Task Handler 的通用约束。

## 推荐抽象

```go
type Handler func(ctx *adk.Context, messages []adk.Message) error
```

实际代码类型可以变化，但语义必须保持一致。

## 输入

Handler 输入应是结构化消息数组。

推荐消息字段：

```json
{
  "role": "user",
  "content": "用户请求"
}
```

规则：

- 不应只接收拼接后的全文字符串。
- 可兼容拼接文本，但 Runtime / Orchestrator 内部必须保留结构化消息边界。
- system prompt 应由 Agent 配置或 Handler 明确注入。

## 输出

Handler 只能通过：

```text
ctx.StreamText
ctx.AddArtifact （输出 ArtifactDraft，非 Core Artifact）
ctx.Fail / return error
```

输出任务结果。

## Handler 可以做

- prompt 组装。
- LLM 调用。
- 解析 LLM 输出。
- 构造 ArtifactDraft（不含平台字段）。
- 调用已授权工具。
- 记录日志。

## Handler 不可以做

- 直接写 HTTP Response。
- 直接写 SSE。
- 直接生成 AG-UI Event。
- 直接调用 Gateway API。
- 直接访问会话数据库。
- 直接调用其他 Child Agent。
- 自行做 Agent 选择。
- 忽略 context cancellation。

## 错误规则

- 返回 error 代表任务失败。
- 不得 panic；Runtime 必须 recover 兜底。
- 错误消息不得泄漏 secret。
- 可重试错误应带 retryable 标识或可映射错误码。

## Review Checklist

- [ ] Handler 只依赖 Runtime Context API。
- [ ] Handler 接收结构化消息。
- [ ] Handler 尊重 context cancellation。
- [ ] Handler 不直接输出 AG-UI。
- [ ] Handler 错误可控且脱敏。
