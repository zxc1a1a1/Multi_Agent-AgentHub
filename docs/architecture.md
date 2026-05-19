# 架构说明：React + Go 多 Agent 框架

## 一句话说明

React 负责用户界面和 A2UI 渲染，Go 负责 Agent 后端、AG-UI 事件流、A2A Agent 服务、ADK 编排适配。

```text
React Web
  │
  │ AG-UI / SSE / WebSocket
  ▼
Go Agent API
  ├── Root Agent
  ├── Research Agent
  ├── Code Agent
  ├── Review Agent
  ├── A2A Remote Agent Client
  └── Tools / MCP / DB / External APIs
  │
  └── A2UI JSON → React Renderer
```

## 各模块职责

### apps/web

- 用户输入。
- 接收 AG-UI 事件流。
- 展示 Agent 状态。
- 渲染 A2UI 声明式 UI。

### apps/api/internal/agent

- Root Agent。
- 子 Agent 编排。
- 后续接入 ADK Go。
- 后续接入远程 A2A Agent。

### apps/api/pkg/protocol/agui

- 前后端事件模型。
- 当前是 starter 版，后续可替换为官方 SDK 类型。

### apps/api/pkg/protocol/a2ui

- A2UI 声明式 UI 数据结构。
- 当前支持 card、text、button、list 等基本组件。

### apps/api/pkg/protocol/a2a

- Agent Card、Skill、Task 等结构占位。
- 后续对齐官方 A2A Go SDK。

## 推荐开发阶段

1. 先完成普通聊天接口。
2. 加 AG-UI 事件流。
3. 加 A2UI 组件渲染。
4. 加本地多 Agent 编排。
5. 加 A2A 远程 Agent 通信。
6. 加 ADK Go 生产级编排。
7. 加鉴权、审计、日志、观测、评估。
