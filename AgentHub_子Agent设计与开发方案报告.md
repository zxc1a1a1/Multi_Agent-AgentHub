# AgentHub 子 Agent 设计与开发方案报告

> 面向：后端/Agent 工程师、前端集成工程师、产品负责人、答辩材料整理人员  
> 版本：v1.0  
> 主题：AgentHub 应该设计哪些子 Agent、每个子 Agent 怎么定义职责、输入输出、权限和开发优先级  
> 结论：v1.0 比赛阶段应优先做“产品可见的 Code Agent + Web Agent + Doc/Markdown 能力”和“内部可演进的 Planner / Context / Review / Test / Artifact 契约”，不要把 UI 功能拟人化成 Agent。

---

## 0. 报告摘要

AgentHub 是一个聊天式多 Agent 协作平台。子 Agent 的设计不能按 UI 模块拆，也不能把“会话列表”“复制按钮”“展开预览”这种功能拟人化为 Agent。真正的子 Agent 应该是被 Orchestrator 调度、拥有独立职责、独立上下文、独立工具权限、独立输出契约的任务执行单元。

本报告建议把 AgentHub 的 Agent 分成三层：

```text
第一层：编排控制层
- Orchestrator / 统一 Agent
- Planner / Intent Engine
- Registry / AgentCard / HealthCheck

第二层：产品可见子 Agent
- Code Agent
- Web Agent
- Document / Markdown Agent
- Custom Agent

第三层：内部工作流子 Agent / 能力 Agent
- Context Agent
- Codebase Explorer Agent
- Diff Agent
- Review Agent
- Test Agent
- Artifact Agent
- Security Agent
- Deploy Agent
- Version Agent
- Agent Builder Agent
```

比赛 v1.0 不建议一次性实现所有 Agent。推荐路线：

```text
v1.0 必做：
Code Agent + Web Agent + Orchestrator 意图编排 + AgentCard Registry + code_preview + web_preview + markdown_render

v1.0 可增强：
Document Agent / Review Agent / Test Agent / Artifact Agent 的轻量版本

后续扩展：
Diff Agent / Deploy Agent / Version Agent / Security Agent / Agent Builder Agent / PPT Agent
```

---

## 1. 输入资料与使用方式

### 1.1 项目内 5 个文档映射

| 文档 | 对本报告的作用 |
|---|---|
| `agenthub-skills-usage-report-v3.md` | 用于确定 Agent 开发必须遵守的 Skill / Contract 边界，尤其是 A2A、ADK Runtime、Artifact、安全、测试、Docker 交付。 |
| `SPRINT-v1.0-Plan.md` | 用于确定近期实际落地范围：code-agent、web-agent、LLM Planner、Registry、群聊、web_preview、markdown、fallback、Docker Demo。 |
| `UML-AgentHub系统图.md` | 用于确定子 Agent 与 Frontend、Gateway、Orchestrator、A2A、AG-UI、MySQL 的关系，以及单聊/群聊/协议转换时序。 |
| `AgentHub- 多Agent协作平台设计.pdf` | 用于对齐赛题要求：IM 聊天式交互、群聊协作、上下文连续、产物内联、多 Agent 接入、自建 Agent、Demo 和评分维度。 |
| `多Agent聊天式工作台_子Agent设计与开发建议报告.md` | 用于抽象“什么才算子 Agent”、MVP/P1/P2 子 Agent 分层、上下文隔离、权限控制、Diff/Review/Test 闭环。 |

### 1.2 外部资料映射

| 资料 | 对子 Agent 设计的启发 |
|---|---|
| OpenAI Agents SDK | Agent 应是能规划、调用工具、协作、保持足够状态完成多步工作的应用；多 Agent 可以通过 handoff 或 agents-as-tools 组织。 |
| LangChain / LangGraph | 子 Agent / supervisor 模式适合让主控 Agent 统一调度专门 Agent，强化上下文隔离。 |
| Anthropic Building Effective Agents | routing、orchestrator-workers、evaluator-optimizer 对应 AgentHub 的 Planner、子 Agent 调度和 Review/Test 闭环。 |
| A2A | 子 Agent 接入要有通用协议、AgentCard、任务、状态和流式输出。 |
| MCP | 文件、搜索、数据库、部署等外部能力应走标准工具网关，Agent 不应直接拥有所有权限。 |
| Microsoft Agent Framework / CrewAI | 多 Agent 系统需要状态化 workflow、guardrails、观测和人工审批。 |

---

## 2. 什么才算 AgentHub 的子 Agent

### 2.1 定义

在 AgentHub 中，子 Agent 是：

> 被 Orchestrator 调度、面向特定任务领域、有独立能力声明、独立上下文输入、独立工具权限、结构化输出、可观测运行记录的执行型智能体。

它至少要满足：

