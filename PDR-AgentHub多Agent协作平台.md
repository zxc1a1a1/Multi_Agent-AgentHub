# AgentHub - 多Agent协作平台 PDR（Product Design Review）

## 文档信息

| 项目 | 内容 |
|------|------|
| 项目名称 | AgentHub - 多Agent协作平台 |
| 文档版本 | v2.0 |
| 创建日期 | 2026-05-21 |
| 开发团队 | 3人 |
| 文档类型 | 产品设计评审文档 |
| 核心协议 | AG-UI（前端通信）+ A2A（Agent间通信） |

---

## 一、产品概述

### 1.1 产品定位

AgentHub 是一个以 **IM 聊天为核心交互范式** 的多 Agent 协作平台。用户像使用飞书/微信一样，通过新建对话、发送消息的方式与不同 AI Agent 进行交互，实现代码生成、网页构建、文档编写等任务的智能协作。

### 1.2 核心价值

- **统一入口**：一个平台接入多个主流 Agent（Claude Code、Codex、OpenCode 等）
- **自然交互**：IM 聊天范式，降低 AI 工具使用门槛
- **协作增强**：群聊模式下多 Agent 自动协调，完成复杂任务
- **产物可视**：代码、网页、文档等产物内联预览，所见即所得
- **标准协议**：基于 AG-UI + A2A 开放协议，可扩展性强

### 1.3 目标用户

- 开发者：需要 AI 辅助编码、代码审查、架构设计
- 产品/设计人员：需要快速生成原型、网页、文档
- 团队协作者：需要多 Agent 协同完成复杂项目

---

## 二、系统架构（核心重点）

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     Frontend (React + AG-UI Client)                      │
│  ┌──────────┐  ┌──────────────┐  ┌──────────┐  ┌────────────────────┐  │
│  │ 对话列表  │  │  聊天窗口     │  │ 产物预览  │  │ AG-UI SDK (skills) │  │
│  └──────────┘  └──────────────┘  └──────────┘  └────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              │ AG-UI Protocol (SSE/WebSocket)
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Gateway 网关层 (Go + LLM Router)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ AG-UI Server │  │  路由/鉴权    │  │ 会话管理      │  │  LLM调用   │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              │ 内部调用
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              统一 Agent (Orchestrator / 编排器)                           │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │  意图理解 (LLM)   │  │  任务编排引擎     │  │  结果聚合器           │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              │ A2A Protocol (Agent-to-Agent)
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      子 Agent 集群 (ADK Runtime)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ Code Agent   │  │ Web Agent    │  │ Doc Agent    │  │ Custom... │  │
│  │ (AgentCard)  │  │ (AgentCard)  │  │ (AgentCard)  │  │(AgentCard)│  │
│  │  ADK Runtime │  │  ADK Runtime │  │  ADK Runtime │  │ADK Runtime│  │
│  └──────────────┘  └──────────────┘  └──────────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Data Layer                                      │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────┐                      │
│  │ PostgreSQL│  │    Redis      │  │  对象存储     │                      │
│  └──────────┘  └──────────────┘  └──────────────┘                      │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 架构分层说明

```mermaid
flowchart TB
    subgraph 前端层["前端层 (React + AG-UI Client)"]
        UI[IM聊天UI]
        AGUI_CLIENT[AG-UI Client SDK]
        SKILLS[Skills - 前端能力扩展]
    end
    
    subgraph 网关层["Gateway 网关层 (Go)"]
        AGUI_SERVER[AG-UI Server Endpoint]
        ROUTER[路由 & 鉴权]
        SESSION[会话管理]
        LLM[LLM 调用层]
    end
    
    subgraph 编排层["统一Agent / Orchestrator"]
        INTENT[意图理解]
        ORCHESTRATE[编排引擎]
        AGGREGATE[结果聚合]
    end
    
    subgraph Agent层["子Agent集群 (ADK Runtime + A2A)"]
        AGENT_A[Code Agent]
        AGENT_B[Web Agent]
        AGENT_C[Doc Agent]
        AGENT_D[Custom Agent]
    end
    
    UI --> AGUI_CLIENT
    AGUI_CLIENT -->|"AG-UI Protocol"| AGUI_SERVER
    AGUI_SERVER --> ROUTER
    ROUTER --> SESSION
    ROUTER --> LLM
    SESSION --> INTENT
    INTENT --> ORCHESTRATE
    ORCHESTRATE -->|"A2A Protocol"| AGENT_A
    ORCHESTRATE -->|"A2A Protocol"| AGENT_B
    ORCHESTRATE -->|"A2A Protocol"| AGENT_C
    ORCHESTRATE -->|"A2A Protocol"| AGENT_D
    AGENT_A --> AGGREGATE
    AGENT_B --> AGGREGATE
    AGENT_C --> AGGREGATE
    AGENT_D --> AGGREGATE
    AGGREGATE -->|"AG-UI Events"| AGUI_SERVER
```

### 2.3 技术选型

| 层级 | 技术方案 | 选型理由 |
|------|----------|----------|
| **前端框架** | React 18 + TypeScript | 生态成熟，AG-UI SDK 原生支持 React |
| **前端通信协议** | AG-UI Protocol | 标准化 Agent-UI 交互协议，支持流式事件 |
| **UI 组件库** | Ant Design 5.x / Shadcn UI | IM 类组件丰富 |
| **状态管理** | Zustand | 轻量、简洁，适合中型项目 |
| **后端语言** | Go (Golang) | 高并发、高性能，适合网关和Agent运行时 |
| **后端框架** | Gin / Fiber | 轻量高性能 HTTP 框架 |
| **Agent间通信** | A2A Protocol (Agent-to-Agent) | Google 开放标准，Agent互操作性 |
| **Agent运行时** | ADK Runtime (Agent Development Kit) | 标准化Agent开发框架 |
| **数据库** | PostgreSQL | 关系型数据，支持 JSONB 存储消息 |
| **缓存** | Redis | 会话缓存、消息队列、在线状态 |
| **部署** | Docker + Docker Compose | 一键启动，便于演示 |

### 2.4 核心协议说明

#### 2.4.1 AG-UI Protocol（前端 ↔ 统一Agent）

> 参考文档：https://docs.ag-ui.com/introduction

AG-UI（Agent-User Interface Protocol）是前端与 Agent 之间的标准化通信协议，定义了：

- **事件流**：Agent 通过 SSE 向前端推送结构化事件（文本chunk、工具调用、状态变更等）
- **Skills**：前端可声明自身能力（如代码编辑、文件上传），Agent 可调用前端 Skills
- **生命周期**：标准化的消息发送、流式响应、中断、重试等生命周期管理

```typescript
// AG-UI 前端集成示例
import { AgUIClient, useAgUI } from '@ag-ui/react';

// AG-UI 事件类型
type AGUIEvent = 
  | { type: 'TEXT_MESSAGE_START'; messageId: string }
  | { type: 'TEXT_MESSAGE_CONTENT'; content: string }
  | { type: 'TEXT_MESSAGE_END' }
  | { type: 'TOOL_CALL_START'; toolCallId: string; toolName: string }
  | { type: 'TOOL_CALL_ARGS'; content: string }
  | { type: 'TOOL_CALL_END' }
  | { type: 'STATE_UPDATE'; state: Record<string, any> }
  | { type: 'RUN_STARTED'; runId: string }
  | { type: 'RUN_FINISHED' }
  | { type: 'RUN_ERROR'; error: string };

// 前端 Skills 声明
const skills = [
  {
    name: 'code_editor',
    description: '在前端打开代码编辑器',
    parameters: { filename: 'string', content: 'string', language: 'string' }
  },
  {
    name: 'web_preview',
    description: '在前端渲染网页预览',
    parameters: { html: 'string', css: 'string', js: 'string' }
  },
  {
    name: 'file_upload',
    description: '让用户上传文件',
    parameters: { accept: 'string', maxSize: 'number' }
  }
];
```

#### 2.4.2 A2A Protocol（统一Agent ↔ 子Agent）

A2A（Agent-to-Agent）是 Google 提出的 Agent 间通信标准协议，定义了：

- **AgentCard**：Agent 的能力描述卡片（类似 API 的 OpenAPI Spec）
- **Task**：Agent 间的任务传递和状态管理
- **Streaming**：支持流式任务结果返回
- **Push Notifications**：支持异步任务完成通知

```go
// A2A AgentCard 定义 (Go)
type AgentCard struct {
    Name         string            `json:"name"`
    Description  string            `json:"description"`
    URL          string            `json:"url"`           // Agent 服务地址
    Version      string            `json:"version"`
    Capabilities AgentCapabilities `json:"capabilities"`
    Skills       []AgentSkill      `json:"skills"`        // Agent 支持的技能
    InputModes   []string          `json:"inputModes"`    // text, file, image
    OutputModes  []string          `json:"outputModes"`   // text, code, file, webpage
}

type AgentCapabilities struct {
    Streaming          bool `json:"streaming"`           // 支持流式
    PushNotifications  bool `json:"pushNotifications"`   // 支持推送通知
    StateTransition    bool `json:"stateTransitionHistory"` // 状态转换历史
}

type AgentSkill struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Tags        []string `json:"tags"`
    Examples    []string `json:"examples"`  // 示例输入
}

// A2A Task 定义
type Task struct {
    ID       string        `json:"id"`
    Status   TaskStatus    `json:"status"`   // submitted, working, completed, failed
    Messages []TaskMessage `json:"messages"`
    Artifacts []Artifact   `json:"artifacts"` // 产出物
}

type TaskStatus string
const (
    TaskStatusSubmitted TaskStatus = "submitted"
    TaskStatusWorking   TaskStatus = "working"
    TaskStatusCompleted TaskStatus = "completed"
    TaskStatusFailed    TaskStatus = "failed"
    TaskStatusCanceled  TaskStatus = "canceled"
)

// A2A 标准接口端点
// POST   /a2a/tasks/send          - 发送任务
// GET    /a2a/tasks/:id            - 查询任务状态
// POST   /a2a/tasks/sendSubscribe  - 发送任务并订阅流式结果
// DELETE /a2a/tasks/:id/cancel     - 取消任务
// GET    /.well-known/agent.json   - 获取 AgentCard
```

#### 2.4.3 ADK Runtime（子Agent运行时）

ADK（Agent Development Kit）是子 Agent 的标准化运行时框架，提供：

- **统一生命周期管理**：Agent 启动、健康检查、优雅关闭
- **工具注册机制**：声明式注册 Agent 可用的工具
- **A2A Server 内置**：自动暴露 A2A 标准端点
- **AgentCard 自动生成**：基于配置自动生成 `/.well-known/agent.json`

```go
// ADK Runtime 子Agent实现示例
package main

import (
    "github.com/agenthub/adk"
)

func main() {
    // 创建 Agent 实例
    agent := adk.NewAgent(adk.AgentConfig{
        Name:        "code-agent",
        Description: "专注于代码生成、审查和重构的Agent",
        Version:     "1.0.0",
        Skills: []adk.Skill{
            {
                ID:          "code_generate",
                Name:        "代码生成",
                Description: "根据需求描述生成代码",
                Tags:        []string{"code", "generate"},
            },
            {
                ID:          "code_review",
                Name:        "代码审查",
                Description: "审查代码质量并给出建议",
                Tags:        []string{"code", "review"},
            },
        },
        InputModes:  []string{"text", "file"},
        OutputModes: []string{"text", "code", "file"},
    })

    // 注册工具
    agent.RegisterTool("read_file", readFileTool)
    agent.RegisterTool("write_file", writeFileTool)
    agent.RegisterTool("run_command", runCommandTool)

    // 设置消息处理器（核心逻辑）
    agent.SetHandler(func(ctx *adk.Context, task *adk.Task) error {
        // 1. 解析任务
        userMessage := task.GetLatestMessage()
        
        // 2. 调用 LLM 处理
        response, err := ctx.LLM.ChatStream(ctx, userMessage)
        if err != nil {
            return err
        }
        
        // 3. 流式返回结果
        for chunk := range response {
            ctx.StreamText(chunk)
        }
        
        // 4. 如果有产物，附加 Artifact
        if hasCodeOutput {
            ctx.AddArtifact(adk.Artifact{
                Type:    "code",
                Title:   "generated_code.go",
                Content: generatedCode,
            })
        }
        
        return nil
    })

    // 启动 Agent（自动暴露 A2A 端点 + AgentCard）
    agent.Serve(":8081")
}
```

### 2.5 核心数据流

```mermaid
sequenceDiagram
    participant U as 用户
    participant FE as 前端(AG-UI Client)
    participant GW as Gateway网关
    participant ORC as 统一Agent(Orchestrator)
    participant SubA as 子Agent A (A2A)
    participant SubB as 子Agent B (A2A)

    U->>FE: 发送消息
    FE->>GW: AG-UI RunRequest (SSE连接)
    GW->>GW: 鉴权 + 会话管理
    GW->>ORC: 转发到统一Agent
    
    Note over ORC: 意图理解 + 编排决策
    
    ORC-->>GW: AG-UI Event: RUN_STARTED
    GW-->>FE: SSE: RUN_STARTED
    
    alt 单Agent任务
        ORC->>SubA: A2A tasks/sendSubscribe
        SubA-->>ORC: A2A Streaming Response
        ORC-->>GW: AG-UI Event: TEXT_MESSAGE_CONTENT (流式)
        GW-->>FE: SSE: TEXT_MESSAGE_CONTENT
        FE-->>U: 实时渲染
    else 多Agent协作任务
        ORC->>SubA: A2A tasks/send (代码任务)
        ORC->>SubB: A2A tasks/send (文档任务)
        SubA-->>ORC: A2A Task Completed + Artifacts
        SubB-->>ORC: A2A Task Completed + Artifacts
        ORC->>ORC: 聚合结果
        ORC-->>GW: AG-UI Events (聚合回复)
        GW-->>FE: SSE: 多条消息事件
        FE-->>U: 渲染多Agent回复
    end
    
    ORC-->>GW: AG-UI Event: RUN_FINISHED
    GW-->>FE: SSE: RUN_FINISHED
```

### 2.6 系统完整时序图

#### 2.6.1 场景一：单聊模式（用户 → 单个Agent）

用户在单聊窗口中发送消息，系统路由到单个子Agent处理并流式返回。

