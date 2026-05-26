# 多模态 A2A Part 补充说明

- A2A Message Part 建议支持 `text`、`image_ref`、`file_ref`、`artifact_ref`、`vision_analysis`、`file_summary`。
- 当前 MVP 可先用项目级 JSON payload 表达，后续再映射到更完整的 A2A Part。
- 子 Agent 不应接收本地文件路径（例如 `file://` 或绝对磁盘路径）作为可信输入。
- 大二进制内容不应直接进入 A2A 文本消息，必须通过 `contentRef` / `attachmentRef` 引用传递。
