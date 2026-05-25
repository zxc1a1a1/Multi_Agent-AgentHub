# artifact-output

## 目的

本文定义 ADK Runtime 的 ArtifactDraft 输出规则。

`ctx.AddArtifact()` 输出的是 **ArtifactDraft**，不是标准 Core Artifact。Handler 只提供最小产物描述，平台字段（`artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt`）由 Orchestrator / ArtifactRegistry 归一化时生成。

## ArtifactDraft 结构

```go
type ArtifactDraft struct {
    Type            string
    Title           string
    Content         string
    ContentRefDraft string
    Metadata        map[string]string
}
```

## ArtifactDraft 最小字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| `Type` | 是 | Artifact 类型，如 `code`、`webpage`、`document` |
| `Title` | 是 | 产物标题或文件名 |
| `Content` | 二选一 | 小型 inline 内容 |
| `ContentRefDraft` | 二选一 | 大型内容的内部引用 |
| `Metadata` | 推荐 | 类型相关元数据 |

ADK Runtime 不要求 Handler 提供 `artifactId`、`mimeType`、`links.*`、`source.*`、`preview.*`、`version`、`status`、`createdAt` 等平台字段。

## v1.0 推荐类型

```text
code
webpage
document
```

具体是否可用，以 `artifact-contract` 为准。

## 通用规则

- 任意 Agent 都可以输出其 AgentCard.outputModes 声明且 Artifact Contract 支持的 ArtifactDraft。
- 不得用 agentName 判断 Artifact 类型。
- ArtifactDraft 必须有 `Type`。
- ArtifactDraft 必须有 `Title`。
- ArtifactDraft 必须有 `Content` 或 `ContentRefDraft`。
- ArtifactDraft 应有 `Metadata["language"]` 或 `Metadata["mimeType"]`。
- Runtime 只把 ArtifactDraft 转成 A2A artifact event。
- Orchestrator / ArtifactRegistry 负责把 ArtifactDraft 归一化为 Core Artifact。
- ProtocolConverter 再把 Core Artifact 转成 AG-UI Tool Call。
- Frontend Runtime Skill 参数由 `frontend-runtime-skills-contract` 定义。

## 示例：code

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main\n",
  "metadata": {
    "language": "go"
  }
}
```

## 示例：webpage

```json
{
  "type": "webpage",
  "title": "index.html",
  "content": "<!DOCTYPE html><html>...</html>",
  "metadata": {
    "language": "html"
  }
}
```

## 示例：document

```json
{
  "type": "document",
  "title": "report.md",
  "content": "# 报告\n",
  "metadata": {
    "language": "markdown",
    "format": "markdown"
  }
}
```

## 示例：contentRefDraft

```json
{
  "type": "webpage",
  "title": "large-page.html",
  "contentRefDraft": "https://internal-agent/storage/temp-001/page.html",
  "metadata": {
    "language": "html"
  }
}
```

## 禁止事项

- Handler 不得直接构造 `code_preview`。
- Handler 不得直接构造 `web_preview`。
- Handler 不得把 ArtifactDraft 伪装成 text chunk。
- ArtifactDraft metadata 不得包含 secret。
- Handler 不得在 ArtifactDraft 中提供 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt`。
- 不得把 ArtifactDraft 当作 Core Artifact。
