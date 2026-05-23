# 前端测试策略

## 1. 目的

本文定义 AgentHub 前端测试重点。

## 2. 必测模块

前端必须测试：

```text
AG-UI SSE parser
TOOL_CALL lifecycle
TOOL_CALL_ARGS chunk append
Tool Call args JSON parse
Runtime Skill registry
CodePreview
StreamingText
message store
error fallback
```

## 3. MVP 必测路径

MVP 阶段至少测试：

- 收到 TEXT_MESSAGE_CONTENT 后显示流式文本。
- 收到 TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END 后渲染 CodePreview。
- `code_preview` 参数缺失时显示 fallback。
- 未知 toolName 不崩溃。
- CodePreview 显示 filename、language、code。
- Copy code 按钮可用。
- 刷新后历史消息存在。

## 4. E2E 原则

E2E 测试应使用用户可见的 selector / role / text。

不得依赖脆弱 DOM 结构。

不得用固定 sleep 代替等待可见状态。

## 5. 禁止事项

不得：

- Tool Call 参数校验失败后继续渲染。
- 未知 toolName 导致 UI 崩溃。
- 在浏览器 console 打印 API key。
- E2E 依赖真实 LLM。
