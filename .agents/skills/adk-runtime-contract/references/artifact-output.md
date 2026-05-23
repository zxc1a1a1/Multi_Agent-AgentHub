# Artifact 输出规则

## 1. 目的

本文定义子 Agent 如何通过 AgentHub ADK Runtime 添加 Artifact。

Artifact 是任务产物事实源。

## 2. Runtime API

子 Agent 必须通过：

```text
ctx.AddArtifact(artifact)
```

添加 Artifact。

不得绕过 Runtime 私自返回私有 Artifact 格式。

## 3. MVP code Artifact

MVP 阶段只允许：

```text
artifact.type = code
```

MVP 示例：

```go
ctx.AddArtifact(adk.Artifact{
    Type: "code",
    Title: "main.go",
    Content: code,
    Metadata: map[string]string{
        "language": "go",
    },
})
```

字段映射：

```text
Type → artifact.type
Title → artifact.title
Content → artifact.content
Metadata["language"] → artifact.metadata.language
```

## 4. 长期 Artifact

长期 Artifact schema、存储策略和预览映射由：

```text
artifact-contract
```

负责。

ADK Runtime 只负责提供添加 Artifact 的标准 API。

## 5. 禁止事项

不得：

- 通过流式文本输出替代 Artifact。
- 输出未注册 Artifact type。
- 在 MVP 阶段输出非 `code` Artifact。
- 在 Artifact 中包含 secret。
- 在 Artifact 中包含内部路径。
- 在 Artifact 中包含对象存储私有地址。
- 让子 Agent 直接生成前端 Runtime Skill Tool Call。