| 条件 | 说明 |
|---|---|
| 独立职责 | 例如代码生成、网页生成、文档生成、测试、审查、部署 |
| 独立 AgentCard | 对外声明 name、description、skills、inputModes、outputModes |
| 独立 System Prompt | 定义行为边界和输出格式 |
| 独立上下文 | 不默认读取所有聊天历史，而是读取 context bundle |
| 独立工具权限 | 例如 Explorer 只读，Code 只生成 diff，Deploy 才能部署 |
| 结构化输出 | 输出 text、artifact、diff、test_result 等可解析结果 |
| 可被调度 | 能被 Orchestrator 通过 A2A 调用 |
| 可观测 | 有 runId、taskId、traceId、status、duration、errorCode |

### 2.2 不应该设计成子 Agent 的东西

以下不是子 Agent：

| 功能 | 正确归类 |
|---|---|
| 会话列表 | Frontend UI + Conversation API |
| 新建会话 | Conversation CRUD |
| 置顶 / 归档 | Conversation metadata |
| 搜索会话 | Search API / DB query |
| 复制代码按钮 | UI action |
| 展开预览 | Artifact renderer |
| 消息气泡 | UI component |
| 头像展示 | Agent profile UI |
| 一键应用 Diff 按钮 | UI action + Patch service |
| Docker healthcheck | Delivery / Ops capability |
| Token 鉴权 | Gateway middleware |

这些模块可以被 Agent 触发或使用，但它们本身不是 Agent。

### 2.3 Orchestrator 与子 Agent 的区别

| 项目 | Orchestrator | 子 Agent |
|---|---|---|
| 角色 | 控制面 / 编排器 | 执行面 / 专业任务处理器 |
| 是否直接面对用户 | 通过 Gateway 间接面对 | 通过 Orchestrator 间接面对 |
| 是否生成计划 | 是 | 通常否 |
| 是否调度其他 Agent | 是 | 默认否 |
| 是否执行具体任务 | 少量，如聚合总结 | 是 |
| 是否拥有全局上下文 | 有摘要和控制视角 | 只拿 context bundle |
| 是否应写业务代码 | 否 | Code Agent 可以 |
| 是否暴露给前端 | 否 | 否，前端只看到消息和 agentName |

---

## 3. 子 Agent 总体分层

### 3.1 比赛 v1.0 推荐最小组合

为了满足赛题“至少 2 个 Agent、多 Agent 调度、IM 核心体验、产物预览、3 分钟 Demo”，v1.0 最小组合建议：

| Agent | 是否独立服务 | 作用 |
|---|---:|---|
| `code-agent` | 是 | 代码生成、代码解释、代码预览 artifact |
| `web-agent` | 是 | 网页/UI 生成、HTML/CSS/JS、web_preview artifact |
| `doc-agent` 或 markdown 能力 | 可先轻量实现 | README、报告、Markdown 文档生成 |
| `orchestrator` | 是，或 v1.0 初期嵌在 Gateway 后拆出 | 意图编排、任务拆解、调度、聚合 |
| `registry` | 可作为 Orchestrator/Gateway 模块 | AgentCard 发现、健康检查、能力索引 |

这套组合能支撑 Demo：

```text
1. 单聊 code-agent：生成 Go 代码 → code_preview
2. 单聊 web-agent：生成登录页 → web_preview
3. 群聊自动编排：用户说“前端页面 + 后端 API” → web-agent + code-agent
4. markdown_render：长文本/报告/README 渲染
```

### 3.2 中期推荐组合

中期加入质量闭环：

| Agent | 作用 |
|---|---|
| `context-agent` | 上下文压缩、pin 消息、任务上下文包 |
| `explorer-agent` | 只读分析代码库、定位文件 |
| `review-agent` | 审查代码和产物是否满足需求 |
| `test-agent` | 运行测试、类型检查、build、解释失败 |
| `artifact-agent` | 生成 artifact manifest，统一产物元数据 |
| `diff-agent` | 生成/校验/解释/合并 diff |

### 3.3 后期扩展组合

| Agent | 作用 |
|---|---|
| `security-agent` | 高危工具、密钥、权限、部署前检查 |
| `deploy-agent` | 构建、部署、预览 URL、部署状态卡片 |
| `version-agent` | 版本快照、回滚、artifact 版本 |
| `agent-builder-agent` | 对话式创建用户自定义 Agent |
| `file-agent` | 上传文件/PDF/ZIP/图片处理 |
| `web-research-agent` | URL 读取、网络检索、引用来源 |
| `ppt-agent` | PPT 结构生成和页面修改 |
| `release-agent` | changelog、打包、发布说明 |

---

## 4. AgentCard 统一规范

每个子 Agent 必须暴露 AgentCard。它既是 Orchestrator 选择 Agent 的依据，也是答辩中解释“为什么能动态接入 Agent”的关键。

### 4.1 AgentCard 基础字段

```json
{
  "name": "code-agent",
  "displayName": "Code Agent",
  "description": "生成、解释和审查代码，输出代码 artifact。",
  "version": "1.0.0",
  "url": "http://code-agent:8081",
  "skills": [
    {
      "id": "code_generation",
      "name": "代码生成",
      "description": "根据自然语言需求生成代码"
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"],
  "capabilities": {
    "streaming": true,
    "artifactTypes": ["code"],
    "supportsContext": true,
    "supportsTools": false
  }
}
```

