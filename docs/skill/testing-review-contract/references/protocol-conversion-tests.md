# 协议转换测试

## 1. 目的

本文定义 AgentHub 协议转换测试规则。

协议转换是 AgentHub 的高风险边界。

## 2. MVP 必测转换

必须测试：

```text
A2A text → AG-UI TEXT_MESSAGE_CONTENT
A2A artifact → ArtifactBuffer
A2A completed → TEXT_MESSAGE_END + TOOL_CALL sequence
code artifact → code_preview args
```

## 3. Tool Call 顺序

必须验证顺序：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

不得乱序。

不得重复执行同一个 toolCallId。

## 4. Artifact 映射

MVP 必测：

```text
artifact.type = code
→ toolName = code_preview
→ args = { code, language, filename }
```

字段映射：

```text
artifact.content → code
artifact.metadata.language → language
artifact.title → filename
```

## 5. 错误路径

必须测试：

- unknown artifact type。
- invalid artifact schema。
- missing metadata.language。
- invalid JSON args。
- unknown toolName。
- duplicate TOOL_CALL_END。
- A2A stream interrupted。

## 6. 禁止事项

不得：

- 改 converter 不补测试。
- 只测 completed 成功路径。
- 把大型 Artifact 塞进 AG-UI token 流。
- 未校验参数就构造 Tool Call。
