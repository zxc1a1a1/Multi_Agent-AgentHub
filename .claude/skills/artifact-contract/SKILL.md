---
name: artifact-contract
description: 当定义、实现、修改或审查 AgentHub 中 A2A Artifact 标准化、AgentHub Artifact schema、content/contentRef、Artifact 生命周期、存储、版本化、ID 关联或 artifact.type 到前端 Runtime Skill 的映射时，使用本 Skill。
---

# artifact-contract

## 1. 目的

本 Skill 定义 AgentHub Artifact 的开发契约。

Artifact 是 AgentHub 中由 Agent 生成的产物事实源。

本 Skill 的核心定位是：

```text
官方 A2A Artifact
→ AgentHub normalized Artifact
→ Artifact identity / storage / versioning
→ artifact.type → frontend runtime skill
→ AG-UI preview notification / Tool Call
```

一句话：

```text
artifact-contract 是 AgentHub 产物事实源契约，不是前端组件契约。
```

## 2. 官方协议边界

涉及上游协议时，必须遵守官方协议边界。

### A2A

A2A Artifact 是 Agent Task 的输出。

A2A Artifact 可由 Parts 组成，Part 可表达文本、文件引用或结构化数据等内容。

AgentHub Artifact 是项目内 normalized object。

不得把 AgentHub MVP 简化格式当成 A2A 官方 Artifact 格式。

### AG-UI

AG-UI Tool Call 事件结构和生命周期由：

```text
agui-event-contract
```

负责。

本 Skill 不定义：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

的事件字段。

本 Skill 只定义：

```text
artifact.type → frontend runtime skill
```

例如：

```text
code → code_preview
```

下面这个映射不属于本 Skill：

```text
code_preview → CodePreview
```

它属于：

```text
frontend-runtime-skills-contract
```

### JSON Schema

项目级 schema 必须使用 JSON Schema draft 2020-12：

```text
https://json-schema.org/draft/2020-12/schema
```

## 3. 文件位置说明

### 项目级 Contract 文件

以下文件属于 AgentHub 项目仓库，是项目级事实源：

```text
<repo-root>/docs/contracts/artifact-schema.md
<repo-root>/docs/contracts/artifact.schema.json
```

### 当前 Skill 的参考文件

以下文件属于当前 Coding Agent Skill：

```text
<current-skill-dir>/references/official-protocol-boundary.md
<current-skill-dir>/references/artifact-registry.md
<current-skill-dir>/references/artifact-schema-policy.md
<current-skill-dir>/references/lifecycle-policy.md
<current-skill-dir>/references/content-storage-policy.md
<current-skill-dir>/references/preview-mapping-policy.md
<current-skill-dir>/references/versioning-policy.md
<current-skill-dir>/references/identity-linking-policy.md
<current-skill-dir>/references/mvp-code-artifact.md
```

如果当前 Skill 安装在 Claude Code 项目目录中，则 `<current-skill-dir>` 通常是：

```text
<repo-root>/.claude/skills/artifact-contract
```

## 4. 适用场景

当进行以下工作时，启用本 Skill：

- 新增 Artifact 类型。
- 修改 Artifact schema。
- 修改 A2A Artifact 到 AgentHub Artifact 的标准化逻辑。
- 修改 ArtifactBuffer。
- 修改 Artifact 生命周期。
- 修改 content / contentRef 规则。
- 修改 Artifact 持久化策略。
- 修改 Artifact 与 conversationId / messageId / runId / a2aTaskId 的关联。
- 修改 Artifact 版本化策略。
- 修改 artifact.type 到 Frontend Runtime Skill 的映射。
- 修改 MVP `code` Artifact。
- 审查大型产物是否错误进入 AG-UI token 流。
- 审查 Artifact 是否丢失 ID 关联。
- 审查 Artifact 是否泄漏私有文件、签名 URL 或敏感信息。

## 5. 长期设计文档约束

长期 Contract Baseline 以 AgentHub 辅助开发 Coding Agent Skills 设计规范为准。

长期链路：

```text
A2A Artifact
→ Orchestrator
→ Gateway
→ AG-UI Tool Call
→ React Preview
→ Database/Object Storage
```

长期要求：

- Artifact 类型统一注册。
- Artifact 必须有 schema。
- Artifact 必须关联 `conversationId / messageId / runId`。
- Artifact 必须可版本化。
- Artifact 必须能映射到 Frontend Runtime Skill。
- AG-UI 不应传输大型 Artifact 内容。
- AG-UI 应只传 `artifactId + summary` 或最小 preview args。
- 完整内容应通过 REST API 查询。
- 大代码包、HTML、图片、文档、zip、日志应进入对象存储。
- Redis 不得作为 Artifact 唯一存储。

## 6. MVP 文档约束

MVP Profile 以 `MVP-AgentHub最小可行产品.md` 为准。

MVP 只要求：

```text
code Artifact
→ code_preview
→ inline small code content
→ message.artifacts JSON 临时保存
```

MVP 允许：

- 只支持 `code` Artifact。
- 小型 code content inline。
- `metadata.language` 作为语言字段。
- `title` 作为 filename。
- `message.artifacts JSON` 临时保存。
- 不要求 Object Storage。
- 不要求独立 Artifact 表。
- completed 后 flush 成 `code_preview` Tool Call。

MVP 不允许：

- 把 inline 规则扩展到所有 Artifact 类型。
- inline 大代码包。
- inline zip。
- inline 图片。
- inline HTML 包。
- inline 私有文件。
- inline 大日志。
- 把 MVP 简化格式当成长期 schema。

## 7. 文档冲突处理规则

当长期设计文档与 MVP 文档存在差异时：

