# llm-client-lifecycle

## 目的

本文定义 ADK Runtime 中 LLMClient 的生命周期。

## 基本规则

- LLMClient 在 Agent 启动时创建。
- HTTP Client 在 LLMClient 内复用。
- HTTP Client 必须设置 timeout。
- Handler 通过依赖注入获得 LLMClient。
- 不得在每个任务请求中重复创建 LLMClient。
- 不得使用无 timeout 的默认 HTTP Client 调 LLM。

## 推荐配置

```yaml
runtime:
  llm:
    enabled: true
    providerRef: default
    timeoutSeconds: 120
```

实际 provider、model、baseURL、API key 由 `llm-provider-contract` 管。

## 推荐启动模式

```go
func main() {
    cfg := adk.MustLoadConfig("config.yaml")
    llm := adk.NewLLMClientFromEnv()
    server := adk.NewA2AServer(cfg, func(ctx *adk.Context, messages []adk.Message) error {
        return handleTask(ctx, messages, llm)
    })
    server.Run()
}
```

## 错误规则

- LLM timeout 应映射为 `ADK_LLM_TIMEOUT`。
- LLM provider 错误应映射为 `ADK_LLM_ERROR`。
- 用户可见错误必须脱敏。
- 日志中不得打印 API key。
- 日志中不得打印完整 Authorization header。

## Review Checklist

- [ ] LLMClient 是否启动时创建？
- [ ] HTTP Client 是否设置 timeout？
- [ ] Handler 是否通过注入使用 LLMClient？
- [ ] 错误是否脱敏？
- [ ] secret 是否不在 config.yaml 中？