```mermaid
sequenceDiagram
    autonumber
    participant User as 👤 用户
    participant React as 🖥️ React前端
    participant AGUI as AG-UI Client
    participant GW as 🚪 Gateway网关
    participant DB as 🗄️ PostgreSQL
    participant ORC as 🧠 Orchestrator
    participant LLM as 🤖 LLM (意图分析)
    participant Registry as 📋 Agent注册表
    participant CodeAgent as 🔧 Code-Agent (A2A)

    User->>React: 输入消息 "帮我写一个Go的HTTP服务器"
    React->>React: 更新UI状态(sending)
    React->>AGUI: 构建RunAgentInput {messages, tools(skills)}
    
    Note over AGUI: 声明前端Skills:<br/>code_preview, web_preview,<br/>diff_preview, terminal_output...
    
    AGUI->>GW: POST /api/agui/run (建立SSE连接)
    
    GW->>GW: JWT鉴权验证
    GW->>DB: 保存用户消息
    GW->>DB: 查询会话历史
    DB-->>GW: 返回历史消息列表
    
    GW->>ORC: 转发请求 {runId, messages, history, tools, context}
    
    Note over ORC: === 意图编排阶段 ===
    
    ORC->>Registry: 获取所有健康Agent的AgentCard
    Registry-->>ORC: [code-agent, web-agent, doc-agent]
    
    ORC->>LLM: 意图分析Prompt + AgentCards + 用户消息
    LLM-->>ORC: ExecutionPlan {strategy:"single", agent:"code-agent"}
    
    Note over ORC: === 协议转换: 构建A2A请求 ===
    
    ORC->>ORC: 初始化ProtocolConverter(前端Skills)
    
    ORC-->>GW: AG-UI Event: RUN_STARTED
    GW-->>AGUI: SSE: RUN_STARTED
    AGUI-->>React: onEvent(RUN_STARTED)
    React-->>User: 显示"Agent正在思考..."

    ORC-->>GW: AG-UI Event: STATE_UPDATE {activeAgent:"code-agent"}
    GW-->>AGUI: SSE: STATE_UPDATE
    AGUI-->>React: 显示Agent头像和名称
    
    Note over ORC,CodeAgent: === A2A调用阶段 ===
    
    ORC->>CodeAgent: POST /a2a/tasks/sendSubscribe {task:{messages}}
    
    Note over CodeAgent: ADK Runtime 处理任务
    CodeAgent->>CodeAgent: 加载SystemPrompt + Tools
    CodeAgent->>CodeAgent: 调用LLM生成代码(流式)
    
    CodeAgent-->>ORC: A2A SSE: {type:"status", status:"working"}
    
    Note over ORC: === 协议转换: A2A→AG-UI ===
    ORC-->>GW: AG-UI: TEXT_MESSAGE_START {messageId}
    GW-->>AGUI: SSE: TEXT_MESSAGE_START
    AGUI-->>React: 创建消息气泡
    
    loop 流式文本chunks
        CodeAgent-->>ORC: A2A SSE: {type:"text", content:"package main..."}
        ORC-->>GW: AG-UI: TEXT_MESSAGE_CONTENT {content}
        GW-->>AGUI: SSE: TEXT_MESSAGE_CONTENT
        AGUI-->>React: 追加文本到气泡
        React-->>User: 实时看到文字逐字出现
    end
    
    CodeAgent-->>ORC: A2A SSE: {type:"artifact", artifact:{type:"code", content:"...", title:"main.go"}}
    Note over ORC: 缓存Artifact，等待消息结束
    
    CodeAgent-->>ORC: A2A SSE: {type:"status", status:"completed"}
    
    ORC-->>GW: AG-UI: TEXT_MESSAGE_END
    GW-->>AGUI: SSE: TEXT_MESSAGE_END
    AGUI-->>React: 消息完成标记
    
    Note over ORC: === 产物→前端Skill转换 ===
    ORC->>ORC: Artifact(code) → ToolCall(code_preview)
    
    ORC-->>GW: AG-UI: TOOL_CALL_START {toolName:"code_preview"}
    GW-->>AGUI: SSE: TOOL_CALL_START
    
    ORC-->>GW: AG-UI: TOOL_CALL_ARGS {code, language:"go", filename:"main.go"}
    GW-->>AGUI: SSE: TOOL_CALL_ARGS
    
    ORC-->>GW: AG-UI: TOOL_CALL_END
    GW-->>AGUI: SSE: TOOL_CALL_END
    
    AGUI->>React: 执行code_preview Skill
    React->>React: 渲染<CodePreview />组件
    React-->>User: 看到语法高亮的代码卡片(含复制/应用/编辑按钮)
    
    React->>AGUI: Skill执行结果 {success:true}
    AGUI->>GW: POST /api/agui/run/{runId}/tool-result
    GW->>ORC: 转发ToolResult
    
    ORC-->>GW: AG-UI: RUN_FINISHED
    GW-->>AGUI: SSE: RUN_FINISHED
    AGUI-->>React: 关闭SSE连接
    
    GW->>DB: 保存Agent回复消息 + Artifact
    
    React-->>User: 显示完成状态，隐藏loading
```

#### 2.6.2 场景二：群聊模式（多Agent协作）

用户在群聊中发送复杂任务，Orchestrator 拆解并分派给多个子Agent。

```mermaid
sequenceDiagram
    autonumber
    participant User as 👤 用户
    participant React as 🖥️ React前端
    participant AGUI as AG-UI Client
    participant GW as 🚪 Gateway网关
    participant ORC as 🧠 Orchestrator
    participant LLM as 🤖 LLM
    participant CodeAgent as 🔧 Code-Agent
    participant WebAgent as 🎨 Web-Agent

    User->>React: "帮我写一个登录页面，前端React组件+后端Go接口"
    React->>AGUI: RunAgentInput {messages, context:{conversationType:"group"}}
    AGUI->>GW: POST /api/agui/run (SSE)
    GW->>ORC: 转发请求
    
    ORC-->>GW: AG-UI: RUN_STARTED
    GW-->>AGUI: SSE → React → User: "开始处理..."
    
    Note over ORC: === 意图编排 ===
    ORC->>LLM: 分析意图 + AgentCards
    LLM-->>ORC: ExecutionPlan {<br/>strategy:"parallel",<br/>tasks:[<br/>{agent:"web-agent", task:"React登录组件"},<br/>{agent:"code-agent", task:"Go登录接口"}<br/>]}
    
    ORC-->>GW: AG-UI: STATE_UPDATE {phase:"orchestrating", plan:"拆解为2个子任务"}
    GW-->>AGUI: SSE → React → User: 显示编排状态卡片
    
    Note over ORC: === 并行A2A调用 ===
    
    par 并行执行
        ORC->>WebAgent: POST /a2a/tasks/sendSubscribe {task:"React登录组件"}
    and
        ORC->>CodeAgent: POST /a2a/tasks/sendSubscribe {task:"Go登录接口"}
    end
    
    Note over ORC: === Web-Agent 响应流 ===
    
    ORC-->>GW: AG-UI: STATE_UPDATE {activeAgent:"web-agent"}
    GW-->>AGUI: SSE → React: 切换显示Web-Agent头像
    
    WebAgent-->>ORC: A2A: status=working
    ORC-->>GW: AG-UI: TEXT_MESSAGE_START {messageId:"msg-web-1"}
    GW-->>AGUI: SSE → React: 创建Web-Agent消息气泡
    
    loop Web-Agent流式输出
        WebAgent-->>ORC: A2A: text chunk
        ORC-->>GW: AG-UI: TEXT_MESSAGE_CONTENT
        GW-->>AGUI: SSE → React → User: 实时渲染
    end
    
    WebAgent-->>ORC: A2A: artifact {type:"webpage", html:"<div>...", css:"..."}
    WebAgent-->>ORC: A2A: artifact {type:"code", content:"LoginForm.tsx"}
    WebAgent-->>ORC: A2A: status=completed
    
    ORC-->>GW: AG-UI: TEXT_MESSAGE_END
    
    ORC-->>GW: AG-UI: TOOL_CALL(web_preview) {html, css, title:"登录页面预览"}
    GW-->>AGUI: SSE → React: 渲染<WebPreview />组件
    React-->>User: 看到登录页面的实时预览(iframe)
    
    ORC-->>GW: AG-UI: TOOL_CALL(code_preview) {code:"LoginForm.tsx", language:"tsx"}
    GW-->>AGUI: SSE → React: 渲染<CodePreview />组件
    
    Note over ORC: === Code-Agent 响应流 ===
    
    ORC-->>GW: AG-UI: STATE_UPDATE {activeAgent:"code-agent"}
    GW-->>AGUI: SSE → React: 切换显示Code-Agent头像
    
    CodeAgent-->>ORC: A2A: status=working
    ORC-->>GW: AG-UI: TEXT_MESSAGE_START {messageId:"msg-code-1"}
    GW-->>AGUI: SSE → React: 创建Code-Agent消息气泡
    
    loop Code-Agent流式输出
        CodeAgent-->>ORC: A2A: text chunk
        ORC-->>GW: AG-UI: TEXT_MESSAGE_CONTENT
        GW-->>AGUI: SSE → React → User: 实时渲染
    end
    
    CodeAgent-->>ORC: A2A: artifact {type:"code", content:"handler.go"}
    CodeAgent-->>ORC: A2A: status=completed
    
    ORC-->>GW: AG-UI: TEXT_MESSAGE_END
    ORC-->>GW: AG-UI: TOOL_CALL(code_preview) {code:"handler.go", language:"go"}
    GW-->>AGUI: SSE → React: 渲染<CodePreview />组件
    React-->>User: 看到Go代码卡片
    
    Note over ORC: === 结果聚合 ===
    ORC-->>GW: AG-UI: STATE_UPDATE {phase:"completed", summary:"已完成2个子任务"}
    ORC-->>GW: AG-UI: RUN_FINISHED
    GW-->>AGUI: SSE → React → User: 显示完成状态
```

#### 2.6.3 场景三：交互式Skill（Agent请求用户输入）

子Agent在执行过程中需要用户确认或提供额外信息。

```mermaid
sequenceDiagram
    autonumber
    participant User as 👤 用户
    participant React as 🖥️ React前端
    participant AGUI as AG-UI Client
    participant GW as 🚪 Gateway网关
    participant ORC as 🧠 Orchestrator
    participant DeployAgent as 🚀 Deploy-Agent

    User->>React: "把刚才的代码部署到线上"
    React->>AGUI: RunAgentInput
    AGUI->>GW: POST /api/agui/run (SSE)
    GW->>ORC: 转发
    
    ORC->>ORC: 意图编排 → deploy-agent
    ORC->>DeployAgent: A2A tasks/sendSubscribe
    
    DeployAgent-->>ORC: A2A: status=working
    DeployAgent-->>ORC: A2A: text "准备部署，检测到以下变更..."
    
    ORC-->>GW: AG-UI: TEXT_MESSAGE_START → CONTENT → END
    GW-->>AGUI: SSE → React → User: 显示部署准备信息
    
    Note over DeployAgent: 需要用户确认危险操作
    DeployAgent-->>ORC: A2A: tool_call {name:"confirm_action", args:{...}}
    
    Note over ORC: 识别为前端Skill → 转发
    ORC-->>GW: AG-UI: TOOL_CALL_START {toolName:"confirm_action"}
    ORC-->>GW: AG-UI: TOOL_CALL_ARGS {title:"确认部署", message:"将部署到生产环境", danger:true}
    ORC-->>GW: AG-UI: TOOL_CALL_END
    GW-->>AGUI: SSE events
    
    AGUI->>React: 执行confirm_action Skill
    React->>React: 渲染<ConfirmDialog />
    React-->>User: 弹出确认对话框 "确认部署到生产环境？"
    
    Note over User: 用户思考后点击"确认"
    User->>React: 点击"确认"按钮
    
    React->>AGUI: Skill结果 {confirmed: true}
    AGUI->>GW: POST /api/agui/run/{runId}/tool-result {confirmed:true}
    GW->>ORC: 转发ToolResult
    
    Note over ORC: 将确认结果转为A2A消息回传
    ORC->>DeployAgent: A2A message {role:"tool", content:"用户已确认部署"}
    
    DeployAgent->>DeployAgent: 执行部署流程
    DeployAgent-->>ORC: A2A: text "正在部署中..."
    ORC-->>GW: AG-UI: TEXT_MESSAGE_CONTENT
    GW-->>AGUI: SSE → React → User: "正在部署中..."
    
    DeployAgent-->>ORC: A2A: artifact {type:"deploy", status:"success", url:"https://app.example.com"}
    DeployAgent-->>ORC: A2A: status=completed
    
    ORC-->>GW: AG-UI: TEXT_MESSAGE_END
    ORC-->>GW: AG-UI: TOOL_CALL(deploy_status) {status:"success", url:"...", projectName:"login-app"}
    GW-->>AGUI: SSE → React: 渲染<DeployCard />
    React-->>User: 看到部署成功卡片(含访问链接)
    
    ORC-->>GW: AG-UI: RUN_FINISHED
```

#### 2.6.4 场景四：Agent注册与发现

系统启动时，子Agent自动注册到Orchestrator。

```mermaid
sequenceDiagram
    autonumber
    participant CodeAgent as 🔧 Code-Agent
    participant WebAgent as 🎨 Web-Agent
    participant DocAgent as 📝 Doc-Agent
    participant Registry as 📋 Agent注册表
    participant ORC as 🧠 Orchestrator
    participant GW as 🚪 Gateway

    Note over CodeAgent,DocAgent: === 系统启动阶段 ===
    
    par 子Agent启动
        CodeAgent->>CodeAgent: ADK Runtime启动<br/>加载config.yaml<br/>暴露A2A端点
    and
        WebAgent->>WebAgent: ADK Runtime启动
    and
        DocAgent->>DocAgent: ADK Runtime启动
    end
    
    Note over Registry: === Agent注册阶段 ===
    
    ORC->>Registry: 初始化注册表(从环境变量读取Agent URLs)
    
    Registry->>CodeAgent: GET /.well-known/agent.json
    CodeAgent-->>Registry: AgentCard {name:"code-agent", skills:[...], outputModes:[...]}
    
    Registry->>WebAgent: GET /.well-known/agent.json
    WebAgent-->>Registry: AgentCard {name:"web-agent", skills:[...], outputModes:[...]}
    
    Registry->>DocAgent: GET /.well-known/agent.json
    DocAgent-->>Registry: AgentCard {name:"doc-agent", skills:[...], outputModes:[...]}
    
    Registry->>Registry: 标记所有Agent为healthy
    
    Note over Registry: === 健康检查(每30s) ===
    
    loop 定期健康检查
        Registry->>CodeAgent: GET /health
        CodeAgent-->>Registry: 200 OK
        Registry->>WebAgent: GET /health
        WebAgent-->>Registry: 200 OK
        Registry->>DocAgent: GET /health
        DocAgent-->>Registry: 200 OK
        Registry->>Registry: 更新状态 + lastCheck时间
    end
    
    Note over GW: === 前端获取Agent列表 ===
    
    GW->>Registry: GetAllCards()
    Registry-->>GW: [code-agent-card, web-agent-card, doc-agent-card]
    GW-->>GW: 返回给前端展示Agent市场/联系人列表
```

#### 2.6.5 场景五：错误处理与降级

子Agent执行失败时的降级处理流程。

```mermaid
sequenceDiagram
    autonumber
    participant User as 👤 用户
    participant React as 🖥️ React前端
    participant GW as 🚪 Gateway
    participant ORC as 🧠 Orchestrator
    participant AgentA as 🔧 Agent-A (主)
    participant AgentB as 🔧 Agent-B (备)

    User->>React: 发送消息
    React->>GW: AG-UI RunRequest
    GW->>ORC: 转发
    
    ORC->>ORC: 意图编排 → 选择Agent-A
    ORC->>AgentA: A2A tasks/sendSubscribe
    
    AgentA-->>ORC: A2A: status=working
    ORC-->>GW: AG-UI: TEXT_MESSAGE_START
    GW-->>React: SSE → 创建消息气泡
    
    Note over AgentA: ⚠️ Agent-A 发生错误
    AgentA-->>ORC: A2A: status=failed {error:"LLM API timeout"}
    
    Note over ORC: === 降级处理 ===
    ORC->>ORC: 检查是否有备选Agent
    ORC-->>GW: AG-UI: STATE_UPDATE {phase:"retrying", message:"Agent-A失败，切换到Agent-B"}
    GW-->>React: SSE → 显示重试提示
    React-->>User: "正在切换到备用Agent..."
    
    ORC->>AgentB: A2A tasks/sendSubscribe (相同任务)
    
    AgentB-->>ORC: A2A: status=working
    
    loop 正常流式输出
        AgentB-->>ORC: A2A: text chunk
        ORC-->>GW: AG-UI: TEXT_MESSAGE_CONTENT
        GW-->>React: SSE → User: 正常渲染
    end
    
    AgentB-->>ORC: A2A: status=completed
    ORC-->>GW: AG-UI: TEXT_MESSAGE_END
    ORC-->>GW: AG-UI: RUN_FINISHED
    GW-->>React: SSE → User: 完成
    
    Note over ORC: 如果Agent-B也失败
    Note over ORC: → 返回RUN_ERROR<br/>→ 前端显示错误信息<br/>→ 用户可点击"重试"
```