### 4.2 AgentCard 不允许包含

```text
真实 API Key
真实 Token
数据库密码
DATABASE_URL
私钥
system prompt 全文
内部服务 token
不应暴露给前端的内网地址
```

### 4.3 AgentCard 与编排的关系

Orchestrator 不应写死：

```text
“网页任务就找 web-agent”
```

更好的方式：

```text
用户意图 → capabilityId=web_generation → Registry 找到支持该 capability 的健康 Agent → 调用
```

这能支持后续：

```text
web-agent-v2
figma-agent
custom-ui-agent
external-codex-agent
```

---

## 5. 核心子 Agent 详细设计

---

## 5.1 Code Agent

### 定位

Code Agent 是 AgentHub 的基础核心 Agent，负责代码生成、代码解释、代码重构建议、代码片段输出和代码 artifact 生成。

### 适合处理

- “用 Go 写一个 HTTP server”
- “帮我改这个接口”
- “给这个函数加测试”
- “解释这段 TypeScript”
- “生成一个 React 组件”
- “把代码整理成 main.go 和 handler.go”

### 不负责

- 不直接部署生产环境。
- 不自行修改 `.env`。
- 不直接删除文件。
- 不绕过 Review/Test。
- 不决定是否调用其他 Agent。
- 不把所有代码直接塞进聊天文本，应该输出 artifact。

### AgentCard

```json
{
  "name": "code-agent",
  "displayName": "Code Agent",
  "description": "负责代码生成、解释、重构建议和代码产物输出。",
  "version": "1.0.0",
  "skills": [
    {
      "id": "code_generation",
      "name": "代码生成",
      "description": "根据需求生成代码文件或代码片段"
    },
    {
      "id": "code_explanation",
      "name": "代码解释",
      "description": "解释现有代码逻辑"
    },
    {
      "id": "code_refactor",
      "name": "代码重构建议",
      "description": "给出小范围重构方案"
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"],
  "capabilities": {
    "streaming": true,
    "artifactTypes": ["code"]
  }
}
```

### 输入契约

```json
{
  "taskId": "task_code_001",
  "taskContent": "用 Go 写一个带日志中间件的 HTTP server",
  "contextBundle": {
    "requirements": ["使用 net/http", "包含日志中间件"],
    "constraints": ["不要读取 .env", "不要写真实 token"]
  },
  "expectedArtifacts": ["code"]
}
```

### 输出契约

```json
{
  "status": "success",
  "summary": "已生成 Go HTTP server 示例。",
  "artifacts": [
    {
      "type": "code",
      "title": "main.go",
      "language": "go",
      "content": "package main\n..."
    }
  ],
  "notes": [
    "代码为示例，可直接复制到本地运行"
  ]
}
```

### 工具权限

| 工具 | 权限 |
|---|---:|
| LLM stream | 允许 |
| read_file | 后续可允许，需 context 限制 |
| write_file | 默认禁止 |
| apply_patch | 默认禁止，后续通过 Diff Agent |
| run_command | 默认禁止 |
| network | 禁止 |
| deploy | 禁止 |

### 测试要点

- 能生成标准代码块。
- 能生成多个 code artifact。
- `parseCodeBlocks` 不误截断普通反引号。
- 无代码块时不生成 artifact。
- 不泄露 prompt / token。
- LLM 超时能返回 safe error。

---

## 5.2 Web Agent

### 定位

Web Agent 负责网页、UI 原型、HTML/CSS/JS、响应式页面和可预览 web artifact。

### 适合处理

- “写一个好看的登录页面”
- “做一个待办清单网页”
- “生成一个计数器应用前端”
- “把页面改成深色主题”
- “生成带动画的 Landing Page”

### 不负责

- 不写后端 API。
- 不部署网页。
- 不访问用户真实文件。
- 不执行任意 JS。
- 不越权生成危险脚本。
- 不直接操作生产站点。

### AgentCard

```json
{
  "name": "web-agent",
  "displayName": "Web Agent",
  "description": "负责生成网页、UI 原型和可预览 HTML/CSS/JS 产物。",
  "version": "1.0.0",
  "skills": [
    {
      "id": "web_generation",
      "name": "网页生成",
      "description": "生成完整 HTML/CSS/JS 页面"
    },
    {
      "id": "ui_design",
      "name": "UI 设计",
      "description": "生成现代化响应式界面"
    },
    {
      "id": "responsive_layout",
      "name": "响应式布局",
      "description": "适配桌面和移动端"
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "webpage", "code"],
  "capabilities": {
    "streaming": true,
    "artifactTypes": ["webpage", "code"]
  }
}
```

### 输入契约

```json
{
  "taskId": "task_web_001",
  "taskContent": "生成一个带动画效果的登录页面",
  "expectedArtifacts": ["webpage"],
  "constraints": [
    "输出自包含 HTML",
    "CSS 和 JS 可内联",
    "不要引用外部不可控脚本"
  ]
}
```

### 输出契约

