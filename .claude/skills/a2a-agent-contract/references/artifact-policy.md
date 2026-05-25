# ArtifactDraft Policy

## 1. 定位

Child Agent 通过 A2A streaming event 输出的是 **ArtifactDraft**，不是标准 Core Artifact。

ArtifactDraft 是 Child Agent 能自主提供的最小产物描述。Orchestrator / ArtifactRegistry 收到后归一化为 Core Artifact（生成 `artifactId`、补齐 `mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 等平台字段）。

## 2. 输出前提

Agent 输出某类 ArtifactDraft 前必须满足：

1. AgentCard.outputModes 声明该能力。
2. `artifact-contract` 支持该 Artifact type。
3. ProtocolConverter 支持映射。
4. `frontend-runtime-skills-contract` 注册对应 Runtime Skill。
5. 安全契约允许展示。

## 3. ArtifactDraft 最小字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| `type` | 是 | Artifact 类型，如 `code`、`webpage`、`document` |
| `title` | 是 | 产物标题或文件名 |
| `content` | 二选一 | 小型 inline 内容 |
| `contentRefDraft` | 二选一 | 大型内容的内部引用（非 Core contentRef 对象） |
| `metadata` | 推荐 | 类型相关元数据，如 `language` |

规则：

- Child Agent 不得在 ArtifactDraft 中提供 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段。
- `contentRefDraft` 是 Child Agent 内部引用，不是 Core Artifact 的 `contentRef` 对象。
- Orchestrator 负责将 `contentRefDraft` 转换为平台可授权的 `contentRef`。

## 4. 由平台生成的字段

以下字段由 Orchestrator / ArtifactRegistry 在归一化时生成或补齐：

| 字段 | 生成者 | 说明 |
|---|---|---|
| `artifactId` | ArtifactRegistry | 全局唯一产物 ID |
| `mimeType` | ArtifactRegistry | 根据 type + metadata 推断或默认 |
| `source.agentName` | Orchestrator | 已知调用目标 |
| `source.taskId` | Orchestrator | 已知 A2A task id |
| `links.conversationId` | Orchestrator | 已知当前会话 |
| `links.messageId` | Orchestrator / Gateway | 消息创建后回填 |
| `links.runId` | Orchestrator | 已知当前 Run |
| `preview.previewType` | ArtifactRegistry | 根据 type 查 Registry |
| `version` | ArtifactRegistry | 默认 1 |
| `status` | ArtifactRegistry | 初始 `pending`，归一化后 `ready` |
| `createdAt` | ArtifactRegistry | 归一化时间 |
| `updatedAt` | ArtifactRegistry | 归一化/更新时间 |

## 5. 推荐类型

### code

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main",
  "metadata": {"language": "go"}
}
```

归一化后映射：`code_preview`

### webpage

```json
{
  "type": "webpage",
  "title": "index.html",
  "content": "<!DOCTYPE html><html>...</html>",
  "metadata": {
    "language": "html",
    "css": "body{}",
    "js": "console.log('ok')"
  }
}
```

归一化后映射：`web_preview`

### document / markdown

```json
{
  "type": "document",
  "title": "report.md",
  "content": "# 标题\n\n正文",
  "metadata": {"format": "markdown"}
}
```

归一化后映射：`markdown_render`

## 6. 禁止事项

- Child Agent 不得直接输出 AG-UI Tool Call。
- Child Agent 不得直接输出 `code_preview` / `web_preview` / `markdown_render`。
- 大型结构化产物不应塞进 text chunk。
- 未在 AgentCard.outputModes 声明的类型不应输出。
- Child Agent 不得在 ArtifactDraft 中提供 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt`。
- 不得把 ArtifactDraft 当作 Core Artifact。
- 不得把 `outputMode` 直接当作 `artifact.type`（除非映射表显式声明兼容）。
- 不得把 `outputMode` 直接当作 `toolName`。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。

## 7. Review 要点

- ArtifactDraft type 是否被声明。
- ArtifactDraft 字段是否只包含 Child Agent 可提供的字段（不含平台字段）。
- ArtifactDraft 是否能归一化为 Core Artifact。
- ArtifactDraft 是否能被前端安全展示。
- 是否有测试覆盖 ArtifactDraft → Core Artifact 归一化链路。
- 是否没有把 ArtifactDraft 当作 Core Artifact。
- `outputMode` 是否没有被直接当作 `artifact.type` 或 `toolName`。
- 是否所有 outputMode → artifact.type → toolName 转换都通过映射表表达。
