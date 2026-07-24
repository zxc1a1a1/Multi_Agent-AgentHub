# 富媒体输出 Artifact 策略

- Artifact 是 Agent 富媒体输出的统一载体。
- 小内容可 inline，大文件必须使用 `contentRef`。
- 需要区分草稿型 artifact 和真实文件型 artifact。
- 产物应记录 `type`、`title`、`mimeType`、`previewSkill`、`provenance`、`lifecycle`。
- 不允许把二进制或大文件塞进文本流。