```json
{
  "status": "success",
  "summary": "已生成登录页。",
  "artifacts": [
    {
      "type": "webpage",
      "title": "index.html",
      "content": "<!DOCTYPE html>...",
      "metadata": {
        "language": "html"
      }
    }
  ]
}
```

### 安全要求

前端渲染 web artifact 时：

- 使用 iframe sandbox。
- 默认不允许同源敏感权限。
- 不把真实 token 注入 srcdoc。
- 大 HTML 后续走 contentRef。
- 如果包含外链脚本，需标记风险或阻止。

### 测试要点

- 生成 HTML artifact。
- Artifact 能转为 `web_preview`。
- iframe 正常渲染。
- 没有 web_preview skill 时降级。
- 不产生真实网络凭据。
- HTML 中危险脚本有安全限制。

---

## 5.3 Document / Markdown Agent

### 定位

Document Agent 负责 Markdown 报告、README、技术方案、PRD、交接报告、API 文档等文本产物。

在 v1.0 中可以先不独立起一个 doc-agent 服务，而是先支持 `markdown_render` 前端 Skill；但从架构上建议预留 Document Agent。

### 适合处理

- “把这次修改整理成报告”
- “生成 README”
- “写一个技术方案”
- “把聊天内容总结成 Markdown”
- “整理接口文档”

### 不负责

- 不生成 PPTX 二进制。
- 不执行代码。
- 不修改源码。
- 不读取未经授权文件。
- 不虚构外部引用。

### AgentCard

```json
{
  "name": "doc-agent",
  "displayName": "Document Agent",
  "description": "负责 Markdown 文档、报告、README、PRD 和技术方案生成。",
  "version": "1.0.0",
  "skills": [
    {
      "id": "markdown_documentation",
      "name": "Markdown 文档生成",
      "description": "生成结构化 Markdown 文档"
    },
    {
      "id": "technical_writing",
      "name": "技术写作",
      "description": "整理架构说明、接口说明和开发报告"
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "document"],
  "capabilities": {
    "streaming": true,
    "artifactTypes": ["document"]
  }
}
```

### 输出契约

```json
{
  "status": "success",
  "summary": "已生成 Markdown 报告。",
  "artifacts": [
    {
      "type": "document",
      "title": "review2-fix-report.md",
      "format": "markdown",
      "content": "# 标题\n..."
    }
  ]
}
```

### 测试要点

- Markdown 能被前端渲染。
- 表格、代码块、列表正常。
- 不输出真实密钥。
- 引用外部资料时给出来源。
- 大文档后续支持 contentRef。

---

## 5.4 Context Agent

### 定位

Context Agent 是上下文打包者，负责把聊天历史、pin 消息、代码库摘要、artifact、文件摘要整理成不同 Agent 需要的 context bundle。

### 为什么重要

如果所有子 Agent 都拿完整聊天历史：

- token 成本高。
- 子 Agent 容易被无关历史干扰。
- 旧需求可能覆盖最新需求。
- 不同 Agent 对任务理解不一致。
- 安全风险更大。

### 适合处理

- 压缩长对话。
- 提取最新需求。
- 识别约束。
- 为 Code Agent 打包代码上下文。
- 为 Review Agent 打包需求 + diff + test result。
- 为 Deploy Agent 打包 artifact + 构建命令。

### AgentCard

```json
{
  "name": "context-agent",
  "displayName": "Context Agent",
  "description": "为不同子 Agent 构造任务相关上下文包，避免上下文污染。",
  "skills": [
    {
      "id": "context_packaging",
      "name": "上下文打包",
      "description": "生成面向目标 Agent 的 context bundle"
    }
  ],
  "inputModes": ["text", "message_history", "artifact"],
  "outputModes": ["context_bundle"],
  "capabilities": {
    "streaming": false,
    "artifactTypes": []
  }
}
```

### 输出契约

```json
{
  "contextBundleId": "ctx_001",
  "targetAgent": "code-agent",
  "summary": "用户正在开发一个多 Agent 聊天平台。",
  "requirements": [
    "支持群聊",
    "支持 web_preview",
    "保持 Gateway/Orchestrator 边界"
  ],
  "constraints": [
    "不修改 .env",
    "不提交真实 token"
  ],
  "relevantMessages": [],
  "relevantArtifacts": [],
  "excludedContextNotes": [
    "无关聊天未注入"
  ]
}
```

### 权限

| 工具 | 权限 |
|---|---:|
| read conversation history | 允许 |
| read pinned messages | 允许 |
| read artifact metadata | 允许 |
| read full file | 按需 |
| write code | 禁止 |
| run command | 禁止 |
| network | 禁止 |

---

## 5.5 Codebase Explorer Agent

### 定位

Explorer Agent 是只读代码库分析 Agent。它负责找文件、理解结构、定位改动范围，不负责写代码。

### 适合处理

- “这个功能应该改哪些文件？”
- “找一下 AG-UI 事件在哪里处理”
- “定位 messageStore 的 streaming 状态”
- “分析现有项目结构”

### 不负责

- 不改文件。
- 不生成最终业务代码。
- 不执行破坏性命令。
- 不做部署。

### 输出契约

