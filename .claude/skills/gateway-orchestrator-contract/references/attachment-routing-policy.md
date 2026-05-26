# 附件路由策略

- Gateway 负责接收上传、校验 `mimeType` / `sizeBytes`、生成 `attachmentRef` / `contentRef`。
- Orchestrator 负责结合 `inputModes`、`mimeType`、用户意图进行路由。
- `image_ref` 优先进入 `vision-agent`。
- `file_ref` 优先进入 `file-agent` 或未来 extractor。
- 高风险输入优先进入 `security-agent` 或安全策略链路。
