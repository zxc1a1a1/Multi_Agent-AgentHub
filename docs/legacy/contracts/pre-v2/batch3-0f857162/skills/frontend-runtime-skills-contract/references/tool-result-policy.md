# ToolResult Policy

## 默认策略

渲染型 capability 默认不返回 ToolResult。

例如：

- `code_preview`
- `web_preview`
- `markdown_render`

## 需要 ToolResult 的情况

- 用户确认。
- 表单输入。
- 文件选择。
- 手动反馈。
- 前端执行结果需要回传。

## 推荐结构

```ts
type ToolResult = {
  toolCallId: string
  status: 'success' | 'cancelled' | 'failed'
  data?: unknown
  error?: {
    code: string
    message: string
  }
}
```

## 规则

- ToolResult 必须关联 `toolCallId`。
- ToolResult 不得包含 secret。
- ToolResult 不得包含未脱敏异常堆栈。
- 交互型 capability 必须支持取消。
