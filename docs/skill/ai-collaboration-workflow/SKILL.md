# ai-collaboration-workflow

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目中的 AI 协作开发流程。

它约束 Codex / Claude Code / OpenCode 在开发 AgentHub 时，如何从需求到 Contract、计划、实现、测试、Review、交付摘要逐步推进。

一句话：

**AgentHub 必须先 spec，再 plan，再 build，再 test，再 review；AI 不能跳过 Contract 和 Review 直接生成业务实现。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 让 Codex 生成或修改代码。
- 让 Codex 生成或修改 Contract。
- 让 Codex 做架构变更。
- 让 Codex 做实现计划。
- 让 Codex 做测试。
- 让 Codex 做 review。
- 让 Codex 修 bug。
- 让 Codex 根据 PDR / MVP / UML 生成代码。
- 让 Codex 检查文件是否齐全。
- 让 Codex 检查内容是否符合 Skill。
- 让 Codex 输出变更摘要。
- 记录重要 Prompt。
- 防止 AI 一次性改太多文件。
- 防止 AI 跳过 Contract-first。

---

## 3. 基础原则

AgentHub AI 协作遵守：

```text
Spec first
Plan before build
Contract first
Mock first
Real integration later
Review always
Small changes
No hidden assumptions
```

对应流程：

```text
1. 明确需求和边界
2. 生成或更新 Contract
3. 生成实现计划
4. 分模块实现
5. 补测试
6. 运行 lint / typecheck / go test
7. 输出变更摘要
8. 记录关键 Prompt
9. 人工 Review
```

---

## 4. 四份设计文档解释原则

每次 AI 协作都必须明确参考：

1. **v1.1 Skills 设计规范**：决定有哪些 Skills、如何启用、哪些 Contract 必须先写。
2. **PDR**：决定完整目标架构和长期设计。
3. **MVP 文档**：决定当前 v0.1 实施范围，不要过度实现。
4. **UML 文档**：决定关键流程、时序和模块关系。

解释规则：

```text
PDR 决定最终目标。
MVP 决定当前范围。
UML 决定流程细节。
Skill / Contract 决定实现约束。
```

如果 PDR 与 MVP 范围不一致：

```text
当前实现优先遵守 MVP。
长期设计保留 PDR。
不得用 MVP 简化破坏 PDR 架构方向。
```

---

## 5. AI 协作阶段

### 5.1 Spec 阶段

目标：

```text
明确要做什么、不做什么、属于哪个协议边界。
```

必须回答：

- 这个任务属于哪个模块？
- 是否是 MVP v0.1 范围？
- 需要启用哪些 Skill？
- 会影响哪些 Contract？
- 是否需要修改 docs？
- 是否允许写业务代码？
- 是否有阶段越界风险？
- 是否需要先问用户确认？

禁止：

- 未确认边界就写代码。
- 把 PDR 完整功能当成 MVP 必做。
- 把 Post-MVP 功能提前实现。
- 跳过 Contract 分析。

---

### 5.2 Plan 阶段

目标：

```text
先给实施计划，再改文件。
```

计划必须包含：

- 将修改哪些文件。
- 不会修改哪些文件。
- 是否创建新文件。
- 是否生成业务代码。
- 是否修改 Contract。
- 是否需要测试。
- 风险点。
- 验收标准。

对于复杂任务，Codex 必须先输出计划，等用户确认后再执行。

---

### 5.3 Contract 阶段

目标：

```text
先固定协议和数据结构，再写实现。
```

必须检查：

- REST API 是否需要更新 OpenAPI。
- AG-UI Event 是否需要更新 AG-UI Contract。
- Gateway-Orchestrator 是否需要更新 internal Contract。
- A2A 是否需要更新 AgentCard / Task Contract。
- Artifact 是否需要更新 Artifact Contract。
- Frontend Runtime Skill 是否需要更新 Skill schema。
- Data Model 是否需要更新 data-persistence-contract。
- Security 是否需要更新 security-boundary-contract。

禁止：

- 先改 Go handler 再补 OpenAPI。
- 先改前端类型再补 schema。
- 先改 Agent 输出再补 A2A Contract。
- 先改 Artifact 结构再补 Artifact Contract。

---

### 5.4 Build 阶段

目标：

```text
小步实现，限制修改范围。
```