#### 2.6.6 完整系统生命周期总览

```mermaid
sequenceDiagram
    autonumber
    participant User as 👤 用户
    participant FE as 🖥️ 前端(React+AG-UI)
    participant GW as 🚪 Gateway(Go)
    participant DB as 🗄️ DB/Redis
    participant ORC as 🧠 Orchestrator
    participant LLM as 🤖 LLM
    participant REG as 📋 Registry
    participant Agent as 🔧 子Agent(ADK+A2A)

    Note over User,Agent: ━━━ 阶段1: 系统初始化 ━━━
    Agent->>Agent: ADK Runtime启动
    REG->>Agent: 获取AgentCard
    Agent-->>REG: 返回AgentCard
    REG->>REG: 注册 + 健康检查

    Note over User,Agent: ━━━ 阶段2: 用户建立会话 ━━━
    User->>FE: 点击"新建对话"
    FE->>GW: POST /api/conversations {agentId}
    GW->>DB: 创建会话记录
    DB-->>GW: conversationId
    GW-->>FE: 返回会话信息
    FE-->>User: 打开聊天窗口

    Note over User,Agent: ━━━ 阶段3: 发送消息 ━━━
    User->>FE: 输入消息 + 点击发送
    FE->>FE: 乐观更新UI(显示发送中)
    FE->>GW: POST /api/agui/run {threadId, messages, tools(skills)}
    
    Note over User,Agent: ━━━ 阶段4: 网关处理 ━━━
    GW->>GW: JWT鉴权
    GW->>DB: 保存消息 + 查询历史
    GW->>ORC: 转发到Orchestrator

    Note over User,Agent: ━━━ 阶段5: 意图编排 ━━━
    ORC->>REG: 获取可用AgentCards
    ORC->>LLM: 意图分析(用户消息+历史+AgentCards)
    LLM-->>ORC: 执行计划(选择Agent+策略)

    Note over User,Agent: ━━━ 阶段6: A2A调用子Agent ━━━
    ORC->>Agent: A2A tasks/sendSubscribe
    Agent->>Agent: ADK处理(LLM+Tools)
    
    Note over User,Agent: ━━━ 阶段7: 协议转换 + 流式返回 ━━━
    Agent-->>ORC: A2A流式事件(text/artifact/status)
    ORC->>ORC: ProtocolConverter: A2A→AG-UI
    ORC-->>GW: AG-UI事件流
    GW-->>FE: SSE事件流
    FE-->>User: 实时渲染(文字+组件)

    Note over User,Agent: ━━━ 阶段8: 产物展示 ━━━
    ORC-->>GW: AG-UI TOOL_CALL(前端Skill)
    GW-->>FE: SSE TOOL_CALL
    FE->>FE: 执行Skill → 渲染组件
    FE-->>User: 看到代码/网页/文件等预览卡片
    FE->>GW: ToolResult回传

    Note over User,Agent: ━━━ 阶段9: 完成 ━━━
    ORC-->>GW: AG-UI RUN_FINISHED
    GW->>DB: 保存Agent回复
    GW-->>FE: SSE RUN_FINISHED
    FE-->>User: 完成状态
```

---

## 三、编排策略设计（核心决策）

### 3.1 编排方案对比分析

我们考虑了两种编排方案：**意图编排（Intent-based）** 和 **DAG Workflow 编排**。

| 维度 | 意图编排（LLM驱动） | DAG Workflow 编排 |
|------|---------------------|-------------------|
| **原理** | LLM 实时理解用户意图，动态决定调用哪些Agent | 预定义有向无环图，按固定流程执行 |
| **灵活性** | ⭐⭐⭐⭐⭐ 极高，可处理任意自然语言指令 | ⭐⭐⭐ 中等，需预定义流程 |
| **可预测性** | ⭐⭐⭐ 中等，LLM可能产生不同决策 | ⭐⭐⭐⭐⭐ 极高，流程确定 |
| **开发复杂度** | ⭐⭐⭐ 中等，核心是Prompt工程 | ⭐⭐⭐⭐ 较高，需DAG引擎+可视化 |
| **适合场景** | 开放式对话、探索性任务 | 固定流程、批量处理 |
| **延迟** | 较高（需LLM推理） | 较低（直接执行） |
| **错误处理** | LLM可自适应重试和降级 | 需预定义错误处理分支 |
| **IM场景适配** | ⭐⭐⭐⭐⭐ 天然适合对话式交互 | ⭐⭐⭐ 需额外适配对话场景 |

### 3.2 最终方案：纯意图编排（Intent-based Orchestration）

**结论：采用纯意图编排方案。**

**理由：**

1. **IM 场景天然适合意图编排**：用户通过自然语言描述需求，LLM 动态理解并分派，无需用户了解底层 Agent 能力
2. **开发周期短**：意图编排核心是 Prompt 工程 + A2A 调用，无需开发复杂的 DAG 引擎和可视化编辑器
3. **灵活性优先**：课题要求"群聊协作"，用户可能提出任意组合需求，固定流程无法覆盖所有场景
4. **LLM 自适应能力强**：意图编排天然支持错误重试、Agent降级、动态调整策略，无需预定义错误处理分支

### 3.3 意图编排架构

```mermaid
flowchart TD
    A[用户消息] --> B{路由决策}
    B -->|"明确@指定Agent"| C[直接路由]
    B -->|"未指定/复杂任务"| D[LLM 意图编排]
    
    D --> E[LLM分析意图 + 读取AgentCards]
    E --> F[动态生成执行计划]
    F --> G{执行策略?}
    G -->|"single"| H[单Agent A2A调用]
    G -->|"parallel"| I[并行A2A调用]
    G -->|"sequential"| J[串行A2A调用]
    C --> K[单Agent A2A调用]
    
    H --> L[A2A Response → AG-UI Events 转换]
    I --> L
    J --> L
    K --> L
    L --> M[通过AG-UI SSE返回前端]
```

### 3.4 意图编排实现

```go
// 统一Agent - 意图编排器 (Go)
type IntentOrchestrator struct {
    llm         LLMClient
    agentRegistry *AgentRegistry  // 已注册的子Agent列表
    taskManager *TaskManager
}

// 编排决策
func (o *IntentOrchestrator) Orchestrate(ctx context.Context, message string, history []Message) (*ExecutionPlan, error) {
    // 1. 获取所有可用Agent的AgentCard
    agentCards := o.agentRegistry.GetAllAgentCards()
    
    // 2. 构建编排Prompt
    prompt := buildOrchestratorPrompt(message, history, agentCards)
    
    // 3. LLM 生成执行计划
    plan, err := o.llm.GenerateStructured(ctx, prompt, ExecutionPlanSchema)
    if err != nil {
        return nil, err
    }
    
    return plan, nil
}

// 执行计划
type ExecutionPlan struct {
    Intent      string       `json:"intent"`       // 识别的意图
    Strategy    string       `json:"strategy"`     // parallel | sequential | single
    Tasks       []TaskPlan   `json:"tasks"`        // 子任务列表
    FallbackPlan *ExecutionPlan `json:"fallback"`  // 降级方案
}

type TaskPlan struct {
    AgentName   string   `json:"agent_name"`   // 目标Agent
    TaskContent string   `json:"task_content"` // 任务描述
    DependsOn   []string `json:"depends_on"`   // 依赖的其他任务ID
    Priority    int      `json:"priority"`
}

// 编排Prompt模板
func buildOrchestratorPrompt(message string, history []Message, agents []AgentCard) string {
    return fmt.Sprintf(`你是一个任务编排器。根据用户消息，决定如何分配任务给可用的Agent。

## 可用Agent列表：
%s

## 对话历史：
%s

## 用户最新消息：
%s

## 输出要求：
返回JSON格式的执行计划，包含：
- intent: 用户意图摘要
- strategy: "single"(单Agent) | "parallel"(并行多Agent) | "sequential"(串行多Agent)
- tasks: 任务列表，每个任务指定目标agent_name和task_content

注意：
1. 如果任务只需要一个Agent，strategy设为"single"
2. 如果多个任务互不依赖，strategy设为"parallel"
3. 如果任务有先后依赖关系，strategy设为"sequential"，并设置depends_on
`, formatAgentCards(agents), formatHistory(history), message)
}
```

### 3.5 A2A ↔ AG-UI 协议转换层（核心设计）

统一Agent（Orchestrator）的核心职责之一是在 **A2A 协议**（子Agent返回）和 **AG-UI 协议**（前端消费）之间进行双向转换。这是整个系统的关键桥梁。

#### 3.5.1 协议转换总览

```mermaid
flowchart LR
    subgraph 前端["前端 (AG-UI Consumer)"]
        FE[React UI]
    end
    
    subgraph 转换层["Orchestrator 协议转换层"]
        direction TB
        A2A_IN[A2A Events 输入] --> CONVERTER[Protocol Converter]
        CONVERTER --> AGUI_OUT[AG-UI Events 输出]
        AGUI_IN[AG-UI Tool Results 输入] --> CONVERTER2[Reverse Converter]
        CONVERTER2 --> A2A_OUT[A2A Messages 输出]
    end
    
    subgraph 子Agent["子Agent (A2A Provider)"]
        AGENT[ADK Runtime]
    end
    
    FE <-->|"AG-UI Protocol"| 转换层
    转换层 <-->|"A2A Protocol"| AGENT
```

#### 3.5.2 A2A → AG-UI 事件映射表

| A2A 事件/字段 | AG-UI 事件 | 说明 |
|--------------|-----------|------|
| `Task.status = "working"` | `RUN_STARTED` | Agent开始处理任务 |
| `Task.status = "completed"` | `RUN_FINISHED` | Agent完成任务 |
| `Task.status = "failed"` | `RUN_ERROR` | Agent处理失败 |
| `streaming.text_chunk` | `TEXT_MESSAGE_CONTENT` | 流式文本内容 |
| `streaming.start` | `TEXT_MESSAGE_START` | 消息开始 |
| `streaming.end` | `TEXT_MESSAGE_END` | 消息结束 |
| `Task.artifacts[]` | `TOOL_CALL_START/ARGS/END` | 产物转为前端Skill调用 |
| `Task.artifacts[].type = "code"` | Tool Call → `code_preview` | 代码产物触发代码预览Skill |
| `Task.artifacts[].type = "webpage"` | Tool Call → `web_preview` | 网页产物触发网页预览Skill |
| `Task.artifacts[].type = "file"` | Tool Call → `file_download` | 文件产物触发下载Skill |
| `Task.metadata.agent_name` | `STATE_UPDATE {activeAgent}` | 当前活跃Agent信息 |
| `Task.metadata.progress` | `STATE_UPDATE {progress}` | 任务进度 |

#### 3.5.3 AG-UI → A2A 请求映射表

| AG-UI 输入 | A2A 请求 | 说明 |
|-----------|---------|------|
| `RunAgent.messages[]` | `Task.messages[]` | 用户消息转为A2A Task消息 |
| `RunAgent.messages[].role = "user"` | `TaskMessage.role = "user"` | 角色映射 |
| `RunAgent.context.mentions` | 路由决策依据 | @的Agent列表用于编排 |
| `RunAgent.tools[]` (前端Skills) | 存储在Orchestrator上下文 | 前端能力声明，供产物转换使用 |
| `ToolResult` (前端Skill执行结果) | `Task.messages[] (tool_result)` | Skill结果回传给子Agent |

#### 3.5.4 协议转换器实现

```go
// 协议转换器 - A2A Events → AG-UI Events
type ProtocolConverter struct {
    frontendSkills []FrontendSkill  // 前端注册的Skills列表
}

// 前端Skill定义（从AG-UI RunRequest中获取）
type FrontendSkill struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  json.RawMessage `json:"parameters"`
}

// A2A流式事件 → AG-UI事件流
func (pc *ProtocolConverter) ConvertA2AStreamToAGUI(
    a2aStream <-chan A2AStreamEvent,
    aguiEvents chan<- AGUIEvent,
    agentName string,
) error {
    // 通知前端当前活跃的Agent
    aguiEvents <- AGUIEvent{
        Type:  "STATE_UPDATE",
        State: map[string]interface{}{"activeAgent": agentName, "phase": "generating"},
    }
    
    var messageID string
    var artifactBuffer []A2AArtifact
    
    for event := range a2aStream {
        switch event.Type {
        
        case A2AEventStatusUpdate:
            switch event.Status {
            case "working":
                messageID = uuid.New().String()
                aguiEvents <- AGUIEvent{
                    Type:      "TEXT_MESSAGE_START",
                    MessageID: messageID,
                }
            case "completed":
                aguiEvents <- AGUIEvent{
                    Type:      "TEXT_MESSAGE_END",
                    MessageID: messageID,
                }
                // 处理产物 → 转为前端Skill调用
                for _, artifact := range artifactBuffer {
                    pc.convertArtifactToToolCall(artifact, aguiEvents)
                }
            case "failed":
                aguiEvents <- AGUIEvent{
                    Type:  "RUN_ERROR",
                    Error: event.Error,
                }
                return fmt.Errorf("agent %s failed: %s", agentName, event.Error)
            }
            
        case A2AEventTextChunk:
            // 流式文本直接转发
            aguiEvents <- AGUIEvent{
                Type:      "TEXT_MESSAGE_CONTENT",
                MessageID: messageID,
                Content:   event.Text,
            }
            
        case A2AEventArtifact:
            // 收集产物，在消息结束后统一处理
            artifactBuffer = append(artifactBuffer, event.Artifact)
            
        case A2AEventToolCall:
            // 子Agent请求调用工具 → 检查是否为前端Skill
            if pc.isFrontendSkill(event.ToolName) {
                // 转发为AG-UI Tool Call，让前端执行
                aguiEvents <- AGUIEvent{
                    Type:       "TOOL_CALL_START",
                    ToolCallID: event.ToolCallID,
                    ToolName:   event.ToolName,
                }
                aguiEvents <- AGUIEvent{
                    Type:       "TOOL_CALL_ARGS",
                    ToolCallID: event.ToolCallID,
                    Content:    event.ToolArgs,
                }
                aguiEvents <- AGUIEvent{
                    Type:       "TOOL_CALL_END",
                    ToolCallID: event.ToolCallID,
                }
            }
        }
    }
    
    return nil
}

// 产物 → 前端Skill Tool Call 转换
func (pc *ProtocolConverter) convertArtifactToToolCall(artifact A2AArtifact, events chan<- AGUIEvent) {
    toolCallID := uuid.New().String()
    
    var toolName string
    var toolArgs map[string]interface{}
    
    switch artifact.Type {
    case "code":
        toolName = "code_preview"
        toolArgs = map[string]interface{}{
            "code":     artifact.Content,
            "language": artifact.Metadata["language"],
            "filename": artifact.Title,
        }
    case "webpage":
        toolName = "web_preview"
        toolArgs = map[string]interface{}{
            "html":  artifact.Content,
            "title": artifact.Title,
            "css":   artifact.Metadata["css"],
            "js":    artifact.Metadata["js"],
        }
    case "file":
        toolName = "file_download"
        toolArgs = map[string]interface{}{
            "url":      artifact.URL,
            "filename": artifact.Title,
            "size":     artifact.Metadata["size"],
            "mimeType": artifact.Metadata["mimeType"],
        }
    case "diff":
        toolName = "diff_preview"
        toolArgs = map[string]interface{}{
            "filename":   artifact.Title,
            "oldContent": artifact.Metadata["oldContent"],
            "newContent": artifact.Content,
        }
    case "image":
        toolName = "image_preview"
        toolArgs = map[string]interface{}{
            "url":   artifact.URL,
            "alt":   artifact.Title,
            "width": artifact.Metadata["width"],
        }
    default:
        // 未知类型，作为通用文件处理
        toolName = "file_download"
        toolArgs = map[string]interface{}{
            "url":      artifact.URL,
            "filename": artifact.Title,
        }
    }
    
    // 检查前端是否注册了该Skill
    if !pc.isFrontendSkill(toolName) {
        // 前端未注册该Skill，降级为文本消息
        events <- AGUIEvent{
            Type:      "TEXT_MESSAGE_START",
            MessageID: toolCallID,
        }
        events <- AGUIEvent{
            Type:      "TEXT_MESSAGE_CONTENT",
            MessageID: toolCallID,
            Content:   fmt.Sprintf("[产物: %s] %s", artifact.Type, artifact.Title),
        }
        events <- AGUIEvent{
            Type:      "TEXT_MESSAGE_END",
            MessageID: toolCallID,
        }
        return
    }
    
    // 发送 Tool Call 事件序列
    argsJSON, _ := json.Marshal(toolArgs)
    events <- AGUIEvent{Type: "TOOL_CALL_START", ToolCallID: toolCallID, ToolName: toolName}
    events <- AGUIEvent{Type: "TOOL_CALL_ARGS", ToolCallID: toolCallID, Content: string(argsJSON)}
    events <- AGUIEvent{Type: "TOOL_CALL_END", ToolCallID: toolCallID}
}

// 检查是否为前端注册的Skill
func (pc *ProtocolConverter) isFrontendSkill(toolName string) bool {
    for _, skill := range pc.frontendSkills {
        if skill.Name == toolName {
            return true
        }
    }
    return false
}
```

