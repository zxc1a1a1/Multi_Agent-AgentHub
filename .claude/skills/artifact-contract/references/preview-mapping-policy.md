# Preview Mapping 规则

## 1. 目的

本文定义 Artifact type 到前端 Runtime Skill 的映射。

本文件只定义：

```text
artifact.type → frontend runtime skill
```

不定义：

```text
frontend runtime skill → React Component
```

## 2. 长期映射

```text
code      → code_preview
webpage   → web_preview
diff      → diff_preview
file      → file_download
image     → image_preview
deploy    → deploy_status
document  → markdown_render
terminal  → terminal_output
chart     → chart_render
```

## 3. MVP 映射

MVP 只启用：

```text
code → code_preview
```

其他映射可以 reserved，但不得执行完整链路。

## 4. Tool args 构建边界

artifact-contract 可以定义 Artifact 到 preview skill 的字段映射。

但具体 Runtime Skill 参数 schema 和 React Component 绑定归：

```text
frontend-runtime-skills-contract
```

## 5. MVP code 字段映射

```text
artifact.content
→ code_preview.args.code

artifact.metadata.language
→ code_preview.args.language

artifact.title
→ code_preview.args.filename
```

## 6. 禁止事项

不得：

- 写 code → CodePreview。
- 把 React Component 写进 preview mapping。
- 把 unknown artifact.type 映射到默认组件。
- 缺少 metadata.language 仍渲染 code_preview。
- implemented=false 的 mapping 进入执行链路。