```json
{
  "status": "success",
  "targetFiles": [
    {
      "path": "frontend/src/stores/messageStore.ts",
      "reason": "流式消息状态在这里维护"
    },
    {
      "path": "server/internal/orchestrator/converter.go",
      "reason": "A2A artifact 转 AG-UI tool call 在这里处理"
    }
  ],
  "architectureNotes": [
    "前端通过 AG-UI SSE 接收事件",
    "Gateway 不应直接调用子 Agent"
  ],
  "risks": [
    "修改 messageStore 会影响多个组件"
  ],
  "recommendedNextAgent": "code-agent"
}
```

### 权限

| 工具 | 权限 |
|---|---:|
| list_directory | 允许 |
| read_file | 允许 |
| grep/search | 允许 |
| git diff | 允许 |
| write_file | 禁止 |
| apply_patch | 禁止 |
| destructive shell | 禁止 |

---

## 5.6 Diff Agent

### 定位

Diff Agent 负责生成、校验、解释和合并 diff，是代码修改闭环的关键。

### v1.0 是否必须

比赛 v1.0 可先不完整实现 Diff Agent 服务，但应在设计中预留 Diff artifact。因为赛题提到 Diff 视图卡片、代码二次编辑、局部修改等扩展能力。

### 适合处理

- 生成 unified diff。
- 校验 patch 是否可应用。
- 合并多个 patch。
- 解释改动。
- 生成回滚 patch。
- 处理多 Agent 改同一文件冲突。

### 输出契约

```json
{
  "patchId": "patch_001",
  "applyStatus": "applicable",
  "riskLevel": "medium",
  "changedFiles": [
    "frontend/src/components/MessageBubble.tsx"
  ],
  "humanSummary": [
    "新增 web_preview 渲染分支"
  ],
  "diff": "diff --git a/...",
  "conflicts": []
}
```

### 权限

| 工具 | 权限 |
|---|---:|
| read file | 允许 |
| git diff | 允许 |
| patch dry-run | 允许 |
| apply patch | 需要确认 |
| write file | 需要确认 |
| deploy | 禁止 |

---

## 5.7 Review Agent

### 定位

Review Agent 负责代码和产物质量审查，不直接改代码。

### 适合处理

- 检查需求是否覆盖。
- 检查代码质量。
- 检查安全风险。
- 检查测试是否补齐。
- 检查是否破坏架构边界。
- 检查是否有真实密钥。

### 输出契约

```json
{
  "status": "changes_requested",
  "findings": [
    {
      "severity": "high",
      "category": "security",
      "file": "docker-compose.yml",
      "issue": "检测到可能写入真实数据库密码",
      "suggestion": "改为环境变量占位"
    }
  ],
  "passedChecks": [
    "Gateway 没有直接调用 LLM Provider",
    "AgentCard 未包含 secret"
  ],
  "recommendedNextAgent": "code-agent"
}
```

### 权限

| 工具 | 权限 |
|---|---:|
| read diff | 允许 |
| read files | 允许 |
| run static checks | 可允许 |
| write code | 禁止 |
| apply patch | 禁止 |
| deploy | 禁止 |

---

## 5.8 Test Agent

### 定位

Test Agent 负责执行测试、构建、类型检查、smoke-test，并解释失败日志。

### 适合处理

- `go test ./...`
- `go build`
- `npm test -- --run`
- `npm run build`
- `docker compose config`
- `bash -n smoke-test.sh`
- smoke-test 结果解释

### 输出契约

```json
{
  "status": "failed",
  "commands": [
    {
      "command": "npm run build",
      "exitCode": 1,
      "summary": "TypeScript 类型错误"
    }
  ],
  "failureAnalysis": [
    "Message 类型缺少 webPreviews 字段"
  ],
  "recommendedFix": "在 frontend/src/types/index.ts 中补充 WebPreviewData 类型"
}
```

### 权限

| 工具 | 权限 |
|---|---:|
| test/build/lint | 允许 |
| docker compose config | 允许 |
| bash -n | 允许 |
| destructive command | 禁止 |
| deploy | 禁止 |
| write file | 禁止 |

---

## 5.9 Artifact Agent

### 定位

Artifact Agent 负责把子 Agent 的输出变成统一产物描述。它不负责 UI 渲染，但负责产物语义和元数据。

### 适合处理

- code artifact。
- webpage artifact。
- markdown document artifact。
- diff artifact。
- deployment artifact。
- file artifact。

### 输出契约

```json
{
  "artifactId": "art_001",
  "type": "webpage",
  "title": "login-page.html",
  "contentRef": null,
  "content": "<!DOCTYPE html>...",
  "source": {
    "agentName": "web-agent",
    "taskId": "task_web"
  },
  "previewType": "web_preview",
  "metadata": {
    "language": "html"
  }
}
```

### 前端映射

| Artifact type | Frontend Skill |
|---|---|
| code | code_preview |
| webpage | web_preview |
| document | markdown_render |
| diff | diff_view |
| deployment | deployment_status |
| file | file_download |

---

## 5.10 Security Agent

### 定位