规则：

- 每次任务尽量只改一个模块或一条链路。
- 不要一次性生成整个系统。
- 只修改用户允许的文件。
- 不要偷偷安装依赖。
- 不要偷偷初始化项目。
- 不要偷偷生成 Docker / DB / LLM 代码。
- 不要把 mock 和真实集成混在一个任务里。
- 输出变更摘要。

---

### 5.5 Test 阶段

目标：

```text
用测试证明实现符合 Contract。
```

按任务类型执行：

- TypeScript：`typecheck`
- React：ESLint / component test
- Go：`go test`
- OpenAPI：schema 校验
- JSON Schema：schema 校验
- AG-UI：event reducer test
- Gateway-Orchestrator：mock event test
- A2A：mock agent test
- Artifact：mapping test
- Security：敏感信息和权限检查

如果无法运行测试，必须说明原因，并给出人工检查清单。

---

### 5.6 Review 阶段

目标：

```text
检查是否符合 Skill / Contract / MVP 范围。
```

Review 必须检查：

- 是否符合当前 Skill。
- 是否修改了不该修改的文件。
- 是否阶段越界。
- 是否破坏 Contract。
- 是否把 REST / AG-UI / A2A 混在一起。
- 是否把 PDR 完整功能误做成 MVP 必做。
- 是否泄漏敏感信息。
- 是否缺少错误处理。
- 是否缺少测试。

---

## 6. 启用 Skills 规则

不要一次性启用所有 Skills。

按任务启用：

### 写 Frontend

```text
project-architecture
code-style-and-conventions
platform-api-contract
agui-event-contract
frontend-runtime-skills-contract
artifact-contract
security-boundary-contract
```

### 写 Gateway

```text
project-architecture
code-style-and-conventions
platform-api-contract
agui-event-contract
gateway-orchestrator-contract
data-persistence-contract
security-boundary-contract
```

### 写 Orchestrator

```text
project-architecture
code-style-and-conventions
gateway-orchestrator-contract
a2a-agent-contract
artifact-contract
intent-orchestration-contract
security-boundary-contract
```

### 写 Child Agent

```text
project-architecture
code-style-and-conventions
a2a-agent-contract
adk-runtime-contract
artifact-contract
security-boundary-contract
```

### 写数据层

```text
data-persistence-contract
platform-api-contract
artifact-contract
security-boundary-contract
```

### 做 Review

```text
testing-review-contract
security-boundary-contract
observability-debugging-contract
code-style-and-conventions
ai-collaboration-workflow
```

---

## 7. Prompt 记录规则

重要 Prompt 必须保留。

建议路径：

```text
docs/ai-prompts/
```

或：

```text
docs/devlog/ai-collaboration-log.md
```

需要记录：

- 生成或修改 Contract 的 Prompt。
- 架构决策 Prompt。
- 大范围代码生成 Prompt。
- 关键修复 Prompt。
- 重要 Review Prompt。
- 改变 MVP / PDR 范围判断的 Prompt。

记录格式：

```markdown
## YYYY-MM-DD 任务标题

### 背景
### 使用的 Skill
### Prompt 摘要
### AI 输出摘要
### 人工 Review 结论
### 后续 TODO
```

禁止记录 API key / token / 用户隐私 / 未脱敏生产日志。

---

## 8. Codex 操作边界

如果用户说“只检查”，Codex 不得修改文件。

如果用户说“只生成文档”，Codex 不得生成业务代码。

如果用户说“不要进入下一个 Skill”，Codex 不得生成下一个 Skill。

如果用户说“不要修改现有文件”，Codex 只能创建用户允许的新文件，或只输出建议。

如果用户说“不要安装依赖”，Codex 不得运行安装命令。

---

## 9. 文件修改报告规则

每次 Codex 修改后必须输出：

```text
1. 修改了哪些文件
2. 创建了哪些文件
3. 删除了哪些文件
4. 每个文件用途
5. 是否修改了用户未授权文件
6. 是否生成业务代码
7. 是否安装依赖
8. 是否运行测试
9. 测试结果
10. 下一步建议
```

如果没有修改文件，也必须明确：

```text
没有修改任何文件。
```

---

## 10. 阶段越界规则

### 10.1 Contract 阶段禁止

