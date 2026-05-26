# AgentHub 子 Agent A2A Smoke 示例

## 1. 说明

- 本文档给同事 API / 测试同学参考。
- A2A payload 需要按当前 ADK `server.go` 的真实 endpoint 和结构调整。
- 当前 ADK 暴露的常用端点为：`POST /` 与 `POST /a2a/tasks/sendSubscribe`。
- 如果当前已有 A2A client，优先使用 client，不必手写 HTTP。

## 2. 通用调用流程

1. 启动 Agent。
2. 检查 health。
3. 检查 AgentCard。
4. 发送 A2A task。
5. 读取文本输出。
6. 读取 artifact 输出。
7. 按 artifact type 做前端预览。

可参考的基础检查：

```powershell
Invoke-RestMethod -Method GET "http://localhost:8092/health"
Invoke-RestMethod -Method GET "http://localhost:8092/.well-known/agent.json"
```

## 3. 示例：vision-agent

目标：输入 `image_ref` 元数据 + `extractedText` + `context`，输出 `vision_analysis`。

说明：

- 示例只包含引用与文本，不包含真实图片二进制和 base64。
- `contentRef` 使用假值：`attachment://att_demo_001`。

示例输入（可作为 message 文本内容）：

```json
{
  "task": "analyze_screenshot_context",
  "image_ref": {
    "contentRef": "attachment://att_demo_001",
    "mimeType": "image/png",
    "name": "ui-login-error.png",
    "sizeBytes": 245612,
    "metadata": {
      "source": "user_upload"
    }
  },
  "extractedText": "Error: invalid token; Login failed",
  "context": "这是登录页截图，用户反馈点击登录后报错。"
}
```

期望：

- 有自然语言 text 流输出；
- 有 `vision_analysis` artifact；
- artifact 中包含 `summary/imageType/detectedText/risks/suggestedNextAgents/sourceRefs/limitations` 等字段。

## 4. 示例：web-agent

输入普通文本或 `vision_analysis` 摘要文本，期望输出 `webpage` artifact。

示例输入要点：

- 直接给页面需求文本，或先拼接 `vision_analysis` 摘要后再调用。
- 结果检查 `artifact.type == "webpage"`，前端走 `web_preview`。

## 5. 示例：document-agent

输入文本或 `file_summary`，期望输出 `document` artifact。

示例输入要点：

- 需求说明 + 结构化摘要（如背景、目标、风险）；
- 结果检查 `artifact.type == "document"`，前端走 `markdown_render`。

## 6. 示例：ppt-agent

输入项目资料摘要或 `file_summary`，期望输出 `slide_deck` artifact。

说明：

- `slide_deck` 是结构化草稿，不是真实 pptx 文件。

## 7. 示例：security-agent

输入 `text/diff/vision_analysis` 摘要，期望输出 `security_report` artifact。

结果检查要点：

- `artifact.type == "security_report"`；
- 包含风险等级、命中项与建议；
- 不应出现真实密钥内容回显。

## 8. 示例：diff-agent

输入改动需求或 diff 描述，期望输出 `diff` artifact。

说明：

- 第一版只生成/解释 diff，不应用 patch。

## 9. 常见失败

- Agent 未启动。
- 端口不一致。
- A2A payload 结构不匹配（建议优先用现有 A2A client）。
- LLM Provider 未配置。
- Agent 只返回文本没有 artifact（模型未按 fenced block 输出）。
- artifact type 与前端映射不一致。
- 传入 `image_ref` 但没有 `extractedText/description`，`vision-agent` 分析会受限。
