---
name: testing-review-contract
description: 当定义、实现、修改或审查 AgentHub 中 contract tests、单元测试、集成测试、协议转换测试、前端测试、E2E smoke、安全 review、PR review 或 CI quality gates 时，使用本 Skill。
---

# testing-review-contract

## 1. 目的

本 Skill 定义 AgentHub 的测试与代码审查开发契约。

本 Skill 的定位是：

```text
AgentHub 的质量门禁与 review 契约
```

它覆盖：

```text
Contract tests
Schema validation tests
Unit tests
Integration tests
Protocol conversion tests
Frontend tests
E2E smoke tests
Security review
PR review checklist
CI quality gates
Demo checklist
```

目标：

- 确保 contract-first 被测试和 review 落实。
- 确保 AG-UI、A2A、Artifact、ExecutionPlan、LLM Provider 等协议边界不被破坏。
- 确保 MVP demo 路径稳定可跑。
- 确保正式开发阶段新增功能必须补测试。
- 确保 review 不只看代码风格，也看 contract、协议、安全、错误路径和脱敏。
- 避免只测 happy path。
- 避免测试依赖真实 LLM、真实外部服务或真实 API key。

## 2. 官方 / 外部标准优先级

本 Skill 没有单一官方规范，但测试实现应遵守相关官方或事实标准。

优先级：

```text
AgentHub 项目 contract
> 官方协议 / 官方测试框架文档
> 项目测试实现细节
```

参考方向：

- Go 后端测试优先遵守 Go 官方 `testing` 包。
- 前端 E2E 测试优先遵守 Playwright 官方最佳实践。
- 前端组件测试可参考 Testing Library 的用户视角测试原则。
- 安全测试和安全 review 可参考 OWASP Web Security Testing Guide。
- JSON Schema、A2A、AG-UI、Artifact、ExecutionPlan 等必须以项目 contract 为准。

判断规则：

```text
项目 contract 决定测什么
官方测试框架文档决定怎么测
```

## 3. 文件位置说明

### 项目级 Contract 文件

以下文件属于 AgentHub 项目仓库，是项目级事实源：

```text
<repo-root>/docs/contracts/testing-review.md
<repo-root>/docs/contracts/testing-review.schema.json
```

### 当前 Skill 的参考文件

以下文件属于当前 Coding Agent Skill：

```text
<current-skill-dir>/references/test-strategy.md
<current-skill-dir>/references/contract-test-policy.md
<current-skill-dir>/references/frontend-testing.md
<current-skill-dir>/references/backend-testing.md
<current-skill-dir>/references/protocol-conversion-tests.md
<current-skill-dir>/references/e2e-smoke-tests.md
<current-skill-dir>/references/security-review-checklist.md
<current-skill-dir>/references/pr-review-checklist.md
<current-skill-dir>/references/ci-quality-gates.md
```

如果当前 Skill 安装在 Claude Code 项目目录中，则 `<current-skill-dir>` 通常是：

```text
<repo-root>/.claude/skills/testing-review-contract
```

## 4. 适用场景

当进行以下工作时，启用本 Skill：

- 新增或修改项目 contract。
- 新增或修改 JSON Schema。
- 新增或修改 AG-UI 事件消费。
- 新增或修改 A2A 转 AG-UI converter。
- 新增或修改 Artifact 映射。
- 新增或修改 ExecutionPlan validation。
- 新增或修改 LLM Provider adapter。
- 新增或修改前端 Runtime Skill。
- 新增或修改 DB persistence。
- 新增或修改 Docker Compose / Demo 流程。
- 新增或修改安全边界。
- 编写 PR review checklist。
- 编写 CI quality gates。
- 审查是否遗漏错误路径测试。
- 审查是否泄漏 secret 或依赖真实外部服务。

## 5. 长期契约基线

长期架构中，AgentHub 必须具备以下测试层级：

```text
contract
schema validation
unit
integration
protocol conversion
frontend component
E2E
security
regression
CI quality gate
PR review
```

必须重点测试：

```text
AG-UI event lifecycle
A2A Task / streaming
A2A Artifact → Artifact normalize
Artifact → Frontend Runtime Skill mapping
TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END 顺序
ExecutionPlan schema validation
Agent Registry / AgentCard.skills validation
LLM stream normalization
message persistence
refresh reload history
error handling / fallback
secret redaction
```

## 6. MVP 约束

MVP 阶段不追求完整测试体系，但必须保证主链路稳定。

MVP 最小质量门禁：

```text
manual demo checklist
minimum backend unit tests
minimum protocol conversion tests
minimum frontend component / hook tests
minimum E2E happy path
docker compose smoke test
secret redaction check
```

MVP 必测路径：

```text
用户发送消息
→ Gateway 保存用户消息
→ Orchestrator 直接路由 code-agent
→ A2A sendSubscribe
→ 文本流式返回
→ Artifact buffer
→ completed 后转换成 code_preview Tool Call
→ 前端 CodePreview 展示
→ 刷新后消息仍在
```

MVP 阶段不强制：

- 高覆盖率门槛。
- 完整性能测试。
- 完整安全扫描。
- 完整 contract test matrix。
- 多 Agent 并发测试。
- fallback 测试。
- Object Storage 测试。
- 复杂 CI pipeline。

