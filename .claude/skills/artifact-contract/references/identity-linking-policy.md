# Identity Linking Policy

## 目的

Artifact 必须可追溯到它的上下文和来源。

## 必填关联

Core Artifact 使用嵌套结构：

```text
artifactId
links.conversationId
links.messageId
links.runId
```

Public API DTO 将这些字段扁平化投影为顶层 `conversationId`、`messageId`、`runId`。Gateway Handler 负责从 `links.*` 展开。

## 推荐来源字段

```text
source.agentName
source.taskId
source.stepId
```

## Artifact ID 三层映射

Core Artifact.artifactId、DB artifacts.id、Public API Artifact.id 是**同一个系统 ID 在不同层级的命名**，不是三套不同 ID。

| 层级 | 字段名 | 说明 |
|---|---|---|
| Core Artifact（内部 JSON / schema） | `artifactId` | 内部事实源字段名 |
| DB（MySQL artifacts 表） | `id` 或 `artifact_id` | 存储 Core Artifact.artifactId 值，不另行生成独立的 public id |
| Public API DTO（对外 JSON） | `id` | `artifactId` 的公开投影 |

规则：

- 内部 JSON / schema 优先使用 `artifactId`。
- DB 主键字段名可以是 `id`，但必须说明其值等于 `artifactId`。
- Public API 的 `id` 字段值是 `artifactId` 的公开投影，非独立 ID。
- 不得为 Public API 另行生成与 `artifactId` 不同的 public id。
- 如果在代码或文档中同时出现 `artifact_id`、`id`、`artifactId`，必须注明它们映射到同一个值。

## 多 Agent 场景

多 Agent 或群聊中：

- 同一 run 可以产生多个 message。
- 同一 message 可以产生多个 Artifact。
- 不同 Agent 可以生成同名文件。
- Artifact ID 不得由 title 单独派生。
- source.agentName 用于展示和调试，不用于判断 Artifact 类型。

## 禁止

- 用 filename 当 artifactId。
- 用 messageId 当 artifactId。
- 用 agentName 决定 artifact.type。
- 缺少 runId。
- 为 Public API 另行生成与 artifactId 不同的 public id。
- 在文档中暗示 artifactId、DB id、API id 是三套不同 ID。
