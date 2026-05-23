# Provider Adapter 规则

Provider Adapter 用于隔离不同 Provider 的 SDK、HTTP API、streaming event、error 和能力差异。

推荐接口：

```text
Generate(ctx, request) -> LLMResponse
Stream(ctx, request) -> Iterator<LLMStreamEvent>
CountTokens(ctx, request) -> TokenUsage
```

MVP 阶段只强制：

```text
Stream(ctx, request) -> Iterator<LLMStreamEvent>
```

Adapter 负责：

- 将 AgentHub `LLMRequest` 转换为 Provider 原始请求。
- 将 Provider 原始响应转换为 `LLMResponse`。
- 将 Provider 原始 streaming event 转换为 `LLMStreamEvent`。
- 将 Provider 原始错误转换为用户安全错误。
- 处理 timeout、context cancellation、retry、rate limit。
- 填充 usage metadata。
- 保护 secret。

Adapter 不负责 Orchestrator 选择 Agent、Artifact schema、A2A Task、AG-UI 事件或 React 组件。

禁止：

- 在业务 handler 中直接散落 Provider SDK 调用。
- 将 Provider 原始 response / streaming event 透传给 AgentHub 上层。
- 在 Adapter 外处理 API key。