## 7. 阶段演进规则

### 7.1 MVP 阶段

MVP 阶段必须保证：

```text
demo path always works
protocol converter has tests
code artifact mapping has tests
message persistence has tests
frontend code_preview path has tests
docker compose smoke passes
```

MVP 阶段不得：

- 只靠人工点一点。
- 改 converter 不补测试。
- 改 Artifact 映射不补测试。
- 让测试依赖真实 LLM。
- 让测试依赖真实外部服务。
- 把真实 API key 放进测试。
- 忽略错误路径。

### 7.2 P1 / 正式开发阶段

正式开发阶段逐步补齐：

- contract test suite。
- schema snapshot tests。
- A2A mock server tests。
- AG-UI event replay tests。
- Artifact mapping golden tests。
- ExecutionPlan validation tests。
- LLM provider adapter fake tests。
- Playwright E2E matrix。
- security regression tests。
- review checklist automation。
- CI quality gates。

### 7.3 多 Agent 阶段

多 Agent 阶段必须新增：

- @Agent routing tests。
- parallel orchestration tests。
- sequential dependsOn tests。
- fallback tests。
- partial failure tests。
- aggregation tests。
- traceId / stepId assertion tests。

### 7.4 Artifact 正式化阶段

Artifact 正式化后必须新增：

- artifactId tests。
- contentRef tests。
- large artifact not in AG-UI stream tests。
- object storage mock tests。
- download permission tests。

## 8. 本 Skill 负责

本 Skill 负责：

- 测试分层策略。
- MVP 最小质量门禁。
- Contract test 策略。
- Schema validation test 策略。
- 前端测试策略。
- 后端测试策略。
- 协议转换测试策略。
- E2E smoke 测试策略。
- 安全 review checklist。
- PR review checklist。
- CI quality gates。
- Demo checklist。
- 测试禁止事项。
- mock / fake / fixture / golden file 使用规则。

## 9. 本 Skill 不负责

本 Skill 不负责：

- 具体业务逻辑实现。
- Go / TypeScript 通用代码风格。
- 安全策略完整定义。
- A2A 协议定义。
- AG-UI 事件定义。
- Artifact schema。
- ExecutionPlan schema。
- LLM Provider adapter 细节。
- Docker Compose 交付细节。
- 数据库完整 DDL。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 10. Contract first 规则

任何新增或修改测试策略、质量门禁、review checklist 或 CI gate 前，必须先更新：

```text
<repo-root>/docs/contracts/testing-review.md
<repo-root>/docs/contracts/testing-review.schema.json
```

如果修改了业务 contract，也必须同步补充对应 contract test。

未更新测试或 review 规则的实现变更不得接受。

## 11. 核心规则

### Contract tests

所有 schema 必须有合法样例和非法样例测试。

至少覆盖：

```text
frontend-runtime-skills.schema.json
artifact.schema.json
execution-plan.schema.json
llm-provider.schema.json
observability-debugging.schema.json
testing-review.schema.json
```

### Unit tests

必须优先覆盖纯函数和协议边界：

```text
mapArtifactToSkill
buildToolArgs
ExecutionPlan validation
Agent Registry lookup
Tool Call args parser
LLM stream normalization
```

### Integration tests

必须覆盖服务间边界：

```text
Gateway → Orchestrator
Orchestrator → A2A mock agent
A2A stream → AG-UI stream converter
DB message persistence
AgentCard discovery
```

### E2E tests

MVP happy path：

```text
open page
new conversation
select code-agent
send prompt
see streaming reply
see CodePreview
copy code
refresh page
history still visible
```

### Security review

必须检查：

```text
auth
authorization
input validation
error redaction
secret redaction
API key not in logs
dangerous operations confirmation
```

## 12. 禁止事项

Coding Agent 不得：

- 只靠手动测试。
- 改 contract 不补 contract test。
- 改协议转换不补 conversion test。
- 改 Artifact 映射不补 artifact mapping test。
- 改 ExecutionPlan schema 不补 validation test。
- 用 sleep 写脆弱 E2E。
- 把真实 LLM 调用作为单元测试依赖。
- 把真实 API key 放进测试。
- 让测试依赖生产外部服务。
- 忽略错误路径。
- 只测 happy path。
- 把 schema validation 当成“以后再补”。
- 在测试 fixture 中放 secret。
- review 时只看代码风格，不看 contract 和协议边界。
- 在 CI 中跳过失败测试后仍允许合并。

## 13. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `frontend-runtime-skills-contract`：负责前端 Runtime Skill 注册、参数校验、组件映射和 ToolResult。
- `artifact-contract`：负责 Artifact schema、生命周期、存储和预览映射。
- `intent-orchestration-contract`：负责 ExecutionPlan、Agent 路由、planner validation。
- `adk-runtime-contract`：负责子 Agent Runtime、Task handler、A2A Server。
- `llm-provider-contract`：负责 Provider Adapter、streaming normalization、structured output。
- `observability-debugging-contract`：负责 traceId、runId、日志、错误码和调试。
- `security-boundary-contract`：负责 secret、权限、沙箱和敏感信息保护。
- `docker-compose-delivery`：负责本地交付和 demo 环境。
