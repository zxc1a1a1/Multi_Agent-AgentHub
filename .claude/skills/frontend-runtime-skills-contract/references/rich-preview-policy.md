# 富媒体预览策略

- 前端根据 artifact `type` 和 `previewSkill` 选择渲染组件。
- 重点支持 `code_preview`、`web_preview`、`markdown_render`、`diff_preview`、`slide_deck_preview`、`image_preview`、`file_card`、`report_card`。
- `web_preview` 必须 sandbox。
- `markdown_render` 必须防 XSS。
- `slide_deck_preview` 只表示草稿预览，不等于 pptx 已生成。
