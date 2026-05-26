# AgentHub 多模态输入与附件处理契约

## 1. 目标与边界

本契约定义用户上传图片、截图、文件、PDF/PPT 资料时，Gateway、Orchestrator、A2A 子 Agent、Artifact、Frontend Runtime Skills 之间如何传递引用和结构化结果。

本契约不要求第一版直接实现 OCR、PDF 解析、PPTX 生成、对象存储。

本契约不要求所有 Agent 直接支持图片或文件。

现有 18 个 Agent 继续作为任务处理能力池，多模态入口负责把非文本内容转换为 `vision_analysis` 或 `file_summary` 等结构化输入。

## 2. 核心概念

- `attachmentRef`：由 Gateway 生成的附件引用标识，用于标识一次上传对象。
- `contentRef`：可解析到附件或产物内容的引用地址，不直接暴露真实存储实现。
- `artifactRef`：已生成 Artifact 的稳定引用。
- `mimeType`：附件媒体类型，例如 `image/png`、`application/pdf`。
- `source`：输入来源，例如 `user_upload`、`clipboard_screenshot`。
- `sizeBytes`：附件大小（字节）。
- `checksum`：内容校验值（例如 SHA-256 摘要字符串）。
- `metadata`：补充字段集合，供路由、安全和预览使用。
- `extractedText`：从附件中提取出的文本（第一版可由上游提供，不要求平台内置提取器）。
- `vision_analysis`：视觉输入经结构化理解后得到的中间产物。
- `file_summary`：文件内容经结构化整理后得到的中间产物。

推荐 `contentRef` 格式：

- `attachment://att_xxx`
- `artifact://art_xxx`

约束：

- 不推荐把大文件或图片 base64 直接塞进消息正文。
- 不推荐把二进制内容放进 AG-UI 文本流。
- `contentRef` 必须由 Gateway 或受信任服务生成，子 Agent 不应伪造本地文件路径。

## 3. 输入 Part 约定

### TextPart

```json
{
  "type": "text",
  "text": "请根据截图生成网页"
}
```

### ImageRefPart

```json
{
  "type": "image_ref",
  "contentRef": "attachment://att_demo_001",
  "mimeType": "image/png",
  "name": "ui-homepage.png",
  "sizeBytes": 245612,
  "metadata": {
    "source": "user_upload"
  }
}
```

### FileRefPart

```json
{
  "type": "file_ref",
  "contentRef": "attachment://att_demo_002",
  "mimeType": "application/pdf",
  "name": "product-requirements.pdf",
  "sizeBytes": 1024312,
  "metadata": {
    "source": "user_upload"
  }
}
```

### ArtifactRefPart

```json
{
  "type": "artifact_ref",
  "artifactRef": "artifact://art_demo_001",
  "metadata": {
    "source": "orchestrator"
  }
}
```

### VisionAnalysisPart

```json
{
  "type": "vision_analysis",
  "artifactRef": "artifact://art_demo_vision_001",
  "summary": "这是一个电商首页草图，包含顶部导航、商品卡片和底部按钮",
  "detectedText": ["New Arrivals", "Buy Now"],
  "uiElements": ["navbar", "card-grid", "cta-button"],
  "risks": ["可能存在价格文案被遮挡"],
  "suggestedNextAgents": ["web-agent", "document-agent"]
}
```

### FileSummaryPart

```json
{
  "type": "file_summary",
  "artifactRef": "artifact://art_demo_file_001",
  "summary": "文档包含目标、范围、验收条件和风险章节",
  "sections": ["背景", "需求", "验收标准"],
  "extractedRequirements": ["支持附件预览", "支持权限校验"],
  "risks": ["部分术语定义不一致"]
}
```

## 4. 推荐处理链路

1. UI 截图转网页

用户上传截图
-> Gateway 保存附件并生成 `image_ref`
-> Orchestrator 路由到 `vision-agent`
-> `vision-agent` 输出 `vision_analysis`
-> `web-agent` 生成 `webpage` artifact

2. 报错截图分析

