# Frontend 架构说明

## 1. Frontend 定位

Frontend 是 AgentHub 的用户交互层，采用 React + TypeScript + AG-UI Client。

它负责 IM 聊天体验、实时事件消费、前端 Skills 执行和产物预览。  
它不负责 Agent 调度和 A2A 通信。

---

## 2. 核心模块

推荐目录：

```text
frontend/src/
  components/
    Chat/
    Preview/
    Agent/
    Common/
  agui/
    client.ts
    events.ts
    skills.ts
    hooks.ts
  stores/
    conversationStore.ts
    messageStore.ts
    agentStore.ts
  services/
    api.ts
  types/
  utils/
```

---

## 3. Frontend 与 Gateway 通信

Frontend 只能访问 Gateway：

```text
REST API: /api/*
AG-UI:    /api/agui/*
```

Frontend 禁止访问：

```text
/internal/*
/a2a/*
Child Agent URL
Orchestrator URL
```

---

## 4. AG-UI 事件处理

Frontend 至少需要处理：

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
STATE_UPDATE
```

---

## 5. Frontend Skills

第一版必须支持：

```text
code_preview
web_preview
diff_preview
file_download
image_preview
deploy_status
markdown_render
terminal_output
chart_render
form_input
confirm_action
file_upload
```

组件映射：

```text
code_preview      → CodePreview
web_preview       → WebPreview
diff_preview      → DiffView
file_download     → FileCard
image_preview     → ImagePreview
deploy_status     → DeployCard
markdown_render   → MarkdownView
terminal_output   → TerminalOutput
chart_render      → ChartView
form_input        → FormInput
confirm_action    → ConfirmDialog
file_upload       → FileUpload
```

---

## 6. Frontend 禁止事项

- 禁止实现 A2A。
- 禁止直接读取 AgentCard 生成调度计划。
- 禁止直接请求 Orchestrator。
- 禁止直接请求 Child Agent。
- 禁止绕过 ToolResult 回传机制。
- 禁止把长代码文件作为普通消息渲染。
