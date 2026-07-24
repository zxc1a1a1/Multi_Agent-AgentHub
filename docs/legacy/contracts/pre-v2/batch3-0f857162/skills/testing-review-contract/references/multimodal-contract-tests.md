# 多模态 Contract 测试补充

- 需要测试 `image_ref` / `file_ref` 路由正确性。
- 需要测试 `contentRef` 非法值与不可信输入拒绝策略。
- 需要测试大文件不会进入文本流事件。
- 需要测试 `vision_analysis` 结构字段与 schema 约束。
- 需要测试安全扫描的误报与漏报边界。
- MVP 阶段可先使用纯 JSON fixture，不依赖真实图片文件。
