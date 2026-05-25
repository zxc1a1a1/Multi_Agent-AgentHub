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

## ArtifactDraft 与 Core Artifact

Child Agent / ADK Handler 输出的是 **ArtifactDraft**，不是标准 Core Artifact。

### ArtifactDraft 最小字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| `type` | 是 | Artifact 类型 |
| `title` | 是 | 产物标题或文件名 |
| `content` | 二选一 | 小型 inline 内容 |
| `contentRefDraft` | 二选一 | 大型内容的内部引用 |
| `metadata` | 推荐 | 类型相关元数据 |

### 平台生成字段

以下字段由 Orchestrator / ArtifactRegistry 归一化时生成，不在 ArtifactDraft 中：

- `artifactId` — 全局唯一产物 ID
- `mimeType` — 根据 type + metadata 推断
- `source.agentName` — 已知调用目标
- `source.taskId` — 已知 A2A task id
- `links.conversationId` — 已知当前会话
- `links.messageId` — 消息创建后回填
- `links.runId` — 已知当前 Run
- `preview.previewType` — 根据 type 查 Registry
- `version` — 默认 1
- `status` — 初始 pending，归一化后 ready
- `createdAt` / `updatedAt` — 归一化时间

### 规则

- 不得把 ArtifactDraft 当作 Core Artifact。
- Child Agent / ADK Handler 不得在 ArtifactDraft 中提供平台字段。
- 不得跳过归一化直接使用 ArtifactDraft 作为公开 API 响应。

## 安全

Artifact 不得包含密钥、token、数据库连接串、私有签名 URL、本地绝对路径、内网地址、完整 system prompt 或未脱敏隐私。

## Artifact ID 三层映射

Core Artifact.artifactId、DB artifacts.id、Public API Artifact.id 是**同一个系统 ID 在不同层级的命名**，不是三套不同 ID：

| 层级 | 字段名 | 说明 |
|---|---|---|
| Core Artifact（内部 JSON / schema） | `artifactId` | 事实源字段名，如 `art_01H...` |
| DB（MySQL `artifacts` 表） | `id`（或 `artifact_id`） | 存储 Core `artifactId` 值，不另行生成独立的 public id |
| Public API DTO（对外 JSON） | `id` | `artifactId` 的公开投影 |

- schema 字段使用 `artifactId`。
- DB 字段 `id` 或 `artifact_id` 必须映射到同一个 `artifactId` 值。
- API `id` 字段值是 `artifactId` 的公开投影，非独立 ID。
- 不强制修改数据库字段名，不新增迁移代码。

## Core ↔ Public API DTO 映射

本文定义的是 Core Artifact（平台内部事实源）。Public API Artifact DTO 是对 Core 的公开投影：

- API `id` = Core `artifactId`（= DB `artifacts.id`，同一值不同命名）
- API `conversationId`/`messageId`/`runId` = Core `links.conversationId`/`links.messageId`/`links.runId`（API 扁平化）
- API `previewType` = Core `preview.previewType`（API 扁平化）
- API `contentRef`（string URL）≠ Core `contentRef`（object）；推荐 API 使用 `contentUrl`
- API `version` = Core `version`（integer）
- API `status` = Core `status`（pending/normalizing/ready/failed/superseded/deleted）
