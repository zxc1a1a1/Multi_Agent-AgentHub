# 前端调试规则

## 1. 目的

本文定义前端 AG-UI SSE 和 Runtime Skill 的调试规则。

## 2. 必须关联的 ID

前端必须能关联：

```text
traceId
runId
messageId
toolCallId
artifactId
```

MVP 阶段如果没有 artifactId，至少要有 messageId、runId 和 toolCallId。

## 3. AG-UI SSE 调试

前端应记录安全调试事件：

```text
sse_connected
sse_event_received
sse_parse_failed
tool_call_started
tool_call_args_received
tool_call_finished
tool_call_failed
runtime_skill_rendered
runtime_skill_failed
```

## 4. 错误处理

前端错误必须：

- 不暴露 stack trace 给用户。
- 不暴露 token。
- 不暴露内部服务地址。
- 显示 safeMessage。
- 保留 traceId / runId 供排障。

## 5. 禁止事项

不得：

- 在浏览器 console 打印 API key。
- 在浏览器 console 打印完整 prompt。
- 在前端错误 UI 展示 stack trace。
- 在没有 toolCallId 的情况下执行 Tool Call。
- 在 Tool Call 参数校验失败后继续渲染 Runtime Skill。
