# Runtime API 规则

## 1. 目的

本文定义 AgentHub ADK Runtime 提供给子 Agent handler 的 Context API。

Runtime API 用于隐藏 A2A streaming 细节，让 handler 专注任务逻辑。

## 2. 推荐 API

长期 Runtime API 可以包括：

```text
ctx.StreamText(chunk)
ctx.AddArtifact(artifact)
ctx.Fail(error)
ctx.Metadata()
ctx.Tools()
ctx.Logger()
```

MVP 阶段只强制：

```text
ctx.StreamText(chunk)
ctx.AddArtifact(artifact)
```

## 3. ctx.StreamText

用途：

- 输出流式文本。
- 输出说明性回复。
- 输出任务进度。
- 输出用户可见状态。

不得用于：

- 输出大型 Artifact。
- 输出私有文件。
- 输出 secret。
- 输出 stack trace。
- 输出内部路径。
- 代替 `ctx.AddArtifact`。

## 4. ctx.AddArtifact

用途：

- 添加任务产物。
- 表达代码、网页、diff、文件、图片等产物。
- 让 Orchestrator / Gateway 后续标准化、持久化和预览。

MVP 只允许：

```text
artifact.type = code
```

## 5. ctx.Fail

用于返回用户安全错误。

错误不得包含：

- stack trace。
- API key。
- token。
- system prompt。
- 内部路径。
- 内部服务地址。

## 6. ctx.Tools

用于访问已注册工具。

使用工具前必须通过工具注册和权限声明。

MVP 阶段 `code-agent` 默认不启用工具。

## 7. 禁止事项

不得：

- 让 handler 直接操作 A2A SSE 原始响应。
- 让 handler 直接返回 AG-UI 事件。
- 让 handler 直接调用 React Component。
- 绕过 `ctx.AddArtifact` 产生产物。
- 绕过工具权限直接执行 shell、filesystem、network 操作。
