# Child Agents 架构说明

## 1. Child Agent 定位

Child Agent 是 AgentHub 的具体能力提供者，使用 ADK Runtime + A2A。

每个 Child Agent 都必须：

- 暴露 AgentCard
- 暴露 A2A endpoints
- 支持 Task
- 支持 Streaming
- 返回 Artifact
- 提供 /health

---

## MVP v0.1 Child Agent 范围

MVP v0.1 只实现一个子 Agent：

```text
code-agent
```

`code-agent` 必须遵守：

- 暴露 AgentCard。
- 提供 A2A Server。
- 支持 A2A `sendSubscribe` 流式输出。
- 使用 ADK Runtime 约定。
- 能输出 text stream。
- 能在检测到代码块时生成 `code` Artifact。
- `code` Artifact 的 metadata 至少包含 `language`。
- `code` Artifact 最终通过 Orchestrator 转换为 AG-UI `code_preview` Tool Call。

MVP v0.1 暂不实现：

- web-agent。
- doc-agent。
- custom-agent。
- deploy-agent。

---

## 2. 推荐目录

```text
agents/
  adk/
    agent.go
    context.go
    task.go
    a2a_server.go
    types.go
  code-agent/
    main.go
    handler.go
    tools/
    config.yaml
    Dockerfile
  web-agent/
    main.go
    handler.go
    tools/
    config.yaml
    Dockerfile
  doc-agent/
    main.go
    handler.go
    tools/
    config.yaml
    Dockerfile
```

---

## 3. 必须暴露的端点

```text
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/{id}
DELETE /a2a/tasks/{id}/cancel
GET    /health
```

---

## 4. AgentCard 必须声明

```text
name
description
url
version
capabilities
skills
inputModes
outputModes
```

---

## 5. outputModes 与 Artifact

Child Agent 只能返回自己声明过的 outputModes 对应 Artifact。

常见映射：

```text
text      → 普通文本消息
code      → code_preview
webpage   → web_preview
diff      → diff_preview
file      → file_download
image     → image_preview
document  → markdown_render
terminal  → terminal_output
chart     → chart_render
deploy    → deploy_status
```

---

## 6. 第一版 Child Agents

### code-agent

负责：

- 代码生成
- 代码审查
- 代码重构
- 终端命令建议

outputModes：

```text
text
code
diff
terminal
file
```

### web-agent

负责：

- 页面生成
- React 组件生成
- HTML/CSS/JS 预览
- UI 优化

outputModes：

```text
text
webpage
code
image
```

### doc-agent

负责：

- 文档生成
- Markdown 输出
- 总结
- 表格和图表

outputModes：

```text
text
document
chart
file
```

---

## 7. Child Agent 禁止事项

- 禁止直接访问 Frontend。
- 禁止直接控制 Frontend Skill。
- 禁止直接访问 Gateway 用户 API。
- 禁止绕过 Orchestrator。
- 禁止返回未声明的 Artifact type。
- 禁止输出过大的 TEXT chunk。
