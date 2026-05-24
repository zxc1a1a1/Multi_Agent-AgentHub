# A2A Artifact Policy

来源：`docs/contracts/a2a-task.md`、`docs/contracts/agui-events.md`。

## 1. MVP

- 强制支持 `type=code` Artifact。
- 推荐包含 `title`、`content`、`metadata.language`。

## 2. 处理边界

- Artifact 在 Orchestrator 缓存并在 completed 后 flush。
- 通过 `TOOL_CALL_*` 映射到前端技能（MVP 为 `code_preview`）。

## 3. 禁止

- 将大产物直接塞进文本流。