#### 3.5.5 反向转换：AG-UI Tool Result → A2A Message

当前端执行完 Skill 后，需要将结果回传给子Agent（如果子Agent需要）：

```go
// 前端Skill执行结果 → A2A消息
func (pc *ProtocolConverter) ConvertToolResultToA2A(toolResult AGUIToolResult) *A2AMessage {
    return &A2AMessage{
        Role: "tool",
        Content: fmt.Sprintf("Tool '%s' executed successfully. Result: %s", 
            toolResult.ToolName, toolResult.Result),
        Metadata: map[string]interface{}{
            "toolCallId": toolResult.ToolCallID,
            "toolName":   toolResult.ToolName,
            "success":    toolResult.Success,
        },
    }
}
```

#### 3.5.6 多Agent场景下的协议转换

群聊模式下，多个子Agent的A2A响应需要合并为一个AG-UI事件流：

```go
// 多Agent A2A响应 → 统一AG-UI事件流
func (pc *ProtocolConverter) MergeMultiAgentResponses(
    agentStreams map[string]<-chan A2AStreamEvent, // agentName → stream
    aguiEvents chan<- AGUIEvent,
    strategy string, // "parallel" | "sequential"
) error {
    switch strategy {
    case "sequential":
        // 串行：按顺序处理每个Agent的响应
        for agentName, stream := range agentStreams {
            err := pc.ConvertA2AStreamToAGUI(stream, aguiEvents, agentName)
            if err != nil {
                // 记录错误但继续处理其他Agent
                aguiEvents <- AGUIEvent{
                    Type:  "STATE_UPDATE",
                    State: map[string]interface{}{"agentError": agentName, "error": err.Error()},
                }
            }
        }
        
    case "parallel":
        // 并行：同时处理所有Agent响应，按完成顺序推送
        var wg sync.WaitGroup
        errChan := make(chan error, len(agentStreams))
        
        for agentName, stream := range agentStreams {
            wg.Add(1)
            go func(name string, s <-chan A2AStreamEvent) {
                defer wg.Done()
                if err := pc.ConvertA2AStreamToAGUI(s, aguiEvents, name); err != nil {
                    errChan <- fmt.Errorf("agent %s: %w", name, err)
                }
            }(agentName, stream)
        }
        
        wg.Wait()
        close(errChan)
        
        // 收集错误
        for err := range errChan {
            aguiEvents <- AGUIEvent{
                Type:  "STATE_UPDATE",
                State: map[string]interface{}{"error": err.Error()},
            }
        }
    }
    
    return nil
}
```

#### 3.5.7 完整转换流程时序图

```mermaid
sequenceDiagram
    participant FE as 前端(AG-UI)
    participant ORC as Orchestrator(转换层)
    participant Agent as 子Agent(A2A)

    FE->>ORC: AG-UI RunRequest {messages, tools(skills)}
    Note over ORC: 保存前端Skills列表
    Note over ORC: 意图编排 → 选择Agent
    
    ORC->>Agent: A2A tasks/sendSubscribe {messages}
    
    Agent-->>ORC: A2A SSE: {type:"status", status:"working"}
    ORC-->>FE: AG-UI SSE: {type:"TEXT_MESSAGE_START"}
    
    Agent-->>ORC: A2A SSE: {type:"text", content:"正在生成..."}
    ORC-->>FE: AG-UI SSE: {type:"TEXT_MESSAGE_CONTENT", content:"正在生成..."}
    
    Agent-->>ORC: A2A SSE: {type:"text", content:"代码如下..."}
    ORC-->>FE: AG-UI SSE: {type:"TEXT_MESSAGE_CONTENT", content:"代码如下..."}
    
    Agent-->>ORC: A2A SSE: {type:"artifact", artifact:{type:"code", content:"..."}}
    Note over ORC: 缓存产物，等待消息结束
    
    Agent-->>ORC: A2A SSE: {type:"status", status:"completed"}
    ORC-->>FE: AG-UI SSE: {type:"TEXT_MESSAGE_END"}
    
    Note over ORC: 产物→Skill Tool Call转换
    ORC-->>FE: AG-UI SSE: {type:"TOOL_CALL_START", toolName:"code_preview"}
    ORC-->>FE: AG-UI SSE: {type:"TOOL_CALL_ARGS", content:"{code,language,filename}"}
    ORC-->>FE: AG-UI SSE: {type:"TOOL_CALL_END"}
    
    Note over FE: 前端执行code_preview Skill
    FE->>ORC: AG-UI ToolResult {success: true}
    
    ORC-->>FE: AG-UI SSE: {type:"RUN_FINISHED"}
```

---

### 3.6 前端组件注册（Skills）详细设计

### 3.6.1 Skills 机制说明

在 AG-UI 协议中，**Skills** 是前端向 Agent 声明的能力。Agent（通过 Orchestrator）可以通过 `TOOL_CALL` 事件调用这些前端 Skills，实现"Agent 驱动前端渲染"的模式。

**核心理念：** 子Agent 产出的产物（代码、网页、文件等）不是简单的文本返回，而是通过 A2A Artifact → AG-UI Tool Call → 前端 Skill 执行的链路，在前端以富组件形式呈现。

```mermaid
flowchart LR
    subgraph 子Agent
        A[生成代码/网页/文件] --> B[A2A Artifact]
    end
    
    subgraph Orchestrator
        B --> C[协议转换器]
        C --> D[匹配前端Skill]
        D --> E[生成AG-UI TOOL_CALL]
    end
    
    subgraph 前端
        E --> F[Skill执行器]
        F --> G[渲染对应组件]
    end
```

### 3.6.2 前端 Skills 注册表（完整列表）

以下是前端需要注册的所有 Skills，子Agent 可以通过产物（Artifact）间接触发这些 Skills：

| Skill 名称 | 触发来源（A2A Artifact Type） | 前端组件 | 说明 |
|------------|-------------------------------|----------|------|
| `code_preview` | `artifact.type = "code"` | `<CodePreview />` | 代码高亮预览 + 操作按钮 |
| `web_preview` | `artifact.type = "webpage"` | `<WebPreview />` | iframe 沙箱网页预览 |
| `diff_preview` | `artifact.type = "diff"` | `<DiffView />` | 代码 Diff 对比视图 |
| `file_download` | `artifact.type = "file"` | `<FileCard />` | 文件下载卡片 |
| `image_preview` | `artifact.type = "image"` | `<ImagePreview />` | 图片预览 |
| `deploy_status` | `artifact.type = "deploy"` | `<DeployCard />` | 部署状态卡片 |
| `markdown_render` | `artifact.type = "document"` | `<MarkdownView />` | Markdown 文档渲染 |
| `terminal_output` | `artifact.type = "terminal"` | `<TerminalOutput />` | 终端输出展示 |
| `chart_render` | `artifact.type = "chart"` | `<ChartView />` | 图表渲染（Echarts） |
| `form_input` | Agent 主动请求用户输入 | `<FormInput />` | 动态表单（Agent向用户收集信息） |
| `confirm_action` | Agent 请求用户确认 | `<ConfirmDialog />` | 确认对话框（危险操作前确认） |
| `file_upload` | Agent 请求用户上传文件 | `<FileUpload />` | 文件上传组件 |

### 3.6.3 Skills 注册代码

```typescript
// skills/registry.ts - 前端Skills注册中心

import { SkillDefinition, SkillExecutor } from '@/agui/types';

/**
 * 所有前端Skills定义
 * 这些Skills会在AG-UI RunRequest中声明给后端
 * Agent可以通过TOOL_CALL事件调用这些Skills
 */
export const FRONTEND_SKILLS: SkillDefinition[] = [
  // ============ 产物展示类 Skills ============
  {
    name: 'code_preview',
    description: '在前端展示代码预览，支持语法高亮、复制、编辑操作',
    parameters: {
      type: 'object',
      properties: {
        code: { type: 'string', description: '代码内容' },
        language: { type: 'string', description: '编程语言（如 typescript, go, python）' },
        filename: { type: 'string', description: '文件名' },
        highlightLines: { 
          type: 'array', 
          items: { type: 'number' },
          description: '需要高亮的行号列表' 
        },
      },
      required: ['code', 'language'],
    },
  },
  {
    name: 'web_preview',
    description: '在前端渲染网页预览（iframe沙箱），支持HTML/CSS/JS',
    parameters: {
      type: 'object',
      properties: {
        html: { type: 'string', description: 'HTML内容' },
        css: { type: 'string', description: 'CSS样式' },
        js: { type: 'string', description: 'JavaScript代码' },
        title: { type: 'string', description: '预览标题' },
        height: { type: 'number', description: '预览高度(px)，默认400' },
      },
      required: ['html', 'title'],
    },
  },
  {
    name: 'diff_preview',
    description: '展示代码Diff对比视图，支持接受/拒绝变更',
    parameters: {
      type: 'object',
      properties: {
        filename: { type: 'string', description: '文件名' },
        oldContent: { type: 'string', description: '修改前的代码' },
        newContent: { type: 'string', description: '修改后的代码' },
        language: { type: 'string', description: '编程语言' },
      },
      required: ['filename', 'oldContent', 'newContent'],
    },
  },
  {
    name: 'file_download',
    description: '展示文件下载卡片，用户可点击下载',
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string', description: '文件下载URL' },
        filename: { type: 'string', description: '文件名' },
        size: { type: 'number', description: '文件大小(bytes)' },
        mimeType: { type: 'string', description: 'MIME类型' },
      },
      required: ['url', 'filename'],
    },
  },
  {
    name: 'image_preview',
    description: '展示图片预览',
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string', description: '图片URL' },
        alt: { type: 'string', description: '图片描述' },
        width: { type: 'number', description: '显示宽度' },
      },
      required: ['url'],
    },
  },
  {
    name: 'deploy_status',
    description: '展示部署状态卡片，包含部署进度和访问链接',
    parameters: {
      type: 'object',
      properties: {
        status: { type: 'string', enum: ['deploying', 'success', 'failed'], description: '部署状态' },
        url: { type: 'string', description: '部署后的访问URL' },
        logs: { type: 'string', description: '部署日志' },
        projectName: { type: 'string', description: '项目名称' },
      },
      required: ['status', 'projectName'],
    },
  },
  {
    name: 'markdown_render',
    description: '渲染Markdown文档，支持目录、代码块、表格等',
    parameters: {
      type: 'object',
      properties: {
        content: { type: 'string', description: 'Markdown内容' },
        title: { type: 'string', description: '文档标题' },
      },
      required: ['content'],
    },
  },
  {
    name: 'terminal_output',
    description: '展示终端/命令行输出',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '执行的命令' },
        output: { type: 'string', description: '命令输出' },
        exitCode: { type: 'number', description: '退出码' },
      },
      required: ['output'],
    },
  },
  {
    name: 'chart_render',
    description: '渲染数据图表（基于ECharts）',
    parameters: {
      type: 'object',
      properties: {
        chartType: { type: 'string', enum: ['line', 'bar', 'pie', 'scatter'], description: '图表类型' },
        data: { type: 'object', description: 'ECharts option配置' },
        title: { type: 'string', description: '图表标题' },
        height: { type: 'number', description: '图表高度' },
      },
      required: ['chartType', 'data'],
    },
  },

  // ============ 交互类 Skills（Agent向用户请求输入） ============
  {
    name: 'form_input',
    description: '向用户展示动态表单，收集结构化输入',
    parameters: {
      type: 'object',
      properties: {
        title: { type: 'string', description: '表单标题' },
        description: { type: 'string', description: '表单说明' },
        fields: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              name: { type: 'string' },
              label: { type: 'string' },
              type: { type: 'string', enum: ['text', 'textarea', 'select', 'number', 'checkbox'] },
              required: { type: 'boolean' },
              options: { type: 'array', items: { type: 'string' } },
              placeholder: { type: 'string' },
            },
          },
          description: '表单字段定义',
        },
      },
      required: ['title', 'fields'],
    },
  },
  {
    name: 'confirm_action',
    description: '向用户展示确认对话框（用于危险操作前的确认）',
    parameters: {
      type: 'object',
      properties: {
        title: { type: 'string', description: '确认标题' },
        message: { type: 'string', description: '确认内容描述' },
        confirmText: { type: 'string', description: '确认按钮文字，默认"确认"' },
        cancelText: { type: 'string', description: '取消按钮文字，默认"取消"' },
        danger: { type: 'boolean', description: '是否为危险操作（红色按钮）' },
      },
      required: ['title', 'message'],
    },
  },
  {
    name: 'file_upload',
    description: '请求用户上传文件',
    parameters: {
      type: 'object',
      properties: {
        accept: { type: 'string', description: '接受的文件类型（如 .ts,.js,.go）' },
        maxSize: { type: 'number', description: '最大文件大小(bytes)' },
        multiple: { type: 'boolean', description: '是否允许多文件' },
        description: { type: 'string', description: '上传说明' },
      },
      required: ['accept'],
    },
  },
];
```

