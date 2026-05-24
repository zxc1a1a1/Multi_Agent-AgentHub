# Tool Call Events

来源：`docs/contracts/agui-events.md`、`docs/contracts/gateway-orchestrator-events.md`。

## 1. 顺序

```text
TOOL_CALL_START
TOOL_CALL_ARGS*
TOOL_CALL_END
```

## 2. MVP code_preview

`code` Artifact 映射到：

```text
toolName = code_preview
args = { code, language, filename }
```

## 3. 约束

- `TOOL_CALL_ARGS` / `END` 必须关联同一 `toolCallId`。
- 未注册 Skill 不应强行触发。
