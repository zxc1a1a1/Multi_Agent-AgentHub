# AgentHub Vision Analysis Artifact 契约

## 1. 目标

`vision_analysis` 是图片、截图、设计稿、报错截图等视觉输入的结构化中间产物。

它不替代 `web-agent`、`code-agent`、`ppt-agent`、`security-agent`，而是为它们提供可消费上下文。

## 2. Artifact 基本字段

`vision_analysis` artifact 基本字段约定：

- `type`: `vision_analysis`
- `title`: 产物标题，例如 `ui-analysis.json`
- `content`: 结构化 JSON 文本
- `metadata.format`: `json`
- `metadata.sourceAttachmentRef`: 来源附件引用
- `metadata.model`: 分析模型标识（可选）
- `metadata.confidence`: 置信度（可选）
- `metadata.createdBy`: `vision-agent`

## 3. content JSON Schema 草案

`content` 建议包含以下字段：

- `summary`: 总结
- `imageType`: 图片类型（UI 截图、报错截图、文档拍照等）
- `detectedText`: 检测到的文本数组
- `uiElements`: UI 元素数组（按钮、输入框、导航栏等）
- `layout`: 布局信息（区域、层次、对齐关系）
- `colors`: 主要配色信息
- `objects`: 普通对象识别结果
- `charts`: 图表信息
- `errors`: 错误提示识别结果
- `sensitiveFindings`: 潜在敏感信息发现
- `risks`: 风险列表
- `suggestedNextAgents`: 建议后续调用的 Agent 列表
- `sourceRefs`: 源引用（`attachmentRef` 或 `artifactRef`）
- `limitations`: 本次识别的局限说明

## 4. 消费方 Agent 约定

- `web-agent` 使用 `layout` / `uiElements` / `colors` 生成网页。
- `code-agent` 使用 `layout` / `uiElements` 生成组件代码。
- `test-agent` 使用 `errors` / `detectedText` 分析报错。
- `security-agent` 使用 `detectedText` / `sensitiveFindings` 检查泄密。
- `ppt-agent` 使用 `summary` / `objects` / `charts` 生成演示页。
- `document-agent` 使用 `summary` / `detectedText` 生成说明文档。

## 5. 示例

### 示例一：UI 截图分析

```json
{
  "summary": "该页面为管理后台仪表盘，顶部为导航，主体为统计卡片与趋势图。",
  "imageType": "ui_screenshot",
  "detectedText": ["Dashboard", "Users", "Revenue"],
  "uiElements": ["top_nav", "stat_cards", "line_chart", "filter_button"],
  "layout": {
    "zones": ["header", "content"],
    "structure": "header + two-column content"
  },
  "colors": ["#0F172A", "#22C55E", "#F8FAFC"],
  "objects": ["icon", "card"],
  "charts": ["line_chart"],
  "errors": [],
  "sensitiveFindings": [],
  "risks": ["图表坐标标签可能不完整"],
  "suggestedNextAgents": ["web-agent", "code-agent"],
  "sourceRefs": ["attachment://att_demo_010"],
  "limitations": ["低分辨率区域可能漏检"]
}
```

### 示例二：报错截图分析

```json
{
  "summary": "截图显示构建失败，核心报错与依赖版本不匹配有关。",
  "imageType": "error_screenshot",
  "detectedText": ["Build failed", "module not found", "version mismatch"],
  "uiElements": ["error_panel", "terminal_block"],
  "layout": {
    "zones": ["left_error_list", "right_stack_preview"],
    "structure": "split_panel"
  },
  "colors": ["#111827", "#EF4444", "#E5E7EB"],
  "objects": ["warning_icon"],
  "charts": [],
  "errors": ["module not found", "version mismatch"],
  "sensitiveFindings": [],
  "risks": ["截图文本可能被裁切，错误链路不完整"],
  "suggestedNextAgents": ["test-agent", "review-agent", "security-agent"],
  "sourceRefs": ["attachment://att_demo_011"],
  "limitations": ["滚动区域外信息不可见"]
}
```

## 6. 局限性

- `vision_analysis` 可能有识别错误。
- 不能把 `detectedText` 当作绝对事实。
- 涉及安全、财务、法律、医疗内容需要人工确认。
- 对含密钥截图要做脱敏。

## 7. 下游富媒体输出

- `vision_analysis` 可以驱动 `web-agent` 输出 `webpage`。
- `vision_analysis` 可以驱动 `code-agent` 输出 `code`。
- `vision_analysis` 可以驱动 `ppt-agent` 输出 `slide_deck`。
- `vision_analysis` 可以驱动 `document-agent` 输出 `document`。
- `vision_analysis` 可以驱动 `security-agent` 输出 `security_report`。
- 这些输出的预览和下载规则由 `rich-artifact-output.md` 定义。