### 3.6.4 Skill 执行器实现

```typescript
// skills/executor.ts - Skill执行器

import { AGUIToolCall } from '@/agui/types';
import { useMessageStore } from '@/stores/messageStore';

/**
 * Skill执行器 - 根据AG-UI TOOL_CALL事件执行对应的前端组件渲染
 */
export class SkillExecutor {
  private skillHandlers: Map<string, SkillHandler> = new Map();

  constructor() {
    this.registerAllSkills();
  }

  private registerAllSkills() {
    // 产物展示类
    this.skillHandlers.set('code_preview', new CodePreviewSkillHandler());
    this.skillHandlers.set('web_preview', new WebPreviewSkillHandler());
    this.skillHandlers.set('diff_preview', new DiffPreviewSkillHandler());
    this.skillHandlers.set('file_download', new FileDownloadSkillHandler());
    this.skillHandlers.set('image_preview', new ImagePreviewSkillHandler());
    this.skillHandlers.set('deploy_status', new DeployStatusSkillHandler());
    this.skillHandlers.set('markdown_render', new MarkdownRenderSkillHandler());
    this.skillHandlers.set('terminal_output', new TerminalOutputSkillHandler());
    this.skillHandlers.set('chart_render', new ChartRenderSkillHandler());
    
    // 交互类
    this.skillHandlers.set('form_input', new FormInputSkillHandler());
    this.skillHandlers.set('confirm_action', new ConfirmActionSkillHandler());
    this.skillHandlers.set('file_upload', new FileUploadSkillHandler());
  }

  /**
   * 执行Skill
   * @returns Skill执行结果，将通过AG-UI ToolResult回传给后端
   */
  async execute(toolCall: AGUIToolCall): Promise<SkillResult> {
    const handler = this.skillHandlers.get(toolCall.toolName);
    if (!handler) {
      return { success: false, error: `Unknown skill: ${toolCall.toolName}` };
    }

    try {
      const params = JSON.parse(toolCall.args);
      const result = await handler.execute(params);
      return { success: true, data: result };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  }
}

// Skill处理器接口
interface SkillHandler {
  execute(params: any): Promise<any>;
}

// ============ 各Skill处理器实现 ============

/** 代码预览Skill */
class CodePreviewSkillHandler implements SkillHandler {
  async execute(params: { code: string; language: string; filename?: string; highlightLines?: number[] }) {
    const messageStore = useMessageStore.getState();
    
    // 在消息流中插入代码预览卡片
    messageStore.insertPreviewCard({
      type: 'code',
      props: {
        code: params.code,
        language: params.language,
        filename: params.filename || 'untitled',
        highlightLines: params.highlightLines || [],
        actions: ['copy', 'apply', 'edit', 'expand'],
      },
    });

    return { rendered: true, filename: params.filename };
  }
}

/** 网页预览Skill */
class WebPreviewSkillHandler implements SkillHandler {
  async execute(params: { html: string; css?: string; js?: string; title: string; height?: number }) {
    const messageStore = useMessageStore.getState();
    
    messageStore.insertPreviewCard({
      type: 'webpage',
      props: {
        html: params.html,
        css: params.css || '',
        js: params.js || '',
        title: params.title,
        height: params.height || 400,
        sandbox: ['allow-scripts', 'allow-same-origin'],
      },
    });

    return { rendered: true, title: params.title };
  }
}

/** Diff预览Skill */
class DiffPreviewSkillHandler implements SkillHandler {
  async execute(params: { filename: string; oldContent: string; newContent: string; language?: string }) {
    const messageStore = useMessageStore.getState();
    
    messageStore.insertPreviewCard({
      type: 'diff',
      props: {
        filename: params.filename,
        oldContent: params.oldContent,
        newContent: params.newContent,
        language: params.language || 'text',
        actions: ['accept', 'reject', 'edit'],
      },
    });

    return { rendered: true, filename: params.filename };
  }
}

/** 确认对话框Skill（交互类，需要等待用户响应） */
class ConfirmActionSkillHandler implements SkillHandler {
  async execute(params: { title: string; message: string; confirmText?: string; cancelText?: string; danger?: boolean }) {
    // 这是一个阻塞式Skill，需要等待用户点击
    return new Promise((resolve) => {
      const messageStore = useMessageStore.getState();
      
      messageStore.insertInteractiveCard({
        type: 'confirm',
        props: {
          title: params.title,
          message: params.message,
          confirmText: params.confirmText || '确认',
          cancelText: params.cancelText || '取消',
          danger: params.danger || false,
          onConfirm: () => resolve({ confirmed: true }),
          onCancel: () => resolve({ confirmed: false }),
        },
      });
    });
  }
}

/** 表单输入Skill（交互类） */
class FormInputSkillHandler implements SkillHandler {
  async execute(params: { title: string; description?: string; fields: FormField[] }) {
    return new Promise((resolve) => {
      const messageStore = useMessageStore.getState();
      
      messageStore.insertInteractiveCard({
        type: 'form',
        props: {
          title: params.title,
          description: params.description,
          fields: params.fields,
          onSubmit: (formData: Record<string, any>) => resolve({ submitted: true, data: formData }),
          onCancel: () => resolve({ submitted: false }),
        },
      });
    });
  }
}

/** 文件上传Skill（交互类） */
class FileUploadSkillHandler implements SkillHandler {
  async execute(params: { accept: string; maxSize?: number; multiple?: boolean; description?: string }) {
    return new Promise((resolve) => {
      const messageStore = useMessageStore.getState();
      
      messageStore.insertInteractiveCard({
        type: 'file_upload',
        props: {
          accept: params.accept,
          maxSize: params.maxSize || 10 * 1024 * 1024, // 默认10MB
          multiple: params.multiple || false,
          description: params.description || '请上传文件',
          onUpload: (files: File[]) => resolve({ uploaded: true, files: files.map(f => ({ name: f.name, size: f.size })) }),
          onCancel: () => resolve({ uploaded: false }),
        },
      });
    });
  }
}
```

### 3.6.5 前端组件与子Agent的对应关系

每个子Agent根据其能力，会产出不同类型的 Artifact，从而触发不同的前端 Skills：

| 子Agent | 产出的 Artifact 类型 | 触发的前端 Skill | 前端组件 |
|---------|---------------------|-----------------|----------|
| **code-agent** | `code` | `code_preview` | `<CodePreview />` |
| **code-agent** | `diff` | `diff_preview` | `<DiffView />` |
| **code-agent** | `terminal` | `terminal_output` | `<TerminalOutput />` |
| **web-agent** | `webpage` | `web_preview` | `<WebPreview />` |
| **web-agent** | `code` (CSS/JS) | `code_preview` | `<CodePreview />` |
| **web-agent** | `image` (设计稿) | `image_preview` | `<ImagePreview />` |
| **doc-agent** | `document` | `markdown_render` | `<MarkdownView />` |
| **doc-agent** | `chart` | `chart_render` | `<ChartView />` |
| **doc-agent** | `file` (PDF/PPT) | `file_download` | `<FileCard />` |
| **deploy-agent** | `deploy` | `deploy_status` | `<DeployCard />` |
| **deploy-agent** | `terminal` | `terminal_output` | `<TerminalOutput />` |
| **任意Agent** | 需要用户确认 | `confirm_action` | `<ConfirmDialog />` |
| **任意Agent** | 需要用户输入 | `form_input` | `<FormInput />` |
| **任意Agent** | 需要用户上传 | `file_upload` | `<FileUpload />` |

### 3.6.6 前端组件实现规范

```typescript
// components/Skills/CodePreview.tsx
import React, { useState } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface CodePreviewProps {
  code: string;
  language: string;
  filename: string;
  highlightLines?: number[];
  actions: ('copy' | 'apply' | 'edit' | 'expand')[];
  onAction?: (action: string) => void;
}

export const CodePreview: React.FC<CodePreviewProps> = ({
  code, language, filename, highlightLines = [], actions, onAction
}) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="code-preview-card">
      {/* 头部：文件名 + 操作按钮 */}
      <div className="code-preview-header">
        <span className="filename">{filename}</span>
        <span className="language-badge">{language}</span>
        <div className="actions">
          {actions.includes('copy') && (
            <button onClick={() => navigator.clipboard.writeText(code)}>复制</button>
          )}
          {actions.includes('apply') && (
            <button onClick={() => onAction?.('apply')}>应用</button>
          )}
          {actions.includes('edit') && (
            <button onClick={() => onAction?.('edit')}>编辑</button>
          )}
          {actions.includes('expand') && (
            <button onClick={() => setExpanded(!expanded)}>
              {expanded ? '收起' : '展开'}
            </button>
          )}
        </div>
      </div>
      {/* 代码区域 */}
      <div className={`code-body ${expanded ? 'expanded' : 'collapsed'}`}>
        <SyntaxHighlighter
          language={language}
          style={vscDarkPlus}
          showLineNumbers
          wrapLines
          lineProps={(lineNumber) => ({
            style: highlightLines.includes(lineNumber) 
              ? { backgroundColor: 'rgba(255,255,0,0.1)' } 
              : {},
          })}
        >
          {code}
        </SyntaxHighlighter>
      </div>
    </div>
  );
};

// components/Skills/WebPreview.tsx
interface WebPreviewProps {
  html: string;
  css: string;
  js: string;
  title: string;
  height: number;
  sandbox: string[];
}

export const WebPreview: React.FC<WebPreviewProps> = ({
  html, css, js, title, height, sandbox
}) => {
  const [fullscreen, setFullscreen] = useState(false);
  
  // 构建完整HTML文档
  const fullHTML = `
    <!DOCTYPE html>
    <html>
    <head><style>${css}</style></head>
    <body>${html}<script>${js}</script></body>
    </html>
  `;
  const srcDoc = fullHTML;

  return (
    <div className={`web-preview-card ${fullscreen ? 'fullscreen' : ''}`}>
      <div className="web-preview-header">
        <span className="title">{title}</span>
        <div className="actions">
          <button onClick={() => setFullscreen(!fullscreen)}>
            {fullscreen ? '退出全屏' : '全屏'}
          </button>
          <button onClick={() => window.open(`data:text/html,${encodeURIComponent(fullHTML)}`)}>
            新窗口打开
          </button>
        </div>
      </div>
      <iframe
        srcDoc={srcDoc}
        sandbox={sandbox.join(' ')}
        style={{ width: '100%', height: `${height}px`, border: 'none' }}
        title={title}
      />
    </div>
  );
};

// components/Skills/DiffView.tsx
interface DiffViewProps {
  filename: string;
  oldContent: string;
  newContent: string;
  language: string;
  actions: ('accept' | 'reject' | 'edit')[];
  onAction?: (action: string) => void;
}

export const DiffView: React.FC<DiffViewProps> = ({
  filename, oldContent, newContent, language, actions, onAction
}) => {
  return (
    <div className="diff-view-card">
      <div className="diff-header">
        <span className="filename">{filename}</span>
        <div className="actions">
          {actions.includes('accept') && (
            <button className="btn-accept" onClick={() => onAction?.('accept')}>
              ✓ 接受变更
            </button>
          )}
          {actions.includes('reject') && (
            <button className="btn-reject" onClick={() => onAction?.('reject')}>
              ✗ 拒绝
            </button>
          )}
        </div>
      </div>
      {/* 使用 react-diff-viewer 或 Monaco Editor diff 模式 */}
      <MonacoDiffEditor
        original={oldContent}
        modified={newContent}
        language={language}
        options={{ readOnly: true, renderSideBySide: true }}
      />
    </div>
  );
};
```

### 3.6.7 Skills 生命周期与 Tool Result 回传

```typescript
// agui/skillLifecycle.ts

/**
 * Skill执行生命周期管理
 * 
 * 1. 收到 TOOL_CALL_START → 开始收集参数
 * 2. 收到 TOOL_CALL_ARGS → 累积参数JSON
 * 3. 收到 TOOL_CALL_END → 解析参数，执行Skill
 * 4. Skill执行完成 → 发送 ToolResult 回后端
 */
export class SkillLifecycleManager {
  private pendingCalls: Map<string, { toolName: string; argsBuffer: string }> = new Map();
  private executor: SkillExecutor;
  private aguiClient: AGUIClient;

  constructor(executor: SkillExecutor, client: AGUIClient) {
    this.executor = executor;
    this.aguiClient = client;
  }

  handleToolCallStart(toolCallId: string, toolName: string) {
    this.pendingCalls.set(toolCallId, { toolName, argsBuffer: '' });
  }

  handleToolCallArgs(toolCallId: string, content: string) {
    const pending = this.pendingCalls.get(toolCallId);
    if (pending) {
      pending.argsBuffer += content;
    }
  }

  async handleToolCallEnd(toolCallId: string) {
    const pending = this.pendingCalls.get(toolCallId);
    if (!pending) return;

    this.pendingCalls.delete(toolCallId);

    // 执行Skill
    const result = await this.executor.execute({
      toolCallId,
      toolName: pending.toolName,
      args: pending.argsBuffer,
    });

    // 将结果回传给后端（Orchestrator会转发给子Agent）
    await this.aguiClient.sendToolResult({
      toolCallId,
      toolName: pending.toolName,
      result: JSON.stringify(result),
      success: result.success,
    });
  }
}
```

### 3.6.8 子Agent 如何声明需要的前端组件

子Agent 通过 AgentCard 中的 `outputModes` 和 `skills` 字段声明它可能产出的产物类型，Orchestrator 据此判断需要哪些前端 Skills：

```yaml
# code-agent/config.yaml
name: "code-agent"
outputModes: ["text", "code", "diff", "terminal"]  # 声明可能的产出类型

# 这意味着code-agent可能触发以下前端Skills:
# - code_preview (对应 outputMode: code)
# - diff_preview (对应 outputMode: diff)  
# - terminal_output (对应 outputMode: terminal)

skills:
  - id: "code_generate"
    name: "代码生成"
    outputTypes: ["code", "terminal"]  # 该skill可能产出的类型
  - id: "code_review"
    name: "代码审查"
    outputTypes: ["diff", "text"]
```

```go
// Orchestrator 根据子Agent的outputModes过滤前端Skills
func (ua *UnifiedAgent) getRelevantSkills(agentCard AgentCard, allFrontendSkills []FrontendSkill) []FrontendSkill {
    // outputMode → skill name 映射
    modeToSkill := map[string]string{
        "code":     "code_preview",
        "webpage":  "web_preview",
        "diff":     "diff_preview",
        "file":     "file_download",
        "image":    "image_preview",
        "deploy":   "deploy_status",
        "document": "markdown_render",
        "terminal": "terminal_output",
        "chart":    "chart_render",
    }
    
    relevant := []FrontendSkill{}
    for _, mode := range agentCard.OutputModes {
        skillName, ok := modeToSkill[mode]
        if !ok {
            continue
        }
        for _, skill := range allFrontendSkills {
            if skill.Name == skillName {
                relevant = append(relevant, skill)
            }
        }
    }
    
    // 交互类Skills始终包含（任何Agent都可能需要）
    interactiveSkills := []string{"form_input", "confirm_action", "file_upload"}
    for _, name := range interactiveSkills {
        for _, skill := range allFrontendSkills {
            if skill.Name == name {
                relevant = append(relevant, skill)
            }
        }
    }
    
    return relevant
}
```

