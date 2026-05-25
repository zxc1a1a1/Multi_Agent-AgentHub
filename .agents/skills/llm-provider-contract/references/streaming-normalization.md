# Streaming Normalization

流式归一化将 Provider 专有事件转换为统一 `LLMStreamEvent`。

## 统一事件类型

```text
message_start
delta_text
tool_call_start
tool_call_delta
tool_call_end
usage_delta
message_end
error
```

## 规则

- Provider 原始 stream event 不得暴露给业务层。
- context cancelled 后不得继续输出 delta。
- error event 必须结束当前请求。
- stream end 后不得再输出普通事件。
- 流式输出应可聚合成非流式响应。
