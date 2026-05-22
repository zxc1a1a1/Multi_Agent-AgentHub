# 官方协议边界

## 1. 目的

本文定义 Artifact 相关的官方协议边界。

本文件用于避免把 AgentHub 项目内 Artifact 简化格式误当成 A2A 官方格式，或把 AG-UI Tool Call 事件定义误写进 artifact-contract。

## 2. A2A 边界

A2A Artifact 是 Agent Task 的输出。

A2A Artifact 可由 Parts 组成。

Part 可表达：

```text
text
file reference
structured data
```

AgentHub Artifact 是项目内 normalized object。

二者关系：

```text
A2A Artifact
→ normalize
→ AgentHub Artifact
```

不得把 MVP 简化格式：

```json
{
  "type": "code",
  "title": "main.go",
  "content": "...",
  "metadata": {
    "language": "go"
  }
}
```

当成 A2A 官方 Artifact 格式。

## 3. AG-UI 边界

AG-UI Tool Call 事件由 `agui-event-contract` 负责。

artifact-contract 不定义：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

的事件字段。

artifact-contract 只定义：

```text
artifact.type → frontend runtime skill
```

例如：

```text
code → code_preview
```

## 4. Frontend Runtime Skill 边界

以下映射不属于 artifact-contract：

```text
code_preview → CodePreview
```

它属于：

```text
frontend-runtime-skills-contract
```

## 5. JSON Schema 边界

`artifact.schema.json` 必须使用：

```text
https://json-schema.org/draft/2020-12/schema
```

## 6. 禁止事项

不得：

- 把 AgentHub MVP Artifact 简化格式当成 A2A 官方格式。
- 在 artifact-contract 中重定义 AG-UI 事件字段。
- 在 artifact-contract 中绑定 React Component。
- 把 Artifact schema 写成只服务 code_preview 的临时结构。