---

## 四、功能模块详细设计

### 4.1 模块优先级划分

| 优先级 | 模块 | 说明 |
|--------|------|------|
| **P0（必须）** | IM 聊天核心体验 | 对话列表、消息收发、AG-UI流式回复 |
| **P0（必须）** | Gateway 网关 | AG-UI Server + 路由 + 鉴权 |
| **P0（必须）** | 统一Agent + 意图编排 | Orchestrator + LLM意图分析 |
| **P0（必须）** | 子Agent (至少2个) | A2A协议 + AgentCard + ADK Runtime |
| **P1（重要）** | 群聊模式 | 多Agent协作、@指定 |
| **P1（重要）** | 产物预览 | 代码高亮、网页 iframe 预览 |
| **P1（重要）** | 自建 Agent | 用户自定义 Agent（A2A兼容） |
| **P2（加分）** | Diff 视图 + 版本历史 | 代码变更可视化 |
| **P2（加分）** | 部署发布 | 一键部署 |

### 4.2 模块一：前端 IM + AG-UI 集成（P0）

#### 4.2.1 AG-UI Client 集成

```typescript
// 前端 AG-UI 集成核心代码
import { RunAgentInput, Message, Event } from '@ag-ui/client';

// AG-UI 客户端配置
const agUIConfig = {
  baseUrl: '/api/agui',  // Gateway 的 AG-UI endpoint
  onEvent: handleAGUIEvent,
};

// 发送消息给统一Agent
async function sendMessage(conversationId: string, content: string, mentions?: string[]) {
  const input: RunAgentInput = {
    threadId: conversationId,
    runId: generateRunId(),
    messages: [
      {
        role: 'user',
        content: content,
      }
    ],
    // 声明前端 Skills
    tools: [
      {
        name: 'code_preview',
        description: '在前端展示代码预览',
        parameters: {
          type: 'object',
          properties: {
            code: { type: 'string' },
            language: { type: 'string' },
            filename: { type: 'string' },
          }
        }
      },
      {
        name: 'web_preview', 
        description: '在前端展示网页预览',
        parameters: {
          type: 'object',
          properties: {
            html: { type: 'string' },
            title: { type: 'string' },
          }
        }
      }
    ],
    // 附加上下文
    context: {
      mentions: mentions,  // @的Agent列表
      conversationType: mentions?.length > 1 ? 'group' : 'single',
    }
  };

  // 建立 SSE 连接，接收流式事件
  const eventSource = await agUIClient.run(input);
  return eventSource;
}

// 处理 AG-UI 事件
function handleAGUIEvent(event: Event) {
  switch (event.type) {
    case 'RUN_STARTED':
      // 显示Agent开始处理
      showAgentThinking();
      break;
      
    case 'TEXT_MESSAGE_START':
      // 创建新的消息气泡
      createMessageBubble(event.messageId);
      break;
      
    case 'TEXT_MESSAGE_CONTENT':
      // 流式追加文本
      appendToMessage(event.messageId, event.content);
      break;
      
    case 'TEXT_MESSAGE_END':
      // 消息完成，解析富媒体
      finalizeMessage(event.messageId);
      break;
      
    case 'TOOL_CALL_START':
      // Agent 调用前端 Skill
      handleToolCallStart(event.toolCallId, event.toolName);
      break;
      
    case 'TOOL_CALL_ARGS':
      // 接收工具参数
      appendToolArgs(event.toolCallId, event.content);
      break;
      
    case 'TOOL_CALL_END':
      // 执行前端 Skill
      executeSkill(event.toolCallId);
      break;
      
    case 'STATE_UPDATE':
      // 更新UI状态（如Agent切换、进度更新）
      updateUIState(event.state);
      break;
      
    case 'RUN_FINISHED':
      hideAgentThinking();
      break;
      
    case 'RUN_ERROR':
      showError(event.error);
      break;
  }
}
```

#### 4.2.2 前端 Skills 实现

```typescript
// 前端 Skills - Agent 可以调用前端能力
class FrontendSkillExecutor {
  
  // 代码预览 Skill
  async executeCodePreview(params: { code: string; language: string; filename: string }) {
    // 在侧边面板打开代码预览
    openSidePanel('code-preview', {
      code: params.code,
      language: params.language,
      filename: params.filename,
      actions: ['copy', 'apply', 'edit']
    });
    return { success: true, message: '代码已展示' };
  }
  
  // 网页预览 Skill
  async executeWebPreview(params: { html: string; title: string }) {
    openSidePanel('web-preview', {
      html: params.html,
      title: params.title,
      sandbox: true
    });
    return { success: true, message: '网页已渲染' };
  }
  
  // 文件上传 Skill（Agent请求用户上传文件）
  async executeFileUpload(params: { accept: string; maxSize: number }) {
    const file = await showFileUploadDialog(params);
    return { success: true, file: file };
  }
}
```

#### 4.2.3 对话列表 & 聊天窗口

数据模型与UI设计保持IM风格：

```typescript
// 会话模型
interface Conversation {
  id: string;
  title: string;
  type: 'single' | 'group';
  participants: Participant[];
  createdAt: Date;
  updatedAt: Date;
  isPinned: boolean;
  isArchived: boolean;
  lastMessage?: MessagePreview;
  unreadCount: number;
}

// 消息模型（兼容AG-UI事件）
interface ChatMessage {
  id: string;
  conversationId: string;
  senderId: string;
  senderType: 'user' | 'agent';
  senderAgent?: AgentCardSummary;  // Agent信息（来自AgentCard）
  content: MessageContent;
  status: 'sending' | 'streaming' | 'sent' | 'failed';
  agUIRunId?: string;              // 关联的AG-UI Run ID
  createdAt: Date;
}
```

### 4.3 模块二：Gateway 网关（P0）

#### 4.3.1 网关职责

```go
// Gateway 核心结构 (Go)
type Gateway struct {
    router      *gin.Engine
    agUIHandler *AGUIHandler       // AG-UI 协议处理
    authMiddleware *AuthMiddleware // 鉴权
    sessionMgr  *SessionManager   // 会话管理
    orchestrator *OrchestratorClient // 统一Agent客户端
    db          *gorm.DB
    redis       *redis.Client
}

// 网关路由注册
func (g *Gateway) SetupRoutes() {
    api := g.router.Group("/api")
    
    // AG-UI 端点（核心）
    api.POST("/agui/run", g.agUIHandler.HandleRun)          // AG-UI Run请求
    api.GET("/agui/run/:runId/events", g.agUIHandler.HandleSSE) // SSE事件流
    
    // 会话管理 REST API
    api.GET("/conversations", g.ListConversations)
    api.POST("/conversations", g.CreateConversation)
    api.GET("/conversations/:id", g.GetConversation)
    api.PATCH("/conversations/:id", g.UpdateConversation)
    api.DELETE("/conversations/:id", g.DeleteConversation)
    
    // 消息管理
    api.GET("/conversations/:id/messages", g.ListMessages)
    api.POST("/conversations/:id/messages/:msgId/regenerate", g.RegenerateMessage)
    
    // Agent 管理
    api.GET("/agents", g.ListAgents)                        // 获取所有Agent（从AgentCard）
    api.POST("/agents/custom", g.CreateCustomAgent)         // 创建自定义Agent
    api.GET("/agents/:id/card", g.GetAgentCard)            // 获取AgentCard
    
    // 产物管理
    api.GET("/artifacts/:id", g.GetArtifact)
    api.GET("/artifacts/:id/preview", g.PreviewArtifact)
}
```

#### 4.3.2 AG-UI Server 实现

```go
// AG-UI Handler - 处理前端AG-UI请求
type AGUIHandler struct {
    orchestrator *OrchestratorClient
    sessionMgr   *SessionManager
    eventBus     *EventBus
}

// 处理 AG-UI Run 请求
func (h *AGUIHandler) HandleRun(c *gin.Context) {
    var input AGUIRunInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 1. 保存消息到DB
    h.sessionMgr.SaveMessage(input.ThreadId, input.Messages)
    
    // 2. 获取会话历史
    history := h.sessionMgr.GetHistory(input.ThreadId)
    
    // 3. 创建SSE通道
    eventChan := make(chan AGUIEvent, 100)
    h.eventBus.Register(input.RunId, eventChan)
    
    // 4. 异步调用统一Agent
    go func() {
        err := h.orchestrator.Process(context.Background(), &OrchestratorRequest{
            RunID:    input.RunId,
            ThreadID: input.ThreadId,
            Messages: input.Messages,
            History:  history,
            Tools:    input.Tools,    // 前端Skills
            Context:  input.Context,
        }, eventChan)
        
        if err != nil {
            eventChan <- AGUIEvent{Type: "RUN_ERROR", Error: err.Error()}
        }
        close(eventChan)
    }()
    
    // 5. SSE 流式返回
    c.Stream(func(w io.Writer) bool {
        event, ok := <-eventChan
        if !ok {
            return false
        }
        c.SSEvent("message", event)
        return true
    })
}

// AG-UI 事件定义
type AGUIEvent struct {
    Type       string      `json:"type"`
    MessageID  string      `json:"messageId,omitempty"`
    Content    string      `json:"content,omitempty"`
    ToolCallID string      `json:"toolCallId,omitempty"`
    ToolName   string      `json:"toolName,omitempty"`
    State      interface{} `json:"state,omitempty"`
    RunID      string      `json:"runId,omitempty"`
    Error      string      `json:"error,omitempty"`
}
```

### 4.4 模块三：统一Agent / Orchestrator（P0）

#### 4.4.1 核心职责

统一Agent是整个系统的"大脑"，负责：
1. 接收 Gateway 转发的用户请求
2. 调用 LLM 进行意图理解和任务编排
3. 通过 A2A 协议调度子 Agent
4. 聚合子 Agent 结果，转换为 AG-UI 事件流返回

```go
// 统一Agent / Orchestrator
type UnifiedAgent struct {
    llm               LLMClient
    agentRegistry     *AgentRegistry       // 子Agent注册表
    a2aClient         *A2AClient           // A2A协议客户端
    intentEngine      *IntentEngine        // 意图编排引擎
    protocolConverter *ProtocolConverter   // A2A↔AG-UI协议转换器
}

// 处理请求（核心入口）
func (ua *UnifiedAgent) Process(ctx context.Context, req *OrchestratorRequest, events chan<- AGUIEvent) error {
    // 发送 RUN_STARTED
    events <- AGUIEvent{Type: "RUN_STARTED", RunID: req.RunID}
    
    // 1. 初始化协议转换器（注入前端Skills）
    ua.protocolConverter = &ProtocolConverter{
        frontendSkills: convertToolsToSkills(req.Tools),
    }
    
    // 2. 判断路由模式
    if hasDirectMention(req.Context) {
        // 用户明确@了某个Agent，直接路由
        return ua.directRoute(ctx, req, events)
    }
    
    // 3. 默认：意图编排
    return ua.intentOrchestrate(ctx, req, events)
}

// 意图编排核心流程
func (ua *UnifiedAgent) intentOrchestrate(ctx context.Context, req *OrchestratorRequest, events chan<- AGUIEvent) error {
    // 1. 获取所有子Agent的AgentCard
    cards := ua.agentRegistry.GetAllCards()
    
    // 2. LLM 生成执行计划
    plan, err := ua.intentEngine.GeneratePlan(ctx, req.Messages, req.History, cards)
    if err != nil {
        return err
    }
    
    // 3. 通知前端当前状态
    events <- AGUIEvent{
        Type:  "STATE_UPDATE",
        State: map[string]interface{}{"phase": "orchestrating", "plan": plan.Intent},
    }
    
    // 4. 执行计划
    switch plan.Strategy {
    case "single":
        return ua.executeSingleTask(ctx, plan.Tasks[0], events)
    case "parallel":
        return ua.executeParallelTasks(ctx, plan.Tasks, events)
    case "sequential":
        return ua.executeSequentialTasks(ctx, plan.Tasks, events)
    }
    
    // 5. 发送 RUN_FINISHED
    events <- AGUIEvent{Type: "RUN_FINISHED"}
    return nil
}

// 通过A2A调用子Agent
func (ua *UnifiedAgent) callSubAgent(ctx context.Context, agentName string, taskContent string, events chan<- AGUIEvent) (*A2ATaskResult, error) {
    // 1. 从注册表获取Agent地址
    agent := ua.agentRegistry.GetAgent(agentName)
    if agent == nil {
        return nil, fmt.Errorf("agent %s not found", agentName)
    }
    
    // 2. 构建A2A Task
    task := &A2ATask{
        ID: uuid.New().String(),
        Messages: []A2AMessage{
            {Role: "user", Content: taskContent},
        },
    }
    
    // 3. 发送A2A请求（流式）
    stream, err := ua.a2aClient.SendSubscribe(ctx, agent.URL, task)
    if err != nil {
        return nil, err
    }
    
    // 4. 转发流式结果为AG-UI事件
    events <- AGUIEvent{
        Type:      "TEXT_MESSAGE_START",
        MessageID: task.ID,
    }
    
    var result A2ATaskResult
    for event := range stream {
        switch event.Type {
        case "text":
            events <- AGUIEvent{
                Type:      "TEXT_MESSAGE_CONTENT",
                MessageID: task.ID,
                Content:   event.Content,
            }
        case "artifact":
            result.Artifacts = append(result.Artifacts, event.Artifact)
        }
    }
    
    events <- AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: task.ID}
    
    return &result, nil
}
```

### 4.5 模块四：子Agent（ADK Runtime + A2A）（P0）

#### 4.5.1 子Agent标准结构

每个子Agent都遵循统一的ADK Runtime规范：

```
sub-agent/
├── main.go              # 入口，ADK Runtime启动
├── agent.go             # Agent核心逻辑
├── tools/               # Agent工具集
│   ├── tool_read_file.go
│   ├── tool_write_file.go
│   └── tool_search.go
├── config.yaml          # Agent配置（生成AgentCard）
└── Dockerfile           # 容器化部署
```

#### 4.5.2 AgentCard 配置

```yaml
# config.yaml - 子Agent配置（自动生成AgentCard）
name: "code-agent"
description: "专注于代码生成、审查和重构的AI Agent"
version: "1.0.0"
url: "http://code-agent:8081"

capabilities:
  streaming: true
  pushNotifications: false
  stateTransitionHistory: true

skills:
  - id: "code_generate"
    name: "代码生成"
    description: "根据自然语言描述生成高质量代码"
    tags: ["code", "generate", "programming"]
    examples:
      - "写一个React登录组件"
      - "实现一个Go的HTTP服务器"
  
  - id: "code_review"
    name: "代码审查"
    description: "审查代码质量，发现潜在问题"
    tags: ["code", "review", "quality"]
    examples:
      - "审查这段代码的安全性"
      - "优化这个函数的性能"

  - id: "code_refactor"
    name: "代码重构"
    description: "重构代码结构，提升可维护性"
    tags: ["code", "refactor"]

inputModes: ["text", "file"]
outputModes: ["text", "code", "file"]

# LLM 配置
llm:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
  maxTokens: 8192
  systemPrompt: |
    你是一个专业的代码开发Agent。你擅长：
    1. 根据需求生成高质量代码
    2. 审查代码并给出改进建议
    3. 重构代码提升质量
    请始终输出可运行的完整代码。
```

