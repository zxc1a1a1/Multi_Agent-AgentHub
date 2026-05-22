# Testing Review Contract

版本：v0.1-p1  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP 最小质量门禁 + P1 正式开发演进  
事实源文件：

```text
<repo-root>/docs/contracts/testing-review.md
<repo-root>/docs/contracts/testing-review.schema.json
```

## 1. 目的

本文定义 AgentHub 的项目级测试与代码审查契约。

定位：

```text
AgentHub 的质量门禁与 review 契约
```

覆盖：

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

## 2. 官方 / 外部标准优先级

优先级：

```text
AgentHub 项目 contract
> 官方协议 / 官方测试框架文档
> 项目测试实现细节
```

项目 contract 决定测什么。

官方测试框架文档决定怎么测。

## 3. Contract first 规则

任何新增或修改测试策略、质量门禁、review checklist 或 CI gate 前，必须先更新：

```text
<repo-root>/docs/contracts/testing-review.md
<repo-root>/docs/contracts/testing-review.schema.json
```

如果修改了业务 contract，也必须同步补充对应 contract test。

## 4. 阶段演进规则

### MVP 阶段

MVP 阶段必须保证：

```text
demo path always works
protocol converter has tests
code artifact mapping has tests
message persistence has tests
frontend code_preview path has tests
docker compose smoke passes
```

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

### P1 / 正式开发阶段

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

## 5. MVP 必测路径

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

## 6. 必测协议边界

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

## 7. PR Review

每次 PR 至少检查：

- 是否改了 contract？
- 如果改了实现，contract 是否先更新？
- 是否影响 AG-UI / A2A / Artifact / ExecutionPlan？
- 是否新增或修改 schema？
- 是否补了合法样例和非法样例测试？
- 是否有流式顺序测试？
- 是否有错误路径测试？
- 是否有敏感信息脱敏检查？
- 是否影响 MVP checklist？
- 是否影响 Docker Compose smoke test？

## 8. CI Quality Gates

MVP 最小门禁：

```text
go test ./...
frontend typecheck
frontend lint
unit tests
protocol conversion tests
schema validation tests
secret scan
```

正式开发阶段增加：

```text
contract tests
integration tests
Playwright E2E smoke
security regression tests
dependency vulnerability scan
coverage report
OpenAPI validation
JSON Schema validation
```

## 9. 禁止事项

不得：

- 只靠手动测试。
- 改 contract 不补 contract test。
- 改协议转换不补 conversion test。
- 改 Artifact 映射不补 artifact mapping test。
- 用 sleep 写脆弱 E2E。
- 把真实 LLM 调用作为单元测试依赖。
- 把真实 API key 放进测试。
- 让测试依赖生产外部服务。
- 忽略错误路径。
- 只测 happy path。
- 把 schema validation 当成以后再补。
- 在测试 fixture 中放 secret。
- review 时只看代码风格，不看 contract 和协议边界。
- 在 CI 中跳过失败测试后仍允许合并。
