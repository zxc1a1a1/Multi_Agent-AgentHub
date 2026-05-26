# AgentHub 多模态 Agent 对接交付说明

## 1. 当前多模态交付范围

- `vision-agent v0.1` 是 `image_ref` 的入口。
- `file-agent` 是 `file_ref / extracted_text` 的入口。
- `vision_analysis / file_summary` 是中间产物。
- `web/code/document/ppt/security/test` 等 Agent 是下游消费者。
- 本交付不包含真实 OCR、真实图片模型、真实文件读取、前端上传实现、Gateway 新接口实现。

## 2. 推荐对接链路

前端上传截图或选择文件  
-> 同事 API 生成或接收 `attachmentRef/contentRef`  
-> API 调用 `vision-agent` 或 `file-agent`  
-> 获得 `vision_analysis / file_summary` artifact  
-> API 再按需要调用 `web-agent / ppt-agent / security-agent`  
-> 前端展示 artifact preview

## 3. 同事 API 需要提供或转发的字段

- `message`
- `attachments[].type`
- `attachments[].contentRef`
- `attachments[].mimeType`
- `attachments[].name`
- `attachments[].sizeBytes`
- `attachments[].metadata.source`
- `extractedText`
- `context`

说明：

- `image_ref` 给 `vision-agent`。
- `file_ref` 给 `file-agent`。
- 不要传 base64。
- 不要传 `file://`。
- 不要传真实本地路径。
- 不要传真实密钥。

示例引用值：

- `attachment://att_demo_001`
- `artifact://art_demo_001`

## 4. 多模态测试路径

- UI 截图 -> `vision-agent` -> `web-agent`
- 报错截图 -> `vision-agent` -> `test-agent` / `review-agent`
- 文件资料 -> `file-agent` -> `document-agent`
- 文件资料 -> `file-agent` -> `ppt-agent`
- 截图/文件疑似泄密 -> `security-agent`