Security Agent 负责敏感操作、密钥、工具权限和高危命令审查。它不能只靠 LLM，需要结合规则引擎。

### 触发场景

- 修改 `.env`
- 读取私钥
- 写真实 token
- 删除文件
- 执行 shell 高危命令
- 安装未知依赖
- 数据库迁移
- 生产部署
- 外发文件到网络

### 输出契约

```json
{
  "riskLevel": "high",
  "blocked": true,
  "reasons": [
    "操作目标包含 .env",
    "命令可能删除文件"
  ],
  "safeAlternatives": [
    "只修改 .env.example",
    "先生成 diff，由用户确认"
  ]
}
```

### 权限

Security Agent 本身不执行危险操作，只做判断和拦截建议。

---

## 5.11 Deploy Agent

### 定位

Deploy Agent 负责构建、部署、返回预览 URL 和部署状态卡片。

### v1.0 建议

比赛阶段可先实现“本地 Docker Demo + smoke-test + 生成部署状态卡片模拟”，不急于接真实云部署。

### 适合处理

- docker build。
- docker compose up。
- 静态站点预览。
- 生成预览 URL。
- 部署日志摘要。
- 回滚建议。

### 输出契约

```json
{
  "deploymentId": "dep_001",
  "target": "preview",
  "status": "success",
  "previewUrl": "http://localhost:3000",
  "logsSummary": "构建完成，服务已启动。",
  "actions": [
    "open_url",
    "view_logs",
    "rollback"
  ]
}
```

### 安全要求

- 生产部署必须人工确认。
- 不保存真实云凭据。
- 不把 token 写进日志。
- 部署失败要脱敏。

---

## 5.12 Agent Builder Agent

### 定位

Agent Builder Agent 负责帮助用户通过对话创建自定义 Agent。

### 适合处理

- 生成 Agent 名称。
- 生成能力标签。
- 生成 System Prompt。
- 配置工具权限。
- 生成 AgentCard。
- 生成默认输出格式。

### 输出契约

```json
{
  "agentProfile": {
    "name": "prd-agent",
    "displayName": "PRD Agent",
    "description": "帮助整理产品需求文档",
    "skills": [
      {
        "id": "prd_writing",
        "name": "PRD 编写"
      }
    ],
    "toolPermissions": [
      "read_context",
      "write_artifact"
    ],
    "riskLevel": "low"
  }
}
```

### 安全要求

- 默认不给写代码权限。
- 默认不给 shell 权限。
- 开启工具必须说明风险。
- 自定义 prompt 自动追加安全边界。

---

## 6. 产品可见 Agent 与内部工作流 Agent 的区别

### 6.1 产品可见 Agent

这些 Agent 会显示在聊天列表或 Agent 列表中：

```text
Code Agent
Web Agent
Document Agent
Custom Agent
```

用户可以选择：

```text
新建对话 → 选择 Web Agent
```

或者在群聊中：

```text
@code-agent 帮我生成 Go API
```

### 6.2 内部工作流 Agent

这些 Agent 不一定作为“联系人”展示：

```text
Context Agent
Explorer Agent
Diff Agent
Review Agent
Test Agent
Artifact Agent
Security Agent
```

它们可以在执行详情里展示，但不一定直接进入用户对话列表。

例如用户看到：

```text
Code Agent：已生成代码。
Review Agent：检查通过。
Test Agent：构建通过。
```

也可以在默认视图中压缩为：

```text
Orchestrator：代码已生成并通过检查。
```

### 6.3 为什么要区分

这样可以避免聊天列表过于复杂，同时保留多 Agent 的可观测性。

推荐 UI：

```text
默认聊天流：
- 用户
- Code Agent
- Web Agent
- Orchestrator 总结

展开执行详情：
- Context Agent
- Explorer Agent
- Review Agent
- Test Agent
- Artifact Agent
```

---

## 7. 子 Agent 接入方式

### 7.1 A2A 服务接口

每个独立子 Agent 至少支持：

```text
GET  /health
GET  /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
POST /a2a/tasks/send
GET  /a2a/tasks/:id
POST /a2a/tasks/:id/cancel
```

v1.0 可以先只做：

```text
GET /health
GET /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
```

### 7.2 目录结构建议

```text
agents/
  adk/
    config.go
    server.go
    context.go
    llm.go
    artifact.go
  code-agent/
    main.go
    handler.go
    handler_test.go
    config.yaml
    Dockerfile
  web-agent/
    main.go
    handler.go
    handler_test.go
    config.yaml
    Dockerfile
  doc-agent/
    main.go
    handler.go
    config.yaml
    Dockerfile
```

### 7.3 config.yaml 建议

```yaml
name: web-agent
displayName: Web Agent
description: Generates responsive web pages and UI prototypes.
version: "1.0.0"
url: "http://web-agent:8082"
skills:
  - id: web_generation
    name: Web generation
    description: Generate HTML/CSS/JS pages.
inputModes:
  - text
outputModes:
  - text
  - webpage
  - code
artifactTypes:
  - webpage
  - code
streaming: true
timeoutSeconds: 120
```

### 7.4 Handler 设计原则

每个 handler 应遵守：

