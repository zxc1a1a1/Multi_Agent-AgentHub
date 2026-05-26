# 多模态 Artifact 策略补充

- 新增或预留 artifact type：`vision_analysis`、`file_summary`、`image_preview`、`file_card`、`slide_deck`。
- Artifact `content` 应保持结构化，便于下游 Agent 与前端消费。
- 大文件应使用 `contentRef`，不直接内嵌到 artifact 文本内容。
- artifact metadata 建议记录可选字段：`mimeType`、`source`、`sizeBytes`、`checksum`。