用户上传报错截图
-> `vision-agent` 提取错误文本和界面上下文
-> `test-agent` 分析失败原因
-> `review-agent` 给修复建议

3. 文件附件生成报告

用户上传 Markdown/PDF/PPT/文本资料
-> Gateway 生成 `file_ref`
-> `file-agent` 或未来 `file-extractor` 生成 `file_summary`
-> `document-agent` 生成 `document` artifact
-> `ppt-agent` 生成 `slide_deck` artifact

4. 截图或文件中疑似泄密

附件内容或提取文本
-> `security-agent` 分析密钥泄漏、危险命令、敏感路径
-> 输出 `security_report` artifact

## 5. Orchestrator 路由策略

- `text-only` 任务继续按现有 Agent 能力路由。
- `image_ref` 优先进入 `vision-agent`。
- `file_ref` 优先进入 `file-agent` 或未来 extractor。
- `vision_analysis` 可以被 `web-agent`、`code-agent`、`document-agent`、`ppt-agent`、`security-agent`、`test-agent` 消费。
- `file_summary` 可以被 `document-agent`、`ppt-agent`、`review-agent`、`qa-acceptance-agent`、`security-agent` 消费。
- 不要让所有 Agent 直接处理 `image_ref` / `file_ref`。
- 高风险附件先经过 `security-agent` 或安全策略检查。

## 6. AgentCard inputModes / outputModes 建议

> 本节为未来建议，不要求本轮修改任何 `config.yaml`。

`vision-agent`：

```yaml
inputModes:
  - text
  - image_ref
outputModes:
  - text
  - vision_analysis
```

`file-agent`：

```yaml
inputModes:
  - text
  - file_ref
  - extracted_text
outputModes:
  - text
  - file_summary
```

`web-agent`：

```yaml
inputModes:
  - text
  - vision_analysis
outputModes:
  - text
  - webpage
  - code
```

`ppt-agent`：

```yaml
inputModes:
  - text
  - vision_analysis
  - file_summary
outputModes:
  - text
  - slide_deck
```

`security-agent`：

```yaml
inputModes:
  - text
  - vision_analysis
  - file_summary
  - diff
outputModes:
  - text
  - security_report
```

## 7. 安全边界

- 禁止子 Agent 读取 `.env`。
- 禁止子 Agent 接受 `file://` 本地路径作为可信输入。
- 禁止 SSRF：禁止直接访问内网地址、`localhost`、metadata IP。
- 禁止在消息正文中传输真实密钥。
- 对附件类型、大小、`mimeType` 做校验。
- 对图片 OCR 或文件提取结果进行敏感信息扫描。
- 对 `contentRef` 设置权限、过期时间和访问范围。
- 日志中不记录完整 `contentRef` 授权 URL，不记录原始密钥。
- 用户上传内容不能直接覆盖 `systemPrompt`。
- 高风险工具调用需要审批或策略拦截。

## 8. MVP 与后续演进

MVP：

- 只定义 contract。
- `vision-agent` 后续作为图片理解入口。
- `file-agent` 第一版只处理调用方提供文本，后续可接 `file_ref`。
- 前端先展示 image/file/vision_analysis 卡片。
- 不做真实 PPTX/PDF/OCR 生成。

后续：

- 对象存储
- 文件解析服务
- OCR
- 图片理解模型
- PDF/PPT 提取
- 多模态 AgentCard 发现
- 工具权限审批
- 前端多模态预览组件

## 9. 与富媒体输出契约的关系

- 本文档关注输入侧 `attachmentRef` / `contentRef`。
- 富媒体输出由 `rich-artifact-output.md` 定义。
- 输入附件经过 `vision-agent` / `file-agent` 后，可生成 `vision_analysis` / `file_summary`。
- 下游 Agent 消费中间产物后，输出 `webpage` / `document` / `slide_deck` / `diff` / `report` 等富媒体 Artifact。
- 输入 `contentRef` 与输出 `artifactRef` 应通过 `sourceRefs` / `provenance` 建立关系。
