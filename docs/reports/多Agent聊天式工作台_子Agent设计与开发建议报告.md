# 多 Agent 聊天式工作台：子 Agent 设计与开发建议报告

> 版本：v1.0  
> 日期：2026-05-25  
> 适用对象：产品负责人、技术负责人、前后端工程师、Agent Runtime/平台工程师  
> 关键词：Subagent、Orchestrator、多 Agent 编排、聊天式 IDE、代码 Agent、Diff、Artifact、部署、上下文管理

---

## 目录

1. [报告结论](#1-报告结论)
2. [系统目标理解](#2-系统目标理解)
3. [什么才算子 Agent](#3-什么才算子-agent)
4. [外部资料参考与启发](#4-外部资料参考与启发)
5. [推荐子 Agent 总览](#5-推荐子-agent-总览)
6. [核心子 Agent 详细设计](#6-核心子-agent-详细设计)
7. [P1/P2 扩展子 Agent 详细设计](#7-p1p2-扩展子-agent-详细设计)
8. [Orchestrator 与子 Agent 的关系](#8-orchestrator-与子-agent-的关系)
9. [单聊模式与群聊模式的 Agent 调度](#9-单聊模式与群聊模式的-agent-调度)
10. [上下文管理设计](#10-上下文管理设计)
11. [工具权限与安全控制](#11-工具权限与安全控制)
12. [数据模型建议](#12-数据模型建议)
13. [Agent Adapter 层设计](#13-agent-adapter-层设计)
14. [消息、产物、Diff、部署卡片设计建议](#14-消息产物diff部署卡片设计建议)
15. [开发优先级与里程碑](#15-开发优先级与里程碑)
16. [工程风险与解决方案](#16-工程风险与解决方案)
17. [评估指标](#17-评估指标)
18. [最终建议](#18-最终建议)
19. [参考资料](#19-参考资料)

---

## 1. 报告结论

你们要做的系统，本质上不是一个普通聊天产品，而是一个 **聊天驱动的多 Agent 协作工作台**。用户通过单聊或群聊提出目标，系统由 **Orchestrator 主协调器**理解需求、拆解任务、选择合适的子 Agent 执行，并把结果以聊天消息、代码块、Diff 卡片、预览卡片、部署状态卡片等形式返回。

本报告的核心结论是：

> 子 Agent 不应该按 UI 功能划分，也不应该把会话列表、消息复制、卡片展开这种功能拟人化成 Agent。真正的子 Agent 应该围绕“任务执行链路”划分。

推荐的 MVP 子 Agent 链路是：

```text
Planner Agent
→ Context Agent
→ Codebase Explorer Agent
→ Code Agent
→ Diff Agent
→ Review Agent
→ Test Agent
→ Artifact Agent
```

P1/P2 再扩展：

```text
File Agent
Web Research Agent
Agent Builder Agent
Security Agent
Deploy Agent
Version Agent
Document Agent
PPT Agent
Release Agent
```

最重要的建议：

1. **先做 Agent Runtime，不要先堆 prompt。**
2. **子 Agent 必须职责清晰、上下文隔离、工具权限受控、输出结构化。**
3. **Orchestrator 要由“状态机 + LLM 决策点 + 工具权限系统”组成，而不是一个巨型 prompt。**
4. **代码修改必须走 Diff → Review → Test → Apply 的闭环。**
5. **用户看到的是自然聊天和产物卡片，内部 Agent 之间应尽量使用结构化 JSON 通信。**

---

## 2. 系统目标理解

根据你们描述的功能，系统包含以下能力：

### 2.1 聊天入口

- 左侧对话列表
- 新建、置顶、归档、搜索
- 最近活跃排序
- 单聊模式
- 群聊模式
- 支持 `@Agent` 指定回复
- Orchestrator 自动分派

### 2.2 多类型消息

系统消息不仅是纯文本，还包括：

- 文本
- 代码块
- 图片
- 文件附件
- 网页预览卡片
- Diff 视图卡片
- 部署状态卡片
- 产物预览卡片
- 文档/PPT 浏览卡片

### 2.3 消息操作

- 回复
- 引用
- 重新生成
- 复制代码
- 一键应用 Diff
- 展开预览
- 对选中代码做局部修改

### 2.4 主 Agent 协调器

Orchestrator 在群聊模式下负责：

- 理解用户意图
- 拆解复杂任务
- 自动分派给合适子 Agent
- 聚合输出
- 汇报结果
- 并行调度
- 失败降级
- 代码冲突处理

### 2.5 多 Agent 接入

系统要通过统一适配器层接入多个 Agent 平台，例如：

- Claude Code
- Codex
- OpenCode
- 未来可能的本地 Agent 或第三方 Agent

同时支持用户自建 Agent，用户可以通过对话配置：

- Agent 名称
- 头像
- 能力标签
- System Prompt
- 工具集
- 权限边界

### 2.6 产物预览与编辑

Agent 的输出会变成可交互产物：

- 网页 iframe 预览
- 文档渲染
- PPT 浏览
- Diff 视图
- 版本历史
- 代码编辑器
- 对话式局部修改

### 2.7 部署发布

用户可以在聊天中直接发送“部署”，系统返回：

- 部署状态卡片
- 构建日志
- 预览 URL
- 静态站点部署
- 容器化部署
- 源码包下载

---

## 3. 什么才算子 Agent

### 3.1 定义

在你们系统里，子 Agent 应定义为：

> 被 Orchestrator 调度、拥有独立职责、独立上下文、独立工具权限、独立输出契约的执行型智能体。

它不是 UI 模块，不是简单后端服务，也不是普通函数。它的核心价值是：在一个复杂任务中承担明确的“智能执行角色”。

### 3.2 子 Agent 应满足的条件

一个东西要被称为子 Agent，至少应满足以下条件：

| 条件 | 说明 |
|---|---|
| 独立职责 | 例如写代码、审查代码、分析文件、部署 |
| 独立 System Prompt | 行为规则、输出格式、工作边界独立 |
| 独立上下文 | 不直接读取全量聊天历史，而是读取 context bundle |
| 独立工具权限 | 不同 Agent 拥有不同工具访问能力 |
| 结构化输出 | 输出可被 Orchestrator 稳定解析 |
| 可被调度 | 由 Orchestrator 调用、串行/并行执行 |
| 可观测 | 每次执行有 task run、日志、状态、结果 |

### 3.3 不应该算作子 Agent 的内容

以下内容是产品功能、UI 模块或后端服务，不应设计成子 Agent：

| 功能 | 正确归类 |
|---|---|
| 左侧会话列表 | Conversation UI + Conversation Service |
| 新建会话 | Conversation CRUD |
| 置顶/归档 | Conversation Metadata |
| 搜索会话 | Search Service |
| 消息复制 | UI Action |
| 展开预览 | UI Action + Artifact Renderer |
| 联系人头像展示 | Agent Profile UI |
| 按最近活跃排序 | 数据库查询逻辑 |
| 一键应用 Diff 按钮 | UI Action + Patch Service |
| 部署状态卡片渲染 | UI Renderer |

这些模块可以被 Agent 触发或消费，但它们本身不是 Agent。

---

## 4. 外部资料参考与启发

### 4.1 Claude Code Subagents

Claude Code 文档中提到，subagents 可以用来并行处理任务、隔离上下文、应用专门指令，并返回总结结果。Claude Code 的功能概览也明确提到 subagents 会在隔离上下文中运行自己的循环并返回 summaries。

对你们的启发：

- 子 Agent 不应该共享完整主上下文。
- 子 Agent 应该有专门能力描述。
- 子 Agent 适合做并行探索、审查、分析、测试。
- 子 Agent 的产出应回到主 Agent/Orchestrator 汇总。

### 4.2 OpenAI Agents：handoff 与 agents-as-tools

OpenAI Agents SDK 文档强调两种多 Agent 编排方式：

1. **Handoff**：一个 Agent 把控制权交给另一个 Agent。
2. **Agents as tools**：主 Agent 保持最终控制权，把专业 Agent 当工具调用。

对你们的启发：

- 群聊默认建议采用 **Orchestrator + agents-as-tools**。
- 用户显式 `@Agent` 时，可以做局部 handoff。
- 最终回复应该由 Orchestrator 聚合，避免多个 Agent 抢最终输出权。

### 4.3 LangChain / LangGraph

LangChain 多 Agent 文档中将多 Agent 模式分为 subagents、handoffs、router、workflow 等。LangGraph 的 supervisor 模式强调由 supervisor 调度多个 specialized agents。

对你们的启发：

- Orchestrator 内部应有 Router/Agent Selector。
- 部分任务走确定性 workflow，部分任务交给 LLM 决策。
- Agent 之间不能随意互传全部消息，需要明确 context engineering。

### 4.4 CrewAI

CrewAI 的文档强调 Crews 是一组协作 Agent，Flows 管理状态和执行流程。

对你们的启发：

- Orchestrator 不应该只是 prompt，而应是有状态工作流。
- Agent 只负责局部任务。
- 任务依赖、状态、失败重试、人工确认应由 Runtime 管理。

### 4.5 AutoGen 与 Microsoft Agent Framework

AutoGen 是多 Agent 对话和协作框架。Microsoft Agent Framework 则是 AutoGen 与 Semantic Kernel 团队后续整合方向，强调 session state、type safety、filters、telemetry、multi-agent workflows。

对你们的启发：

- 生产级 Agent 系统需要状态、类型、安全、遥测。
- 要记录每个 Agent run。
- 要能复现和 debug Agent 的输出。
- 多 Agent 编排应支持 sequential、concurrent、handoff、group chat、manager-style coordination 等模式。

### 4.6 MCP

Model Context Protocol 是连接 AI 应用与外部数据源、工具和工作流的开放协议。

对你们的启发：

- 文件、数据库、部署平台、搜索、CI/CD 等外部能力可以抽象成 MCP server 或类似工具层。
- Agent 不应直接访问所有系统，而应通过权限化工具网关访问。
- 长期可减少对单一模型/平台的绑定。

---

## 5. 推荐子 Agent 总览

### 5.1 MVP 必备子 Agent

| 子 Agent | 是否必备 | 核心用途 |
|---|---:|---|
| Planner Agent | 必备 | 拆任务、生成任务树、标注依赖 |
| Context Agent | 必备 | 压缩聊天历史、整理 pin 消息、打包上下文 |
| Codebase Explorer Agent | 必备 | 只读分析代码库、定位相关文件 |
| Code Agent | 必备 | 写代码、改代码、输出 Diff |
| Diff Agent | 必备 | 生成、校验、解释、合并 Diff |
| Review Agent | 必备 | 检查需求覆盖、代码质量、安全问题 |
| Test Agent | 强烈建议 | 运行测试、类型检查、解释失败 |
| Artifact Agent | 必备 | 把产出转成预览卡片/产物描述 |

### 5.2 P1 推荐子 Agent

| 子 Agent | 核心用途 |
|---|---|
| File Agent | 处理用户上传文件、图片、PDF、ZIP |
| Web Research Agent | 读取 URL、生成网页预览、网络资料摘要 |
| Agent Builder Agent | 帮用户创建自定义 Agent |
| Security Agent | 审查工具权限、高危操作、敏感文件 |
| QA Acceptance Agent | 根据原始需求做最终验收 |

### 5.3 P2 推荐子 Agent

| 子 Agent | 核心用途 |
|---|---|
| Deploy Agent | 构建、部署、生成预览 URL |
| Version Agent | 版本快照、回滚、版本对比 |
| Document Agent | Markdown/DOCX 文档生成与编辑 |
| PPT Agent | PPT 生成、浏览、逐页修改 |
| Release Agent | 打包源码、生成 changelog、发布说明 |

---

## 6. 核心子 Agent 详细设计

---

## 6.1 Planner Agent

### 定位

Planner Agent 是任务规划者。它负责把复杂用户目标拆成可执行任务树，但不直接写代码、不审查代码、不部署。

### 主要用途

- 理解复杂需求
- 拆分任务
- 判断任务依赖
- 判断并行/串行
- 推荐执行子 Agent
- 生成验收标准

### 典型触发

用户输入：

```text
帮我做一个多 Agent 聊天系统，支持 Claude Code、Codex、文件预览、Diff 和部署。
```

Planner Agent 应输出：

- 任务列表
- 每个任务的目标
- 依赖关系
- 推荐 Agent
- 验收标准

### 输入示例

```json
{
  "user_goal": "实现多 Agent 群聊系统",
  "conversation_summary": "用户正在设计一个多 Agent 聊天式开发工作台",
  "available_agents": [
    "context_agent",
    "codebase_explorer_agent",
    "code_agent",
    "diff_agent",
    "review_agent",
    "test_agent",
    "artifact_agent"
  ],
  "constraints": {
    "priority": "MVP",
    "mode": "group_chat"
  }
}
```

### 输出示例

```json
{
  "plan_id": "plan_001",
  "tasks": [
    {
      "id": "task_1",
      "title": "设计消息类型模型",
      "agent": "code_agent",
      "depends_on": [],
      "parallelizable": false,
      "acceptance_criteria": [
        "支持 text/code/image/file/web_preview/diff/deployment_status",
        "使用 discriminated union"
      ]
    },
    {
      "id": "task_2",
      "title": "实现 Agent 群聊消息流",
      "agent": "code_agent",
      "depends_on": ["task_1"],
      "parallelizable": false,
      "acceptance_criteria": [
        "支持 @Agent 指定",
        "支持 Orchestrator 自动分派"
      ]
    }
  ]
}
```

### 开发建议

- Planner 输出必须结构化。
- 不要让 Planner 写实现细节。
- Planner 的结果应允许用户查看和修改。
- 复杂任务应先 plan，再执行。
- 简单任务可以跳过 Planner，直接路由到 Code Agent 或 File Agent。

---

## 6.2 Context Agent

### 定位

Context Agent 是上下文管理者。它负责把聊天历史、pin 消息、文件摘要、代码信息整理成适合某个子 Agent 使用的上下文包。

### 主要用途

- 压缩长对话
- 识别关键需求
- 处理用户 pin 的长期上下文
- 为不同子 Agent 生成不同 context bundle
- 避免无关历史污染 Agent

### 为什么重要

多 Agent 系统最大的问题之一是上下文混乱。如果所有 Agent 都拿完整聊天历史，会出现：

- token 成本高
- 子 Agent 注意力分散
- 历史错误信息污染当前任务
- 最新指令被忽略
- 不同 Agent 对需求理解不一致

### 输入示例

```json
{
  "target_agent": "code_agent",
  "task": "实现 MessageCard 组件",
  "conversation_messages": [],
  "pinned_messages": [],
  "artifacts": [],
  "files": []
}
```

### 输出示例

```json
{
  "context_bundle_id": "ctx_001",
  "summary": "用户正在开发一个聊天式多 Agent 工作台。",
  "requirements": [
    "支持单聊和群聊模式",
    "群聊可通过 @ 指定 Agent 或由 Orchestrator 自动分派",
    "消息支持文本、代码块、图片、文件、网页预览、Diff、部署状态"
  ],
  "constraints": [
    "不要把 UI 功能误认为子 Agent",
    "MVP 优先实现代码修改闭环"
  ],
  "relevant_artifacts": [],
  "relevant_files": [
    "src/components/chat/MessageCard.tsx",
    "src/types/message.ts"
  ],
  "excluded_context_reason": [
    {
      "message_id": "msg_12",
      "reason": "与当前任务无关的闲聊"
    }
  ]
}
```

### 开发建议

- 每次 Agent run 都要记录使用了哪个 context bundle。
- 最新用户指令优先级最高。
- pin 消息不等于永远全量注入，应按任务选择。
- 对代码任务，Context Agent 应结合 Codebase Explorer 的结果。
- 对 Review Agent，context 应包含原始需求、Diff、测试结果，不需要全量聊天。

---

## 6.3 Codebase Explorer Agent

### 定位

Codebase Explorer Agent 是代码库探索者。它是只读 Agent，负责理解项目结构、定位相关文件、识别架构约束。

### 为什么单独拆出来

如果 Code Agent 既要探索又要修改，容易：

- 盲目改错文件
- 改动范围过大
- 虚构文件
- 忽略现有架构
- 浪费上下文窗口

Explorer 先读代码、再把结果交给 Code Agent，可以提升稳定性。

### 主要用途

- 扫描目录结构
- 搜索组件/API/类型定义
- 找到相关文件
- 总结架构约定
- 推荐修改位置

### 工具权限

| 工具 | 权限 |
|---|---|
| list_directory | 允许 |
| read_file | 允许 |
| search_files | 允许 |
| grep | 允许 |
| write_file | 禁止 |
| apply_patch | 禁止 |
| shell | 仅允许只读命令 |

### 输出示例

```json
{
  "target_files": [
    {
      "path": "src/types/message.ts",
      "reason": "定义消息类型，需要扩展消息种类"
    },
    {
      "path": "src/components/chat/MessageCard.tsx",
      "reason": "负责消息卡片渲染"
    },
    {
      "path": "src/components/chat/MessageList.tsx",
      "reason": "负责消息流展示，需要适配新消息类型"
    }
  ],
  "architecture_notes": [
    "项目使用 React + TypeScript",
    "组件采用函数组件",
    "样式使用 Tailwind CSS",
    "消息类型目前只支持 text/code"
  ],
  "risks": [
    "修改 Message 类型可能影响 MessageList 和 MessageInput"
  ],
  "recommended_next_agent": "code_agent"
}
```

### 开发建议

- Explorer Agent 默认只读。
- 输出应给出“为什么这些文件相关”。
- 不允许它直接改代码。
- 可以并行启动多个 Explorer，例如 UI Explorer、API Explorer、Data Model Explorer。

---

## 6.4 Code Agent

### 定位

Code Agent 是核心生产 Agent，负责写代码、改代码、生成组件、实现 API、输出 patch。

### 主要用途

- 生成 React 组件
- 修改已有代码
- 实现接口
- 创建类型定义
- 编写测试
- 根据 Review 反馈修复
- 生成统一 Diff

### 底层可接入

- Claude Code
- Codex
- OpenCode
- 本地 Agent
- 其他第三方 coding agent

### 输入示例

```json
{
  "task": "实现 MessageCard 组件，支持多种消息类型",
  "context_bundle_id": "ctx_001",
  "target_files": [
    "src/types/message.ts",
    "src/components/chat/MessageCard.tsx"
  ],
  "coding_standards": [
    "TypeScript strict",
    "React functional components",
    "Tailwind CSS",
    "输出 unified diff"
  ],
  "expected_output": "unified_diff"
}
```

### 输出示例

```json
{
  "summary": "新增 MessageCard 组件并扩展 Message 类型。",
  "changed_files": [
    "src/types/message.ts",
    "src/components/chat/MessageCard.tsx"
  ],
  "diff": "diff --git a/src/types/message.ts b/src/types/message.ts ...",
  "assumptions": [
    "项目已配置 Tailwind CSS",
    "部署状态卡片先只做展示，不触发真实部署"
  ],
  "needs_review": true
}
```

### 开发建议

- Code Agent 默认输出 Diff，而不是直接写文件。
- 修改必须尽量小。
- 不要让 Code Agent 自己决定部署。
- 不要让 Code Agent 自己绕过 Review。
- 可以根据任务规模选择不同模型：小任务用便宜模型，大型重构用强模型。

---

## 6.5 Diff Agent

### 定位

Diff Agent 是代码修改管理者。它负责生成、校验、解释、合并 Diff。

### 主要用途

- 生成 unified diff
- 判断 patch 是否能应用
- 解释改动
- 合并多个 patch
- 处理冲突
- 支持一键应用 Diff
- 支持选中代码后的局部修改
- 生成回滚 patch

### 输入示例

```json
{
  "base_version": "ver_001",
  "proposed_changes": [
    {
      "source_agent": "code_agent",
      "diff": "..."
    }
  ],
  "conflict_policy": "prefer_latest_user_instruction",
  "output_format": "unified_diff"
}
```

### 输出示例

```json
{
  "patch_id": "patch_001",
  "apply_status": "applicable",
  "risk_level": "medium",
  "changed_files": [
    "src/types/message.ts",
    "src/components/chat/MessageCard.tsx"
  ],
  "human_summary": [
    "扩展 Message 类型",
    "新增多类型消息卡片渲染",
    "未修改后端 API"
  ],
  "diff": "...",
  "conflicts": []
}
```

### 开发建议

- Diff Agent 必须结合真实 patch 工具验证可应用性。
- 不要完全依赖 LLM 判断 Diff 是否能应用。
- 多 Agent 并发修改时，每个 patch 要绑定 base version。
- 冲突不可自动安全合并时，应交给用户确认。
- Diff 卡片应展示风险等级和影响文件。

---

## 6.6 Review Agent

### 定位

Review Agent 是质量审查者。它负责检查代码或产物是否满足需求、是否有 bug、是否有安全风险。

### 主要用途

- 检查需求覆盖
- 检查代码质量
- 检查类型安全
- 检查边界情况
- 检查安全问题
- 检查可维护性
- 判断是否需要返工

### 输入示例

```json
{
  "original_requirement": "实现支持多消息类型的 MessageCard",
  "context_bundle_id": "ctx_001",
  "patch_id": "patch_001",
  "diff": "...",
  "test_results": []
}
```

### 输出示例

```json
{
  "status": "changes_requested",
  "findings": [
    {
      "severity": "high",
      "file": "src/components/chat/MessageCard.tsx",
      "issue": "diff 消息没有处理空 diff 内容",
      "suggestion": "增加 empty state"
    },
    {
      "severity": "medium",
      "file": "src/types/message.ts",
      "issue": "deployment_status 缺少 failed 状态下的错误信息字段",
      "suggestion": "增加 errorMessage?: string"
    }
  ],
  "passed_checks": [
    "消息类型使用 discriminated union",
    "组件职责清晰",
    "未引入多余依赖"
  ],
  "recommended_next_agent": "code_agent"
}
```

### 开发建议

- Review Agent 不直接改代码。
- Review 输出必须分 severity。
- Review 应绑定原始需求，否则容易只做代码风格审查。
- Review 应能触发 Code Agent 返工。
- 对高危问题，Orchestrator 应阻止 apply patch 或 deploy。

---

## 6.7 Test Agent

### 定位

Test Agent 是验证者。它负责运行测试、类型检查、lint、build，并解释失败原因。

### 主要用途

- 跑单元测试
- 跑类型检查
- 跑 lint
- 跑 build
- 分析失败日志
- 给 Code Agent 提供修复建议

### 工具权限

| 工具 | 权限 |
|---|---|
| npm test / pnpm test | 允许 |
| npm run typecheck | 允许 |
| npm run lint | 允许 |
| npm run build | 允许 |
| destructive shell command | 禁止 |
| deploy command | 禁止 |

### 输出示例

```json
{
  "status": "failed",
  "commands": [
    {
      "command": "pnpm typecheck",
      "exit_code": 1,
      "summary": "Message 类型缺少 deployment_status 分支"
    },
    {
      "command": "pnpm test",
      "exit_code": 0,
      "summary": "测试通过"
    }
  ],
  "failure_analysis": [
    "MessageCard 对 deployment_status 类型进行了渲染，但 Message union type 中未定义该类型"
  ],
  "recommended_fix": "更新 src/types/message.ts，加入 DeploymentStatusMessage 类型"
}
```

### 开发建议

- Test Agent 可使用低成本模型，重点是工具执行与日志解释。
- 测试失败应自动回传给 Code Agent 修复。
- 不要让 Test Agent 自己改代码。
- 对无法运行测试的项目，应说明原因，而不是假装通过。

---

## 6.8 Artifact Agent

### 定位

Artifact Agent 是产物语义描述者。它不直接渲染 UI，而是把 Agent 产出转换成可被前端渲染的 artifact manifest。

### 主要用途

- 生成网页预览 artifact
- 生成文档 artifact
- 生成代码编辑 artifact
- 生成 Diff artifact
- 生成部署状态 artifact
- 维护产物与消息、patch、version 的关系

### 输入示例

```json
{
  "source_agent": "code_agent",
  "patch_id": "patch_001",
  "output_files": [
    "src/components/chat/MessageCard.tsx"
  ],
  "artifact_type_hint": "web_preview"
}
```

### 输出示例

```json
{
  "artifact_id": "art_001",
  "type": "web_preview",
  "title": "MessageCard Preview",
  "source": {
    "kind": "local_dev_server",
    "url": "http://localhost:5173/preview/message-card"
  },
  "actions": [
    "open_fullscreen",
    "edit_code",
    "regenerate",
    "create_diff",
    "deploy"
  ],
  "metadata": {
    "created_by_agent": "code_agent",
    "related_patch_id": "patch_001",
    "related_version_id": "ver_001"
  }
}
```

### 开发建议

- Artifact Agent 输出语义，不负责 UI 细节。
- 前端由 Artifact Renderer 根据 type 渲染卡片。
- Artifact 应可追溯到 Agent、Patch、Version、Message。
- Artifact 可以是代码、网页、文档、PPT、部署结果等统一抽象。

---

## 7. P1/P2 扩展子 Agent 详细设计

---

## 7.1 File Agent

### 定位

File Agent 负责处理用户上传的文件、图片、PDF、ZIP、代码文件，并把它们转换成 Agent 可使用的上下文。

### 主要用途

- 解析 PDF
- 解析 DOCX/Markdown
- 识别图片内容
- 解压 ZIP
- 总结代码文件
- 提取需求
- 生成文件摘要
- 给 Planner/Code Agent 提供文件上下文

### 输出示例

```json
{
  "file_context_id": "file_ctx_001",
  "files": [
    {
      "name": "requirements.pdf",
      "type": "pdf",
      "summary": "该文件描述了多 Agent 聊天系统的功能需求。",
      "extracted_requirements": [
        "支持单聊和群聊",
        "支持 Orchestrator 自动分派",
        "支持 Diff 卡片和部署状态卡片"
      ]
    }
  ],
  "recommended_agents": [
    "planner_agent",
    "context_agent"
  ]
}
```

### 开发建议

- 文件解析结果要缓存。
- 敏感文件要先经过 Security Agent。
- 大文件不要直接塞进上下文，应摘要化。
- ZIP 项目应先由 File Agent 解压，再交给 Explorer Agent 分析。

---

## 7.2 Web Research Agent

### 定位

Web Research Agent 负责处理 URL、网页预览、网络资料检索和引用来源整理。

### 主要用途

- 读取用户发送的 URL
- 生成网页预览卡片
- 提取标题、描述、favicon
- 摘要网页正文
- 为报告生成参考资料
- 给其他 Agent 提供当前网络信息

### 输出示例

```json
{
  "web_context_id": "web_ctx_001",
  "url": "https://code.claude.com/docs/en/agents",
  "title": "Run agents in parallel - Claude Code Docs",
  "summary": "该页面介绍 Claude Code 中并行运行 agents 的方式，包括 subagents、agent view、agent teams 等。",
  "preview_card": {
    "title": "Run agents in parallel",
    "description": "Compare the ways Claude Code can take on multiple tasks at once.",
    "favicon": "..."
  },
  "citations": [
    {
      "label": "Claude Code Docs",
      "url": "https://code.claude.com/docs/en/agents"
    }
  ]
}
```

### 开发建议

- Web Research Agent 与 Code Agent 分离。
- 所有网络引用要可追踪。
- 对时效性内容要记录访问时间。
- 不要让普通 Agent 随意联网，统一通过 Web Research Agent。

---

## 7.3 Agent Builder Agent

### 定位

Agent Builder Agent 帮用户通过对话创建自定义 Agent。

### 主要用途

- 生成 Agent 名称
- 生成头像建议
- 生成能力标签
- 生成 System Prompt
- 配置工具集
- 配置权限等级
- 生成 Agent Profile

### 输出示例

```json
{
  "agent_profile": {
    "name": "PRD Writer",
    "description": "用于产品需求分析、PRD 编写和用户故事拆解。",
    "tags": [
      "PRD",
      "产品设计",
      "需求分析"
    ],
    "system_prompt": "你是一个严谨的产品需求文档专家...",
    "tools": [
      "read_context",
      "write_markdown",
      "create_artifact"
    ],
    "permission_level": "write_artifact",
    "default_output_format": "markdown"
  }
}
```

### 开发建议

- 自定义 Agent 默认只读或只写 artifact。
- 开启写代码、执行命令、部署等权限必须确认。
- System Prompt 要自动加入安全边界。
- 自定义 Agent 的能力描述要用于 Router 选择。

---

## 7.4 Security Agent

### 定位

Security Agent 负责检查权限、安全风险、敏感文件和高危工具调用。

### 触发场景

- Agent 要执行 shell 命令
- Agent 要访问网络
- Agent 要读取私密文件
- Agent 要修改 `.env`
- Agent 要部署
- Agent 要删除文件
- Agent 要安装依赖
- Agent 要访问外部 API

### 输出示例

```json
{
  "risk_level": "high",
  "blocked": true,
  "reasons": [
    "检测到操作会修改 .env 文件",
    "部署目标为 production，需要用户确认"
  ],
  "safe_alternatives": [
    "改为 preview 环境部署",
    "先生成 Diff，不直接应用"
  ]
}
```

### 开发建议

- Security Agent 不能只靠 LLM。
- 应结合规则引擎、权限系统、审计日志。
- 高危操作必须人类确认。
- 每次工具调用都要记录 agentId、taskId、userId、timestamp。

---

## 7.5 Deploy Agent

### 定位

Deploy Agent 负责构建、部署、返回部署状态和预览 URL。

### 主要用途

- 静态站点部署
- 容器化部署
- 生成预览 URL
- 生成部署状态卡片
- 解释部署失败
- 下载源码包
- 回滚部署

### 输出示例

```json
{
  "deployment_id": "dep_001",
  "status": "success",
  "target": "preview",
  "preview_url": "https://preview.example.com/dep_001",
  "logs_summary": "Build completed successfully.",
  "actions": [
    "open_url",
    "rollback",
    "download_source"
  ]
}
```

### 开发建议

- MVP 可先只做“生成源码包/本地 preview”。
- P2 再接 Vercel、Netlify、Cloudflare Pages、Docker。
- 生产部署必须确认。
- 部署状态要用事件流推送给前端。

---

## 7.6 Version Agent

### 定位

Version Agent 管理产物版本、代码快照、patch 关系、回滚和版本对比。

### 主要用途

- 记录每次 Agent 修改
- 保存 artifact 版本
- 保存 patch 与 base version
- 支持回滚
- 支持版本对比
- 支持命名版本

### 输出示例

```json
{
  "version_id": "ver_003",
  "parent_version_id": "ver_002",
  "created_by": "code_agent",
  "summary": "新增多类型消息卡片。",
  "related_patch_id": "patch_001",
  "rollback_patch_id": "patch_rollback_001"
}
```

### 开发建议

- 代码版本仍建议落到 Git。
- 产品内 Version Agent 管 artifact、patch、预览版本。
- 每个 patch 必须绑定 base version。
- 回滚要生成反向 patch，而不是直接覆盖。

---

## 7.7 Document Agent

### 定位

Document Agent 负责生成和修改 Markdown、README、PRD、技术方案等文档产物。

### 主要用途

- 生成 Markdown 报告
- 生成 PRD
- 生成技术方案
- 生成 README
- 生成 API 文档
- 根据聊天内容整理文档

### 输出示例

```json
{
  "artifact_type": "document",
  "format": "markdown",
  "title": "多 Agent 系统技术方案",
  "content": "...",
  "sections": [
    "背景",
    "目标",
    "架构",
    "子 Agent 设计",
    "开发计划"
  ]
}
```

### 开发建议

- Document Agent 与 PPT Agent 分离。
- 文档 Agent 应支持引用来源。
- 文档生成后应创建 artifact，支持继续对话式修改。

---

## 7.8 PPT Agent

### 定位

PPT Agent 负责生成、浏览、修改演示文稿结构。

### 主要用途

- 从文档生成 PPT 大纲
- 生成 slide JSON
- 修改指定页
- 生成演讲备注
- 输出可渲染 PPT artifact

### 输出示例

```json
{
  "artifact_type": "presentation",
  "slides": [
    {
      "page": 1,
      "title": "产品定位",
      "bullets": [
        "聊天式多 Agent 工作台",
        "面向开发与创作场景",
        "统一管理代码、文档、预览和部署"
      ],
      "speaker_notes": "介绍产品核心价值和使用场景。"
    }
  ]
}
```

### 开发建议

- P2 再做 PPT。
- 先输出结构化 slide JSON。
- 真正 PPTX 生成由后端渲染服务负责。
- 支持“修改第 3 页标题”“把第 5 页改成对比页”等局部修改。

---

## 8. Orchestrator 与子 Agent 的关系

### 8.1 Orchestrator 不是普通子 Agent

Orchestrator 是主协调器。它不应该和 Code Agent、Review Agent 同级对待。

它负责：

- 理解用户意图
- 判断是否需要拆任务
- 选择子 Agent
- 管理任务状态
- 管理上下文
- 管理工具权限
- 控制串行/并行
- 处理失败重试
- 聚合结果
- 向用户汇报
- 生成最终消息和卡片

### 8.2 Orchestrator 应由三部分组成

```text
Orchestrator = 状态机 + LLM 决策点 + 工具/权限 Runtime
```

不要做成：

```text
Orchestrator = 一个超长 System Prompt
```

### 8.3 推荐内部组件

| 组件 | 用途 |
|---|---|
| Intent Classifier | 判断用户意图 |
| Agent Selector | 选择子 Agent |
| Task Planner | 调用 Planner 或生成轻量 plan |
| Task State Machine | 管理任务状态 |
| Context Builder | 调用 Context Agent |
| Tool Permission Manager | 管理工具权限 |
| Event Stream Manager | 推送执行事件 |
| Result Aggregator | 汇总子 Agent 输出 |
| Failure Handler | 重试、降级、要求用户确认 |

---

## 9. 单聊模式与群聊模式的 Agent 调度

### 9.1 单聊模式

单聊模式下，用户看起来是在和一个 Agent 对话，例如：

```text
用户 ↔ Claude Code Agent
```

但底层仍建议经过 Orchestrator：

```text
User
→ Orchestrator
→ Context Agent
→ Code Agent
→ Diff Agent
→ Review/Test Agent
→ Orchestrator
→ User
```

原因：

- 需要统一记录消息
- 需要权限控制
- 需要上下文打包
- 需要 Diff/Review/Test 闭环
- 需要卡片生成

### 9.2 群聊模式

群聊模式下，用户可以指定：

```text
@CodeAgent 帮我实现这个组件
@ReviewAgent 检查刚才的代码
```

也可以不指定：

```text
帮我完成这个页面，写代码、检查、生成预览。
```

推荐流程：

```text
User
→ Orchestrator
→ Planner Agent
→ Context Agent
→ Codebase Explorer Agent
→ Code Agent
→ Diff Agent
→ Review Agent + Test Agent
→ Artifact Agent
→ Orchestrator 汇总
```

### 9.3 Agent 依次回复的产品表现

用户看到的聊天流可以是：

```text
Planner Agent：我已拆解为 4 个任务。
Codebase Explorer Agent：我找到了 3 个相关文件。
Code Agent：我生成了一个 Diff。
Review Agent：发现 1 个问题。
Code Agent：已修复。
Test Agent：类型检查通过。
Artifact Agent：已生成预览卡片。
Orchestrator：任务完成，这是结果。
```

底层可以并行执行，但 UI 可以按逻辑顺序展示。

---

## 10. 上下文管理设计

### 10.1 上下文来源

| 来源 | 说明 |
|---|---|
| 聊天历史 | 当前会话历史 |
| Pin 消息 | 用户手动标记的长期上下文 |
| 文件内容 | 用户上传文件或项目文件 |
| 代码库摘要 | Explorer Agent 生成 |
| Artifact | 已生成产物 |
| Diff/Patch | 代码修改记录 |
| Version | 历史版本 |
| 用户偏好 | 项目级偏好、技术栈、风格要求 |

### 10.2 上下文优先级

建议优先级：

```text
最新用户指令
> 显式 @Agent 指令
> 当前任务上下文
> Pin 消息
> 项目约束
> 最近相关聊天
> 长期历史摘要
```

### 10.3 Context Bundle 数据结构

```ts
type ContextBundle = {
  id: string;
  targetAgentId: string;
  taskId: string;

  userGoal: string;
  latestInstruction: string;

  requirements: string[];
  constraints: string[];
  assumptions: string[];

  pinnedContext: {
    messageId: string;
    content: string;
    reason: string;
  }[];

  relevantMessages: {
    messageId: string;
    content: string;
    reason: string;
  }[];

  relevantFiles: {
    path: string;
    summary: string;
    reason: string;
  }[];

  relevantArtifacts: {
    artifactId: string;
    type: string;
    summary: string;
  }[];

  excludedContextNotes: string[];
};
```

### 10.4 开发建议

- 不要把全量聊天历史直接给所有 Agent。
- 不同 Agent 需要不同上下文。
- 代码 Agent 需要目标文件和需求。
- Review Agent 需要需求、Diff、测试结果。
- Deploy Agent 需要可部署 artifact、构建命令、目标环境。
- Security Agent 需要工具调用、权限、敏感文件信息。

---

## 11. 工具权限与安全控制

### 11.1 权限矩阵

| Agent | 读文件 | 写文件 | 生成 Diff | 应用 Diff | 执行命令 | 网络 | 部署 |
|---|---:|---:|---:|---:|---:|---:|---:|
| Planner | 否 | 否 | 否 | 否 | 否 | 否 | 否 |
| Context | 读上下文 | 否 | 否 | 否 | 否 | 否 | 否 |
| Explorer | 是 | 否 | 否 | 否 | 只读 | 否 | 否 |
| Code | 是 | 否 | 是 | 否 | 限制 | 否 | 否 |
| Diff | 是 | 否 | 是 | 需确认 | 否 | 否 | 否 |
| Review | 是 | 否 | 否 | 否 | 可运行检查 | 否 | 否 |
| Test | 是 | 否 | 否 | 否 | 是 | 否 | 否 |
| Artifact | 读产物 | 写 artifact | 否 | 否 | 否 | 否 | 否 |
| File | 读上传文件 | 写摘要 | 否 | 否 | 否 | 否 | 否 |
| Web Research | 否 | 写摘要 | 否 | 否 | 否 | 是 | 否 |
| Deploy | 是 | 写构建产物 | 否 | 否 | 是 | 是 | 是 |
| Security | 审计 | 否 | 否 | 否 | 否 | 否 | 否 |

### 11.2 高危操作

以下操作必须确认：

- 生产部署
- 删除文件
- 覆盖文件
- 修改 `.env`
- 执行数据库迁移
- 安装未知依赖
- 执行 shell 高危命令
- 对外发送用户文件
- 访问外部网络
- 修改 CI/CD 配置
- 修改权限配置

### 11.3 工具调用审计

每次工具调用都应记录：

```ts
type ToolCallAuditLog = {
  id: string;
  agentId: string;
  taskId: string;
  conversationId: string;
  toolName: string;
  argsHash: string;
  riskLevel: "low" | "medium" | "high";
  approvedBy?: string;
  startedAt: string;
  completedAt?: string;
  status: "success" | "failed" | "blocked";
};
```

---

## 12. 数据模型建议

### 12.1 AgentProfile

```ts
type AgentProfile = {
  id: string;
  name: string;
  avatarUrl?: string;
  description: string;
  tags: string[];

  role:
    | "planner"
    | "context"
    | "explorer"
    | "coder"
    | "diff"
    | "reviewer"
    | "tester"
    | "artifact"
    | "file"
    | "web"
    | "deployer"
    | "document"
    | "ppt"
    | "security"
    | "custom";

  systemPrompt: string;

  modelPreference: {
    provider: "anthropic" | "openai" | "local" | "custom";
    model: string;
    fallbackModels?: string[];
  };

  toolPermissions: string[];

  inputSchema?: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;

  routing: {
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

### 12.2 AgentTaskRun

```ts
type AgentTaskRun = {
  id: string;
  conversationId: string;
  parentTaskId?: string;

  requestedBy: "user" | "orchestrator" | "agent";
  assignedAgentId: string;

  status:
    | "queued"
    | "running"
    | "success"
    | "failed"
    | "blocked"
    | "cancelled";

  input: unknown;
  output?: unknown;

  dependencies: string[];

  contextBundleId?: string;
  artifacts: string[];
  patches: string[];

  startedAt?: string;
  completedAt?: string;

  traceId: string;
};
```

### 12.3 AgentRunOutput

```ts
type AgentRunOutput = {
  agentId: string;
  taskId: string;
  status: "success" | "failed" | "needs_input" | "blocked";
  confidence?: number;

  summary: string;

  artifacts?: string[];
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
    tokensUsed?: number;
    durationMs?: number;
  };
};
```

---

## 13. Agent Adapter 层设计

### 13.1 为什么需要 Adapter

你们要接入 Claude Code、Codex/OpenCode 等不同 Agent 平台。如果业务层直接绑定具体供应商，后续扩展和替换会很困难。

应设计统一 Agent Adapter：

```ts
interface AgentAdapter {
  provider: string;

  run(input: AgentRunInput): Promise<AgentRunOutput>;

  stream?(input: AgentRunInput): AsyncIterable<AgentRunEvent>;

  cancel?(runId: string): Promise<void>;

  getCapabilities(): AgentCapabilities;
}
```

### 13.2 AgentCapabilities

```ts
type AgentCapabilities = {
  supportsStreaming: boolean;
  supportsToolUse: boolean;
  supportsFileRead: boolean;
  supportsFileWrite: boolean;
  supportsShell: boolean;
  supportsSubagents: boolean;
  supportsVision: boolean;
  supportsLongContext: boolean;
};
```

### 13.3 开发建议

- Claude Code 可作为第一个 Code Agent backend。
- Codex/OpenCode 可作为第二个 backend。
- Orchestrator 不应假设所有 backend 都支持同样能力。
- 根据 capabilities 决定是否允许某些任务。
- Adapter 层应统一处理 streaming、取消、错误、超时、日志。

---

## 14. 消息、产物、Diff、部署卡片设计建议

### 14.1 Message 类型

```ts
type Message =
  | TextMessage
  | CodeMessage
  | ImageMessage
  | FileMessage
  | WebPreviewMessage
  | DiffMessage
  | ArtifactMessage
  | DeploymentStatusMessage
  | AgentTaskStatusMessage;

type BaseMessage = {
  id: string;
  conversationId: string;
  senderType: "user" | "agent" | "system";
  senderId: string;
  createdAt: string;
  replyToMessageId?: string;
  quotedMessageId?: string;
  pinned?: boolean;
};
```

### 14.2 DiffMessage

```ts
type DiffMessage = BaseMessage & {
  type: "diff";
  patchId: string;
  summary: string;
  changedFiles: string[];
  riskLevel: "low" | "medium" | "high";
  applyStatus: "pending" | "applied" | "failed" | "rejected";
};
```

### 14.3 DeploymentStatusMessage

```ts
type DeploymentStatusMessage = BaseMessage & {
  type: "deployment_status";
  deploymentId: string;
  status: "queued" | "building" | "deploying" | "success" | "failed";
  previewUrl?: string;
  logsUrl?: string;
  errorMessage?: string;
};
```

### 14.4 ArtifactManifest

```ts
type ArtifactManifest = {
  id: string;
  type:
    | "web_preview"
    | "document"
    | "presentation"
    | "code"
    | "diff"
    | "deployment"
    | "file";

  title: string;

  source: {
    kind:
      | "local_file"
      | "local_dev_server"
      | "remote_url"
      | "generated_content";
    path?: string;
    url?: string;
    contentId?: string;
  };

  actions: string[];

  createdByAgentId: string;
  relatedMessageId?: string;
  relatedPatchId?: string;
  relatedVersionId?: string;
};
```

---

## 15. 开发优先级与里程碑

### 15.1 阶段 1：Runtime 骨架

目标：

- AgentProfile
- AgentTaskRun
- AgentRunOutput
- Orchestrator Runtime
- Agent Adapter
- 基础事件流
- 工具权限模型

子 Agent：

- Planner
- Context
- Code

### 15.2 阶段 2：代码修改闭环

目标：

- Explorer 读代码
- Code Agent 生成 Diff
- Diff Agent 校验 patch
- Review Agent 审查
- Test Agent 执行检查
- 前端展示 Diff 卡片

子 Agent：

- Explorer
- Code
- Diff
- Review
- Test

### 15.3 阶段 3：产物预览

目标：

- Artifact 数据模型
- 网页预览卡片
- 文档预览卡片
- 代码编辑器入口
- Artifact 与消息绑定

子 Agent：

- Artifact
- File

### 15.4 阶段 4：群聊体验

目标：

- 多 Agent 会话
- `@Agent` 指定
- Orchestrator 自动分派
- Agent 依次回复
- 任务状态可视化

子 Agent：

- 所有 MVP Agent

### 15.5 阶段 5：高级能力

目标：

- 用户自建 Agent
- 部署
- 版本历史
- 局部修改
- PPT/文档产物

子 Agent：

- Agent Builder
- Deploy
- Version
- Document
- PPT
- Security

---

## 16. 工程风险与解决方案

### 16.1 风险：Agent 职责混乱

表现：

- Planner 写代码
- Review 直接改代码
- Code Agent 自己部署
- Deploy Agent 修改业务代码

解决：

- 每个 Agent 明确“负责/不负责”。
- 输出 schema 强校验。
- Orchestrator 拒绝不符合契约的输出。
- 工具权限从系统层限制，不只靠 prompt。

### 16.2 风险：上下文污染

表现：

- 子 Agent 看到太多无关历史。
- 历史错误信息影响当前任务。
- 最新用户指令被忽略。

解决：

- Context Agent 生成 context bundle。
- 最新指令优先级最高。
- pin 消息按任务选择注入。
- 每个 Agent run 记录 contextBundleId。

### 16.3 风险：多 Agent 改代码冲突

表现：

- 两个 Agent 改同一文件。
- patch 覆盖。
- Diff 卡片无法应用。

解决：

- 每个 patch 绑定 base version。
- Diff Agent 做冲突检测。
- Version Agent 记录快照。
- 不能安全合并时请求用户选择。

### 16.4 风险：工具权限过大

表现：

- Agent 删除文件。
- Agent 泄露密钥。
- Agent 自动生产部署。
- Agent 执行危险命令。

解决：

- 默认最小权限。
- 高危工具需确认。
- Security Agent + 规则引擎。
- 所有工具调用审计。
- 命令在 sandbox 中执行。

### 16.5 风险：Orchestrator 变成巨型 Prompt

表现：

- 所有逻辑写在一个 prompt 里。
- 难测试。
- 难 debug。
- 稍微改动就不稳定。

解决：

- Orchestrator 用状态机实现。
- LLM 只做必要决策。
- 确定性路由用代码。
- 所有 Agent 输出结构化。

### 16.6 风险：用户体验过于技术化

表现：

- 用户看到大量内部日志。
- Agent 发太多无用状态。
- 聊天流噪音大。

解决：

- 内部日志和用户消息分离。
- 用户只看关键状态。
- 提供“展开执行详情”。
- 默认展示：计划、产物、Diff、错误、下一步。

---

## 17. 评估指标

### 17.1 Agent 质量指标

| 指标 | 含义 |
|---|---|
| task_success_rate | 子任务成功率 |
| patch_apply_rate | Diff 可应用率 |
| review_pass_rate | Review 一次通过率 |
| test_pass_rate | 测试通过率 |
| hallucinated_file_rate | 虚构文件比例 |
| unnecessary_change_rate | 无关改动比例 |
| retry_count | 平均重试次数 |
| user_acceptance_rate | 用户接受率 |

### 17.2 系统体验指标

| 指标 | 含义 |
|---|---|
| time_to_first_agent_event | 用户看到 Agent 开始工作的时间 |
| time_to_first_artifact | 首个产物出现时间 |
| artifact_open_rate | 用户打开预览卡片比例 |
| diff_apply_rate | 用户应用 Diff 比例 |
| deployment_success_rate | 部署成功率 |
| conversation_reuse_rate | 用户复用同一会话比例 |

### 17.3 成本指标

| 指标 | 含义 |
|---|---|
| tokens_per_task | 每个任务 token 消耗 |
| model_cost_per_success | 每次成功任务成本 |
| wasted_runs | 失败/取消的 Agent run |
| context_compression_ratio | 上下文压缩比 |
| cache_hit_rate | 文件摘要、网页摘要缓存命中率 |

---

## 18. 最终建议

### 18.1 MVP 子 Agent 清单

MVP 建议只做以下 8 个：

1. Planner Agent
2. Context Agent
3. Codebase Explorer Agent
4. Code Agent
5. Diff Agent
6. Review Agent
7. Test Agent
8. Artifact Agent

这 8 个 Agent 能形成完整闭环：

```text
需求 → 拆解 → 上下文 → 探索代码 → 写代码 → Diff → 审查/测试 → 预览产物
```

### 18.2 P1 增强

P1 加：

- File Agent
- Web Research Agent
- Agent Builder Agent
- Security Agent
- QA Acceptance Agent

### 18.3 P2 扩展

P2 加：

- Deploy Agent
- Version Agent
- Document Agent
- PPT Agent
- Release Agent

### 18.4 产品落地原则

1. **不要把 UI 功能设计成 Agent。**
2. **不要一开始做太多 Agent。**
3. **先打通代码生产闭环。**
4. **每个 Agent 都要有明确输入/输出。**
5. **工具权限一定要系统层控制。**
6. **Diff、Review、Test 是代码 Agent 产品的生命线。**
7. **Orchestrator 必须可观测、可中断、可重试。**
8. **用户看到聊天，系统内部跑 workflow。**

### 18.5 一句话总结

> 你们真正需要的子 Agent，不是“会话列表 Agent”“消息操作 Agent”这种 UI 拟人化模块，而是围绕任务执行链路设计的专业智能体：Planner、Context、Explorer、Code、Diff、Review、Test、Artifact，再逐步扩展 File、Web、Agent Builder、Security、Deploy、Version、Document、PPT。

---

## 19. 参考资料

以下资料用于本报告的设计参考：

1. Claude Code Docs - Run agents in parallel  
   https://code.claude.com/docs/en/agents

2. Claude Code Docs - Features Overview  
   https://code.claude.com/docs/en/features-overview

3. OpenAI Agents SDK - Agents  
   https://developers.openai.com/api/docs/guides/agents

4. OpenAI Agents SDK - Orchestration and handoffs  
   https://developers.openai.com/api/docs/guides/agents/orchestration

5. OpenAI Agents Python SDK - Agent orchestration  
   https://openai.github.io/openai-agents-python/multi_agent/

6. OpenAI Agents Python SDK - Handoffs  
   https://openai.github.io/openai-agents-python/handoffs/

7. LangChain Docs - Multi-agent  
   https://docs.langchain.com/oss/python/langchain/multi-agent

8. LangChain Docs - Handoffs  
   https://docs.langchain.com/oss/python/langchain/multi-agent/handoffs

9. LangGraph Supervisor Reference  
   https://reference.langchain.com/python/langgraph-supervisor

10. CrewAI Documentation  
    https://docs.crewai.com/

11. CrewAI Introduction  
    https://docs.crewai.com/en/introduction

12. CrewAI Crews  
    https://docs.crewai.com/en/concepts/crews

13. CrewAI Flows  
    https://docs.crewai.com/en/concepts/flows

14. CrewAI Tasks  
    https://docs.crewai.com/en/concepts/tasks

15. Microsoft AutoGen Documentation  
    https://microsoft.github.io/autogen/stable/

16. Microsoft AutoGen Research Publication  
    https://www.microsoft.com/en-us/research/publication/autogen-enabling-next-gen-llm-applications-via-multi-agent-conversation-framework/

17. Microsoft Agent Framework Overview  
    https://learn.microsoft.com/en-us/agent-framework/overview/

18. Microsoft Agent Framework Workflows  
    https://learn.microsoft.com/en-us/agent-framework/workflows/

19. Microsoft Agent Framework Workflow Orchestrations  
    https://learn.microsoft.com/en-us/agent-framework/workflows/orchestrations/

20. Microsoft Agent Framework - Workflow-oriented multi-agent patterns  
    https://learn.microsoft.com/en-us/agents/architecture/multi-agent-workflow-oriented

21. Model Context Protocol Introduction  
    https://modelcontextprotocol.io/docs/getting-started/intro

22. Anthropic - Introducing the Model Context Protocol  
    https://www.anthropic.com/news/model-context-protocol

23. Google Cloud - What is Model Context Protocol  
    https://cloud.google.com/discover/what-is-model-context-protocol

24. A2A Project GitHub  
    https://github.com/a2aproject/A2A/
