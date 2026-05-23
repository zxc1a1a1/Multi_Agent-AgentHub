# ToolResult 规则

## 1. 目的

本文定义前端 Runtime Skill 的 ToolResult 规则。

## 2. 何时需要 ToolResult

展示型 `render` Skill 可以不返回 ToolResult。

交互型 `interactive` Skill 通常需要返回 ToolResult。

副作用型 `side_effect` Skill 必须返回 ToolResult。

## 3. ToolResult 推荐结构

```ts
type FrontendToolResult = {
  toolCallId: string;
  skillName: string;
  status: "success" | "cancelled" | "failed";
  data?: unknown;
  error?: {
    code: string;
    safeMessage: string;
  };
};
```

## 4. 错误结果

错误结果不得包含：

- stack trace。
- API key。
- access token。
- refresh token。
- internal path。
- internal service URL。
- raw provider error。
- database connection string。

## 5. MVP code_preview

`code_preview` 是 render-only Skill。

MVP 不要求 ToolResult。

如果记录复制行为，可以作为可选非阻塞结果：

```json
{
  "toolCallId": "tc_123",
  "skillName": "code_preview",
  "status": "success",
  "data": {
    "copied": true
  }
}
```

## 6. 禁止事项

不得：

- 让展示型 Skill 阻塞 run。
- ToolResult 中暴露敏感信息。
- 用户取消后返回 success。
- failed 结果缺少 safeMessage。
