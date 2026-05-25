# AgentHub Artifact Schema Contract

## 目的

本文定义 AgentHub 标准 Artifact 对象。

Artifact 是系统中的产物事实源，用于保存、预览、下载、复用和追踪 Agent 或系统模块生成的结果。

## 标准对象

```json
{
  "artifactId": "art_01H00000000000000000000000",
  "type": "code",
  "title": "main.go",
  "mimeType": "text/x-go",
  "content": "package main\n",
  "contentRef": null,
  "summary": "Go 入口文件",
  "metadata": {
    "language": "go",
    "filename": "main.go"
  },
  "source": {
    "agentName": "agent-name",
    "taskId": "task_001"
  },
  "links": {
    "conversationId": "conv_001",
    "messageId": "msg_001",
    "runId": "run_001"
  },
  "preview": {
    "previewType": "code_preview",
    "available": true
  },
  "version": 1,
  "status": "ready",
  "createdAt": "2026-05-25T00:00:00Z",
  "updatedAt": "2026-05-25T00:00:00Z"
}
```

## 必填字段

- `artifactId`
- `type`
- `title`
- `mimeType`
- `links.conversationId`
- `links.messageId`
- `links.runId`
- `version`
- `status`
- `createdAt`

`content` 与 `contentRef` 至少存在一个。

## 类型

v1.0 required：

- `code`
- `webpage`
- `markdown`

planned：

- `document`
- `data`
- `image`
- `archive`
- `audio`
- `video`
- `diff`
- `terminal_log`

## 状态

- `pending`
- `normalizing`
- `ready`
- `failed`
- `superseded`
- `deleted`

只有 `ready` 状态可以预览。

## 安全

Artifact 不得包含密钥、token、数据库连接串、私有签名 URL、本地绝对路径、内网地址、完整 system prompt 或未脱敏隐私。