```text
1. 输入转换：A2A messages → LLM messages
2. 上下文使用：只使用 Orchestrator 提供的 context bundle
3. LLM 调用：有 timeout，有 safe error
4. 流式输出：ctx.StreamText(chunk)
5. 产物输出：ctx.AddArtifact(...)
6. 错误脱敏：不返回内部栈、不返回 key、不返回完整请求头
7. 测试覆盖：标准输入、空输入、LLM error、artifact parse
```

---

## 8. 数据模型建议

### 8.1 AgentProfile

```ts
type AgentProfile = {
  id: string;
  name: string;
  displayName: string;
  avatarUrl?: string;
  description: string;
  tags: string[];

  role:
    | "coder"
    | "web"
    | "document"
    | "reviewer"
    | "tester"
    | "artifact"
    | "security"
    | "deployer"
    | "custom";

  systemPromptRef: string;
  modelPreference: {
    provider: "anthropic" | "openai" | "local" | "custom";
    model: string;
    fallbackModels?: string[];
  };

  toolPermissions: string[];
  inputSchema?: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;

  routing: {
    capabilityIds: string[];
    triggerDescription: string;
    explicitMentions: string[];
    priority: number;
  };

  runtime: {
    maxTurns: number;
    timeoutMs: number;
    allowParallel: boolean;
    requiresApprovalForTools: string[];
  };

  visibility: "system" | "workspace" | "user_created";
};
```

### 8.2 AgentTaskRun

```ts
type AgentTaskRun = {
  id: string;
  runId: string;
  conversationId: string;
  assignedAgentName: string;
  capabilityId: string;
  status: "queued" | "running" | "success" | "failed" | "blocked" | "cancelled";
  inputSummary: string;
  outputSummary?: string;
  contextBundleId?: string;
  artifactIds: string[];
  traceId: string;
  startedAt?: string;
  completedAt?: string;
  safeError?: string;
};
```

### 8.3 AgentRunOutput

```ts
type AgentRunOutput = {
  agentName: string;
  taskId: string;
  status: "success" | "failed" | "needs_input" | "blocked";
  summary: string;
  text?: string;
  artifacts?: ArtifactDraft[];
  patches?: string[];
  nextActions?: {
    label: string;
    action: string;
  }[];
  requiresUserConfirmation?: boolean;
  errors?: {
    code: string;
    message: string;
  }[];
  metadata?: {
    model?: string;
    durationMs?: number;
  };
};
```

---

## 9. 开发优先级建议

### 9.1 Day 1：巩固 MVP 和统一契约

目标：

- P0/P1/P2 bug 修复完成。
- SSE、timeout、client reuse、history、ErrorBoundary、DB pool、parseCodeBlocks 等稳定。
- Skill / Contract 文档明确边界。

状态：你们目前已经完成这一阶段的大部分修复。

### 9.2 Day 2：web-agent + Registry

目标：

- 新增 web-agent。
- 暴露 AgentCard。
- Registry 发现 code-agent 和 web-agent。
- `/api/agents` 返回 Registry 信息。
- 前端支持 web_preview。

### 9.3 Day 3：LLM 意图编排

目标：

- Planner 生成 OrchestrationPlan。
- Plan Validator 校验。
- 支持 single / ordered_parallel。
- Planner 失败 fallback 关键词路由。
- 群聊中自动分派 code-agent / web-agent。

### 9.4 Day 4：多 Agent 消息与协议转换

目标：

- 每个 Agent 有独立 messageId / senderName。
- A2A artifact → AG-UI TOOL_CALL。
- code → code_preview。
- webpage → web_preview。
- STATE_UPDATE 显示编排过程。

### 9.5 Day 5：群聊和 @Agent

目标：

- conversationType 支持 single/group。
- @Agent 解析。
- 群聊 UI 显示多个 Agent。
- Agent 头像/标签区分。
- Planner 自动编排无 @ 请求。

### 9.6 Day 6：降级、测试、Docker

目标：

- Agent 失败 fallback。
- Docker compose 启动 5 个服务。
- smoke-test 覆盖 web-agent / orchestrator。
- Test Agent 思路可先体现在测试脚本里。

### 9.7 Day 7：Demo 和文档

目标：

- 3 分钟 Demo。
- README / 架构图 / AI 协作记录。
- 展示 Skills / Contract-first。
- 展示多 Agent 群聊和产物预览。

---

## 10. 测试与验收标准

### 10.1 AgentCard 测试

- `GET /.well-known/agent.json` 返回合法 JSON。
- 不包含 secret。
- skills 不为空。
- outputModes 包含实际 artifact 类型。
- url 与部署环境一致。

### 10.2 Health 测试

- `GET /health` 返回 200。
- Agent 挂掉时 Registry 标记 unhealthy。
- unhealthy Agent 不参与默认 plan。

### 10.3 A2A 流测试

- status working。
- text chunks。
- artifact。
- status completed。
- error safe message。

### 10.4 Orchestrator 调度测试

