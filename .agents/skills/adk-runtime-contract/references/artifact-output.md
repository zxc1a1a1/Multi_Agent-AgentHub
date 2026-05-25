# artifact-output

## 目的

本文定义 ADK Runtime 的 Artifact 输出规则。

Artifact 是结构化产物，不是普通文本流。

## 推荐结构

```go
type Artifact struct {
    Type     string
    Title    string
    Content  string
    Metadata map[string]string
}
```

## v1.0 推荐类型

```text
code
webpage
document
```

具体是否可用，以 `artifact-contract` 为准。

## 通用规则

- 任意 Agent 都可以输出其 AgentCard.outputModes 声明且 Artifact Contract 支持的 Artifact。
- 不得用 agentName 判断 Artifact 类型。
- Artifact 必须有 `type`。
- Artifact 必须有 `title`。
- Artifact 必须有 `content` 或外部引用。
- Artifact 应有 `metadata.language` 或 `metadata.mimeType`。
- Runtime 只把 Artifact 转成 A2A artifact event。
- Orchestrator 才负责转成 AG-UI Tool Call。
- Frontend Runtime Skill 参数由 `frontend-runtime-skills-contract` 定义。

## 示例：code

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main\n",
  "metadata": {
    "language": "go",
    "mimeType": "text/x-go"
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
    "language": "html",
    "mimeType": "text/html"
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
    "mimeType": "text/markdown"
  }
}
```

## 禁止事项

- Handler 不得直接构造 `code_preview`。
- Handler 不得直接构造 `web_preview`。
- Handler 不得把 Artifact 伪装成 text chunk。
- Artifact metadata 不得包含 secret。
