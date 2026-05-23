# MVP code Artifact 规则

## 1. 目的

本文定义 MVP 阶段唯一实现的 Artifact 类型：`code`。

## 2. MVP 输入格式

MVP code Artifact 可使用项目内简化格式：

```ts
type MvpCodeArtifact = {
  type: "code";
  title: string;
  content: string;
  metadata: {
    language: string;
  };
};
```

注意：

```text
这是 AgentHub MVP 简化格式，不是 A2A 官方 Artifact 格式。
```

## 3. MVP 映射

```text
artifact.type = code
→ toolName = code_preview

artifact.content
→ args.code

artifact.metadata.language
→ args.language

artifact.title
→ args.filename
```

## 4. MVP 生命周期

```text
A2A artifact received
→ ArtifactBuffer
→ A2A completed
→ TEXT_MESSAGE_END
→ code → code_preview
→ TOOL_CALL_START
→ TOOL_CALL_ARGS
→ TOOL_CALL_END
→ RUN_FINISHED
→ messages.artifacts JSON
```

## 5. MVP 允许

MVP 允许：

- small code inline。
- message.artifacts JSON 临时保存。
- version = 1。
- 不使用 Object Storage。
- 不建立独立 Artifact 表。

## 6. MVP 不允许

MVP 不允许：

- inline 大代码包。
- inline zip。
- inline 图片。
- inline HTML 包。
- inline 大日志。
- inline 私有文件。
- 缺少 language 仍渲染。
- 将 code Artifact 直接映射到 CodePreview。
- 执行 code。
- eval code。
- 把 code_preview 变成代码执行器。

## 7. 参数约束引用

`code_preview` 参数 schema 由：

```text
frontend-runtime-skills-contract
```

负责。

本文件只记录 Artifact 到 `code_preview` 的字段映射。
