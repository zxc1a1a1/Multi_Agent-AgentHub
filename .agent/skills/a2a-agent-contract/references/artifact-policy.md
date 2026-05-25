# Artifact Policy

## 1. 定位

Child Agent 可以通过 A2A streaming event 输出 Artifact。

Artifact 是结构化产物，不是普通文本。

## 2. 输出前提

Agent 输出某类 Artifact 前必须满足：

1. AgentCard.outputModes 声明该能力。
2. `artifact-contract` 支持该 Artifact type。
3. ProtocolConverter 支持映射。
4. `frontend-runtime-skills-contract` 注册对应 Runtime Skill。
5. 安全契约允许展示。

## 3. 推荐类型

### code

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main",
  "metadata": {"language": "go"}
}
```

映射：`code_preview`

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

映射：`web_preview`

### document / markdown

```json
{
  "type": "document",
  "title": "report.md",
  "content": "# 标题\n\n正文",
  "metadata": {"format": "markdown"}
}
```

映射：`markdown_render`

## 4. 禁止事项

- Child Agent 不得直接输出 AG-UI Tool Call。
- Child Agent 不得直接输出 `code_preview` / `web_preview` / `markdown_render`。
- 大型结构化产物不应塞进 text chunk。
- 未在 AgentCard.outputModes 声明的类型不应输出。

## 5. Review 要点

- Artifact type 是否被声明。
- Artifact 字段是否完整。
- Artifact 是否能被前端安全展示。
- 是否有测试覆盖映射链路。
