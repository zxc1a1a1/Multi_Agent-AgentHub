# 富媒体 Artifact 契约测试补充

- 需要测试 artifact type 到 previewSkill 的映射。
- 需要测试大文件不进入文本流。
- 需要测试 `slide_deck` 不被误认为 `pptx_file`。
- 需要测试 `webpage` sandbox 策略。
- 需要测试 markdown XSS 样例。
- 需要测试 `artifactRef/contentRef` 权限和非法值。
- MVP 阶段可先用 JSON fixture，不需要真实文件。
