# Artifact 持久化规则

## 目的

Artifact 是 AgentHub 的产物事实源，必须能被历史回放、调试、预览和追踪。

## artifacts 表字段与 Core Artifact 映射

DB `artifacts.id` 存储 Core `artifactId` 值。Core `artifactId` = DB `id` = Public API `id` 是**同一个系统 ID**，不是三套不同 ID。DB 不另行生成与 `artifactId` 不同的 public id。

| DB 列（snake_case） | Core Artifact 字段（camelCase） | 说明 |
|---|---|---|
| `id` | `artifactId` | DB `id` 存储 Core `artifactId` 值。Public API 中 `id` 是对此值的公开投影。 |
| `conversation_id` | `links.conversationId` | Core 嵌套于 `links` 对象下。 |
| `message_id` | `links.messageId` | Core 嵌套于 `links` 对象下。 |
| `run_id` | `links.runId` | Core 嵌套于 `links` 对象下。 |
| `agent_name` | `source.agentName` | Core 嵌套于 `source` 对象下。 |
| `type` | `type` | 直接对应。 |
| `title` | `title` | 直接对应。 |
| `mime_type` | `mimeType` | 直接对应。 |
| `content` | `content` | 小型文本直接存储。 |
| `content_ref` | `contentRef` | 存储 Core contentRef 对象的序列化 JSON，非裸 URL。 |
| `metadata` | `metadata` | JSON 对象。 |
| `version` | `version` | integer，从 1 开始。 |
| `status` | `status` | Core 枚举：pending / normalizing / ready / failed / superseded / deleted。 |
| `created_at` | `createdAt` | 直接对应。 |
| `updated_at` | `updatedAt` | 直接对应。 |
| `deleted_at` | N/A | 软删除时间戳，Core 中无直接对应。 |

## content 与 content_ref

- 小型文本内容可使用 `content`。
- 大型内容、二进制内容、下载内容必须使用 `content_ref`。
- `content_ref` 必须是后端可控引用，不是永久公开 URL。

## v1.0 类型

至少应支持持久化：

```text
code
webpage
markdown
```

可规划：

```text
document
data
image
zip
log
```

## 关联规则

- Artifact 必须关联 conversation。
- Artifact 应关联 message。
- Artifact 应关联 run。
- 多 Agent 场景下 Artifact 应保存 agent_name。
- 重名 title 不得造成覆盖。

## 禁止

- 产物只存在实时事件中。
- 大文件直接塞数据库。
- 永久公开下载 URL 入库。
- secret、系统 prompt、内部路径入库。