#### 4.5.3 ADK Runtime 实现

```go
// ADK Runtime 核心框架
package adk

import (
    "context"
    "net/http"
    "encoding/json"
)

// Agent 接口
type Agent struct {
    config    AgentConfig
    handler   HandlerFunc
    tools     map[string]ToolFunc
    server    *http.Server
}

type HandlerFunc func(ctx *Context, task *Task) error
type ToolFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)

// 创建Agent
func NewAgent(config AgentConfig) *Agent {
    return &Agent{
        config: config,
        tools:  make(map[string]ToolFunc),
    }
}

// 注册工具
func (a *Agent) RegisterTool(name string, fn ToolFunc) {
    a.tools[name] = fn
}

// 设置处理器
func (a *Agent) SetHandler(handler HandlerFunc) {
    a.handler = handler
}

// 启动服务（自动暴露A2A端点）
func (a *Agent) Serve(addr string) error {
    mux := http.NewServeMux()
    
    // A2A 标准端点
    mux.HandleFunc("/.well-known/agent.json", a.handleAgentCard)
    mux.HandleFunc("/a2a/tasks/send", a.handleTaskSend)
    mux.HandleFunc("/a2a/tasks/sendSubscribe", a.handleTaskSendSubscribe)
    mux.HandleFunc("/a2a/tasks/", a.handleTaskQuery)  // GET /a2a/tasks/:id
    
    // 健康检查
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status":"healthy"}`))
    })
    
    a.server = &http.Server{Addr: addr, Handler: mux}
    return a.server.ListenAndServe()
}

// 处理 AgentCard 请求
func (a *Agent) handleAgentCard(w http.ResponseWriter, r *http.Request) {
    card := AgentCard{
        Name:         a.config.Name,
        Description:  a.config.Description,
        URL:          a.config.URL,
        Version:      a.config.Version,
        Capabilities: a.config.Capabilities,
        Skills:       a.config.Skills,
        InputModes:   a.config.InputModes,
        OutputModes:  a.config.OutputModes,
    }
    json.NewEncoder(w).Encode(card)
}

// 处理 A2A 流式任务
func (a *Agent) handleTaskSendSubscribe(w http.ResponseWriter, r *http.Request) {
    var taskReq A2ATaskRequest
    json.NewDecoder(r.Body).Decode(&taskReq)
    
    // 设置SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    flusher := w.(http.Flusher)
    
    // 创建上下文
    ctx := &Context{
        writer:  w,
        flusher: flusher,
        tools:   a.tools,
    }
    
    task := &Task{
        ID:       taskReq.ID,
        Messages: taskReq.Messages,
        Status:   TaskStatusWorking,
    }
    
    // 发送状态更新
    sendSSE(w, flusher, A2AEvent{Type: "status", Status: "working"})
    
    // 执行Handler
    err := a.handler(ctx, task)
    if err != nil {
        sendSSE(w, flusher, A2AEvent{Type: "status", Status: "failed", Error: err.Error()})
        return
    }
    
    // 发送完成状态
    sendSSE(w, flusher, A2AEvent{Type: "status", Status: "completed"})
}
```

#### 4.5.4 预置子Agent列表

| Agent名称 | 职责 | 核心Skills | LLM |
|-----------|------|-----------|-----|
| **code-agent** | 代码生成与审查 | code_generate, code_review, code_refactor | Claude Sonnet |
| **web-agent** | 网页/UI生成 | web_generate, component_design, style_optimize | Claude Sonnet |
| **doc-agent** | 文档编写 | doc_write, doc_format, summary | GPT-4 |
| **deploy-agent** | 部署发布 | deploy_static, deploy_container, generate_url | - (工具调用) |

### 4.6 模块五：Agent 注册与发现

```go
// Agent 注册表 - 管理所有子Agent
type AgentRegistry struct {
    agents map[string]*RegisteredAgent
    mu     sync.RWMutex
}

type RegisteredAgent struct {
    Card      AgentCard  `json:"card"`
    URL       string     `json:"url"`
    Status    string     `json:"status"`    // healthy, unhealthy, unknown
    LastCheck time.Time  `json:"lastCheck"`
}

// 注册Agent（启动时自动发现或手动注册）
func (r *AgentRegistry) Register(url string) error {
    // 1. 获取AgentCard
    resp, err := http.Get(url + "/.well-known/agent.json")
    if err != nil {
        return err
    }
    
    var card AgentCard
    json.NewDecoder(resp.Body).Decode(&card)
    
    // 2. 注册
    r.mu.Lock()
    r.agents[card.Name] = &RegisteredAgent{
        Card:   card,
        URL:    url,
        Status: "healthy",
    }
    r.mu.Unlock()
    
    return nil
}

// 健康检查（定期执行）
func (r *AgentRegistry) HealthCheck(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        r.mu.RLock()
        for name, agent := range r.agents {
            go func(name string, agent *RegisteredAgent) {
                resp, err := http.Get(agent.URL + "/health")
                r.mu.Lock()
                if err != nil || resp.StatusCode != 200 {
                    r.agents[name].Status = "unhealthy"
                } else {
                    r.agents[name].Status = "healthy"
                }
                r.agents[name].LastCheck = time.Now()
                r.mu.Unlock()
            }(name, agent)
        }
        r.mu.RUnlock()
    }
}

// 获取所有AgentCard（供Orchestrator编排使用）
func (r *AgentRegistry) GetAllCards() []AgentCard {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    cards := make([]AgentCard, 0)
    for _, agent := range r.agents {
        if agent.Status == "healthy" {
            cards = append(cards, agent.Card)
        }
    }
    return cards
}
```

### 4.7 模块六：产物预览与编辑（P1）

产物通过 AG-UI 的 Tool Call 机制，由 Agent 主动调用前端 Skills 来展示：

```typescript
// Agent 通过 AG-UI Tool Call 触发前端预览
// 流程：Agent产出代码 → 发送TOOL_CALL事件 → 前端执行code_preview Skill → 展示预览

// 预览卡片类型
interface PreviewCard {
  type: 'code' | 'webpage' | 'diff' | 'file' | 'deploy_status';
  data: CodePreviewData | WebPreviewData | DiffData | FileData | DeployData;
}

interface CodePreviewData {
  code: string;
  language: string;
  filename: string;
  actions: ('copy' | 'apply' | 'edit' | 'expand')[];
}

interface WebPreviewData {
  html: string;
  css?: string;
  js?: string;
  title: string;
  sandboxed: boolean;
}
```

### 4.8 模块七：自建Agent（P1）

用户自建Agent也需要遵循A2A协议，系统提供简化的创建流程：

```go
// 自建Agent - 基于模板快速创建
type CustomAgentTemplate struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    SystemPrompt string  `json:"systemPrompt"`
    LLMModel    string   `json:"llmModel"`     // 底层模型
    Skills      []Skill  `json:"skills"`
    Tools       []string `json:"tools"`         // 可用工具列表
}

// 自建Agent运行时（轻量版ADK）
// 系统内置一个通用Agent Runtime，根据配置动态加载
type CustomAgentRuntime struct {
    config CustomAgentTemplate
    llm    LLMClient
}

