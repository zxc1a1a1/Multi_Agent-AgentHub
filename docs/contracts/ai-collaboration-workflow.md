# AI Collaboration Workflow Contract

## 1. 文档目的

本文档定义 AgentHub 中 Codex / Claude Code / OpenCode 的协作开发流程。

核心原则：

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

## 2. 标准流程

```text
1. 明确需求和边界
2. 生成或更新 Contract
3. 生成实施计划
4. 分模块实现
5. 补测试
6. 运行 lint / typecheck / go test
7. 输出变更摘要
8. 记录关键 Prompt
9. 人工 Review
```

## 3. Spec 阶段

必须回答：

- 任务属于哪个模块？
- 是否属于 MVP v0.1 范围？
- 需要启用哪些 Skill？
- 会影响哪些 Contract？
- 是否允许写业务代码？
- 是否存在阶段越界风险？

禁止：

- 未确认边界就写代码。
- 把 PDR 完整功能当成 MVP 必做。
- 把 Post-MVP 功能提前实现。
- 跳过 Contract 分析。

## 4. Plan 阶段

计划必须说明：

- 将修改哪些文件。
- 不会修改哪些文件。
- 是否创建新文件。
- 是否生成业务代码。
- 是否修改 Contract。
- 是否需要测试。
- 风险点。
- 验收标准。

复杂任务必须先输出计划，等用户确认后再执行。

## 5. Contract 阶段

协议和数据结构必须先于实现：

- REST API 变更先更新 OpenAPI。
- AG-UI Event 变更先更新 AG-UI Contract。
- Gateway-Orchestrator 变更先更新 internal Contract。
- A2A 变更先更新 AgentCard / Task Contract。
- Artifact 变更先更新 Artifact Contract。
- Frontend Runtime Skill 变更先更新 Skill schema。
- Data Model 变更先更新 data-persistence-contract。
- Security 变更先更新 security-boundary-contract。

## 6. Build 阶段

规则：

- 每次尽量只改一个模块或一条链路。
- 不要一次性生成整个系统。
- 只修改用户允许的文件。
- 不要偷偷安装依赖。
- 不要把 mock 和真实集成混在一个任务里。
- 必须输出变更摘要。

## 7. Test 阶段

按任务类型执行：

- TypeScript：typecheck。
- React：ESLint / component test。
- Go：go test。
- OpenAPI：schema 校验。
- JSON Schema：schema 校验。
- AG-UI：event reducer test。
- Gateway-Orchestrator：mock event test。
- A2A：mock agent test。
- Artifact：mapping test。
- Security：敏感信息和权限检查。

无法运行测试时，必须说明原因并给出人工检查清单。

## 8. Review 阶段

Review 必须检查：

- 是否符合用户授权范围。
- 是否修改了不该修改的文件。
- 是否阶段越界。
- 是否破坏 Contract。
- 是否把 REST / AG-UI / A2A 混在一起。
- 是否把 Orchestrator 逻辑写进 Gateway handler。
- 是否泄漏 token / API key。
- 是否缺少错误处理。
- 是否缺少测试。

## 9. Prompt 记录

重要 Prompt 建议记录到：

```text
docs/ai-prompts/
```

或：

```text
docs/devlog/ai-collaboration-log.md
```

记录模板：

```markdown
## YYYY-MM-DD 任务标题

### 背景
### 使用的 Skill
### Prompt 摘要
### AI 输出摘要
### 人工 Review 结论
### 后续 TODO
```

不得记录 API key、token、用户隐私、未脱敏生产日志。

## 10. Codex 操作边界

- 用户说“只检查”时，不得修改文件。
- 用户说“只生成文档”时，不得生成业务代码。
- 用户说“不要进入下一个 Skill”时，不得生成下一个 Skill。
- 用户说“不要安装依赖”时，不得运行安装命令。

## 11. 修改报告格式

每次修改后必须输出：

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

## 12. 阶段越界禁止

Contract 阶段禁止生成：

- React 业务代码。
- Go Gateway handler。
- Orchestrator 实现。
- A2A Client 实现。
- ADK Runtime 实现。
- Child Agent 实现。
- 数据库 migration。
- SQL 初始化脚本。
- Docker Compose。
- 真实 LLM 调用。

MVP 阶段默认不做：

- 群聊。
- 自建 Agent。
- web-agent / doc-agent。
- 复杂 Agent Registry。
- 多 Agent parallel / sequential。
- 完整对象存储。
- 完整 Redis 缓存。
- 部署发布。

## 13. 按任务启用 Skills

写 Frontend：

```text
project-architecture
code-style-and-conventions
platform-api-contract
agui-event-contract
frontend-runtime-skills-contract
artifact-contract
security-boundary-contract
```

写 Gateway：

```text
project-architecture
code-style-and-conventions
platform-api-contract
agui-event-contract
gateway-orchestrator-contract
data-persistence-contract
security-boundary-contract
```

写 Orchestrator：

```text
project-architecture
code-style-and-conventions
gateway-orchestrator-contract
a2a-agent-contract
artifact-contract
intent-orchestration-contract
security-boundary-contract
```

写 Child Agent：

```text
project-architecture
code-style-and-conventions
a2a-agent-contract
adk-runtime-contract
artifact-contract
security-boundary-contract
```