- “写 Go HTTP server” → code-agent。
- “写登录页面” → web-agent。
- “做计数器应用，前端+后端” → code-agent + web-agent。
- “@web-agent 改深色主题” → web-agent。
- Planner invalid JSON → fallback。
- web-agent down → fallback 或 safe error。

### 10.5 前端渲染测试

- senderName 显示正确。
- 多 Agent 多消息不串。
- code_preview 正常。
- web_preview 正常。
- markdown_render 正常。
- RUN_ERROR 不白屏。
- ErrorBoundary 生效。

### 10.6 安全测试

- AgentCard 无密钥。
- staged diff 无真实 API key。
- `.env` 不被提交。
- HTML preview sandbox。
- Markdown 不执行危险 HTML。
- 高危工具需确认。

---

## 11. 风险与规避

| 风险 | 表现 | 规避 |
|---|---|---|
| Agent 过多导致实现失控 | 每个都只做半成品 | v1.0 只做 code-agent + web-agent + doc/markdown 轻量能力 |
| Orchestrator 变成巨型 prompt | 难测试、难 debug | 使用状态机 + Plan Validator + fallback |
| 子 Agent 职责混乱 | Planner 写代码、Review 改代码 | 每个 Agent 写清负责/不负责 |
| 上下文污染 | Agent 被旧消息误导 | Context Bundle |
| 多 Agent 流式输出混乱 | 前端消息串流 | ordered_parallel + 独立 messageId |
| Artifact 不安全 | iframe 执行危险代码 | sandbox + security policy |
| 密钥泄露 | .env 被提交 | 安全扫描 + git ignore + review checklist |
| Docker Demo 失败 | compose 配置不稳定 | smoke-test + healthcheck + README |

---

## 12. 与五个项目文档的对应关系

| 本报告内容 | 对应文档 |
|---|---|
| 子 Agent 定义、不要 UI 拟人化、核心 Agent 链路 | `多Agent聊天式工作台_子Agent设计与开发建议报告.md` |
| code-agent / web-agent / registry / planner / docker demo 优先级 | `SPRINT-v1.0-Plan.md` |
| Frontend / Gateway / Orchestrator / A2A / 子 Agent 分层 | `UML-AgentHub系统图.md` |
| IM 聊天、群聊、产物内联、多 Agent 接入、评分点 | `AgentHub- 多Agent协作平台设计.pdf` |
| Skill / Contract 边界、安全、测试、交付规范 | `agenthub-skills-usage-report-v3.md` |

---

## 13. 外部参考资料

1. OpenAI Agents SDK - Agent orchestration  
   https://openai.github.io/openai-agents-python/multi_agent/

2. OpenAI Agents SDK - Guardrails  
   https://openai.github.io/openai-agents-python/guardrails/

3. OpenAI API Docs - Agents  
   https://developers.openai.com/api/docs/guides/agents

4. LangChain Docs - Multi-agent  
   https://docs.langchain.com/oss/python/langchain/multi-agent

5. LangChain Docs - Subagents / Supervisor Pattern  
   https://docs.langchain.com/oss/python/langchain/multi-agent/subagents-personal-assistant

6. LangGraph Supervisor Reference  
   https://reference.langchain.com/python/langgraph-supervisor

7. Anthropic - Building Effective Agents  
   https://www.anthropic.com/research/building-effective-agents

8. Google Developers Blog - Announcing Agent2Agent Protocol  
   https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/

9. a2aproject/A2A GitHub  
   https://github.com/a2aproject/A2A

10. Model Context Protocol Introduction  
    https://modelcontextprotocol.io/docs/getting-started/intro

11. Anthropic - Introducing the Model Context Protocol  
    https://www.anthropic.com/news/model-context-protocol

12. Microsoft Agent Framework Overview  
    https://learn.microsoft.com/en-us/agent-framework/overview/

13. Microsoft Agent Framework Workflows  
    https://learn.microsoft.com/en-us/agent-framework/workflows/

14. CrewAI Introduction  
    https://docs.crewai.com/en/introduction

15. CrewAI Crews  
    https://docs.crewai.com/en/concepts/crews

---

## 14. 最终建议

### 14.1 v1.0 应该真正落地的子 Agent

```text
code-agent
web-agent
doc-agent 或 markdown_render 能力
```

配套：

```text
Orchestrator
Registry
Planner
Artifact Converter
Frontend Skills
```

### 14.2 架构上应预留的子 Agent

```text
context-agent
explorer-agent
diff-agent
review-agent
test-agent
artifact-agent
security-agent
deploy-agent
version-agent
agent-builder-agent
```

### 14.3 答辩总结话术

> AgentHub 的子 Agent 不是 UI 功能模块，而是可被 Orchestrator 调度的专业任务执行单元。v1.0 中我们先落地 Code Agent 和 Web Agent，用 AgentCard 描述能力，通过 A2A 接入 Orchestrator，再由 Orchestrator 基于用户意图和 Registry 生成执行计划，最终把代码、网页和文档产物转换成 AG-UI 前端 Skills 渲染。后续可以按同一契约扩展 Document、Review、Test、Deploy、Security 和 Custom Agent，而不需要重写 Gateway 或前端协议。
