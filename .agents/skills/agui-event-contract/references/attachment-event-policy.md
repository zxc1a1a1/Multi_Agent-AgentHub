# 附件事件传输策略

- AG-UI 文本事件（如 `TEXT_MESSAGE_CONTENT`）不承载大二进制内容。
- 附件上传由前端与 Gateway 协作完成，Gateway 生成 attachment metadata。
- AG-UI 事件可传递 `attachmentRef` / `contentRef` 与预览事件，不直接传附件实体。
- Artifact 预览由 frontend runtime skills 渲染，避免在事件流中混入不可控二进制。
