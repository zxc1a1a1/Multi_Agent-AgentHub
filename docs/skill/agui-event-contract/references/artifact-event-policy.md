# Artifact 事件传递策略

- AG-UI 文本事件只承载自然语言。
- Artifact 通过 `ARTIFACT_CREATED/UPDATED/PREVIEW_READY/FAILED/EXPIRED` 等项目级事件或 tool result 传递引用。
- 事件中只传 `artifactRef`、`type`、`title`、`previewSkill`、`metadata`、`status`。
- 不通过 AG-UI 文本流传输大二进制。
