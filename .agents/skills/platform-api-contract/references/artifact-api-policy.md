# Artifact API 策略

Artifact API 返回平台资源形态，是对 Core Artifact 的公开投影，不定义前端组件实现。

## 字段映射

Public API DTO 字段来源：

| Public API DTO | Core Artifact |
|---|---|
| `id` | `artifactId`（公开投影）。Core `artifactId` = DB `artifacts.id` = Public API `id`，是同一个系统 ID，不是三套不同 ID。API 不另行生成独立的 public id。 |
| `type` | `type` |
| `title` | `title` |
| `mimeType` | `mimeType` |
| `summary` | `summary` |
| `contentRef` | 公开授权访问入口 URL（非 Core contentRef 对象） |
| `contentUrl` | 公开下载/访问 URL（推荐） |
| `previewType` | `preview.previewType`（扁平投影） |
| `status` | `status`（直接对齐 Core 枚举） |
| `version` | `version`（integer） |
| `messageId` | `links.messageId`（扁平投影） |
| `conversationId` | `links.conversationId`（扁平投影） |
| `runId` | `links.runId`（扁平投影） |

## previewType 与 toolName

API 返回的 `previewType` 字段应优先等于 Frontend Runtime Registry 中的 `toolName`。`artifact.type` 不是 `toolName`，`outputMode` 不是 `toolName`。前端消费时直接使用 `previewType` 值查询 Runtime Capability Registry。

## 推荐资源

- `GET /api/artifacts/{artifactId}`
- `GET /api/artifacts/{artifactId}/content`
- `GET /api/artifacts/{artifactId}/preview`
- `GET /api/artifacts/{artifactId}/download`

大内容必须使用 `contentRef` 或下载入口，下载 URL 必须短期有效，不返回对象存储原始凭证或内部路径。

推荐 Public API 使用 `contentUrl` 字段提供公开访问地址，避免与 Core `contentRef` 对象混名。