func (r *CustomAgentRuntime) HandleTask(ctx *adk.Context, task *adk.Task) error {
    // 使用用户配置的SystemPrompt + LLM处理
    messages := []LLMMessage{
        {Role: "system", Content: r.config.SystemPrompt},
    }
    for _, msg := range task.Messages {
        messages = append(messages, LLMMessage{Role: msg.Role, Content: msg.Content})
    }
    
    // 流式调用LLM
    stream, err := r.llm.ChatStream(ctx, messages)
    if err != nil {
        return err
    }
    
    for chunk := range stream {
        ctx.StreamText(chunk)
    }
    return nil
}
```

---

## 五、数据库设计

### 5.1 ER 图

```mermaid
erDiagram
    USER ||--o{ CONVERSATION : creates
    CONVERSATION ||--o{ MESSAGE : contains
    CONVERSATION ||--o{ CONVERSATION_PARTICIPANT : has
    AGENT ||--o{ CONVERSATION_PARTICIPANT : joins
    MESSAGE ||--o{ ARTIFACT : produces
    USER ||--o{ AGENT : creates_custom
    
    USER {
        uuid id PK
        string username
        string email
        string avatar
        timestamp created_at
    }
    
    CONVERSATION {
        uuid id PK
        uuid creator_id FK
        string title
        enum type "single|group"
        boolean is_pinned
        boolean is_archived
        timestamp created_at
        timestamp updated_at
    }
    
    CONVERSATION_PARTICIPANT {
        uuid id PK
        uuid conversation_id FK
        string participant_type "user|agent"
        uuid participant_id
        enum role "orchestrator|member"
    }
    
    MESSAGE {
        uuid id PK
        uuid conversation_id FK
        uuid sender_id
        enum sender_type "user|agent"
        jsonb content
        uuid reply_to
        text[] mentions
        enum status "sending|sent|failed|streaming"
        string agui_run_id
        string a2a_task_id
        timestamp created_at
    }
    
    AGENT {
        uuid id PK
        string name
        string url
        enum type "builtin|custom"
        string avatar
        string description
        jsonb agent_card
        text system_prompt
        jsonb config
        uuid creator_id FK "nullable"
        boolean is_active
        timestamp created_at
    }
    
    ARTIFACT {
        uuid id PK
        uuid message_id FK
        enum type "code|webpage|document|file"
        string title
        text content
        string file_url
        jsonb metadata
        integer version
        timestamp created_at
    }
    

```

### 5.2 Go 数据模型（GORM）

```go
// models/conversation.go
type Conversation struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    CreatorID   uuid.UUID `gorm:"type:uuid;not null"`
    Title       string    `gorm:"type:varchar(255)"`
    Type        string    `gorm:"type:varchar(20);default:'single'"` // single, group
    IsPinned    bool      `gorm:"default:false"`
    IsArchived  bool      `gorm:"default:false"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    
    Messages    []Message    `gorm:"foreignKey:ConversationID"`
    Participants []ConversationParticipant `gorm:"foreignKey:ConversationID"`
}

// models/message.go
type Message struct {
    ID             uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    ConversationID uuid.UUID       `gorm:"type:uuid;not null;index"`
    SenderID       uuid.UUID       `gorm:"type:uuid;not null"`
    SenderType     string          `gorm:"type:varchar(20)"` // user, agent
    Content        datatypes.JSON  `gorm:"type:jsonb"`
    ReplyTo        *uuid.UUID      `gorm:"type:uuid"`
    Mentions       pq.StringArray  `gorm:"type:text[]"`
    Status         string          `gorm:"type:varchar(20);default:'sent'"`
    AGUIRunID      string          `gorm:"type:varchar(100)"`
    A2ATaskID      string          `gorm:"type:varchar(100)"`
    CreatedAt      time.Time       `gorm:"index"`
}

// models/agent.go
type Agent struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name        string         `gorm:"type:varchar(100);uniqueIndex"`
    URL         string         `gorm:"type:varchar(500)"`
    Type        string         `gorm:"type:varchar(20)"` // builtin, custom
    Avatar      string         `gorm:"type:varchar(500)"`
    Description string         `gorm:"type:text"`
    AgentCard   datatypes.JSON `gorm:"type:jsonb"`       // 完整AgentCard JSON
    SystemPrompt string        `gorm:"type:text"`
    Config      datatypes.JSON `gorm:"type:jsonb"`
    CreatorID   *uuid.UUID     `gorm:"type:uuid"`
    IsActive    bool           `gorm:"default:true"`
    CreatedAt   time.Time
}
```

---

## 六、API 设计

### 6.1 AG-UI 端点（核心）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/agui/run` | 发起 AG-UI Run（SSE流式响应） |
| POST | `/api/agui/run/:runId/cancel` | 取消正在执行的 Run |
| POST | `/api/agui/run/:runId/tool-result` | 前端 Skill 执行结果回传 |

### 6.2 REST API

#### 会话管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/conversations` | 获取会话列表 |
| POST | `/api/conversations` | 创建新会话 |
| GET | `/api/conversations/:id` | 获取会话详情 |
| PATCH | `/api/conversations/:id` | 更新会话 |
| DELETE | `/api/conversations/:id` | 删除会话 |
| GET | `/api/conversations/:id/messages` | 获取消息列表（分页） |

#### Agent 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/agents` | 获取所有Agent列表（含AgentCard摘要） |
| GET | `/api/agents/:name/card` | 获取完整AgentCard |
| POST | `/api/agents/custom` | 创建自定义Agent |
| PATCH | `/api/agents/custom/:id` | 更新自定义Agent |
| DELETE | `/api/agents/custom/:id` | 删除自定义Agent |

#### A2A 内部端点（子Agent暴露）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/.well-known/agent.json` | AgentCard（自动生成） |
| POST | `/a2a/tasks/send` | 发送任务 |
| POST | `/a2a/tasks/sendSubscribe` | 发送任务+订阅流式结果 |
| GET | `/a2a/tasks/:id` | 查询任务状态 |
| DELETE | `/a2a/tasks/:id/cancel` | 取消任务 |

### 6.3 统一响应格式

```go
// 统一响应
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

// 分页响应
type PageResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    PageData    `json:"data"`
}

type PageData struct {
    List     interface{} `json:"list"`
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"pageSize"`
}
```

---

## 七、项目目录结构

```
agenthub/
├── frontend/                        # 前端项目 (React + AG-UI)
│   ├── src/
│   │   ├── components/
│   │   │   ├── Chat/                # 聊天组件
│   │   │   │   ├── MessageBubble.tsx
│   │   │   │   ├── MessageInput.tsx
│   │   │   │   ├── ConversationList.tsx
│   │   │   │   ├── StreamingText.tsx
│   │   │   │   └── AgentIndicator.tsx
│   │   │   ├── Preview/            # 预览组件
│   │   │   │   ├── CodePreview.tsx
│   │   │   │   ├── WebPreview.tsx
│   │   │   │   ├── DiffView.tsx
│   │   │   │   └── PreviewCard.tsx
│   │   │   ├── Agent/              # Agent相关
│   │   │   │   ├── AgentCard.tsx
│   │   │   │   ├── AgentMarket.tsx
│   │   │   │   └── AgentBuilder.tsx
│   │   │   └── Common/
│   │   ├── agui/                    # AG-UI 集成层
│   │   │   ├── client.ts           # AG-UI Client 封装
│   │   │   ├── events.ts           # 事件处理
│   │   │   ├── skills.ts           # 前端Skills定义
│   │   │   └── hooks.ts            # AG-UI React Hooks
│   │   ├── pages/
│   │   │   ├── ChatPage.tsx
│   │   │   ├── AgentPage.tsx
│   │   │   └── SettingsPage.tsx
│   │   ├── stores/                  # Zustand状态管理
│   │   │   ├── conversationStore.ts
│   │   │   ├── messageStore.ts
│   │   │   └── agentStore.ts
│   │   ├── services/
│   │   │   └── api.ts
│   │   ├── types/
│   │   └── utils/
│   ├── package.json
│   └── vite.config.ts
│
├── gateway/                         # Gateway 网关 (Go)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # 入口
│   ├── internal/
│   │   ├── handler/                 # HTTP处理器
│   │   │   ├── agui_handler.go     # AG-UI 端点
│   │   │   ├── conversation.go
│   │   │   ├── message.go
│   │   │   └── agent.go
│   │   ├── middleware/              # 中间件
│   │   │   ├── auth.go
│   │   │   ├── cors.go
│   │   │   └── logger.go
│   │   ├── service/                 # 业务逻辑
│   │   │   ├── session.go
│   │   │   └── artifact.go
│   │   ├── model/                   # 数据模型
│   │   │   ├── conversation.go
│   │   │   ├── message.go
│   │   │   └── agent.go
│   │   └── config/
│   │       └── config.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── orchestrator/                    # 统一Agent / 编排器 (Go)
│   ├── cmd/
│   │   └── orchestrator/
│   │       └── main.go
│   ├── internal/
│   │   ├── engine/
│   │   │   ├── intent_engine.go    # 意图编排引擎
│   │   │   ├── converter.go       # A2A↔AG-UI协议转换器
│   │   │   └── planner.go         # 执行计划生成
│   │   ├── a2a/
│   │   │   ├── client.go          # A2A客户端
│   │   │   └── types.go           # A2A类型定义
│   │   ├── registry/
│   │   │   └── agent_registry.go  # Agent注册表
│   │   ├── aggregator/
│   │   │   └── result_aggregator.go
│   │   └── llm/
│   │       ├── client.go          # LLM调用封装
│   │       └── prompts.go         # Prompt模板
│   ├── go.mod
│   └── Dockerfile
│
├── agents/                          # 子Agent集群
│   ├── adk/                         # ADK Runtime 公共库
│   │   ├── agent.go
│   │   ├── context.go
│   │   ├── task.go
│   │   ├── a2a_server.go
│   │   └── types.go
│   ├── code-agent/                  # 代码Agent
│   │   ├── main.go
│   │   ├── handler.go
│   │   ├── tools/
│   │   ├── config.yaml
│   │   └── Dockerfile
│   ├── web-agent/                   # 网页Agent
│   │   ├── main.go
│   │   ├── handler.go
│   │   ├── tools/
│   │   ├── config.yaml
│   │   └── Dockerfile
│   └── doc-agent/                   # 文档Agent
│       ├── main.go
│       ├── handler.go
│       ├── tools/
│       ├── config.yaml
│       └── Dockerfile
│
├── docker-compose.yml               # 一键启动所有服务
├── .env.example
└── Makefile                         # 构建脚本
```

---

## 八、开发计划与分工

### 8.1 团队分工（3人）

| 成员 | 角色 | 负责模块 |
|------|------|----------|
| **成员A** | 前端主力 | React IM UI、AG-UI Client集成、前端Skills、产物预览组件 |
| **成员B** | 后端主力（Gateway） | Go Gateway网关、AG-UI Server、会话/消息API、数据库、部署 |
| **成员C** | Agent主力 | 统一Agent/Orchestrator、ADK Runtime、子Agent开发、A2A协议实现 |

### 8.2 开发阶段

#### 第一阶段：基础框架搭建（第1-3天）

| 任务 | 负责人 | 产出 |
|------|--------|------|
| React项目初始化 + AG-UI SDK集成 + 基础布局 | 成员A | 前端骨架 + AG-UI连接 |
| Go Gateway项目初始化 + AG-UI Server端点 + DB | 成员B | 可运行的网关服务 |
| ADK Runtime框架 + A2A协议实现 + 第一个子Agent | 成员C | code-agent可运行 |
| 联调：前端→Gateway→Orchestrator→子Agent 全链路 | 全员 | 端到端消息流通 |

**里程碑：** 前端发消息 → AG-UI → Gateway → 子Agent → 流式回复，全链路跑通

#### 第二阶段：核心功能开发（第4-6天）

| 任务 | 负责人 | 产出 |
|------|--------|------|
| 对话列表UI + 消息流式渲染 + 代码高亮 | 成员A | 完整IM体验 |
| 前端Skills实现（代码预览、网页预览） | 成员A | Agent可调用前端能力 |
| 会话管理API + 消息持久化 + 分页 | 成员B | 完整后端数据层 |
| Agent注册表 + 健康检查 + AgentCard管理 | 成员B | Agent发现机制 |
| 意图编排引擎 + LLM Prompt工程 | 成员C | 智能任务分派 |
| 第二个子Agent（web-agent） | 成员C | 多Agent可用 |

**里程碑：** 单聊完整可用，意图编排基础可用

#### 第三阶段：功能完善（第7-9天）

| 任务 | 负责人 | 产出 |
|------|--------|------|
| 群聊UI（多Agent回复、@提及、Agent头像） | 成员A | 群聊体验 |
| 自建Agent前端（创建流程、配置表单） | 成员A | Agent创建页 |
| 自建Agent后端（动态注册、运行时加载） | 成员B | 自定义Agent运行 |
| 多Agent并行调度 + 结果聚合 + 降级 | 成员C | 稳定编排 |
| 第三个子Agent（doc-agent）+ 协议转换优化 | 成员C | 更多Agent + 转换层完善 |
| 产物全屏预览 + Monaco编辑器 | 成员A | 产物编辑 |

**里程碑：** 所有P0+P1功能完成，群聊多Agent协作可用

#### 第四阶段：打磨与交付（第10-12天）

| 任务 | 负责人 | 产出 |
|------|--------|------|
| UI打磨、动画、响应式 | 成员A | 精致体验 |
| 性能优化、错误处理、日志完善 | 成员B | 稳定后端 |
| Docker Compose编排 + 一键启动 | 成员B | 演示环境 |
| Agent能力优化 + Prompt调优 | 成员C | 更好的生成效果 |
| Demo视频录制 + 文档整理 | 全员 | 交付物 |

**里程碑：** 项目交付

### 8.3 甘特图

```mermaid
gantt
    title AgentHub 开发计划
    dateFormat  YYYY-MM-DD
    
    section 第一阶段-基础搭建
    React+AG-UI初始化       :a1, 2026-05-22, 2d
    Go Gateway初始化        :a2, 2026-05-22, 2d
    ADK Runtime+首个Agent   :a3, 2026-05-22, 3d
    全链路联调              :a4, 2026-05-24, 1d
    
    section 第二阶段-核心功能
    IM聊天UI完整实现        :b1, 2026-05-25, 3d
    前端Skills开发          :b2, 2026-05-26, 2d
    会话API+数据持久化      :b3, 2026-05-25, 2d
    Agent注册发现           :b4, 2026-05-27, 1d
    意图编排引擎            :b5, 2026-05-25, 3d
    第二子Agent             :b6, 2026-05-27, 1d
    
    section 第三阶段-功能完善
    群聊UI                  :c1, 2026-05-28, 2d
    自建Agent               :c2, 2026-05-29, 2d
    多Agent并行调度         :c3, 2026-05-28, 2d
    第三子Agent+协议转换    :c4, 2026-05-30, 2d
    产物编辑器              :c5, 2026-05-30, 1d
    
    section 第四阶段-打磨交付
    UI打磨                  :d1, 2026-06-01, 2d
    性能优化+Docker         :d2, 2026-06-01, 2d
    Agent调优               :d3, 2026-06-01, 1d
    Demo视频+文档           :d4, 2026-06-02, 1d
```

---

## 九、前端页面设计

### 9.1 页面结构

```
┌─────────────────────────────────────────────────────────────────────┐
│  Header: Logo + 搜索 + 用户头像                                       │
├────────────┬────────────────────────────────┬───────────────────────┤
│            │                                │                       │
│  对话列表   │        聊天主区域               │    侧边面板            │
│            │                                │  (产物预览/Agent信息)   │
│  - 搜索框   │  ┌─────────────────────────┐  │                       │
│  - 新建对话  │  │  Agent头像 + 名称        │  │  ┌─────────────────┐  │
│  - 会话列表  │  ├─────────────────────────┤  │  │  代码预览        │  │
│    (按时间)  │  │                         │  │  │  (AG-UI Skill)   │  │
│            │  │  消息流                   │  │  │  或              │  │
│            │  │  - 用户消息气泡           │  │  │  网页预览        │  │
│            │  │  - Agent回复(流式)        │  │  │  (iframe沙箱)    │  │
│            │  │  - 产物预览卡片           │  │  │  或              │  │
│            │  │  - 多Agent标识            │  │  │  Agent详情       │  │
│            │  │                         │  │  │  (AgentCard)     │  │
│            │  ├─────────────────────────┤  │  └─────────────────┘  │
│            │  │  输入框 + 工具栏          │  │                       │
│            │  │  [@Agent] [附件] [发送]   │  │                       │
│            │  └─────────────────────────┘  │                       │
├────────────┴────────────────────────────────┴───────────────────────┤
│  底部状态栏: Agent在线状态(AgentCard) + 连接状态(AG-UI)                   │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.2 核心页面

| 页面 | 路由 | 说明 |
|------|------|------|
| 聊天主页 | `/chat` | 对话列表 + 聊天窗口 |
| 聊天详情 | `/chat/:conversationId` | 特定会话 |
| Agent 市场 | `/agents` | 浏览所有Agent（展示AgentCard） |
| 创建 Agent | `/agents/create` | 对话式创建自定义Agent |
| Agent 详情 | `/agents/:name` | AgentCard详情 + 能力展示 |
| 设置 | `/settings` | API Key、偏好设置 |

---

## 十、非功能性需求

### 10.1 性能要求

| 指标 | 目标 |
|------|------|
| 首屏加载时间 | < 2s |
| AG-UI 首事件延迟 | < 500ms |
| 流式首字节延迟 | < 1s（含LLM推理） |
| A2A 任务分派延迟 | < 100ms |
| 并发会话数 | 支持 10+ 并行 |
| 子Agent健康检查 | 30s 间隔 |

### 10.2 可靠性

- AG-UI SSE 断线自动重连
- A2A 任务超时处理（30s）+ 自动重试
- Agent 不可用时自动降级
- 消息发送失败重试
- Gateway 无状态设计，支持水平扩展

### 10.3 安全性

- API Key 加密存储
- AG-UI 端点鉴权（JWT）
- A2A 内部通信网络隔离
- iframe 沙箱隔离（产物预览）
- 输入内容 XSS 防护

---

## 十一、风险与应对

| 风险 | 影响 | 应对策略 |
|------|------|----------|
| AG-UI 协议学习成本 | 前端开发延迟 | 提前阅读文档，先用最小集成验证 |
| A2A 协议实现复杂度 | Agent通信不稳定 | 先实现核心端点，逐步完善 |
| LLM 意图编排不准确 | 任务分派错误 | 支持用户@手动指定 + Prompt迭代优化 |
| 多Agent并行调度复杂 | 结果聚合困难 | 先实现串行，再优化为并行 |
| 子Agent开发工作量大 | 功能不完整 | ADK Runtime复用，快速创建Agent |
| 开发周期紧张 | 功能不完整 | 严格P0>P1>P2，确保核心链路 |

---

## 十二、考核对标

| 维度 | 权重 | 我们的策略 |
|------|------|------------|
| **AI协作能力** | 30% | 全程AI辅助开发，沉淀Spec/Rules/Skill，AG-UI+A2A协议本身就是AI协作最佳实践 |
| **功能完整度** | 25% | 确保AG-UI全链路流畅，A2A多Agent调度跑通，Demo完整演示 |
| **生成效果质量** | 20% | 精心打磨聊天UI，流式渲染体验，产物预览效果出色 |
| **代码理解度** | 15% | 清晰的四层架构（前端→网关→编排→子Agent），每人熟悉自己模块 |
| **创新与产品感** | 10% | AG-UI+A2A标准协议、意图编排、AgentCard能力发现、前端Skills机制 |

---

## 十三、交付物清单

| 交付物 | 说明 | 负责人 |
|--------|------|--------|
| 产品设计文档（本文档） | PDR + 架构设计 | 全员 |
| 技术设计文档 | 协议说明 + API文档 + DB设计 | 成员B |
| 可运行Demo | Docker Compose一键启动 | 成员B |
| AI协作开发记录 | Spec + Rules + 协作日志 | 全员 |
| 3分钟Demo视频 | 核心功能演示 | 全员 |
| 源代码 | GitHub仓库 | 全员 |

---

## 附录A：AI协作规范（Spec）

### 开发Spec

```markdown
# AgentHub 开发规范

## 代码风格
- 前端：TypeScript strict mode + ESLint + Prettier
- 后端：Go 标准规范 + golangci-lint
- 组件命名：PascalCase（前端）
- 包命名：小写（Go）
- 提交信息：feat/fix/docs/refactor + 简短描述

## 分支策略
- main: 稳定分支
- dev: 开发分支
- feature/*: 功能分支

## AI协作规则
- 每次AI生成代码需人工Review
- 复杂逻辑先写注释再让AI实现
- 前端代码必须通过TypeScript类型检查
- Go代码必须通过golangci-lint
- 保留AI协作的Prompt记录
```

### 接口规范

```markdown
# API 响应格式统一（Go）

## 成功响应
{
  "code": 0,
  "data": { ... },
  "message": "success"
}

## 错误响应
{
  "code": 错误码,
  "data": null,
  "message": "错误描述"
}

## 分页响应
{
  "code": 0,
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

---

## 附录B：环境变量配置

```env
# 数据库
DATABASE_URL=postgresql://user:password@localhost:5432/agenthub

# Redis
REDIS_URL=redis://localhost:6379

# LLM API Keys
ANTHROPIC_API_KEY=sk-ant-xxx
OPENAI_API_KEY=sk-xxx

# Gateway 配置
GATEWAY_PORT=8080
GATEWAY_JWT_SECRET=your-secret-key

# Orchestrator 配置
ORCHESTRATOR_PORT=8090
ORCHESTRATOR_LLM_MODEL=claude-sonnet-4-20250514

# 子Agent 配置
CODE_AGENT_PORT=8081
WEB_AGENT_PORT=8082
DOC_AGENT_PORT=8083

# Agent 注册地址
AGENT_REGISTRY_URLS=http://code-agent:8081,http://web-agent:8082,http://doc-agent:8083
```

---

## 附录C：Docker Compose 编排

```yaml
version: '3.8'

services:
  # 前端
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - gateway

  # Gateway 网关
  gateway:
    build: ./gateway
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgresql://user:password@postgres:5432/agenthub
      - REDIS_URL=redis://redis:6379
      - ORCHESTRATOR_URL=http://orchestrator:8090
    depends_on:
      - postgres
      - redis
      - orchestrator

  # 统一Agent / Orchestrator
  orchestrator:
    build: ./orchestrator
    ports:
      - "8090:8090"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - AGENT_REGISTRY_URLS=http://code-agent:8081,http://web-agent:8082,http://doc-agent:8083
    depends_on:
      - code-agent
      - web-agent
      - doc-agent

  # 子Agent - Code
  code-agent:
    build: ./agents/code-agent
    ports:
      - "8081:8081"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}

  # 子Agent - Web
  web-agent:
    build: ./agents/web-agent
    ports:
      - "8082:8082"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}

  # 子Agent - Doc
  doc-agent:
    build: ./agents/doc-agent
    ports:
      - "8083:8083"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}

  # 数据库
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: agenthub
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  # 缓存
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pgdata:
```

---

## 附录D：关键协议参考

| 协议 | 文档地址 | 用途 |
|------|----------|------|
| AG-UI | https://docs.ag-ui.com/introduction | 前端↔Agent通信 |
| A2A | https://google.github.io/A2A/ | Agent间通信 |
| ADK | https://google.github.io/adk-docs/ | Agent开发框架 |
