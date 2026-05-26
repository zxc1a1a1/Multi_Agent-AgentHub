# ADK Runtime 多模态方向补充

- ADK Runtime 第一版可保持文本 handler 模式不变。
- 多模态引用应通过结构化消息或 `contentRef` 传递，不直接读取本地路径。
- 子 Agent 不应自行解析本地文件路径或绕过 Gateway 附件引用。
- 后续可扩展 helper：`ReadAttachmentRef`、`EmitVisionAnalysis`、`ValidateContentRef`。
- 当前阶段不实现以上 helper，仅定义后续扩展方向。