```text
长期 Contract Baseline:
  以 AgentHub 辅助开发 Coding Agent Skills 设计规范为准。

MVP Profile:
  以 MVP-AgentHub最小可行产品.md 为准。

冲突处理:
  MVP 可以作为临时例外，但必须标注为 mvpOnly。
  MVP 例外不得反向污染长期 contract。
```

典型例子：

```text
长期:
  AG-UI 只传 artifactId + summary，大内容经 REST / Object Storage。

MVP:
  small code 可以通过 TOOL_CALL_ARGS inline 传给 code_preview。

处理:
  仅 small code 可 inline，且标记为 MVP exception。
```

## 8. 本 Skill 负责

本 Skill 负责：

- A2A Artifact 到 AgentHub Artifact 的标准化边界。
- Artifact Type Registry。
- AgentHub Artifact 标准结构。
- Artifact metadata 规则。
- content / contentRef 边界。
- Artifact lifecycle。
- Artifact storage policy。
- Artifact versioning。
- Artifact identity linking。
- artifact.type 到 Frontend Runtime Skill 的 preview mapping。
- MVP code Artifact profile。
- 大型 Artifact 不进入 AG-UI token 流的规则。

## 9. 本 Skill 不负责

本 Skill 不负责：

- A2A 官方协议字段定义。
- AG-UI Tool Call 事件字段定义。
- Tool Call args 的前端 schema 校验。
- React Component 映射。
- CodePreview 组件实现。
- 数据库 DDL。
- Object Storage SDK。
- REST API endpoint 细节。
- LLM 如何生成代码。
- 子 Agent handler 具体代码。
- 前端 store 具体实现。
- Docker / CI / 测试策略。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 10. Contract first 规则

任何新增或修改 Artifact 类型、schema、生命周期、contentRef、存储策略、预览映射、版本化或 ID 关联前，必须先更新：

```text
<repo-root>/docs/contracts/artifact-schema.md
<repo-root>/docs/contracts/artifact.schema.json
```

未更新 contract 的实现变更不得接受。

不得先改 converter 或 storage，再补 Artifact contract。

## 11. 使用 references 的规则

修改不同区域时，应先读取对应 reference：

```text
官方协议边界:
  <current-skill-dir>/references/official-protocol-boundary.md

新增 Artifact type:
  <current-skill-dir>/references/artifact-registry.md

修改 Artifact 字段:
  <current-skill-dir>/references/artifact-schema-policy.md

修改 A2A → Artifact → preview 生命周期:
  <current-skill-dir>/references/lifecycle-policy.md

修改 inline / contentRef / Object Storage:
  <current-skill-dir>/references/content-storage-policy.md

修改 artifact.type → frontend runtime skill:
  <current-skill-dir>/references/preview-mapping-policy.md

修改版本化:
  <current-skill-dir>/references/versioning-policy.md

修改 ID 关联:
  <current-skill-dir>/references/identity-linking-policy.md

修改 MVP code Artifact:
  <current-skill-dir>/references/mvp-code-artifact.md
```

## 12. 开发 / 审查 Checklist

开发或审查 Artifact 相关变更时，必须确认：

```text
是否区分了 A2A Artifact 和 AgentHub Artifact？
是否更新了 artifact.schema.json？
artifact.type 是否已注册？
metadata schema 是否明确？
conversationId / messageId / runId 是否存在？
a2aTaskId / stepId / agentName 是否在需要时关联？
content 和 contentRef 是否边界清楚？
大型内容是否避免进入 AG-UI token 流？
preview mapping 是否只写 artifact.type → frontend runtime skill？
是否没有写 React Component 映射？
MVP inline 是否只限 small code？
Object Storage / REST / 权限是否交给相关 contract？
```

## 13. 禁止事项

Coding Agent 不得：

- 把 AgentHub MVP Artifact 简化格式当成 A2A 官方 Artifact 格式。
- 在本 Skill 中重新定义 AG-UI Tool Call 事件字段。
- 在 preview mapping 中写 React Component。
- 把 `artifact.type` 当作 React Component。
- 把 `toolName` 当作 Artifact type。
- 未注册 Artifact type 就持久化或预览。
- Artifact 缺少 `conversationId / messageId / runId`。
- 大型 Artifact 进入 AG-UI token 流。
- 图片、zip、HTML 包、日志、私有文件 inline 进入 Tool Call args。
- 把私有对象存储签名 URL 直接写入 Artifact 给前端。
- Redis 作为 Artifact 唯一存储。
- 重新生成 Artifact 时覆盖旧版本。
- MVP inline 规则扩展到所有 Artifact。
- 把 code Artifact 直接映射到 `CodePreview`。
- 把 code_preview 参数 schema 写进 artifact-contract，除非作为映射说明引用。

## 14. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `adk-runtime-contract`：负责子 Agent Runtime、Task handler、A2A Server 和 Artifact 输出 API。
- `a2a-agent-contract`：负责 A2A Task、Message、Artifact、Streaming 和 AgentCard。
- `agui-event-contract`：负责 AG-UI 事件名称、字段和 Tool Call 生命周期。
- `frontend-runtime-skills-contract`：负责 `code_preview → CodePreview` 等前端 Runtime Skill 注册、参数校验和组件绑定。
- `platform-api-contract`：负责大型 Artifact 内容查询、下载和 REST API。
- `security-boundary-contract`：负责鉴权、下载权限、对象存储签名 URL、sandbox 和敏感信息保护。
- `data-persistence-contract`：负责数据库、对象存储、迁移和事实源。
- `observability-debugging-contract`：负责 artifactId、runId、traceId、a2aTaskId 的日志和调试。
- `testing-review-contract`：负责 Artifact schema、mapping、storage 和 converter 的测试策略。