在只写 Skill / Contract 时，禁止生成：

- React 业务代码。
- Go Gateway handler。
- Orchestrator 实现。
- A2A Client 实现。
- ADK Runtime 实现。
- Child Agent 实现。
- 数据库模型实现。
- Docker Compose。
- SQL 初始化脚本。
- 真实 LLM 调用。

### 10.2 Mock 阶段禁止

在 Mock 阶段，禁止直接接：

- 真实 LLM。
- 真实部署。
- 复杂多 Agent 编排。
- Agent 市场。
- 自建 Agent。
- 生产级安全系统。

### 10.3 MVP 阶段禁止

除非用户明确要求，MVP v0.1 不做：

- 群聊。
- 自建 Agent。
- web-agent / doc-agent。
- 复杂 Agent Registry。
- 完整 JWT 注册登录。
- 多 Agent parallel / sequential。
- 完整对象存储。
- 完整 Redis 缓存。
- 部署发布。
- 高级 UI 动画。

---

## 11. MVP v0.1 协作规则

MVP v0.1 的目标是跑通：

```text
用户发消息
→ Gateway
→ Orchestrator
→ code-agent
→ A2A 流式响应
→ AG-UI 流式回复
→ code_preview 代码预览
```

AI 生成内容必须优先支持该链路。

MVP v0.1 允许：

- Orchestrator 嵌入 Gateway 进程。
- MySQL 8。
- 固定 Token。
- 配置文件注册 `code-agent`。
- Mock A2A Agent。
- Mock LLM 输出。
- 只实现 `code_preview`。

MVP v0.1 不允许：

- 为了完整 PDR 一次性实现所有功能。
- 为了未来扩展把第一版做复杂。
- 绕过 Contract 快速写死数据结构。
- 把 A2A / AG-UI / REST 混在一起。

---

## 12. AI Review Checklist

在接受 AI 输出前，必须检查：

- 是否符合用户本次授权？
- 是否只改了允许的文件？
- 是否符合 PDR / MVP / UML？
- 是否符合相关 Skill？
- 是否需要先更新 Contract？
- 是否把 REST / AG-UI / A2A 混在一起？
- 是否把 Orchestrator 逻辑写进 Gateway handler？
- 是否把 A2A endpoint 暴露给 Frontend？
- 是否手写了 API response 类型？
- 是否破坏 camelCase JSON？
- 是否泄漏 token / API key？
- 是否引入不必要依赖？
- 是否跳过测试？
- 是否阶段越界？
- 是否输出了清楚的变更摘要？

---

## 13. 硬性规则

1. 先 spec。
2. 再 plan。
3. 再 build。
4. 再 test。
5. 再 review。
6. Contract 变更必须先于实现变更。
7. Mock 必须先于真实集成。
8. 每次 AI 生成代码必须人工 Review。
9. 复杂逻辑先写注释 / 计划，再让 AI 实现。
10. 重要 Prompt 必须保留记录。
11. Codex 必须遵守用户授权范围。
12. “只检查”时不得修改文件。
13. “只生成文档”时不得写业务代码。
14. 不得偷偷安装依赖。
15. 不得偷偷生成 Docker / SQL / LLM 代码。
16. 不得一次性生成整个系统。
17. 不得把 MVP 外功能提前实现。
18. 不得绕过 Skill / Contract。
19. 必须输出修改报告。
20. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 14. 必须维护的文件

本 Skill 本体：

```text
skills/ai-collaboration-workflow/SKILL.md
```

后续可选 references：

```text
skills/ai-collaboration-workflow/references/spec-plan-build-review.md
skills/ai-collaboration-workflow/references/prompt-log-policy.md
skills/ai-collaboration-workflow/references/codex-operation-boundaries.md
```

正式项目文档可选落地：

```text
docs/ai-prompts/
docs/devlog/ai-collaboration-log.md
docs/contracts/ai-collaboration-workflow.md
```

---

## 15. 完成定义

本 Skill 视为完成，当 `SKILL.md` 已经明确：

- spec / plan / build / test / review 流程。
- Contract first。
- Mock first。
- Real integration later。
- Review always。
- Prompt 记录规则。
- Codex 操作边界。
- 文件修改报告规则。
- 阶段越界规则。
- MVP v0.1 协作规则。
- AI Review Checklist。
